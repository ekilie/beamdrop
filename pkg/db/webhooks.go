package db

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"time"

	"github.com/ekilie/beamdrop/pkg/crypto"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Webhook represents a registered webhook destination.
type Webhook struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	Name            string     `gorm:"size:255;not null" json:"name"`
	TargetURL       string     `gorm:"column:target_url;size:2048;not null" json:"targetUrl"`
	SecretEncrypted string     `gorm:"column:secret_encrypted;size:512;not null" json:"-"`
	Enabled         bool       `gorm:"default:true" json:"enabled"`
	EventTypes      string     `gorm:"column:event_types;type:text;not null" json:"eventTypes"` // JSON array
	BucketScope     string     `gorm:"column:bucket_scope;size:255" json:"bucketScope,omitempty"`
	CreatedAt       time.Time  `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updatedAt"`
	LastDeliveryAt  *time.Time `gorm:"column:last_delivery_at" json:"lastDeliveryAt,omitempty"`
	LastError       string     `gorm:"column:last_error;size:1024" json:"lastError,omitempty"`
}

func (Webhook) TableName() string { return "webhooks" }

// WebhookEvent represents a recorded event.
type WebhookEvent struct {
	ID           string    `gorm:"primaryKey;size:36" json:"id"` // UUID
	EventType    string    `gorm:"column:event_type;size:64;not null;index" json:"eventType"`
	ResourceType string    `gorm:"column:resource_type;size:32;not null" json:"resourceType"` // bucket, object, share, presign
	ResourcePath string    `gorm:"column:resource_path;size:1280" json:"resourcePath"`        // bucket/key
	Actor        string    `gorm:"column:actor;size:255" json:"actor"`                        // access_key_id or "session"
	PayloadJSON  string    `gorm:"column:payload_json;type:text" json:"payloadJson"`
	CreatedAt    time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP;index" json:"createdAt"`
}

func (WebhookEvent) TableName() string { return "webhook_events" }

// WebhookDelivery tracks delivery attempts for an event to a webhook.
type WebhookDelivery struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	WebhookID      uint       `gorm:"column:webhook_id;not null;index:idx_wh_evt,unique" json:"webhookId"`
	EventID        string     `gorm:"column:event_id;size:36;not null;index:idx_wh_evt,unique" json:"eventId"`
	Status         string     `gorm:"column:status;size:20;not null;default:pending;index" json:"status"` // pending, delivering, delivered, failed, dead_letter
	AttemptCount   int        `gorm:"column:attempt_count;default:0" json:"attemptCount"`
	NextAttemptAt  *time.Time `gorm:"column:next_attempt_at;index" json:"nextAttemptAt,omitempty"`
	LastHTTPStatus int        `gorm:"column:last_http_status" json:"lastHttpStatus,omitempty"`
	LastError      string     `gorm:"column:last_error;size:1024" json:"lastError,omitempty"`
	LastDurationMs int        `gorm:"column:last_duration_ms" json:"lastDurationMs,omitempty"`
	CreatedAt      time.Time  `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updatedAt"`
	DeliveredAt    *time.Time `gorm:"column:delivered_at" json:"deliveredAt,omitempty"`
}

func (WebhookDelivery) TableName() string { return "webhook_deliveries" }

// Webhook delivery status constants
const (
	DeliveryPending    = "pending"
	DeliveryDelivering = "delivering"
	DeliveryDelivered  = "delivered"
	DeliveryFailed     = "failed"
	DeliveryDeadLetter = "dead_letter"
)

// GenerateWebhookSecret creates a random 32-byte hex secret for webhook signing.
func GenerateWebhookSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "whsec_" + hex.EncodeToString(b), nil
}

