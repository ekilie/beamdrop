package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ekilie/beamdrop/config"
	"github.com/ekilie/beamdrop/pkg/db"
	"github.com/ekilie/beamdrop/pkg/errors"
	"github.com/ekilie/beamdrop/pkg/metrics"
	"github.com/ekilie/beamdrop/pkg/storage"
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
		if f.Name() == "beamdrop.db" {
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
	filePath, err := ResolvePath(h.sharedDir, filename)
	if err != nil {
		slog.Error("Invalid download path", "file", filename, "error", err)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	info, err := os.Stat(filePath)
	if err != nil {
		slog.Error("Failed to stat path", "path", filePath, "error", err)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	slog.Info("Download request", "file", filename)

	if info.IsDir() {
		n, err := streamDirectoryAsZIP(w, filePath, filename)
		if err != nil {
			slog.Error("Failed to stream ZIP", "path", filePath, "error", err)
			http.Error(w, "Failed to create ZIP", http.StatusInternalServerError)
			return
		}
		recordDownloadMetrics(n)
		slog.Info("ZIP download completed", "file", filename)
		return
	}

	f, err := os.Open(filePath)
	if err != nil {
		slog.Error("Failed to open file", "path", filePath, "error", err)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	slog.Info("Serving download", "file", filename)
	n, _ := io.Copy(w, f)
	recordDownloadMetrics(n)
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
	if err := r.ParseMultipartForm(config.MultipartFormMaxMemory); err != nil {
		slog.Error("Invalid upload request", "error", err)
		if err.Error() == "http: request body too large" {
			errors.FileTooLarge(FormatFileSize(config.MaxUploadSize)).WriteHTTPResponse(w)
			return
		}
		errors.InvalidRequest("Invalid upload").WriteHTTPResponse(w)
		return
	}

	uploadPath := strings.TrimSpace(r.FormValue("path"))
	if uploadPath == "." {
		uploadPath = ""
	}
	uploadDir, err := ResolvePath(h.sharedDir, uploadPath)
	if err != nil {
		errors.InvalidPath("Invalid upload path").WriteHTTPResponse(w)
		return
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		slog.Error("Failed to create upload directory", "path", uploadDir, "error", err)
		errors.IOError("Failed to prepare upload directory").WithCause(err).WriteHTTPResponse(w)
		return
	}

	headers := append(r.MultipartForm.File["file"], r.MultipartForm.File["files"]...)
	if len(headers) == 0 {
		errors.InvalidRequest("Invalid upload").WriteHTTPResponse(w)
		return
	}

	var totalUploaded int64
	firstUploadedFile := ""
	for _, header := range headers {
		// Check file size against limit
		if header.Size > config.MaxUploadSize {
			slog.Error("File size exceeds limit", "size", header.Size, "limit", config.MaxUploadSize)
			errors.FileTooLarge(FormatFileSize(config.MaxUploadSize)).WriteHTTPResponse(w)
			return
		}

		file, err := header.Open()
		if err != nil {
			slog.Error("Failed to open uploaded file", "file", header.Filename, "error", err)
			errors.InvalidRequest("Invalid upload").WriteHTTPResponse(w)
			return
		}

		filename := filepath.Base(header.Filename)
		filePath, err := ResolvePath(uploadDir, filename)
		if err != nil {
			file.Close()
			errors.InvalidPath("Invalid file name").WriteHTTPResponse(w)
			return
		}

		var reader io.Reader = file
		// Validate MIME type if restrictions are configured
		if len(config.AllowedMIMETypes) > 0 {
			// Read the first 512 bytes to detect content type
			buffer := make([]byte, 512)
			n, err := file.Read(buffer)
			if err != nil && err != io.EOF {
				file.Close()
				slog.Error("Failed to read file for MIME detection", "error", err)
				errors.IOError("Failed to process file").WithCause(err).WriteHTTPResponse(w)
				return
			}

			// Detect MIME type
			detectedMIME := http.DetectContentType(buffer[:n])
			slog.Debug("Detected MIME type", "mime", detectedMIME, "file", filename)

			// Extract base MIME type (remove parameters like charset)
			baseMIME := detectedMIME
			if idx := strings.Index(detectedMIME, ";"); idx != -1 {
				baseMIME = strings.TrimSpace(detectedMIME[:idx])
			}

			// Check if MIME type is allowed
			if !slices.Contains(config.AllowedMIMETypes, baseMIME) {
				file.Close()
				slog.Error("File type not allowed", "mime", baseMIME, "file", filename)
				errors.InvalidMIMEType(baseMIME).WriteHTTPResponse(w)
				return
			}

			reader = io.MultiReader(bytes.NewReader(buffer[:n]), file)
		}

		slog.Info("Uploading file", "file", filename, "size", FormatFileSize(header.Size), "path", uploadPath)
		// Use atomic write for crash safety
		n, err := storage.AtomicWriteFile(filePath, reader)
		file.Close()
		if err != nil {
			slog.Error("Failed to write file", "path", filePath, "error", err)
			errors.WriteFailed("Failed to save file").WithCause(err).WriteHTTPResponse(w)
			return
		}

		if firstUploadedFile == "" {
			firstUploadedFile = filename
		}
		totalUploaded += n
		slog.Info("File uploaded successfully", "file", filename, "bytes", n)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	db.IncrementUploads()
	db.AddBytesUploaded(totalUploaded)
	metrics.UploadsTotal.Inc()
	metrics.UploadSizeBytes.Observe(float64(totalUploaded))
	json.NewEncoder(w).Encode(map[string]string{"message": "Uploaded", "file": firstUploadedFile})
}

// Helper function - kept for backward compatibility with file_operations.go
func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	errors.New(errors.CodeInternalError, errors.CategoryInternal, message, statusCode).WriteHTTPResponse(w)
}

type countingWriter struct {
	writer io.Writer
	total  int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.writer.Write(p)
	c.total += int64(n)
	return n, err
}

func streamDirectoryAsZIP(w http.ResponseWriter, dirPath, requestedPath string) (int64, error) {
	zipName := strings.TrimSpace(filepath.Base(filepath.Clean(requestedPath)))
	if zipName == "" || zipName == "." || zipName == string(filepath.Separator) {
		zipName = "folder"
	}
	zipName += ".zip"

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, zipName))

	cw := &countingWriter{writer: w}
	zw := zip.NewWriter(cw)

	err := filepath.WalkDir(dirPath, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(dirPath, current)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		zipPath := filepath.ToSlash(relPath)

		if entry.IsDir() {
			_, err := zw.Create(zipPath + "/")
			return err
		}

		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(fileInfo)
		if err != nil {
			return err
		}
		header.Name = zipPath
		header.Method = zip.Deflate

		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(current)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, file)
		closeErr := file.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		return nil
	})
	if err != nil {
		_ = zw.Close()
		return 0, err
	}

	if err := zw.Close(); err != nil {
		return 0, err
	}

	return cw.total, nil
}

func recordDownloadMetrics(n int64) {
	if db.GetDB() == nil {
		return
	}
	db.IncrementDownloads()
	db.AddBytesDownloaded(n)
	metrics.DownloadsTotal.Inc()
}
