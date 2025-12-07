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
		sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SourcePath string `json:"sourcePath"`
		TargetPath string `json:"targetPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Invalid move request: %v", err)
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	sourcePath, err := ResolvePath(h.sharedDir, req.SourcePath)
	if err != nil {
		sendJSONError(w, "Invalid source path", http.StatusBadRequest)
		return
	}

	targetPath, err := ResolvePath(h.sharedDir, req.TargetPath)
	if err != nil {
		sendJSONError(w, "Invalid target path", http.StatusBadRequest)
		return
	}

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		sendJSONError(w, "Source file not found", http.StatusNotFound)
		return
	}

	if err := os.Rename(sourcePath, targetPath); err != nil {
		logger.Error("Failed to move file from %s to %s: %v", sourcePath, targetPath, err)
		sendJSONError(w, "Failed to move file", http.StatusInternalServerError)
		return
	}

	logger.Info("File moved from %s to %s", req.SourcePath, req.TargetPath)
	sendJSONSuccess(w, map[string]string{
		"message": "File moved successfully",
		"from":    req.SourcePath,
		"to":      req.TargetPath,
	})
}

func (h *FileOperationsHandler) Copy(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SourcePath string `json:"sourcePath"`
		TargetPath string `json:"targetPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Invalid copy request: %v", err)
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	sourcePath, err := ResolvePath(h.sharedDir, req.SourcePath)
	if err != nil {
		sendJSONError(w, "Invalid source path", http.StatusBadRequest)
		return
	}

	targetPath, err := ResolvePath(h.sharedDir, req.TargetPath)
	if err != nil {
		sendJSONError(w, "Invalid target path", http.StatusBadRequest)
		return
	}

	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		logger.Error("Failed to open source file %s: %v", sourcePath, err)
		sendJSONError(w, "Source file not found", http.StatusNotFound)
		return
	}
	defer sourceFile.Close()

	targetFile, err := os.Create(targetPath)
	if err != nil {
		logger.Error("Failed to create target file %s: %v", targetPath, err)
		sendJSONError(w, "Failed to create target file", http.StatusInternalServerError)
		return
	}
	defer targetFile.Close()

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		logger.Error("Failed to copy file from %s to %s: %v", sourcePath, targetPath, err)
		sendJSONError(w, "Failed to copy file", http.StatusInternalServerError)
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
		sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DirPath string `json:"dirPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Invalid mkdir request: %v", err)
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	targetPath, err := ResolvePath(h.sharedDir, req.DirPath)
	if err != nil {
		sendJSONError(w, "Invalid directory path", http.StatusBadRequest)
		return
	}

	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		sendJSONError(w, "Directory already exists", http.StatusConflict)
		return
	}

	if err := os.MkdirAll(targetPath, 0755); err != nil {
		logger.Error("Failed to create directory %s: %v", targetPath, err)
		sendJSONError(w, "Failed to create directory", http.StatusInternalServerError)
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
		sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OldPath string `json:"oldPath"`
		NewName string `json:"newName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Invalid rename request: %v", err)
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	oldPath, err := ResolvePath(h.sharedDir, req.OldPath)
	if err != nil {
		sendJSONError(w, "Invalid old path", http.StatusBadRequest)
		return
	}

	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		sendJSONError(w, "File or directory not found", http.StatusNotFound)
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
		sendJSONError(w, "Invalid new name", http.StatusBadRequest)
		return
	}

	if _, err := os.Stat(newFullPath); !os.IsNotExist(err) {
		sendJSONError(w, "Target name already exists", http.StatusConflict)
		return
	}

	if err := os.Rename(oldPath, newFullPath); err != nil {
		logger.Error("Failed to rename %s to %s: %v", oldPath, newFullPath, err)
		sendJSONError(w, "Failed to rename", http.StatusInternalServerError)
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
		sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FilePath string `json:"filePath"`
		Content  string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Invalid write request: %v", err)
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.FilePath == "" {
		sendJSONError(w, "File path is required", http.StatusBadRequest)
		return
	}

	targetPath, err := ResolvePath(h.sharedDir, req.FilePath)
	if err != nil {
		sendJSONError(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	// Create parent directories if they don't exist
	parentDir := path.Dir(targetPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		logger.Error("Failed to create parent directory %s: %v", parentDir, err)
		sendJSONError(w, "Failed to create parent directory", http.StatusInternalServerError)
		return
	}

	// Write file content
	if err := os.WriteFile(targetPath, []byte(req.Content), 0644); err != nil {
		logger.Error("Failed to write file %s: %v", targetPath, err)
		sendJSONError(w, "Failed to write file", http.StatusInternalServerError)
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
		sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		sendJSONError(w, "Search query is required", http.StatusBadRequest)
		return
	}

	searchPath := r.URL.Query().Get("path")
	if searchPath == "" {
		searchPath = ""
	}

	targetPath, err := ResolvePath(h.sharedDir, searchPath)
	if err != nil {
		sendJSONError(w, "Invalid search path", http.StatusBadRequest)
		return
	}

	var results []File
	err = searchFiles(targetPath, query, searchPath, &results)
	if err != nil {
		logger.Error("Search failed: %v", err)
		sendJSONError(w, "Search failed", http.StatusInternalServerError)
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
		sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FilePath string `json:"filePath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Invalid star request: %v", err)
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	target, err := ResolvePath(h.sharedDir, req.FilePath)
	if err != nil {
		sendJSONError(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	if _, err := os.Stat(target); os.IsNotExist(err) {
		sendJSONError(w, "File not found", http.StatusNotFound)
		return
	}

	// TODO: Implement database storage for starred files
	logger.Info("File starred: %s", req.FilePath)
	sendJSONSuccess(w, map[string]string{"message": "File starred", "filePath": req.FilePath})
}

func (h *FileOperationsHandler) Starred(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement database retrieval of starred files
	starredFiles := []map[string]string{} // Empty list for now

	logger.Debug("Retrieved starred files list")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"starred": starredFiles,
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
			file := File{
				Name:    info.Name(),
				IsDir:   info.IsDir(),
				Size:    FormatFileSize(info.Size()),
				ModTime: FormatModTime(info.ModTime().Format(time.RFC3339)),
				Path:    strings.ReplaceAll(fullRelPath, "\\", "/"), // Normalize path separators
			}
			*results = append(*results, file)
		}

		return nil
	})
}