// CreateWebhook creates a new webhook and returns the plaintext secret (shown once).
func CreateWebhook(name, targetURL, eventTypes, bucketScope string) (*Webhook, string, error) {
	secret, err := GenerateWebhookSecret()
	if err != nil {
		return nil, "", err
	}

	encrypted, err := crypto.Encrypt(secret, crypto.GetEncryptionKey())
	if err != nil {
		return nil, "", err
	}

	wh := &Webhook{
		Name:            name,
		TargetURL:       targetURL,
		SecretEncrypted: encrypted,
		Enabled:         true,
		EventTypes:      eventTypes,
		BucketScope:     bucketScope,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	d := GetDB()
	if err := d.Create(wh).Error; err != nil {
		slog.Error("Failed to create webhook", "error", err)
		return nil, "", err
	}

	slog.Info("Webhook created", "id", wh.ID, "name", name, "url", targetURL)
	return wh, secret, nil
}

// ListWebhooks returns all webhooks (without secrets).
func ListWebhooks() ([]Webhook, error) {
	d := GetDB()
	var hooks []Webhook
	if err := d.Order("created_at DESC").Find(&hooks).Error; err != nil {
		return nil, err
	}
	return hooks, nil
}

// GetWebhook returns a single webhook by ID.
func GetWebhook(id uint) (*Webhook, error) {
	d := GetDB()
	var wh Webhook
	if err := d.First(&wh, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &wh, nil
}

// UpdateWebhook updates mutable webhook fields.
func UpdateWebhook(id uint, updates map[string]any) error {
	d := GetDB()
	updates["updated_at"] = time.Now()
	result := d.Model(&Webhook{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("webhook not found")
	}
	return nil
}

// RotateWebhookSecret generates a new secret for a webhook and returns it.
func RotateWebhookSecret(id uint) (string, error) {
	secret, err := GenerateWebhookSecret()
	if err != nil {
		return "", err
	}

	encrypted, err := crypto.Encrypt(secret, crypto.GetEncryptionKey())
	if err != nil {
		return "", err
	}

	d := GetDB()
	result := d.Model(&Webhook{}).Where("id = ?", id).Updates(map[string]any{
		"secret_encrypted": encrypted,
		"updated_at":       time.Now(),
	})
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected == 0 {
		return "", errors.New("webhook not found")
	}
	return secret, nil
}

// DeleteWebhook deletes a webhook and its deliveries.
func DeleteWebhook(id uint) error {
	d := GetDB()
	return d.Transaction(func(tx *gorm.DB) error {
		// Delete deliveries
		tx.Where("webhook_id = ?", id).Delete(&WebhookDelivery{})
		// Delete webhook
		result := tx.Delete(&Webhook{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("webhook not found")
		}
		slog.Info("Webhook deleted", "id", id)
		return nil
	})
}

// GetWebhookSecret decrypts and returns the webhook secret.
func GetWebhookSecret(wh *Webhook) (string, error) {
	return crypto.Decrypt(wh.SecretEncrypted, crypto.GetEncryptionKey())
}

// CreateWebhookEvent persists an event and fans out pending deliveries.
func CreateWebhookEvent(eventType, resourceType, resourcePath, actor, payloadJSON string) error {
	d := GetDB()
	eventID := uuid.New().String()

	event := &WebhookEvent{
		ID:           eventID,
		EventType:    eventType,
		ResourceType: resourceType,
		ResourcePath: resourcePath,
		Actor:        actor,
		PayloadJSON:  payloadJSON,
		CreatedAt:    time.Now(),
	}

	return d.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(event).Error; err != nil {
			return err
		}

		// Find matching webhooks (batch-process to avoid loading all into memory)
		now := time.Now()
		batchSize := 50
		var cursor uint

		for {
			var batch []Webhook
			if err := tx.Where("enabled = ? AND id > ?", true, cursor).
				Order("id ASC").Limit(batchSize).Find(&batch).Error; err != nil {
				return err
			}
			if len(batch) == 0 {
				break
			}

			for _, wh := range batch {
				if !webhookMatchesEvent(wh, eventType, resourcePath) {
					continue
				}
				delivery := &WebhookDelivery{
					WebhookID:     wh.ID,
					EventID:       eventID,
					Status:        DeliveryPending,
					AttemptCount:  0,
					NextAttemptAt: &now,
					CreatedAt:     now,
					UpdatedAt:     now,
				}
				if err := tx.Create(delivery).Error; err != nil {
					slog.Error("Failed to create webhook delivery", "webhook_id", wh.ID, "event_id", eventID, "error", err)
				}
			}

			cursor = batch[len(batch)-1].ID
		}

		return nil
	})
}

// webhookMatchesEvent checks if a webhook subscription matches a given event.
func webhookMatchesEvent(wh Webhook, eventType, resourcePath string) bool {
	// Check event type match
	if wh.EventTypes != "" && wh.EventTypes != "[]" {
		matched := false
		// Simple contains check for JSON array like ["beamdrop.object.created"]
		if containsEventType(wh.EventTypes, eventType) {
			matched = true
		}
		// Support wildcard like "beamdrop.object.*"
		parts := splitEventType(eventType)
		if len(parts) >= 2 {
			wildcard := parts[0] + "." + parts[1] + ".*"
			if containsEventType(wh.EventTypes, wildcard) {
				matched = true
			}
		}
		if !matched {
			return false
		}
	}

	// Check bucket scope
	if wh.BucketScope != "" {
		// resourcePath format is "bucket/key" or just "bucket"
		bucket := resourcePath
		if idx := indexByte(resourcePath, '/'); idx >= 0 {
			bucket = resourcePath[:idx]
		}
		if bucket != wh.BucketScope {
			return false
		}
	}

	return true
}

func containsEventType(jsonArray, eventType string) bool {
	return len(jsonArray) > 0 && contains(jsonArray, `"`+eventType+`"`)
}

func splitEventType(et string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(et); i++ {
		if et[i] == '.' {
			parts = append(parts, et[start:i])
			start = i + 1
		}
	}
	parts = append(parts, et[start:])
	return parts
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// GetPendingDeliveries returns deliveries due for processing.
func GetPendingDeliveries(limit int) ([]WebhookDelivery, error) {
	d := GetDB()
	var deliveries []WebhookDelivery
	now := time.Now()
	err := d.Where("status IN ? AND (next_attempt_at IS NULL OR next_attempt_at <= ?)",
		[]string{DeliveryPending, DeliveryFailed}, now).
		Order("next_attempt_at ASC").
		Limit(limit).
		Find(&deliveries).Error
	return deliveries, err
}

// UpdateDelivery updates delivery status after an attempt.
func UpdateDelivery(id uint, status string, httpStatus int, errMsg string, durationMs int) error {
	d := GetDB()
	updates := map[string]any{
		"status":           status,
		"last_http_status": httpStatus,
		"last_error":       errMsg,
		"last_duration_ms": durationMs,
		"updated_at":       time.Now(),
		"attempt_count":    gorm.Expr("attempt_count + 1"),
	}
	if status == DeliveryDelivered {
		now := time.Now()
		updates["delivered_at"] = now
	}
	return d.Model(&WebhookDelivery{}).Where("id = ?", id).Updates(updates).Error
}

// SetDeliveryNextAttempt sets the next retry time.
func SetDeliveryNextAttempt(id uint, nextAt time.Time) error {
	d := GetDB()
	return d.Model(&WebhookDelivery{}).Where("id = ?", id).
		Update("next_attempt_at", nextAt).Error
}

// GetDeliveriesForWebhook returns recent deliveries for a webhook.
func GetDeliveriesForWebhook(webhookID uint, limit int) ([]WebhookDelivery, error) {
	d := GetDB()
	var deliveries []WebhookDelivery
	err := d.Where("webhook_id = ?", webhookID).
		Order("created_at DESC").
		Limit(limit).
		Find(&deliveries).Error
	return deliveries, err
}

// CleanupOldWebhookEvents removes events older than the given duration.
func CleanupOldWebhookEvents(maxAge time.Duration) int {
	d := GetDB()
	cutoff := time.Now().Add(-maxAge)

	// Delete old deliveries first (FK dependency)
	d.Where("created_at < ?", cutoff).Delete(&WebhookDelivery{})

	result := d.Where("created_at < ?", cutoff).Delete(&WebhookEvent{})
	if result.Error != nil {
		slog.Error("Failed to cleanup old webhook events", "error", result.Error)
		return 0
	}
	return int(result.RowsAffected)
}
