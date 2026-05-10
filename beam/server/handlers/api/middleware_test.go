package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ekilie/beamdrop/config"
	"github.com/ekilie/beamdrop/pkg/crypto"
	"github.com/ekilie/beamdrop/pkg/db"
)

var middlewareTestDBOnce sync.Once

func setupMiddlewareTestDB(t *testing.T) {
	t.Helper()

	middlewareTestDBOnce.Do(func() {
		tempDir, err := os.MkdirTemp("", "beamdrop-api-middleware-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		config.SetDBPath(filepath.Join(tempDir, "beamdrop.db"))
		crypto.SetEncryptionKey(bytes.Repeat([]byte("k"), 32))
		db.Init()
		db.AutoMigrate()
	})

	if err := db.GetDB().Exec("DELETE FROM api_keys").Error; err != nil {
		t.Fatalf("failed to cleanup api_keys table: %v", err)
	}
}

func newSignedRequest(method, path, accessKeyID, secretKey string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	timestamp := time.Now().UTC().Format(time.RFC3339)
	signature := crypto.GenerateSignature(secretKey, method, path, timestamp)
	req.Header.Set("X-Beamdrop-Date", timestamp)
	req.Header.Set("Authorization", "Bearer "+accessKeyID+":"+signature)
	return req
}

func TestAPIAuthMiddleware_DeniesWriteForReadOnlyKey(t *testing.T) {
	setupMiddlewareTestDB(t)

	apiKey, secretKey, err := db.CreateAPIKey("read-only", "read", "", nil)
	if err != nil {
		t.Fatalf("failed to create API key: %v", err)
	}

	nextCalled := false
	middleware := NewAPIAuthMiddleware(true)
	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := newSignedRequest(http.MethodPut, "/api/v1/buckets/photos/object.jpg", apiKey.AccessKeyID, secretKey)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if nextCalled {
		t.Fatal("expected next handler not to be called for denied request")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}
}

func TestAPIAuthMiddleware_AllowsReadForReadOnlyKey(t *testing.T) {
	setupMiddlewareTestDB(t)

	apiKey, secretKey, err := db.CreateAPIKey("read-only", "read", "", nil)
	if err != nil {
		t.Fatalf("failed to create API key: %v", err)
	}

	middleware := NewAPIAuthMiddleware(true)
	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := newSignedRequest(http.MethodGet, "/api/v1/buckets/photos/object.jpg", apiKey.AccessKeyID, secretKey)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}
}

func TestAPIAuthMiddleware_EnforcesBucketScope(t *testing.T) {
	setupMiddlewareTestDB(t)

	apiKey, secretKey, err := db.CreateAPIKey("scoped", "read,write", "photos", nil)
	if err != nil {
		t.Fatalf("failed to create API key: %v", err)
	}

	middleware := NewAPIAuthMiddleware(true)
	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := newSignedRequest(http.MethodGet, "/api/v1/buckets/videos/object.jpg", apiKey.AccessKeyID, secretKey)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}
}
