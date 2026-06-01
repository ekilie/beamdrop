package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/ekilie/beamdrop/pkg/db"
	"github.com/ekilie/beamdrop/pkg/errors"
	"github.com/ekilie/beamdrop/pkg/webhooks"
)

// WebhookHandler manages webhook CRUD operations.
type WebhookHandler struct{}

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{}
}

// Handle routes webhook requests based on path and method.
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// /api/v1/webhooks          → list or create
	// /api/v1/webhooks/{id}     → update or delete
	// /api/v1/webhooks/{id}/test       → send test event
	// /api/v1/webhooks/{id}/deliveries → list deliveries

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/webhooks")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		switch r.Method {
		case http.MethodGet:
			h.list(w, r)
		case http.MethodPost:
			h.create(w, r)
		default:
			errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
		}
		return
	}

	// Parse ID from path
	parts := strings.SplitN(path, "/", 2)
	id, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		errors.MissingField("id").WriteHTTPResponse(w)
		return
	}

	subpath := ""
	if len(parts) > 1 {
		subpath = parts[1]
	}

	switch subpath {
	case "test":
		if r.Method != http.MethodPost {
			errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
			return
		}
		h.test(w, r, uint(id))
	case "deliveries":
		if r.Method != http.MethodGet {
			errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
			return
		}
		h.deliveries(w, r, uint(id))
	case "":
		switch r.Method {
		case http.MethodPatch:
			h.update(w, r, uint(id))
		case http.MethodDelete:
			h.delete(w, r, uint(id))
		default:
			errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
		}
	default:
		errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Unknown webhook sub-resource", http.StatusNotFound).WriteHTTPResponse(w)
	}
}

func (h *WebhookHandler) list(w http.ResponseWriter, _ *http.Request) {
	hooks, err := db.ListWebhooks()
	if err != nil {
		errors.ReadFailed("Failed to list webhooks").WithCause(err).WriteHTTPResponse(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"webhooks": hooks,
		"count":    len(hooks),
	})
}

type createWebhookRequest struct {
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	EventTypes  []string `json:"event_types"`
	BucketScope string   `json:"bucket_scope"`
}

func (h *WebhookHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Invalid JSON body", http.StatusBadRequest).WriteHTTPResponse(w)
		return
	}

	if req.URL == "" {
		errors.MissingField("url").WriteHTTPResponse(w)
		return
	}
	if req.Name == "" {
		req.Name = "webhook"
	}

	// Validate URL scheme
	if !strings.HasPrefix(req.URL, "https://") && !strings.HasPrefix(req.URL, "http://") {
		errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "URL must start with https:// or http://", http.StatusBadRequest).WriteHTTPResponse(w)
		return
	}

	// Validate event types
	validEvents := map[string]bool{
		"beamdrop.object.created":  true, "beamdrop.object.updated": true,
		"beamdrop.object.deleted":  true, "beamdrop.bucket.created": true,
		"beamdrop.bucket.deleted":  true, "beamdrop.share.created": true,
		"beamdrop.share.deleted":   true, "beamdrop.presign.created": true,
		"beamdrop.presign.deleted": true,
		"beamdrop.object.*": true, "beamdrop.bucket.*": true,
		"beamdrop.share.*": true, "beamdrop.presign.*": true,
	}
	for _, et := range req.EventTypes {
		if !validEvents[et] {
			errors.New(errors.CodeInvalidRequest, errors.CategoryValidation,
				"Invalid event type: "+et, http.StatusBadRequest).WriteHTTPResponse(w)
			return
		}
	}

	eventTypesJSON, _ := json.Marshal(req.EventTypes)

	wh, secret, err := db.CreateWebhook(req.Name, req.URL, string(eventTypesJSON), req.BucketScope)
	if err != nil {
		errors.WriteFailed("Failed to create webhook").WithCause(err).WriteHTTPResponse(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"webhook": wh,
		"secret":  secret,
		"warning": "Save the secret now — it will not be shown again.",
	})
}

type updateWebhookRequest struct {
	Name        *string  `json:"name,omitempty"`
	URL         *string  `json:"url,omitempty"`
	Enabled     *bool    `json:"enabled,omitempty"`
	EventTypes  []string `json:"event_types,omitempty"`
	BucketScope *string  `json:"bucket_scope,omitempty"`
	RotateSecret bool    `json:"rotate_secret,omitempty"`
}

func (h *WebhookHandler) update(w http.ResponseWriter, r *http.Request, id uint) {
	var req updateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "Invalid JSON body", http.StatusBadRequest).WriteHTTPResponse(w)
		return
	}

	updates := make(map[string]any)
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.URL != nil {
		if !strings.HasPrefix(*req.URL, "https://") && !strings.HasPrefix(*req.URL, "http://") {
			errors.New(errors.CodeInvalidRequest, errors.CategoryValidation, "URL must start with https:// or http://", http.StatusBadRequest).WriteHTTPResponse(w)
			return
		}
		updates["target_url"] = *req.URL
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if len(req.EventTypes) > 0 {
		data, _ := json.Marshal(req.EventTypes)
		updates["event_types"] = string(data)
	}
	if req.BucketScope != nil {
		updates["bucket_scope"] = *req.BucketScope
	}

	if len(updates) > 0 {
		if err := db.UpdateWebhook(id, updates); err != nil {
			errors.WriteFailed("Failed to update webhook").WithCause(err).WriteHTTPResponse(w)
			return
		}
	}

	resp := map[string]any{"updated": true}

	if req.RotateSecret {
		newSecret, err := db.RotateWebhookSecret(id)
		if err != nil {
			errors.WriteFailed("Failed to rotate secret").WithCause(err).WriteHTTPResponse(w)
			return
		}
		resp["secret"] = newSecret
		resp["warning"] = "New secret generated. Save it now — it will not be shown again."
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *WebhookHandler) delete(w http.ResponseWriter, _ *http.Request, id uint) {
	if err := db.DeleteWebhook(id); err != nil {
		errors.WriteFailed("Failed to delete webhook").WithCause(err).WriteHTTPResponse(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WebhookHandler) test(w http.ResponseWriter, _ *http.Request, id uint) {
	wh, err := db.GetWebhook(id)
	if err != nil || wh == nil {
		errors.New(errors.CodeNotFound, errors.CategoryNotFound, "Webhook not found", http.StatusNotFound).WriteHTTPResponse(w)
		return
	}

	// Emit a synthetic test event
	webhooks.Emit("beamdrop.test", "test", "test", "api", map[string]any{
		"webhook_id": id,
		"message":    "This is a test event from Beamdrop",
	})

	slog.Info("Webhook test event emitted", "webhook_id", id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"sent":    true,
		"message": "Test event queued for delivery",
	})
}

func (h *WebhookHandler) deliveries(w http.ResponseWriter, _ *http.Request, id uint) {
	deliveries, err := db.GetDeliveriesForWebhook(id, 50)
	if err != nil {
		errors.ReadFailed("Failed to list deliveries").WithCause(err).WriteHTTPResponse(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"deliveries": deliveries,
		"count":      len(deliveries),
	})
}
