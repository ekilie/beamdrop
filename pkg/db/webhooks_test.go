package db

import (
	"testing"
	"time"

	"github.com/ekilie/beamdrop/pkg/crypto"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestGenerateWebhookSecret(t *testing.T) {
	s1, err := GenerateWebhookSecret()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s1) < 10 {
		t.Fatalf("expected non-empty secret, got %q", s1)
	}
	if s1[:6] != "whsec_" {
		t.Fatalf("expected whsec_ prefix, got %q", s1[:6])
	}

	s2, _ := GenerateWebhookSecret()
	if s1 == s2 {
		t.Fatal("secrets should be unique")
	}
}

func setupWebhookTestDB(t *testing.T) {
	t.Helper()
	var err error
	db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(&Webhook{}, &WebhookEvent{}, &WebhookDelivery{}, &ServerStats{}, &Config{}, &StarredFile{}, &APIKey{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
}

func TestCreateAndGetWebhook(t *testing.T) {
	setupWebhookTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	wh, secret, err := CreateWebhook("test-hook", "https://example.com/hook", `["beamdrop.object.created"]`, "my-bucket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wh.Name != "test-hook" {
		t.Fatalf("expected name 'test-hook', got %q", wh.Name)
	}
	if wh.TargetURL != "https://example.com/hook" {
		t.Fatalf("expected URL 'https://example.com/hook', got %q", wh.TargetURL)
	}
	if !wh.Enabled {
		t.Fatal("webhook should be enabled by default")
	}
	if secret == "" {
		t.Fatal("expected non-empty secret")
	}

	found, err := GetWebhook(wh.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find webhook")
	}
	if found.Name != "test-hook" {
		t.Fatalf("expected name 'test-hook', got %q", found.Name)
	}
}

func TestGetWebhook_NotFound(t *testing.T) {
	setupWebhookTestDB(t)

	wh, err := GetWebhook(999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wh != nil {
		t.Fatal("expected nil for non-existent webhook")
	}
}

func TestListWebhooks(t *testing.T) {
	setupWebhookTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	CreateWebhook("hook1", "https://example.com/1", "[]", "")
	CreateWebhook("hook2", "https://example.com/2", "[]", "")

	hooks, err := ListWebhooks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hooks) != 2 {
		t.Fatalf("expected 2 webhooks, got %d", len(hooks))
	}
}

func TestUpdateWebhook(t *testing.T) {
	setupWebhookTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	wh, _, _ := CreateWebhook("test", "https://example.com/hook", "[]", "")

	err := UpdateWebhook(wh.ID, map[string]any{"enabled": false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, _ := GetWebhook(wh.ID)
	if found.Enabled {
		t.Fatal("webhook should be disabled")
	}
}

func TestUpdateWebhook_NotFound(t *testing.T) {
	setupWebhookTestDB(t)

	err := UpdateWebhook(999, map[string]any{"enabled": false})
	if err == nil {
		t.Fatal("expected error for non-existent webhook")
	}
}

func TestRotateWebhookSecret(t *testing.T) {
	setupWebhookTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	wh, secret, _ := CreateWebhook("test", "https://example.com/hook", "[]", "")

	newSecret, err := RotateWebhookSecret(wh.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newSecret == secret {
		t.Fatal("new secret should differ from old")
	}
	if newSecret[:6] != "whsec_" {
		t.Fatalf("expected whsec_ prefix, got %q", newSecret[:6])
	}
}

func TestRotateWebhookSecret_NotFound(t *testing.T) {
	setupWebhookTestDB(t)

	_, err := RotateWebhookSecret(999)
	if err == nil {
		t.Fatal("expected error for non-existent webhook")
	}
}

func TestDeleteWebhook(t *testing.T) {
	setupWebhookTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	wh, _, _ := CreateWebhook("test", "https://example.com/hook", "[]", "")

	if err := DeleteWebhook(wh.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, _ := GetWebhook(wh.ID)
	if found != nil {
		t.Fatal("webhook should be deleted")
	}
}

func TestDeleteWebhook_NotFound(t *testing.T) {
	setupWebhookTestDB(t)

	err := DeleteWebhook(999)
	if err == nil {
		t.Fatal("expected error for non-existent webhook")
	}
}

func TestGetWebhookSecret(t *testing.T) {
	setupWebhookTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	wh, originalSecret, _ := CreateWebhook("test", "https://example.com/hook", "[]", "")

	decrypted, err := GetWebhookSecret(wh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decrypted != originalSecret {
		t.Fatalf("decrypted secret doesn't match: got %q, want %q", decrypted, originalSecret)
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	setupWebhookTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	// Create a webhook that matches the event
	CreateWebhook("test", "https://example.com/hook", `["beamdrop.object.created"]`, "")

	err := CreateWebhookEvent("beamdrop.object.created", "object", "my-bucket/my-file.txt", "test-actor", `{"size": 123}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the event was created and a delivery was queued
	deliveries, err := GetPendingDeliveries(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deliveries) != 1 {
		t.Fatalf("expected 1 pending delivery, got %d", len(deliveries))
	}
}

func TestGetPendingDeliveries(t *testing.T) {
	setupWebhookTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	deliveries, err := GetPendingDeliveries(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deliveries) != 0 {
		t.Fatalf("expected 0 pending deliveries, got %d", len(deliveries))
	}
}

func TestWebhookMatchesEvent_ExactMatch(t *testing.T) {
	wh := Webhook{EventTypes: `["beamdrop.object.created"]`}
	if !webhookMatchesEvent(wh, "beamdrop.object.created", "bucket/key") {
		t.Fatal("exact match should succeed")
	}
}

func TestWebhookMatchesEvent_WildcardMatch(t *testing.T) {
	wh := Webhook{EventTypes: `["beamdrop.object.*"]`}
	if !webhookMatchesEvent(wh, "beamdrop.object.created", "bucket/key") {
		t.Fatal("wildcard match should succeed")
	}
}

func TestWebhookMatchesEvent_NoMatch(t *testing.T) {
	wh := Webhook{EventTypes: `["beamdrop.object.created"]`}
	if webhookMatchesEvent(wh, "beamdrop.bucket.created", "bucket") {
		t.Fatal("non-matching event type should fail")
	}
}

func TestWebhookMatchesEvent_BucketScopeMatch(t *testing.T) {
	wh := Webhook{EventTypes: `["beamdrop.object.created"]`, BucketScope: "my-bucket"}
	if !webhookMatchesEvent(wh, "beamdrop.object.created", "my-bucket/my-file.txt") {
		t.Fatal("bucket scope match should succeed")
	}
}

func TestWebhookMatchesEvent_BucketScopeNoMatch(t *testing.T) {
	wh := Webhook{EventTypes: `["beamdrop.object.created"]`, BucketScope: "other-bucket"}
	if webhookMatchesEvent(wh, "beamdrop.object.created", "my-bucket/my-file.txt") {
		t.Fatal("bucket scope mismatch should fail")
	}
}

func TestWebhookMatchesEvent_EmptyEventTypes(t *testing.T) {
	wh := Webhook{EventTypes: ""}
	if !webhookMatchesEvent(wh, "any.event", "bucket/key") {
		t.Fatal("empty event types should match all")
	}
}

func TestWebhookMatchesEvent_EmptyArray(t *testing.T) {
	wh := Webhook{EventTypes: "[]"}
	if !webhookMatchesEvent(wh, "any.event", "bucket/key") {
		t.Fatal("empty array should match all")
	}
}

func TestSplitEventType(t *testing.T) {
	parts := splitEventType("beamdrop.object.created")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d: %v", len(parts), parts)
	}
	if parts[0] != "beamdrop" || parts[1] != "object" || parts[2] != "created" {
		t.Fatalf("unexpected parts: %v", parts)
	}
}

func TestContainsEventType(t *testing.T) {
	if !containsEventType(`["beamdrop.object.created"]`, "beamdrop.object.created") {
		t.Fatal("event type should be found")
	}
	if containsEventType(`["beamdrop.object.created"]`, "beamdrop.bucket.created") {
		t.Fatal("non-matching event type should not be found")
	}
}

func TestContainsStr(t *testing.T) {
	if !containsStr("hello world", "world") {
		t.Fatal("substring should be found")
	}
	if containsStr("hello", "world") {
		t.Fatal("substring should not be found")
	}
}

func TestIndexByte(t *testing.T) {
	if idx := indexByte("hello/world", '/'); idx != 5 {
		t.Fatalf("expected index 5, got %d", idx)
	}
	if idx := indexByte("hello", '/'); idx != -1 {
		t.Fatalf("expected -1, got %d", idx)
	}
}

func TestUpdateDelivery(t *testing.T) {
	setupWebhookTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	CreateWebhook("test", "https://example.com/hook", `["test.event"]`, "")
	CreateWebhookEvent("test.event", "object", "bucket/key", "actor", "")

	deliveries, _ := GetPendingDeliveries(10)
	if len(deliveries) == 0 {
		t.Fatal("expected at least one delivery")
	}

	err := UpdateDelivery(deliveries[0].ID, DeliveryDelivered, 200, "", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetDeliveryNextAttempt(t *testing.T) {
	setupWebhookTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	CreateWebhook("test", "https://example.com/hook", `["test.event"]`, "")
	CreateWebhookEvent("test.event", "object", "bucket/key", "actor", "")

	deliveries, _ := GetPendingDeliveries(10)
	if len(deliveries) == 0 {
		t.Fatal("expected at least one delivery")
	}

	nextAt := time.Now().Add(1 * time.Hour)
	err := SetDeliveryNextAttempt(deliveries[0].ID, nextAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetDeliveriesForWebhook(t *testing.T) {
	setupWebhookTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	wh, _, _ := CreateWebhook("test", "https://example.com/hook", `["test.event"]`, "")
	CreateWebhookEvent("test.event", "object", "bucket/key", "actor", "")

	deliveries, err := GetDeliveriesForWebhook(wh.ID, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deliveries) != 1 {
		t.Fatalf("expected 1 delivery, got %d", len(deliveries))
	}
}

func TestCleanupOldWebhookEvents(t *testing.T) {
	setupWebhookTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	removed := CleanupOldWebhookEvents(0) // Remove everything
	_ = removed
}
