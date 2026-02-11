package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/tachRoutine/beamdrop-go/pkg/db"
	"github.com/tachRoutine/beamdrop-go/pkg/errors"
	"github.com/tachRoutine/beamdrop-go/pkg/logger"
)

type FileOperationsHandler struct {
	sharedDir string
}

func NewFileOperationsHandler(sharedDir string) *FileOperationsHandler {
	return &FileOperationsHandler{sharedDir: sharedDir}
}

func (h *FileOperationsHandler) Move(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
		return
	}

	var req struct {
		SourcePath string `json:"sourcePath"`
		TargetPath string `json:"targetPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Invalid move request: %v", err)
		errors.InvalidRequest("Invalid request body").WriteHTTPResponse(w)
		return
	}

	sourcePath, err := ResolvePath(h.sharedDir, req.SourcePath)
	if err != nil {
		errors.InvalidPath("Invalid source path").WriteHTTPResponse(w)
		return
	}

	targetPath, err := ResolvePath(h.sharedDir, req.TargetPath)
	if err != nil {
		errors.InvalidPath("Invalid target path").WriteHTTPResponse(w)
		return
	}

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		errors.FileNotFound(req.SourcePath).WriteHTTPResponse(w)
		return
	}

	if err := os.Rename(sourcePath, targetPath); err != nil {
		logger.Error("Failed to move file from %s to %s: %v", sourcePath, targetPath, err)
		errors.IOError("Failed to move file").WithCause(err).WriteHTTPResponse(w)
		return
	}

	logger.Info("File moved from %s to %s", req.SourcePath, req.TargetPath)
	sendJSONSuccess(w, map[string]string{
		"message": "File moved successfully",
		"from":    req.SourcePath,
		"to":      req.TargetPath,
	})
}

func (h *FileOperationsHandler) Trash(w http.ResponseWriter, r *http.Request) {
	trashBinPath := filepath.Join(h.sharedDir, ".beamdrop_trash")
	if r.Method != "POST" {
		errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
		return
	}

	var req struct {
		SourcePath string `json:"sourcePath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Invalid trash request: %v", err)
		errors.InvalidRequest("Invalid request body").WriteHTTPResponse(w)
		return
	}

	sourcePath, err := ResolvePath(h.sharedDir, req.SourcePath)
	if err != nil {
		errors.InvalidPath("Invalid source path").WriteHTTPResponse(w)
		return
	}

	targetPath := filepath.Join(trashBinPath, path.Base(req.SourcePath))
	trashRelativePath := path.Join(".beamdrop_trash", path.Base(req.SourcePath))

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		errors.FileNotFound(req.SourcePath).WriteHTTPResponse(w)
		return
	}

	if err := os.Rename(sourcePath, targetPath); err != nil {
		logger.Error("Failed to move file to trash from %s to %s: %v", sourcePath, targetPath, err)
		errors.IOError("Failed to move file to trash").WithCause(err).WriteHTTPResponse(w)
		return
	}

	logger.Info("File moved to trash: %s -> %s", req.SourcePath, trashRelativePath)
	sendJSONSuccess(w, map[string]string{
		"message": "File moved to trash successfully",
		"from":    req.SourcePath,
		"to":      trashRelativePath,
	})
}

