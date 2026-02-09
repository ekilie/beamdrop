package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/tachRoutine/beamdrop-go/pkg/logger"
	"github.com/tachRoutine/beamdrop-go/pkg/storage"
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
		logger.Error("Failed to create buckets directory: %v", err)
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
			sendAPIError(w, "BucketNameRequired", "Bucket name is required", http.StatusBadRequest)
			return
		}
		h.createBucket(w, r, bucketName)
	case http.MethodDelete:
		if bucketName == "" {
			sendAPIError(w, "BucketNameRequired", "Bucket name is required", http.StatusBadRequest)
			return
		}
		h.deleteBucket(w, r, bucketName)
	case http.MethodHead:
		if bucketName == "" {
			sendAPIError(w, "BucketNameRequired", "Bucket name is required", http.StatusBadRequest)
			return
		}
		h.headBucket(w, r, bucketName)
	default:
		sendAPIError(w, "MethodNotAllowed", "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *BucketHandler) listBuckets(w http.ResponseWriter, r *http.Request) {
	buckets, err := h.bucketManager.ListBuckets()
	if err != nil {
		logger.Error("Failed to list buckets: %v", err)
		sendAPIError(w, "InternalError", "Failed to list buckets", http.StatusInternalServerError)
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
			sendAPIError(w, "InvalidBucketName", "Invalid bucket name. Must be 3-63 lowercase alphanumeric characters, hyphens, or dots.", http.StatusBadRequest)
		case storage.ErrBucketExists:
			sendAPIError(w, "BucketAlreadyExists", "Bucket already exists", http.StatusConflict)
		default:
			logger.Error("Failed to create bucket %s: %v", name, err)
			sendAPIError(w, "InternalError", "Failed to create bucket", http.StatusInternalServerError)
		}
		return
	}

	logger.Info("Bucket created: %s", name)
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
			sendAPIError(w, "InvalidBucketName", "Invalid bucket name", http.StatusBadRequest)
		case storage.ErrBucketNotFound:
			sendAPIError(w, "BucketNotFound", "Bucket not found", http.StatusNotFound)
		case storage.ErrBucketNotEmpty:
			sendAPIError(w, "BucketNotEmpty", "Bucket is not empty", http.StatusConflict)
		default:
			logger.Error("Failed to delete bucket %s: %v", name, err)
			sendAPIError(w, "InternalError", "Failed to delete bucket", http.StatusInternalServerError)
		}
		return
	}

	logger.Info("Bucket deleted: %s", name)
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
		sendAPIError(w, "BucketNotFound", "Bucket not found", http.StatusNotFound)
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

func sendAPIError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
