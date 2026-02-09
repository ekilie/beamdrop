package api

import (
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tachRoutine/beamdrop-go/pkg/logger"
	"github.com/tachRoutine/beamdrop-go/pkg/storage"
)

// ObjectHandler handles object operations
type ObjectHandler struct {
	objectManager *storage.ObjectManager
	bucketManager *storage.BucketManager
}

// NewObjectHandler creates a new object handler
func NewObjectHandler(sharedDir string) *ObjectHandler {
	return &ObjectHandler{
		objectManager: storage.NewObjectManager(sharedDir),
		bucketManager: storage.NewBucketManager(sharedDir),
	}
}

// Handle routes object requests based on method
func (h *ObjectHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Extract bucket and key from path: /api/v1/buckets/{bucket}/{key...}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/buckets/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) < 1 || parts[0] == "" {
		sendAPIError(w, "InvalidRequest", "Bucket name is required", http.StatusBadRequest)
		return
	}

	bucket := parts[0]
	key := ""
	if len(parts) > 1 {
		key = parts[1]
	}

	// If no key, this might be a list objects request
	if key == "" {
		if r.Method == http.MethodGet {
			h.listObjects(w, r, bucket)
			return
		}
		sendAPIError(w, "InvalidRequest", "Object key is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getObject(w, r, bucket, key)
	case http.MethodPut:
		h.putObject(w, r, bucket, key)
	case http.MethodPost:
		h.putObjectMultipart(w, r, bucket, key)
	case http.MethodDelete:
		h.deleteObject(w, r, bucket, key)
	case http.MethodHead:
		h.headObject(w, r, bucket, key)
	default:
		sendAPIError(w, "MethodNotAllowed", "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ObjectHandler) getObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	file, info, err := h.objectManager.GetObject(bucket, key)
	if err != nil {
		switch err {
		case storage.ErrBucketNotFound:
			sendAPIError(w, "BucketNotFound", "Bucket not found", http.StatusNotFound)
		case storage.ErrObjectNotFound:
			sendAPIError(w, "KeyNotFound", "Object not found", http.StatusNotFound)
		case storage.ErrInvalidKey:
			sendAPIError(w, "InvalidKey", "Invalid object key", http.StatusBadRequest)
		default:
			logger.Error("Failed to get object %s/%s: %v", bucket, key, err)
			sendAPIError(w, "InternalError", "Failed to retrieve object", http.StatusInternalServerError)
		}
		return
	}
	defer file.Close()

	// Set response headers
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
	w.Header().Set("Last-Modified", info.LastModified.UTC().Format(http.TimeFormat))
	if info.ETag != "" {
		w.Header().Set("ETag", `"`+info.ETag+`"`)
	}

	// Detect content type
	ext := strings.ToLower(filepath.Ext(key))
	contentType := getContentType(ext)
	w.Header().Set("Content-Type", contentType)

	// Handle Range requests for partial content
	http.ServeContent(w, r, key, info.LastModified, file)
}

func (h *ObjectHandler) putObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	// Check bucket exists
	if !h.bucketManager.BucketExists(bucket) {
		sendAPIError(w, "BucketNotFound", "Bucket not found", http.StatusNotFound)
		return
	}

	info, err := h.objectManager.PutObject(bucket, key, r.Body)
	if err != nil {
		switch err {
		case storage.ErrInvalidKey:
			sendAPIError(w, "InvalidKey", "Invalid object key", http.StatusBadRequest)
		default:
			logger.Error("Failed to put object %s/%s: %v", bucket, key, err)
			sendAPIError(w, "InternalError", "Failed to store object", http.StatusInternalServerError)
		}
		return
	}

	logger.Info("Object uploaded: %s/%s (%d bytes)", bucket, key, info.Size)

	w.Header().Set("ETag", `"`+info.ETag+`"`)
	sendJSON(w, map[string]any{
		"bucket": bucket,
		"key":    key,
		"etag":   info.ETag,
		"size":   info.Size,
		"url":    "/api/v1/buckets/" + bucket + "/" + key,
	}, http.StatusOK)
}

func (h *ObjectHandler) putObjectMultipart(w http.ResponseWriter, r *http.Request, bucket, key string) {
	// Check bucket exists
	if !h.bucketManager.BucketExists(bucket) {
		sendAPIError(w, "BucketNotFound", "Bucket not found", http.StatusNotFound)
		return
	}

	// Parse multipart form (max 10GB)
	if err := r.ParseMultipartForm(10 << 30); err != nil {
		sendAPIError(w, "InvalidRequest", "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		sendAPIError(w, "InvalidRequest", "File field required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Use provided key or fall back to uploaded filename
	if key == "" {
		key = header.Filename
	}

	info, err := h.objectManager.PutObject(bucket, key, file)
	if err != nil {
		switch err {
		case storage.ErrInvalidKey:
			sendAPIError(w, "InvalidKey", "Invalid object key", http.StatusBadRequest)
		default:
			logger.Error("Failed to put object %s/%s: %v", bucket, key, err)
			sendAPIError(w, "InternalError", "Failed to store object", http.StatusInternalServerError)
		}
		return
	}

	logger.Info("Object uploaded (multipart): %s/%s (%d bytes)", bucket, key, info.Size)

	w.Header().Set("ETag", `"`+info.ETag+`"`)
	sendJSON(w, map[string]any{
		"bucket": bucket,
		"key":    key,
		"etag":   info.ETag,
		"size":   info.Size,
		"url":    "/api/v1/buckets/" + bucket + "/" + key,
	}, http.StatusOK)
}

func (h *ObjectHandler) deleteObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	err := h.objectManager.DeleteObject(bucket, key)
	if err != nil {
		switch err {
		case storage.ErrBucketNotFound:
			sendAPIError(w, "BucketNotFound", "Bucket not found", http.StatusNotFound)
		case storage.ErrObjectNotFound:
			sendAPIError(w, "KeyNotFound", "Object not found", http.StatusNotFound)
		case storage.ErrInvalidKey:
			sendAPIError(w, "InvalidKey", "Invalid object key", http.StatusBadRequest)
		default:
			logger.Error("Failed to delete object %s/%s: %v", bucket, key, err)
			sendAPIError(w, "InternalError", "Failed to delete object", http.StatusInternalServerError)
		}
		return
	}

	logger.Info("Object deleted: %s/%s", bucket, key)
	w.WriteHeader(http.StatusNoContent)
}

func (h *ObjectHandler) headObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	info, err := h.objectManager.HeadObject(bucket, key)
	if err != nil {
		switch err {
		case storage.ErrBucketNotFound:
			w.WriteHeader(http.StatusNotFound)
		case storage.ErrObjectNotFound:
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
	w.Header().Set("Last-Modified", info.LastModified.UTC().Format(http.TimeFormat))
	if info.ETag != "" {
		w.Header().Set("ETag", `"`+info.ETag+`"`)
	}

	ext := strings.ToLower(filepath.Ext(key))
	contentType := getContentType(ext)
	w.Header().Set("Content-Type", contentType)

	w.WriteHeader(http.StatusOK)
}

func (h *ObjectHandler) listObjects(w http.ResponseWriter, r *http.Request, bucket string) {
	prefix := r.URL.Query().Get("prefix")
	delimiter := r.URL.Query().Get("delimiter")
	maxKeysStr := r.URL.Query().Get("max-keys")

	maxKeys := 1000
	if maxKeysStr != "" {
		if mk, err := strconv.Atoi(maxKeysStr); err == nil && mk > 0 && mk <= 1000 {
			maxKeys = mk
		}
	}

	result, err := h.objectManager.ListObjects(bucket, prefix, delimiter, maxKeys)
	if err != nil {
		switch err {
		case storage.ErrBucketNotFound:
			sendAPIError(w, "BucketNotFound", "Bucket not found", http.StatusNotFound)
		default:
			logger.Error("Failed to list objects in %s: %v", bucket, err)
			sendAPIError(w, "InternalError", "Failed to list objects", http.StatusInternalServerError)
		}
		return
	}

	sendJSON(w, map[string]any{
		"bucket":         bucket,
		"prefix":         result.Prefix,
		"delimiter":      result.Delimiter,
		"maxKeys":        result.MaxKeys,
		"isTruncated":    result.IsTruncated,
		"contents":       result.Contents,
		"commonPrefixes": result.CommonPrefixes,
	}, http.StatusOK)
}

// getContentType returns the MIME type for a file extension
func getContentType(ext string) string {
	types := map[string]string{
		".html":  "text/html",
		".css":   "text/css",
		".js":    "application/javascript",
		".json":  "application/json",
		".xml":   "application/xml",
		".txt":   "text/plain",
		".md":    "text/markdown",
		".pdf":   "application/pdf",
		".zip":   "application/zip",
		".tar":   "application/x-tar",
		".gz":    "application/gzip",
		".png":   "image/png",
		".jpg":   "image/jpeg",
		".jpeg":  "image/jpeg",
		".gif":   "image/gif",
		".svg":   "image/svg+xml",
		".webp":  "image/webp",
		".ico":   "image/x-icon",
		".mp3":   "audio/mpeg",
		".wav":   "audio/wav",
		".mp4":   "video/mp4",
		".webm":  "video/webm",
		".woff":  "font/woff",
		".woff2": "font/woff2",
		".ttf":   "font/ttf",
		".eot":   "application/vnd.ms-fontobject",
	}

	if ct, ok := types[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}

// Ensure io package is used
var _ = io.EOF