func (h *FileOperationsHandler) Copy(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
		return
	}

	var req struct {
		SourcePath string `json:"sourcePath"`
		TargetPath string `json:"targetPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Invalid copy request: %v", err)
		errors.InvalidRequest("Invalid request body").WriteHTTPResponse(w)
		return
	}

	sourcePath, err := ResolvePath(h.sharedDir, req.SourcePath)
	if err != nil {
		errors.InvalidPath("Invalid source path").WriteHTTPResponse(w)
		return
	}

	targetPath, err := ResolvePath(h.sharedDir, req.TargetPath)
	if err != nil {
		errors.InvalidPath("Invalid target path").WriteHTTPResponse(w)
		return
	}

	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		logger.Error("Failed to open source file %s: %v", sourcePath, err)
		errors.FileNotFound(req.SourcePath).WriteHTTPResponse(w)
		return
	}
	defer sourceFile.Close()

	targetFile, err := os.Create(targetPath)
	if err != nil {
		logger.Error("Failed to create target file %s: %v", targetPath, err)
		errors.WriteFailed("Failed to create target file").WithCause(err).WriteHTTPResponse(w)
		return
	}
	defer targetFile.Close()

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		logger.Error("Failed to copy file from %s to %s: %v", sourcePath, targetPath, err)
		errors.IOError("Failed to copy file").WithCause(err).WriteHTTPResponse(w)
		return
	}

	logger.Info("File copied from %s to %s", req.SourcePath, req.TargetPath)
	sendJSONSuccess(w, map[string]string{
		"message": "File copied successfully",
		"from":    req.SourcePath,
		"to":      req.TargetPath,
	})
}

func (h *FileOperationsHandler) Mkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
		return
	}

	var req struct {
		DirPath string `json:"dirPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Invalid mkdir request: %v", err)
		errors.InvalidRequest("Invalid request body").WriteHTTPResponse(w)
		return
	}

	targetPath, err := ResolvePath(h.sharedDir, req.DirPath)
	if err != nil {
		errors.InvalidPath("Invalid directory path").WriteHTTPResponse(w)
		return
	}

	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		errors.FileExists(req.DirPath).WriteHTTPResponse(w)
		return
	}

	if err := os.MkdirAll(targetPath, 0755); err != nil {
		logger.Error("Failed to create directory %s: %v", targetPath, err)
		errors.IOError("Failed to create directory").WithCause(err).WriteHTTPResponse(w)
		return
	}

	logger.Info("Directory created: %s", req.DirPath)
	sendJSONSuccess(w, map[string]string{
		"message": "Directory created successfully",
		"path":    req.DirPath,
	})
}

func (h *FileOperationsHandler) Rename(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
		return
	}

	var req struct {
		OldPath string `json:"oldPath"`
		NewName string `json:"newName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Invalid rename request: %v", err)
		errors.InvalidRequest("Invalid request body").WriteHTTPResponse(w)
		return
	}

	oldPath, err := ResolvePath(h.sharedDir, req.OldPath)
	if err != nil {
		errors.InvalidPath("Invalid old path").WriteHTTPResponse(w)
		return
	}

	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		errors.FileNotFound(req.OldPath).WriteHTTPResponse(w)
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

	newFullPath, err := ResolvePath(h.sharedDir, newPath)
	if err != nil {
		errors.InvalidPath("Invalid new name").WriteHTTPResponse(w)
		return
	}

	if _, err := os.Stat(newFullPath); !os.IsNotExist(err) {
		errors.FileExists(newPath).WriteHTTPResponse(w)
		return
	}

	if err := os.Rename(oldPath, newFullPath); err != nil {
		logger.Error("Failed to rename %s to %s: %v", oldPath, newFullPath, err)
		errors.IOError("Failed to rename").WithCause(err).WriteHTTPResponse(w)
		return
	}

	logger.Info("Renamed %s to %s", req.OldPath, newPath)
	sendJSONSuccess(w, map[string]string{
		"message": "Renamed successfully",
		"oldPath": req.OldPath,
		"newPath": newPath,
	})
}

func (h *FileOperationsHandler) Write(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
		return
	}

	var req struct {
		FilePath string `json:"filePath"`
		Content  string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Invalid write request: %v", err)
		errors.InvalidRequest("Invalid request body").WriteHTTPResponse(w)
		return
	}

	if req.FilePath == "" {
		errors.MissingField("filePath").WriteHTTPResponse(w)
		return
	}

	targetPath, err := ResolvePath(h.sharedDir, req.FilePath)
	if err != nil {
		errors.InvalidPath("Invalid file path").WriteHTTPResponse(w)
		return
	}

	// Create parent directories if they don't exist
	parentDir := path.Dir(targetPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		logger.Error("Failed to create parent directory %s: %v", parentDir, err)
		errors.IOError("Failed to create parent directory").WithCause(err).WriteHTTPResponse(w)
		return
	}

	// Write file content
	if err := os.WriteFile(targetPath, []byte(req.Content), 0644); err != nil {
		logger.Error("Failed to write file %s: %v", targetPath, err)
		errors.WriteFailed("Failed to write file").WithCause(err).WriteHTTPResponse(w)
		return
	}

	logger.Info("File written successfully: %s", req.FilePath)
	sendJSONSuccess(w, map[string]string{
		"message":  "File written successfully",
		"filePath": req.FilePath,
	})
}

func (h *FileOperationsHandler) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		errors.MissingField("q").WriteHTTPResponse(w)
		return
	}

	searchPath := r.URL.Query().Get("path")
	if searchPath == "" {
		searchPath = ""
	}

	targetPath, err := ResolvePath(h.sharedDir, searchPath)
	if err != nil {
		errors.InvalidPath("Invalid search path").WriteHTTPResponse(w)
		return
	}

	var results []File
	err = searchFiles(targetPath, query, searchPath, &results)
	if err != nil {
		logger.Error("Search failed: %v", err)
		errors.InternalError("Search failed").WithCause(err).WriteHTTPResponse(w)
		return
	}

	logger.Info("Search completed for query '%s' in path '%s', found %d results", query, searchPath, len(results))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"query":   query,
		"path":    searchPath,
		"results": results,
		"count":   len(results),
	})
}

func (h *FileOperationsHandler) Star(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
		return
	}

	var req struct {
		FilePath string `json:"filePath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Invalid star request: %v", err)
		errors.InvalidRequest("Invalid request body").WriteHTTPResponse(w)
		return
	}

	target, err := ResolvePath(h.sharedDir, req.FilePath)
	if err != nil {
		errors.InvalidPath("Invalid file path").WriteHTTPResponse(w)
		return
	}

	if _, err := os.Stat(target); os.IsNotExist(err) {
		errors.FileNotFound(req.FilePath).WriteHTTPResponse(w)
		return
	}

	// Toggle star status: if already starred, unstars it; otherwise stars it
	isStarred := db.IsStarred(req.FilePath)
	if isStarred {
		if err := db.UnstarFile(req.FilePath); err != nil {
			logger.Error("Failed to unstar file %s: %v", req.FilePath, err)
			errors.DatabaseError("Failed to unstar file").WithCause(err).WriteHTTPResponse(w)
			return
		}
		logger.Info("File unstarred: %s", req.FilePath)
		sendJSONSuccess(w, map[string]string{"message": "File unstarred", "filePath": req.FilePath, "starred": "false"})
	} else {
		if err := db.StarFile(req.FilePath); err != nil {
			logger.Error("Failed to star file %s: %v", req.FilePath, err)
			errors.DatabaseError("Failed to star file").WithCause(err).WriteHTTPResponse(w)
			return
		}
		logger.Info("File starred: %s", req.FilePath)
		sendJSONSuccess(w, map[string]string{"message": "File starred", "filePath": req.FilePath, "starred": "true"})
	}
}

