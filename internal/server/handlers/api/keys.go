package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ekilie/beamdrop/internal"
	"github.com/ekilie/beamdrop/pkg/db"
	"github.com/ekilie/beamdrop/pkg/errors"
)

// KeysHandler handles API key management
type KeysHandler struct{}

// NewKeysHandler creates a new keys handler
func NewKeysHandler() *KeysHandler {
	return &KeysHandler{}
}

// Handle routes key management requests
func (h *KeysHandler) Handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listKeys(w, r)
	case http.MethodPost:
		h.createKey(w, r)
	case http.MethodDelete:
		h.deleteKey(w, r)
	default:
		errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
	}
}

func (h *KeysHandler) listKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := db.ListAPIKeys()
	if err != nil {
		slog.Error("Failed to list API keys", "error", err)
		errors.DatabaseError("Failed to list API keys").WithCause(err).WriteHTTPResponse(w)
		return
	}

	// Transform keys for JSON response (exclude sensitive data)
	type keyResponse struct {
		ID          uint       `json:"id"`
		Name        string     `json:"name"`
		AccessKeyID string     `json:"accessKeyId"`
		Permissions string     `json:"permissions,omitempty"`
		BucketScope string     `json:"bucketScope,omitempty"`
		ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
		LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
		CreatedAt   time.Time  `json:"createdAt"`
		Disabled    bool       `json:"disabled"`
	}

	response := make([]keyResponse, len(keys))
	for i, key := range keys {
		response[i] = keyResponse{
			ID:          key.ID,
			Name:        key.Name,
			AccessKeyID: key.AccessKeyID,
			Permissions: key.Permissions,
			BucketScope: key.BucketScope,
			ExpiresAt:   key.ExpiresAt,
			LastUsedAt:  key.LastUsedAt,
			CreatedAt:   key.CreatedAt,
			Disabled:    key.Disabled,
		}
	}

	beam.SendJSON(w, map[string]any{
		"keys":  response,
		"count": len(response),
	}, http.StatusOK)
}

func (h *KeysHandler) createKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Permissions string `json:"permissions,omitempty"`
		BucketScope string `json:"bucketScope,omitempty"`
		ExpiresIn   *int64 `json:"expiresIn,omitempty"` // Duration in seconds
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.InvalidRequest("Invalid request body").WriteHTTPResponse(w)
		return
	}

	if req.Name == "" {
		errors.MissingField("name").WriteHTTPResponse(w)
		return
	}

	permissions, err := normalizePermissions(req.Permissions)
	if err != nil {
		errors.InvalidRequest(err.Error()).WriteHTTPResponse(w)
		return
	}

	var expiresIn *time.Duration
	if req.ExpiresIn != nil {
		d := time.Duration(*req.ExpiresIn) * time.Second
		expiresIn = &d
	}

	apiKey, secretKey, err := db.CreateAPIKey(req.Name, permissions, req.BucketScope, expiresIn)
	if err != nil {
		slog.Error("Failed to create API key", "error", err)
		errors.DatabaseError("Failed to create API key").WithCause(err).WriteHTTPResponse(w)
		return
	}

	slog.Info("API key created", "name", apiKey.Name, "access_key_id", apiKey.AccessKeyID)

	// Return the key with the secret (only shown once!)
	beam.SendJSON(w, map[string]any{
		"id":          apiKey.ID,
		"name":        apiKey.Name,
		"accessKeyId": apiKey.AccessKeyID,
		"secretKey":   secretKey,
		"permissions": apiKey.Permissions,
		"bucketScope": apiKey.BucketScope,
		"expiresAt":   apiKey.ExpiresAt,
		"createdAt":   apiKey.CreatedAt,
		"warning":     "Save the secret key now. It cannot be retrieved later.",
	}, http.StatusCreated)
}

func (h *KeysHandler) deleteKey(w http.ResponseWriter, r *http.Request) {
	accessKeyID := r.URL.Query().Get("accessKeyId")
	if accessKeyID == "" {
		errors.MissingField("accessKeyId").WriteHTTPResponse(w)
		return
	}

	err := db.DeleteAPIKey(accessKeyID)
	if err != nil {
		if err.Error() == "API key not found" {
			errors.New(errors.CodeObjectNotFound, errors.CategoryNotFound, "API key not found", http.StatusNotFound).WriteHTTPResponse(w)
			return
		}
		slog.Error("Failed to delete API key", "error", err)
		errors.DatabaseError("Failed to delete API key").WithCause(err).WriteHTTPResponse(w)
		return
	}

	slog.Info("API key deleted", "access_key_id", accessKeyID)
	w.WriteHeader(http.StatusNoContent)
}

func normalizePermissions(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "read,write", nil
	}

	hasRead := false
	hasWrite := false

	for _, part := range strings.Split(trimmed, ",") {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "read":
			hasRead = true
		case "write":
			hasWrite = true
		case "":
			// Ignore empty entries.
		default:
			return "", fmt.Errorf("permissions must only include read and/or write")
		}
	}

	if !hasRead && !hasWrite {
		return "", fmt.Errorf("permissions must include read and/or write")
	}
	if hasRead && hasWrite {
		return "read,write", nil
	}
	if hasRead {
		return "read", nil
	}
	return "write", nil
}
