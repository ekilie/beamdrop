package api

import (
	stderrors "errors"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ekilie/beamdrop/internal"
	"github.com/ekilie/beamdrop/config"
	"github.com/ekilie/beamdrop/pkg/errors"
	"github.com/ekilie/beamdrop/pkg/reqctx"
	"github.com/ekilie/beamdrop/pkg/storage"
	"github.com/ekilie/beamdrop/pkg/webhooks"
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
		errors.MissingField("bucket").WriteHTTPResponse(w)
		return
	}

	bucket := parts[0]
	key := ""
	if len(parts) > 1 {
		key = parts[1]
	}

	// If no key, this will be a list objects request
	if key == "" {
		if r.Method == http.MethodGet {
			h.listObjects(w, r, bucket)
			return
		}
		errors.MissingField("key").WriteHTTPResponse(w)
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
		errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
	}
}

func (h *ObjectHandler) getObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	file, info, unlock, err := h.objectManager.GetObject(bucket, key)
	if err != nil {
		switch {
		case stderrors.Is(err, storage.ErrBucketNotFound):
			errors.BucketNotFound(bucket).WriteHTTPResponse(w)
		case stderrors.Is(err, storage.ErrObjectNotFound):
			errors.ObjectNotFound(key).WriteHTTPResponse(w)
		case stderrors.Is(err, storage.ErrInvalidKey):
			errors.InvalidObjectKey("Invalid object key").WriteHTTPResponse(w)
		case stderrors.Is(err, storage.ErrLockTimeout):
			errors.ObjectLocked(key).WithCause(err).WriteHTTPResponse(w)
		default:
			slog.Error("Failed to get object", "bucket", bucket, "key", key, "error", err)
			errors.ReadFailed("Failed to retrieve object").WithCause(err).WriteHTTPResponse(w)
		}
		return
	}
	defer file.Close()
	defer unlock()

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
		errors.BucketNotFound(bucket).WriteHTTPResponse(w)
		return
	}

	info, err := h.objectManager.PutObject(bucket, key, r.Body)
	if err != nil {
		switch {
		case stderrors.Is(err, storage.ErrInvalidKey):
			errors.InvalidObjectKey("Invalid object key").WriteHTTPResponse(w)
		case stderrors.Is(err, storage.ErrLockTimeout):
			errors.ObjectLocked(key).WithCause(err).WriteHTTPResponse(w)
		default:
			slog.Error("Failed to put object", "bucket", bucket, "key", key, "error", err)
			errors.WriteFailed("Failed to store object").WithCause(err).WriteHTTPResponse(w)
		}
		return
	}

	slog.Info("Object uploaded", "bucket", bucket, "key", key, "bytes", info.Size)

	actor := reqctx.GetAccessKeyID(r.Context())
	webhooks.EmitObjectCreated(bucket, key, actor, info.Size, info.ETag)

	w.Header().Set("ETag", `"`+info.ETag+`"`)
	beam.SendJSON(w, map[string]any{
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
		errors.BucketNotFound(bucket).WriteHTTPResponse(w)
		return
	}

	// Parse multipart form (max 10GB)
	if err := r.ParseMultipartForm(config.MultipartFormMaxMemory); err != nil {
		errors.InvalidRequest("Failed to parse multipart form").WithCause(err).WriteHTTPResponse(w)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		errors.MissingField("file").WriteHTTPResponse(w)
		return
	}
	defer file.Close()

	// Use provided key or fall back to uploaded filename
	if key == "" {
		key = header.Filename
	}

	info, err := h.objectManager.PutObject(bucket, key, file)
	if err != nil {
		switch {
		case stderrors.Is(err, storage.ErrInvalidKey):
			errors.InvalidObjectKey("Invalid object key").WriteHTTPResponse(w)
		case stderrors.Is(err, storage.ErrLockTimeout):
			errors.ObjectLocked(key).WithCause(err).WriteHTTPResponse(w)
		default:
			slog.Error("Failed to put object", "bucket", bucket, "key", key, "error", err)
			errors.WriteFailed("Failed to store object").WithCause(err).WriteHTTPResponse(w)
		}
		return
	}

	slog.Info("Object uploaded (multipart)", "bucket", bucket, "key", key, "bytes", info.Size)

	actor := reqctx.GetAccessKeyID(r.Context())
	webhooks.EmitObjectCreated(bucket, key, actor, info.Size, info.ETag)

	w.Header().Set("ETag", `"`+info.ETag+`"`)
	beam.SendJSON(w, map[string]any{
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
		switch {
		case stderrors.Is(err, storage.ErrBucketNotFound):
			errors.BucketNotFound(bucket).WriteHTTPResponse(w)
		case stderrors.Is(err, storage.ErrObjectNotFound):
			errors.ObjectNotFound(key).WriteHTTPResponse(w)
		case stderrors.Is(err, storage.ErrInvalidKey):
			errors.InvalidObjectKey("Invalid object key").WriteHTTPResponse(w)
		case stderrors.Is(err, storage.ErrLockTimeout):
			errors.ObjectLocked(key).WithCause(err).WriteHTTPResponse(w)
		default:
			slog.Error("Failed to delete object", "bucket", bucket, "key", key, "error", err)
			errors.New(errors.CodeDeleteFailed, errors.CategoryStorage, "Failed to delete object", http.StatusInternalServerError).WithCause(err).WriteHTTPResponse(w)
		}
		return
	}

	slog.Info("Object deleted", "bucket", bucket, "key", key)

	actor := reqctx.GetAccessKeyID(r.Context())
	webhooks.EmitObjectDeleted(bucket, key, actor)

	w.WriteHeader(http.StatusNoContent)
}

func (h *ObjectHandler) headObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	info, err := h.objectManager.HeadObject(bucket, key)
	if err != nil {
		switch {
		case stderrors.Is(err, storage.ErrBucketNotFound):
			w.WriteHeader(http.StatusNotFound)
		case stderrors.Is(err, storage.ErrObjectNotFound):
			w.WriteHeader(http.StatusNotFound)
		case stderrors.Is(err, storage.ErrLockTimeout):
			w.WriteHeader(http.StatusLocked)
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
			errors.BucketNotFound(bucket).WriteHTTPResponse(w)
		default:
			slog.Error("Failed to list objects", "bucket", bucket, "error", err)
			errors.InternalError("Failed to list objects").WithCause(err).WriteHTTPResponse(w)
		}
		return
	}

	beam.SendJSON(w, map[string]any{
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
		// Text
		".html": "text/html",
		".htm":  "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".mjs":  "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",
		".txt":  "text/plain",
		".md":   "text/markdown",
		".csv":  "text/csv",
		".tsv":  "text/tab-separated-values",
		".yaml": "text/yaml",
		".yml":  "text/yaml",
		".toml": "text/plain",
		".ini":  "text/plain",
		".cfg":  "text/plain",
		".conf": "text/plain",
		".log":  "text/plain",
		".env":  "text/plain",

		// Code / script
		".ts":      "text/typescript",
		".tsx":     "text/typescript",
		".jsx":     "text/javascript",
		".py":      "text/x-python",
		".go":      "text/x-go",
		".rs":      "text/x-rust",
		".java":    "text/x-java",
		".c":       "text/x-c",
		".cpp":     "text/x-c++",
		".h":       "text/x-c",
		".hpp":     "text/x-c++",
		".rb":      "text/x-ruby",
		".php":     "text/x-php",
		".sh":      "application/x-sh",
		".bash":    "application/x-sh",
		".sql":     "application/sql",
		".lua":     "text/x-lua",
		".swift":   "text/x-swift",
		".kt":      "text/x-kotlin",
		".scala":   "text/x-scala",
		".dart":    "text/x-dart",
		".r":       "text/x-r",
		".scss":    "text/x-scss",
		".sass":    "text/x-sass",
		".less":    "text/x-less",
		".vue":     "text/html",
		".graphql": "application/graphql",

		// Documents
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".odt":  "application/vnd.oasis.opendocument.text",
		".ods":  "application/vnd.oasis.opendocument.spreadsheet",
		".rtf":  "application/rtf",

		// Archives
		".zip": "application/zip",
		".tar": "application/x-tar",
		".gz":  "application/gzip",
		".tgz": "application/gzip",
		".bz2": "application/x-bzip2",
		".xz":  "application/x-xz",
		".7z":  "application/x-7z-compressed",
		".rar": "application/vnd.rar",
		".zst": "application/zstd",
		".dmg": "application/x-apple-diskimage",
		".iso": "application/x-iso9660-image",
		".deb": "application/vnd.debian.binary-package",
		".rpm": "application/x-rpm",

		// Images
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
		".webp": "image/webp",
		".ico":  "image/x-icon",
		".bmp":  "image/bmp",
		".tiff": "image/tiff",
		".tif":  "image/tiff",
		".avif": "image/avif",
		".heic": "image/heic",
		".heif": "image/heif",

		// Audio
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".ogg":  "audio/ogg",
		".flac": "audio/flac",
		".aac":  "audio/aac",
		".m4a":  "audio/mp4",
		".opus": "audio/opus",
		".wma":  "audio/x-ms-wma",

		// Video
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".mkv":  "video/x-matroska",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
		".wmv":  "video/x-ms-wmv",
		".flv":  "video/x-flv",
		".m4v":  "video/mp4",
		".3gp":  "video/3gpp",

		// Fonts
		".woff":  "font/woff",
		".woff2": "font/woff2",
		".ttf":   "font/ttf",
		".otf":   "font/otf",
		".eot":   "application/vnd.ms-fontobject",

		// Binary / misc
		".wasm": "application/wasm",
		".bin":  "application/octet-stream",
		".exe":  "application/vnd.microsoft.portable-executable",
		".apk":  "application/vnd.android.package-archive",
		".ipa":  "application/octet-stream",
	}

	if ct, ok := types[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}
