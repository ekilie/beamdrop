package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/tachRoutine/beamdrop-go/pkg/db"
	"github.com/tachRoutine/beamdrop-go/pkg/logger"
)

type FileHandler struct {
	sharedDir string
}

func NewFileHandler(sharedDir string) *FileHandler {
	return &FileHandler{sharedDir: sharedDir}
}

func (h *FileHandler) ListFiles(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Listing files from directory: %s", h.sharedDir)
	w.Header().Set("Content-Type", "application/json")

	reqPath := r.URL.Query().Get("path")
	target, err := ResolvePath(h.sharedDir, reqPath)
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
		http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
		return
	}

	var fileList []File
	for _, f := range files {
		info, err := f.Info()
		if err != nil {
			continue
		}
		filePath := path.Join(reqPath, info.Name())
		fileList = append(fileList, File{
			Name:      info.Name(),
			IsDir:     info.IsDir(),
			Size:      FormatFileSize(info.Size()),
			ModTime:   FormatModTime(info.ModTime().Format(time.RFC3339)),
			Path:      filePath,
			IsStarred: db.IsStarred(filePath),
		})
	}

	json.NewEncoder(w).Encode(fileList)
}

func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("file")
	filePath := h.sharedDir + "/" + filename

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
}

func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	logger.Info("Upload request received")
	file, header, err := r.FormFile("file")
	if err != nil {
		logger.Error("Invalid upload request: %v", err)
		sendJSONError(w, "Invalid upload", http.StatusBadRequest)
		return
	}
	defer file.Close()

	filePath := h.sharedDir + "/" + header.Filename
	logger.Info("Uploading file: %s (size: %s)", header.Filename, FormatFileSize(header.Size))

	out, err := os.Create(filePath)
	if err != nil {
		logger.Error("Failed to create file %s: %v", filePath, err)
		sendJSONError(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		logger.Error("Failed to write file %s: %v", filePath, err)
		sendJSONError(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	logger.Info("File uploaded successfully: %s", header.Filename)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	db.IncrementUploads()
	json.NewEncoder(w).Encode(map[string]string{"message": "Uploaded", "file": header.Filename})
}

// Helper function
func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
