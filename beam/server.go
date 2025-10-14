package beam

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/tachRoutine/beamdrop-go/config"
	"github.com/tachRoutine/beamdrop-go/pkg/db"
	"github.com/tachRoutine/beamdrop-go/pkg/logger"
	"github.com/tachRoutine/beamdrop-go/pkg/qr"
	"github.com/tachRoutine/beamdrop-go/static"
)

type File struct {
	Name    string `json:"name"`
	Size    string `json:"size"`
	IsDir   bool   `json:"isDir"`
	ModTime string `json:"modTime"`
	Path    string `json:"path"`
}

func StartServer(sharedDir string, flags config.Flags) {
	logger.Info("Initializing HTTP handlers")
	db.AutoMigrate()

	if flags.Password != "" {
		logger.Info("Password is enabled")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		db.IncrementRequests()
		urlPath := r.URL.Path
		if urlPath == "/" {
			urlPath = "/index.html"
		}

		logger.Debug("Serving static file: %s", urlPath)
		file, err := static.FrontendFiles.Open("frontend/dist" + urlPath)
		if err != nil {
			logger.Warn("Static file not found: %s", urlPath)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Not found"})
			return
		}
		defer file.Close()

		ext := strings.ToLower(path.Ext(urlPath))
		if mimeType := mime.TypeByExtension(ext); mimeType != "" {
			w.Header().Set("Content-Type", mimeType)
		} else {
			w.Header().Set("Content-Type", "application/octet-stream")
		}

		io.Copy(w, file)
	})

	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		db.IncrementRequests()

		stats, err := db.GetStats()
		if err != nil {
			logger.Error("Failed to get server stats: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get server stats"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	// File APIs
	http.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
		db.IncrementRequests()

		logger.Debug("Listing files from directory: %s", sharedDir)
		w.Header().Set("Content-Type", "application/json")

		reqPath := r.URL.Query().Get("path")
		target, err := ResolvePath(sharedDir, reqPath)
		if err != nil {
			http.Error(w, `{"error":"invalid path"}`, http.StatusBadRequest)
			return
		}

		if IsFile(target) {
			http.ServeFile(w, r, target)
			return
		}
		files, err := os.ReadDir(target)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError) //TODO: add better error handling
			return
		}

		var fileList []File
		for _, f := range files {
			info, err := f.Info()
			if err != nil {
				continue
			}
			fileList = append(fileList, File{
				Name:    info.Name(),
				IsDir:   info.IsDir(),
				Size:    FormatFileSize(info.Size()),
				ModTime: FormatModTime(info.ModTime().Format(time.RFC3339)),
				Path:    path.Join(reqPath, info.Name()), // relative path for client
			})
		}

		json.NewEncoder(w).Encode(fileList)
	})

	http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		db.IncrementRequests()

		//TODO: allow folder download (zip them first)
		filename := r.URL.Query().Get("file")
		filePath := sharedDir + "/" + filename

		logger.Info("Download request for file: %s", filename)
		f, err := os.Open(filePath)
		if err != nil {
			logger.Error("Failed to open file %s: %v", filePath, err)
			http.Error(w, "File not found", 404)
			return
		}
		defer f.Close()

		logger.Info("Serving download for file: %s", filename)
		io.Copy(w, f)
		db.IncrementDownloads()
		logger.Info("Download completed for file: %s", filename)
	})

	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		db.IncrementRequests()

		logger.Info("Upload request received")
		file, header, err := r.FormFile("file")
		if err != nil {
			logger.Error("Invalid upload request: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid upload"})
			return
		}
		defer file.Close()

		filePath := sharedDir + "/" + header.Filename
		logger.Info("Uploading file: %s (size: %s)", header.Filename, FormatFileSize(header.Size))

		out, err := os.Create(filePath)
		if err != nil {
			logger.Error("Failed to create file %s: %v", filePath, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save file"})
			return
		}
		defer out.Close()

		_, err = io.Copy(out, file)
		if err != nil {
			logger.Error("Failed to write file %s: %v", filePath, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to write file"})
			return
		}

		logger.Info("File uploaded successfully: %s", header.Filename)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		db.IncrementUploads()
		json.NewEncoder(w).Encode(map[string]string{"message": "Uploaded", "file": header.Filename})
	})

	// Star/favorite files endpoint
	http.HandleFunc("/star", func(w http.ResponseWriter, r *http.Request) {
		db.IncrementRequests()

		if r.Method != "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
			return
		}

		var req struct {
			FilePath string `json:"filePath"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("Invalid star request: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		target, err := ResolvePath(sharedDir, req.FilePath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid file path"})
			return
		}

		if _, err := os.Stat(target); os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "File not found"})
			return
		}

		// TODO: Implement database storage for starred files
		logger.Info("File starred: %s", req.FilePath)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "File starred", "filePath": req.FilePath})
	})

	// Get starred files list endpoint
	http.HandleFunc("/starred", func(w http.ResponseWriter, r *http.Request) {
		db.IncrementRequests()

		if r.Method != "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
			return
		}

		// TODO: Implement database retrieval of starred files
		starredFiles := []map[string]string{} // Empty list for now

		logger.Debug("Retrieved starred files list")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"starred": starredFiles,
		})
	})

	// Move/organize files endpoint
	http.HandleFunc("/move", func(w http.ResponseWriter, r *http.Request) {
		db.IncrementRequests()

		if r.Method != "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
			return
		}

		var req struct {
			SourcePath string `json:"sourcePath"`
			TargetPath string `json:"targetPath"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("Invalid move request: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		sourcePath, err := ResolvePath(sharedDir, req.SourcePath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid source path"})
			return
		}

		targetPath, err := ResolvePath(sharedDir, req.TargetPath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid target path"})
			return
		}

		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Source file not found"})
			return
		}

		if err := os.Rename(sourcePath, targetPath); err != nil {
			logger.Error("Failed to move file from %s to %s: %v", sourcePath, targetPath, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to move file"})
			return
		}

		logger.Info("File moved from %s to %s", req.SourcePath, req.TargetPath)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "File moved successfully",
			"from":    req.SourcePath,
			"to":      req.TargetPath,
		})
	})

	// Copy files endpoint
	http.HandleFunc("/copy", func(w http.ResponseWriter, r *http.Request) {
		db.IncrementRequests()

		if r.Method != "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
			return
		}

		var req struct {
			SourcePath string `json:"sourcePath"`
			TargetPath string `json:"targetPath"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("Invalid copy request: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		sourcePath, err := ResolvePath(sharedDir, req.SourcePath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid source path"})
			return
		}

		targetPath, err := ResolvePath(sharedDir, req.TargetPath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid target path"})
			return
		}

		sourceFile, err := os.Open(sourcePath)
		if err != nil {
			logger.Error("Failed to open source file %s: %v", sourcePath, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Source file not found"})
			return
		}
		defer sourceFile.Close()

		targetFile, err := os.Create(targetPath)
		if err != nil {
			logger.Error("Failed to create target file %s: %v", targetPath, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create target file"})
			return
		}
		defer targetFile.Close()

		if _, err := io.Copy(targetFile, sourceFile); err != nil {
			logger.Error("Failed to copy file from %s to %s: %v", sourcePath, targetPath, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to copy file"})
			return
		}

		logger.Info("File copied from %s to %s", req.SourcePath, req.TargetPath)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "File copied successfully",
			"from":    req.SourcePath,
			"to":      req.TargetPath,
		})
	})

	// Create directories endpoint
	http.HandleFunc("/mkdir", func(w http.ResponseWriter, r *http.Request) {
		db.IncrementRequests()

		if r.Method != "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
			return
		}

		var req struct {
			DirPath string `json:"dirPath"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("Invalid mkdir request: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		targetPath, err := ResolvePath(sharedDir, req.DirPath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid directory path"})
			return
		}

		if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "Directory already exists"})
			return
		}

		if err := os.MkdirAll(targetPath, 0755); err != nil {
			logger.Error("Failed to create directory %s: %v", targetPath, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create directory"})
			return
		}

		logger.Info("Directory created: %s", req.DirPath)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Directory created successfully",
			"path":    req.DirPath,
		})
	})

	// Rename files/folders endpoint
	http.HandleFunc("/rename", func(w http.ResponseWriter, r *http.Request) {
		db.IncrementRequests()

		if r.Method != "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
			return
		}

		var req struct {
			OldPath string `json:"oldPath"`
			NewName string `json:"newName"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("Invalid rename request: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		oldPath, err := ResolvePath(sharedDir, req.OldPath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid old path"})
			return
		}

		if _, err := os.Stat(oldPath); os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "File or directory not found"})
			return
		}

		// Get the parent directory and create new path
		parentDir := path.Dir(req.OldPath)
		var newPath string
		if parentDir == "." || parentDir == "" {
			newPath = req.NewName
		} else {
			newPath = path.Join(parentDir, req.NewName)
		}

		newFullPath, err := ResolvePath(sharedDir, newPath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid new name"})
			return
		}

		if _, err := os.Stat(newFullPath); !os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "Target name already exists"})
			return
		}

		if err := os.Rename(oldPath, newFullPath); err != nil {
			logger.Error("Failed to rename %s to %s: %v", oldPath, newFullPath, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to rename"})
			return
		}

		logger.Info("Renamed %s to %s", req.OldPath, newPath)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Renamed successfully",
			"oldPath": req.OldPath,
			"newPath": newPath,
		})
	})

	// Search files by name/content endpoint
	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		db.IncrementRequests()

		if r.Method != "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
			return
		}

		query := r.URL.Query().Get("q")
		if query == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Search query is required"})
			return
		}

		searchPath := r.URL.Query().Get("path")
		if searchPath == "" {
			searchPath = ""
		}

		targetPath, err := ResolvePath(sharedDir, searchPath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid search path"})
			return
		}

		var results []File
		err = searchFiles(targetPath, query, searchPath, &results)
		if err != nil {
			logger.Error("Search failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Search failed"})
			return
		}

		logger.Info("Search completed for query '%s' in path '%s', found %d results", query, searchPath, len(results))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"query":   query,
			"path":    searchPath,
			"results": results,
			"count":   len(results),
		})
	})

	// Find an available port from the default ports list
	port, err := config.FindAvailablePort()
	if err != nil {
		logger.Fatal("Failed to find available port: %v", err)
	}

	ip := GetLocalIP()
	url := fmt.Sprintf("http://%s:%d", ip, port)

	if !flags.NoQR {
		qr.ShowQrCode(url)
	}
	logger.Info("Server started at %s sharing directory: %s", url, sharedDir)

	err = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		logger.Fatal("Server error: %v", err)
	}
}