func (h *FileOperationsHandler) Starred(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
		return
	}

	starredFiles, err := db.GetStarredFiles()
	if err != nil {
		logger.Error("Failed to retrieve starred files: %v", err)
		errors.DatabaseError("Failed to retrieve starred files").WithCause(err).WriteHTTPResponse(w)
		return
	}

	// Convert to a more frontend-friendly format
	result := make([]map[string]string, len(starredFiles))
	for i, sf := range starredFiles {
		result[i] = map[string]string{
			"filePath":  sf.FilePath,
			"createdAt": sf.CreatedAt.Format(time.RFC3339),
		}
	}

	logger.Debug("Retrieved %d starred files", len(starredFiles))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"starred": result,
	})
}

// Helper functions
func sendJSONSuccess(w http.ResponseWriter, data map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

// searchFiles recursively searches for files matching the query in the given directory
func searchFiles(rootPath, query, relativePath string, results *[]File) error {
	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Warn("Error accessing path %s: %v", path, err)
			return nil // Continue searching other files
		}

		// Get relative path from the search root
		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return nil
		}

		// Construct the path relative to the shared directory
		var fullRelPath string
		if relativePath == "" {
			fullRelPath = relPath
		} else {
			fullRelPath = filepath.Join(relativePath, relPath)
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Check if filename contains the search query (case-insensitive)
		if strings.Contains(strings.ToLower(info.Name()), strings.ToLower(query)) {
			filePath := strings.ReplaceAll(fullRelPath, "\\", "/") // Normalize path separators
			file := File{
				Name:      info.Name(),
				IsDir:     info.IsDir(),
				Size:      FormatFileSize(info.Size()),
				ModTime:   FormatModTime(info.ModTime().Format(time.RFC3339)),
				Path:      filePath,
				IsStarred: db.IsStarred(filePath),
			}
			*results = append(*results, file)
		}

		return nil
	})
}
