package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/tachRoutine/beamdrop-go/config"
	"github.com/tachRoutine/beamdrop-go/pkg/db"
	"github.com/tachRoutine/beamdrop-go/pkg/errors"
	"github.com/tachRoutine/beamdrop-go/pkg/storage"
)

type FileHandler struct {
	sharedDir string
}

func NewFileHandler(sharedDir string) *FileHandler {
	return &FileHandler{sharedDir: sharedDir}
}

func (h *FileHandler) ListFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
		return
	}

	slog.Debug("Listing files from directory", "dir", h.sharedDir)
	w.Header().Set("Content-Type", "application/json")

	reqPath := r.URL.Query().Get("path")
	target, err := ResolvePath(h.sharedDir, reqPath)
	if err != nil {
		errors.InvalidPath("Invalid path").WriteHTTPResponse(w)
		return
	}

	if IsFile(target) {
		http.ServeFile(w, r, target)
		return
	}

	files, err := os.ReadDir(target)
	if err != nil {
		errors.InternalError("Failed to read directory").WithCause(err).WriteHTTPResponse(w)
		return
	}

	fileList := make([]File, 0, len(files))
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
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filename := r.URL.Query().Get("file")
	filePath := h.sharedDir + "/" + filename

	slog.Info("Download request", "file", filename)
	f, err := os.Open(filePath)
	if err != nil {
		slog.Error("Failed to open file", "path", filePath, "error", err)
		http.Error(w, "File not found", 404)
		return
	}
	defer f.Close()

	slog.Info("Serving download", "file", filename)
	io.Copy(w, f)
	db.IncrementDownloads()
	slog.Info("Download completed", "file", filename)
}

func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
		return
	}

	slog.Info("Upload request received")

	// Set max upload size limit on the request body
	r.Body = http.MaxBytesReader(w, r.Body, config.MaxUploadSize)

	file, header, err := r.FormFile("file")
	if err != nil {
		slog.Error("Invalid upload request", "error", err)
		if err.Error() == "http: request body too large" {
			errors.FileTooLarge(FormatFileSize(config.MaxUploadSize)).WriteHTTPResponse(w)
			return
		}
		errors.InvalidRequest("Invalid upload").WriteHTTPResponse(w)
		return
	}
	defer file.Close()

	// Check file size against limit
	if header.Size > config.MaxUploadSize {
		slog.Error("File size exceeds limit", "size", header.Size, "limit", config.MaxUploadSize)
		errors.FileTooLarge(FormatFileSize(config.MaxUploadSize)).WriteHTTPResponse(w)
		return
	}

	// Validate MIME type if restrictions are configured
	if len(config.AllowedMIMETypes) > 0 {
		// Read the first 512 bytes to detect content type
		buffer := make([]byte, 512)
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			slog.Error("Failed to read file for MIME detection", "error", err)
			errors.IOError("Failed to process file").WithCause(err).WriteHTTPResponse(w)
			return
		}

		// Detect MIME type
		detectedMIME := http.DetectContentType(buffer[:n])
		slog.Debug("Detected MIME type", "mime", detectedMIME, "file", header.Filename)

		// Extract base MIME type (remove parameters like charset)
		baseMIME := detectedMIME
		if idx := strings.Index(detectedMIME, ";"); idx != -1 {
			baseMIME = strings.TrimSpace(detectedMIME[:idx])
		}

		// Check if MIME type is allowed
		allowed := slices.Contains(config.AllowedMIMETypes, baseMIME)

		if !allowed {
			slog.Error("File type not allowed", "mime", baseMIME, "file", header.Filename)
			errors.InvalidMIMEType(baseMIME).WriteHTTPResponse(w)
			return
		}

		// Reset file pointer to beginning after reading for MIME detection
		if _, err := file.Seek(0, 0); err != nil {
			slog.Error("Failed to reset file pointer", "error", err)
			errors.IOError("Failed to process file").WithCause(err).WriteHTTPResponse(w)
			return
		}
	}

	filePath := h.sharedDir + "/" + header.Filename
	slog.Info("Uploading file", "file", header.Filename, "size", FormatFileSize(header.Size))

	// Use atomic write for crash safety
	n, err := storage.AtomicWriteFile(filePath, file)
	if err != nil {
		slog.Error("Failed to write file", "path", filePath, "error", err)
		errors.WriteFailed("Failed to save file").WithCause(err).WriteHTTPResponse(w)
		return
	}

	slog.Info("File uploaded successfully", "file", header.Filename, "bytes", n)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	db.IncrementUploads()
	json.NewEncoder(w).Encode(map[string]string{"message": "Uploaded", "file": header.Filename})
}

// Helper function - kept for backward compatibility with file_operations.go
func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	errors.New(errors.CodeInternalError, errors.CategoryInternal, message, statusCode).WriteHTTPResponse(w)
}
