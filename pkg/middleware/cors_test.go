package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultCORSConfig(t *testing.T) {
	cfg := DefaultCORSConfig()
	if len(cfg.AllowedOrigins) != 0 {
		t.Fatal("default CORS should have no allowed origins")
	}
	if !cfg.AllowCredentials {
		t.Fatal("default CORS should allow credentials")
	}
}

func TestCORS_Disabled(t *testing.T) {
	cfg := DefaultCORSConfig()
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("no CORS headers should be set when disabled")
	}
}

func TestCORS_DisabledPreflight(t *testing.T) {
	cfg := DefaultCORSConfig()
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("OPTIONS", "/test", nil)
	r.Header.Set("Origin", "http://evil.com")
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for preflight when CORS disabled, got %d", w.Code)
	}
}

func TestCORS_AllowedOrigin(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins:   []string{"http://example.com"},
		AllowCredentials: true,
	}
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("Origin", "http://example.com")
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if v := w.Header().Get("Access-Control-Allow-Origin"); v != "http://example.com" {
		t.Fatalf("expected origin header, got %q", v)
	}
	if v := w.Header().Get("Access-Control-Allow-Credentials"); v != "true" {
		t.Fatalf("expected credentials header, got %q", v)
	}
}

func TestCORS_WildcardOrigin(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	}
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("Origin", "http://anywhere.com")
	handler.ServeHTTP(w, r)

	if v := w.Header().Get("Access-Control-Allow-Origin"); v != "http://anywhere.com" {
		t.Fatalf("expected origin echo, got %q", v)
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins:   []string{"http://example.com"},
		AllowCredentials: true,
	}
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("Origin", "http://evil.com")
	handler.ServeHTTP(w, r)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("no CORS headers for disallowed origin")
	}
}

func TestCORS_AllowedPreflight(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins:   []string{"http://example.com"},
		AllowCredentials: true,
	}
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("OPTIONS", "/test", nil)
	r.Header.Set("Origin", "http://example.com")
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for allowed preflight, got %d", w.Code)
	}
}

func TestCORS_DisallowedPreflight(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins:   []string{"http://example.com"},
		AllowCredentials: true,
	}
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("OPTIONS", "/test", nil)
	r.Header.Set("Origin", "http://evil.com")
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disallowed preflight, got %d", w.Code)
	}
}

func TestIsOriginAllowed(t *testing.T) {
	if IsOriginAllowed("http://example.com", nil) {
		t.Fatal("nil origins should not allow")
	}
	if IsOriginAllowed("http://example.com", []string{}) {
		t.Fatal("empty origins should not allow")
	}
	if !IsOriginAllowed("http://example.com", []string{"http://example.com"}) {
		t.Fatal("exact match should allow")
	}
	if !IsOriginAllowed("http://anything.com", []string{"*"}) {
		t.Fatal("wildcard should allow any")
	}
	if IsOriginAllowed("http://evil.com", []string{"http://good.com"}) {
		t.Fatal("non-matching should not allow")
	}
}
