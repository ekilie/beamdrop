package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ekilie/beamdrop/pkg/errors"
	"github.com/ekilie/beamdrop/pkg/storage"
)

// BucketHandler handles bucket operations
type BucketHandler struct {
	bucketManager *storage.BucketManager
}

// NewBucketHandler creates a new bucket handler
func NewBucketHandler(sharedDir string) *BucketHandler {
	bm := storage.NewBucketManager(sharedDir)
	// Ensure buckets directory exists
	if err := bm.EnsureBucketsDir(); err != nil {
		slog.Error("Failed to create buckets directory", "error", err)
	}
	return &BucketHandler{bucketManager: bm}
}

// Handle routes bucket requests based on method
func (h *BucketHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Extract bucket name from path: /api/v1/buckets/{bucket}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/buckets")
	path = strings.TrimPrefix(path, "/")
	bucketName := strings.Split(path, "/")[0]

	switch r.Method {
	case http.MethodGet:
		if bucketName == "" {
			h.listBuckets(w, r)
		} else {
			// This would be handled by object handler for listing objects
			h.getBucketInfo(w, r, bucketName)
		}
	case http.MethodPut:
		if bucketName == "" {
			errors.MissingField("bucket").WriteHTTPResponse(w)
			return
		}
		// if r.Requ
		h.createBucket(w, r, bucketName)
	case http.MethodDelete:
		if bucketName == "" {
			errors.MissingField("bucket").WriteHTTPResponse(w)
			return
		}
		h.deleteBucket(w, r, bucketName)
	case http.MethodHead:
		if bucketName == "" {
			errors.MissingField("bucket").WriteHTTPResponse(w)
			return
		}
		h.headBucket(w, r, bucketName)
	default:
		errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
	}
}

func (h *BucketHandler) listBuckets(w http.ResponseWriter, r *http.Request) {
	buckets, err := h.bucketManager.ListBuckets()
	if err != nil {
		slog.Error("Failed to list buckets", "error", err)
		errors.InternalError("Failed to list buckets").WithCause(err).WriteHTTPResponse(w)
		return
	}

	response := map[string]any{
		"buckets": buckets,
		"count":   len(buckets),
	}

	sendJSON(w, response, http.StatusOK)
}

func (h *BucketHandler) createBucket(w http.ResponseWriter, r *http.Request, name string) {
	err := h.bucketManager.CreateBucket(name)
	if err != nil {
		switch err {
		case storage.ErrInvalidBucketName:
			errors.InvalidBucketName("Invalid bucket name. Must be 3-63 lowercase alphanumeric characters, hyphens, or dots.").WriteHTTPResponse(w)
		case storage.ErrBucketExists:
			errors.BucketExists(name).WriteHTTPResponse(w)
		default:
			slog.Error("Failed to create bucket", "bucket", name, "error", err)
			errors.InternalError("Failed to create bucket").WithCause(err).WriteHTTPResponse(w)
		}
		return
	}

	slog.Info("Bucket created", "bucket", name)
	sendJSON(w, map[string]any{
		"bucket":   name,
		"created":  time.Now().UTC().Format(time.RFC3339),
		"location": "/api/v1/buckets/" + name,
	}, http.StatusCreated)
}

func (h *BucketHandler) createBucketIfNotExist(w http.ResponseWriter, r *http.Request, name string) {
	created, err := h.bucketManager.CreateBucketIfNotExists(name)
	if err != nil {
		switch err {
		case storage.ErrInvalidBucketName:
			errors.InvalidBucketName("Invalid bucket name. Must be 3-63 lowercase alphanumeric characters, hyphens, or dots.").WriteHTTPResponse(w)
		default:
			slog.Error("Failed to create bucket", "bucket", name, "error", err)
			errors.InternalError("Failed to create bucket").WithCause(err).WriteHTTPResponse(w)
		}
		return
	}

	if !created {
		errors.BucketExists(name).WriteHTTPResponse(w)
		return
	}

	slog.Info("Bucket created", "bucket", name)
	sendJSON(w, map[string]any{
		"bucket":   name,
		"created":  time.Now().UTC().Format(time.RFC3339),
		"location": "/api/v1/buckets/" + name,
	}, http.StatusCreated)
}

func (h *BucketHandler) deleteBucket(w http.ResponseWriter, r *http.Request, name string) {
	err := h.bucketManager.DeleteBucket(name)
	if err != nil {
		switch err {
		case storage.ErrInvalidBucketName:
			errors.InvalidBucketName("Invalid bucket name").WriteHTTPResponse(w)
		case storage.ErrBucketNotFound:
			errors.BucketNotFound(name).WriteHTTPResponse(w)
		case storage.ErrBucketNotEmpty:
			errors.BucketNotEmpty(name).WriteHTTPResponse(w)
		default:
			slog.Error("Failed to delete bucket", "bucket", name, "error", err)
			errors.InternalError("Failed to delete bucket").WithCause(err).WriteHTTPResponse(w)
		}
		return
	}

	slog.Info("Bucket deleted", "bucket", name)
	w.WriteHeader(http.StatusNoContent)
}

func (h *BucketHandler) headBucket(w http.ResponseWriter, r *http.Request, name string) {
	if !h.bucketManager.BucketExists(name) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *BucketHandler) getBucketInfo(w http.ResponseWriter, r *http.Request, name string) {
	if !h.bucketManager.BucketExists(name) {
		errors.BucketNotFound(name).WriteHTTPResponse(w)
		return
	}

	sendJSON(w, map[string]any{
		"bucket": name,
		"exists": true,
	}, http.StatusOK)
}

// Helper functions

func sendJSON(w http.ResponseWriter, data any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
