package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders_Default(t *testing.T) {
	handler := SecurityHeaders(false, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if v := w.Header().Get("X-Frame-Options"); v != "DENY" {
		t.Fatalf("expected X-Frame-Options: DENY, got %q", v)
	}
	if v := w.Header().Get("X-Content-Type-Options"); v != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options: nosniff, got %q", v)
	}
	if v := w.Header().Get("Referrer-Policy"); v != "strict-origin-when-cross-origin" {
		t.Fatalf("expected Referrer-Policy header, got %q", v)
	}
	if v := w.Header().Get("Permissions-Policy"); v == "" {
		t.Fatal("expected Permissions-Policy header")
	}
	if v := w.Header().Get("Content-Security-Policy"); v == "" {
		t.Fatal("expected Content-Security-Policy header")
	}
}

func TestSecurityHeaders_NoCSP(t *testing.T) {
	handler := SecurityHeaders(false, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(w, r)

	if v := w.Header().Get("Content-Security-Policy"); v != "" {
		t.Fatal("CSP should be empty when disabled")
	}
}

func TestSecurityHeaders_HSTS(t *testing.T) {
	handler := SecurityHeaders(true, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Without TLS, HSTS should NOT be set
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(w, r)

	if v := w.Header().Get("Strict-Transport-Security"); v != "" {
		t.Fatal("HSTS should not be set without TLS")
	}

	// With TLS, HSTS should be set
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/test", nil)
	r2.TLS = &tls.ConnectionState{}
	handler.ServeHTTP(w2, r2)

	if v := w2.Header().Get("Strict-Transport-Security"); v == "" {
		t.Fatal("HSTS should be set with TLS")
	}
}
