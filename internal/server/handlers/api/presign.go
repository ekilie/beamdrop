package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ekilie/beamdrop/internal"
	"github.com/ekilie/beamdrop/pkg/db"
	"github.com/ekilie/beamdrop/pkg/errors"
	"github.com/ekilie/beamdrop/pkg/storage"
)

type PresignHandler struct {
	objectManager *storage.ObjectManager
}

func NewPresignHandler(sharedDir string) *PresignHandler {
	return &PresignHandler{
		objectManager: storage.NewObjectManager(sharedDir),
	}
}

type CreatePresignRequest struct {
	Bucket       string `json:"bucket"`
	Key          string `json:"key"`
	Method       string `json:"method"`       // GET or PUT, default GET
	ExpiresIn    *int64 `json:"expiresIn"`    // seconds
	MaxDownloads *int   `json:"maxDownloads"` // optional
}

func (h *PresignHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Strip /api/v1/presign to get optional /{token}
	token := strings.TrimPrefix(r.URL.Path, "/api/v1/presign")
	token = strings.TrimPrefix(token, "/")

	switch r.Method {
	case http.MethodPost:
		h.create(w, r)
	case http.MethodGet:
		if token != "" {
			h.getOne(w, r, token)
		} else {
			h.list(w, r)
		}
	case http.MethodDelete:
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token == "" {
			errors.MissingField("token").WriteHTTPResponse(w)
			return
		}
		h.delete(w, r, token)
	default:
		errors.New(errors.CodeInvalidRequest, errors.CategoryValidation,
			"Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
	}
}

func (h *PresignHandler) create(w http.ResponseWriter, r *http.Request) {
	var req CreatePresignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.InvalidRequest("Invalid request body").WriteHTTPResponse(w)
		return
	}

	if req.Bucket == "" {
		errors.MissingField("bucket").WriteHTTPResponse(w)
		return
	}
	if req.Key == "" {
		errors.MissingField("key").WriteHTTPResponse(w)
		return
	}

	method := strings.ToUpper(req.Method)
	if method == "" {
		method = "GET"
	}
	if method != "GET" && method != "PUT" {
		errors.InvalidRequest("Method must be GET or PUT").WriteHTTPResponse(w)
		return
	}

	// Get the access_key_id from the request context (set by API auth middleware)
	createdBy := r.Header.Get("X-Access-Key-Id") // You may need to adjust this

	var expiresIn *time.Duration
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		d := time.Duration(*req.ExpiresIn) * time.Second
		expiresIn = &d
	}

	p, err := db.CreatePresignedURL(req.Bucket, req.Key, method, createdBy, expiresIn, req.MaxDownloads)
	if err != nil {
		slog.Error("Failed to create presigned URL", "error", err)
		errors.InternalError("Failed to create presigned URL").WriteHTTPResponse(w)
		return
	}

	// Build the pretty URL
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	url := scheme + "://" + r.Host + "/dl/" + p.Token

	beam.SendJSON(w, map[string]any{
		"token":        p.Token,
		"url":          url,
		"bucket":       p.Bucket,
		"key":          p.Key,
		"method":       p.Method,
		"expiresAt":    p.ExpiresAt,
		"maxDownloads": p.MaxDownloads,
		"createdAt":    p.CreatedAt,
	}, http.StatusCreated)
}

func (h *PresignHandler) list(w http.ResponseWriter, r *http.Request) {
	urls, err := db.ListPresignedURLs()
	if err != nil {
		errors.InternalError("Failed to list presigned URLs").WriteHTTPResponse(w)
		return
	}
	beam.SendJSON(w, map[string]any{
		"urls":  urls,
		"count": len(urls),
	}, http.StatusOK)
}

func (h *PresignHandler) getOne(w http.ResponseWriter, r *http.Request, token string) {
	p, err := db.GetPresignedURLByToken(token)
	if err != nil || p == nil {
		errors.New(errors.CodeLinkNotFound, errors.CategoryNotFound,
			"Presigned URL not found or expired", http.StatusNotFound).WriteHTTPResponse(w)
		return
	}
	beam.SendJSON(w, p, http.StatusOK)
}

func (h *PresignHandler) delete(w http.ResponseWriter, r *http.Request, token string) {
	if err := db.DeletePresignedURL(token); err != nil {
		errors.New(errors.CodeLinkNotFound, errors.CategoryNotFound,
			err.Error(), http.StatusNotFound).WriteHTTPResponse(w)
		return
	}
	beam.SendJSON(w, map[string]string{"message": "Presigned URL revoked"}, http.StatusOK)
}
