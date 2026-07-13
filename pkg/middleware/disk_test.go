package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestMaxStorageCheck_Disabled(t *testing.T) {
	handler := MaxStorageCheck("/tmp", 0)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/upload", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when storage check disabled, got %d", w.Code)
	}
}

func TestMaxStorageCheck_SafeMethods(t *testing.T) {
	handler := MaxStorageCheck("/tmp", 1)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, method := range []string{"GET", "HEAD", "OPTIONS", "DELETE"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/test", nil)
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", method, w.Code)
		}
	}
}

func TestMaxStorageCheck_NegativeMaxBytes(t *testing.T) {
	handler := MaxStorageCheck("/tmp", -1)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/upload", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for negative maxBytes, got %d", w.Code)
	}
}

func TestMaxStorageCheck_WriteMethods(t *testing.T) {
	tmpDir := t.TempDir()
	// Create some content so usage > 1
	if err := os.WriteFile(filepath.Join(tmpDir, "payload.bin"), make([]byte, 100), 0644); err != nil {
		t.Fatal(err)
	}

	handler := MaxStorageCheck(tmpDir, 1)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, method := range []string{"POST", "PUT", "PATCH"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/test", nil)
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusInsufficientStorage {
			t.Fatalf("expected 507 for %s with small limit, got %d", method, w.Code)
		}
	}
}
