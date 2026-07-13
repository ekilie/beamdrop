package db

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func setupTestDBWithAllTables(t *testing.T) {
	t.Helper()
	var err error
	db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(&PresignedURL{}, &ShareableLink{}, &APIKey{}, &Webhook{}, &WebhookEvent{}, &WebhookDelivery{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
}

func TestCreatePresignedURL(t *testing.T) {
	setupTestDBWithAllTables(t)

	p, err := CreatePresignedURL("my-bucket", "my-key.txt", "GET", "test-actor", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if p.Bucket != "my-bucket" {
		t.Fatalf("expected bucket 'my-bucket', got %q", p.Bucket)
	}
	if p.Key != "my-key.txt" {
		t.Fatalf("expected key 'my-key.txt', got %q", p.Key)
	}
	if p.Method != "GET" {
		t.Fatalf("expected method 'GET', got %q", p.Method)
	}
	if p.CreatedBy != "test-actor" {
		t.Fatalf("expected createdBy 'test-actor', got %q", p.CreatedBy)
	}
}

func TestCreatePresignedURL_WithExpiryAndDownloads(t *testing.T) {
	setupTestDBWithAllTables(t)

	expiry := 1 * time.Hour
	maxDownloads := 5
	p, err := CreatePresignedURL("bucket", "key", "PUT", "actor", &expiry, &maxDownloads)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ExpiresAt == nil {
		t.Fatal("expected non-nil ExpiresAt")
	}
	if p.MaxDownloads == nil || *p.MaxDownloads != 5 {
		t.Fatalf("expected MaxDownloads 5, got %v", p.MaxDownloads)
	}
}

func TestGetPresignedURLByToken(t *testing.T) {
	setupTestDBWithAllTables(t)

	created, _ := CreatePresignedURL("bucket", "key", "GET", "actor", nil, nil)

	found, err := GetPresignedURLByToken(created.Token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find presigned URL")
	}
	if found.Bucket != "bucket" {
		t.Fatalf("expected bucket 'bucket', got %q", found.Bucket)
	}
}

func TestGetPresignedURLByToken_NotFound(t *testing.T) {
	setupTestDBWithAllTables(t)

	found, err := GetPresignedURLByToken("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Fatal("expected nil for non-existent token")
	}
}

func TestGetPresignedURLByToken_Expired(t *testing.T) {
	setupTestDBWithAllTables(t)

	past := time.Now().Add(-1 * time.Hour)
	p := &PresignedURL{
		Token:     "expired_token_1234",
		Bucket:    "bucket",
		Key:       "key",
		Method:    "GET",
		ExpiresAt: &past,
		CreatedAt: time.Now(),
	}
	db.Create(p)

	_, err := GetPresignedURLByToken("expired_token_1234")
	if err == nil {
		t.Fatal("expected error for expired URL")
	}
}

func TestGetPresignedURLByToken_MaxDownloadsReached(t *testing.T) {
	setupTestDBWithAllTables(t)

	maxDownloads := 1
	p := &PresignedURL{
		Token:         "maxed_out_token",
		Bucket:        "bucket",
		Key:           "key",
		Method:        "GET",
		MaxDownloads:  &maxDownloads,
		DownloadCount: 1,
		CreatedAt:     time.Now(),
	}
	db.Create(p)

	_, err := GetPresignedURLByToken("maxed_out_token")
	if err == nil {
		t.Fatal("expected error when download limit reached")
	}
}

func TestIncrementPresignedURLDownloads(t *testing.T) {
	setupTestDBWithAllTables(t)

	created, _ := CreatePresignedURL("bucket", "key", "GET", "actor", nil, nil)

	if err := IncrementPresignedURLDownloads(created.Token); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, _ := GetPresignedURLByToken(created.Token)
	if found.DownloadCount != 1 {
		t.Fatalf("expected download count 1, got %d", found.DownloadCount)
	}
}

func TestListPresignedURLs(t *testing.T) {
	setupTestDBWithAllTables(t)

	CreatePresignedURL("bucket1", "key1", "GET", "actor", nil, nil)
	CreatePresignedURL("bucket2", "key2", "PUT", "actor", nil, nil)

	urls, err := ListPresignedURLs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 2 {
		t.Fatalf("expected 2 URLs, got %d", len(urls))
	}
}

func TestDeletePresignedURL(t *testing.T) {
	setupTestDBWithAllTables(t)

	created, _ := CreatePresignedURL("bucket", "key", "GET", "actor", nil, nil)

	if err := DeletePresignedURL(created.Token); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, _ := GetPresignedURLByToken(created.Token)
	if found != nil {
		t.Fatal("URL should be deleted")
	}
}

func TestDeletePresignedURL_NotFound(t *testing.T) {
	setupTestDBWithAllTables(t)

	err := DeletePresignedURL("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent URL")
	}
}

func TestCleanupExpiredPresignedURLs(t *testing.T) {
	setupTestDBWithAllTables(t)

	past := time.Now().Add(-1 * time.Hour)
	db.Create(&PresignedURL{
		Token:     "expired_token",
		Bucket:    "bucket",
		Key:       "key",
		Method:    "GET",
		ExpiresAt: &past,
		CreatedAt: time.Now(),
	})

	future := time.Now().Add(1 * time.Hour)
	db.Create(&PresignedURL{
		Token:     "valid_token",
		Bucket:    "bucket",
		Key:       "key",
		Method:    "GET",
		ExpiresAt: &future,
		CreatedAt: time.Now(),
	})

	if err := CleanupExpiredPresignedURLs(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	urls, _ := ListPresignedURLs()
	if len(urls) != 1 {
		t.Fatalf("expected 1 URL after cleanup, got %d", len(urls))
	}
	if urls[0].Token != "valid_token" {
		t.Fatalf("expected valid token to remain, got %q", urls[0].Token)
	}
}
