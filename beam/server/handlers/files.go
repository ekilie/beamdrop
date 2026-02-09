package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/tachRoutine/beamdrop-go/config"
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
	
	// Set max upload size limit on the request body
	r.Body = http.MaxBytesReader(w, r.Body, config.MaxUploadSize)
	
	file, header, err := r.FormFile("file")
	if err != nil {
		logger.Error("Invalid upload request: %v", err)
		if err.Error() == "http: request body too large" {
			sendJSONError(w, fmt.Sprintf("File too large. Maximum size is %s", FormatFileSize(config.MaxUploadSize)), http.StatusRequestEntityTooLarge)
			return
		}
		sendJSONError(w, "Invalid upload", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check file size against limit
	if header.Size > config.MaxUploadSize {
		logger.Error("File size %d exceeds limit %d", header.Size, config.MaxUploadSize)
		sendJSONError(w, fmt.Sprintf("File size exceeds maximum allowed size of %s", FormatFileSize(config.MaxUploadSize)), http.StatusRequestEntityTooLarge)
		return
	}

	// Validate MIME type if restrictions are configured
	if len(config.AllowedMIMETypes) > 0 {
		// Read the first 512 bytes to detect content type
		buffer := make([]byte, 512)
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			logger.Error("Failed to read file for MIME detection: %v", err)
			sendJSONError(w, "Failed to process file", http.StatusInternalServerError)
			return
		}

		// Detect MIME type
		detectedMIME := http.DetectContentType(buffer[:n])
		logger.Debug("Detected MIME type: %s for file: %s", detectedMIME, header.Filename)

		// Extract base MIME type (remove parameters like charset)
		baseMIME := detectedMIME
		if idx := strings.Index(detectedMIME, ";"); idx != -1 {
			baseMIME = strings.TrimSpace(detectedMIME[:idx])
		}

		// Check if MIME type is allowed
		allowed := false
		for _, allowedType := range config.AllowedMIMETypes {
			if baseMIME == allowedType {
				allowed = true
				break
			}
		}

		if !allowed {
			logger.Error("File type %s not allowed for file: %s", baseMIME, header.Filename)
			sendJSONError(w, fmt.Sprintf("File type '%s' is not allowed", baseMIME), http.StatusUnsupportedMediaType)
			return
		}

		// Reset file pointer to beginning after reading for MIME detection
		if _, err := file.Seek(0, 0); err != nil {
			logger.Error("Failed to reset file pointer: %v", err)
			sendJSONError(w, "Failed to process file", http.StatusInternalServerError)
			return
		}
	}

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
