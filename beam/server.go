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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		urlPath := r.URL.Path
		if urlPath == "/" {
			urlPath = "/index.html"
		}

		logger.Debug("Serving static file: %s", urlPath)
		file, err := static.FrontendFiles.Open("frontend" + urlPath)
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

	// File APIs
	http.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("Listing files from directory: %s", sharedDir)
		w.Header().Set("Content-Type", "application/json")

		reqPath := r.URL.Query().Get("path")
		target, err := ResolvePath(sharedDir, reqPath)
		if err != nil {
			http.Error(w, `{"error":"invalid path"}`, http.StatusBadRequest)
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
		//TODO: allow multiple file download (zip them first)
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
		logger.Info("Download completed for file: %s", filename)
	})

	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		//TODO: allow multiple file upload
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
		logger.Info("Uploading file: %s (size: %d bytes)", header.Filename, header.Size)

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
		json.NewEncoder(w).Encode(map[string]string{"message": "Uploaded", "file": header.Filename})
	})

	ip := GetLocalIP()
	url := fmt.Sprintf("http://%s:%d", ip, config.GetConfig().PORT)

	if !flags.NoQR {
		qr.ShowQrCode(url)
	}
	logger.Info("Server started at %s sharing directory: %s", url, sharedDir)

	err := http.ListenAndServe(fmt.Sprintf(":%d", config.GetConfig().PORT), nil)
	if err != nil {
		logger.Fatal("Server error: %v", err)
	}
}
