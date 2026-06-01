package webhooks

import (
	"encoding/json"
	"log/slog"

	"github.com/ekilie/beamdrop/pkg/db"
)

// Event type constants
const (
	EventObjectCreated  = "beamdrop.object.created"
	EventObjectUpdated  = "beamdrop.object.updated"
	EventObjectDeleted  = "beamdrop.object.deleted"
	EventBucketCreated  = "beamdrop.bucket.created"
	EventBucketDeleted  = "beamdrop.bucket.deleted"
	EventShareCreated   = "beamdrop.share.created"
	EventShareDeleted   = "beamdrop.share.deleted"
	EventPresignCreated = "beamdrop.presign.created"
	EventPresignDeleted = "beamdrop.presign.deleted"
)

// Emit records an event and fans out deliveries to matching webhooks.
// It runs asynchronously to avoid blocking the request handler.
func Emit(eventType, resourceType, resourcePath, actor string, data any) {
	go func() {
		payloadJSON := ""
		if data != nil {
			if b, err := json.Marshal(data); err == nil {
				payloadJSON = string(b)
			}
		}

		if err := db.CreateWebhookEvent(eventType, resourceType, resourcePath, actor, payloadJSON); err != nil {
			slog.Error("Failed to emit webhook event",
				"event_type", eventType,
				"resource", resourcePath,
				"error", err)
		}
	}()
}

// EmitObjectCreated emits an object.created event.
func EmitObjectCreated(bucket, key, actor string, size int64, etag string) {
	Emit(EventObjectCreated, "object", bucket+"/"+key, actor, map[string]any{
		"bucket": bucket, "key": key, "size": size, "etag": etag,
	})
}

// EmitObjectDeleted emits an object.deleted event.
func EmitObjectDeleted(bucket, key, actor string) {
	Emit(EventObjectDeleted, "object", bucket+"/"+key, actor, map[string]any{
		"bucket": bucket, "key": key,
	})
}

// EmitBucketCreated emits a bucket.created event.
func EmitBucketCreated(bucket, actor string) {
	Emit(EventBucketCreated, "bucket", bucket, actor, map[string]any{
		"bucket": bucket,
	})
}

// EmitBucketDeleted emits a bucket.deleted event.
func EmitBucketDeleted(bucket, actor string) {
	Emit(EventBucketDeleted, "bucket", bucket, actor, map[string]any{
		"bucket": bucket,
	})
}

// EmitShareCreated emits a share.created event.
func EmitShareCreated(path, token, actor string) {
	Emit(EventShareCreated, "share", path, actor, map[string]any{
		"path": path, "token": token,
	})
}

// EmitShareDeleted emits a share.deleted event.
func EmitShareDeleted(token, actor string) {
	Emit(EventShareDeleted, "share", token, actor, map[string]any{
		"token": token,
	})
}

// EmitPresignCreated emits a presign.created event.
func EmitPresignCreated(bucket, key, token, actor string) {
	Emit(EventPresignCreated, "presign", bucket+"/"+key, actor, map[string]any{
		"bucket": bucket, "key": key, "token": token,
	})
}

// EmitPresignDeleted emits a presign.deleted event.
func EmitPresignDeleted(token, actor string) {
	Emit(EventPresignDeleted, "presign", token, actor, map[string]any{
		"token": token,
	})
}
