package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/tachRoutine/beamdrop-go/pkg/db"
	"github.com/tachRoutine/beamdrop-go/pkg/errors"
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
		slog.Error("Invalid move request", "error", err)
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
		slog.Error("Failed to move file", "from", sourcePath, "to", targetPath, "error", err)
		errors.IOError("Failed to move file").WithCause(err).WriteHTTPResponse(w)
		return
	}

	slog.Info("File moved", "from", req.SourcePath, "to", req.TargetPath)
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
		slog.Error("Invalid trash request", "error", err)
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
		slog.Error("Failed to move file to trash", "from", sourcePath, "to", targetPath, "error", err)
		errors.IOError("Failed to move file to trash").WithCause(err).WriteHTTPResponse(w)
		return
	}

	slog.Info("File moved to trash", "from", req.SourcePath, "to", trashRelativePath)
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
		slog.Error("Invalid copy request", "error", err)
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
		slog.Error("Failed to open source file", "path", sourcePath, "error", err)
		errors.FileNotFound(req.SourcePath).WriteHTTPResponse(w)
		return
	}
	defer sourceFile.Close()

	targetFile, err := os.Create(targetPath)
	if err != nil {
		slog.Error("Failed to create target file", "path", targetPath, "error", err)
		errors.WriteFailed("Failed to create target file").WithCause(err).WriteHTTPResponse(w)
		return
	}
	defer targetFile.Close()

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		slog.Error("Failed to copy file", "from", sourcePath, "to", targetPath, "error", err)
		errors.IOError("Failed to copy file").WithCause(err).WriteHTTPResponse(w)
		return
	}

	slog.Info("File copied", "from", req.SourcePath, "to", req.TargetPath)
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
		slog.Error("Invalid mkdir request", "error", err)
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
		slog.Error("Failed to create directory", "path", targetPath, "error", err)
		errors.IOError("Failed to create directory").WithCause(err).WriteHTTPResponse(w)
		return
	}

	slog.Info("Directory created", "path", req.DirPath)
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
		slog.Error("Invalid rename request", "error", err)
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
		slog.Error("Failed to rename", "from", oldPath, "to", newFullPath, "error", err)
		errors.IOError("Failed to rename").WithCause(err).WriteHTTPResponse(w)
		return
	}

	slog.Info("Renamed", "from", req.OldPath, "to", newPath)
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
		slog.Error("Invalid write request", "error", err)
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
		slog.Error("Failed to create parent directory", "path", parentDir, "error", err)
		errors.IOError("Failed to create parent directory").WithCause(err).WriteHTTPResponse(w)
		return
	}

	// Write file content
	if err := os.WriteFile(targetPath, []byte(req.Content), 0644); err != nil {
		slog.Error("Failed to write file", "path", targetPath, "error", err)
		errors.WriteFailed("Failed to write file").WithCause(err).WriteHTTPResponse(w)
		return
	}

	slog.Info("File written successfully", "path", req.FilePath)
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
		slog.Error("Search failed", "error", err)
		errors.InternalError("Search failed").WithCause(err).WriteHTTPResponse(w)
		return
	}

	slog.Info("Search completed", "query", query, "path", searchPath, "results", len(results))
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
		slog.Error("Invalid star request", "error", err)
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
			slog.Error("Failed to unstar file", "path", req.FilePath, "error", err)
			errors.DatabaseError("Failed to unstar file").WithCause(err).WriteHTTPResponse(w)
			return
		}
		slog.Info("File unstarred", "path", req.FilePath)
		sendJSONSuccess(w, map[string]string{"message": "File unstarred", "filePath": req.FilePath, "starred": "false"})
	} else {
		if err := db.StarFile(req.FilePath); err != nil {
			slog.Error("Failed to star file", "path", req.FilePath, "error", err)
			errors.DatabaseError("Failed to star file").WithCause(err).WriteHTTPResponse(w)
			return
		}
		slog.Info("File starred", "path", req.FilePath)
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
		slog.Error("Failed to retrieve starred files", "error", err)
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

	slog.Debug("Retrieved starred files", "count", len(starredFiles))
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
			slog.Warn("Error accessing path", "path", path, "error", err)
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
