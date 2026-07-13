package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRFProtection_Disabled(t *testing.T) {
	handler := CSRFProtection(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when disabled, got %d", w.Code)
	}
}

func TestCSRFProtection_SafeMethods(t *testing.T) {
	handler := CSRFProtection(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, method := range []string{"GET", "HEAD", "OPTIONS"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/test", nil)
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", method, w.Code)
		}
	}
}

func TestCSRFProtection_SafeMethodsSetsCookie(t *testing.T) {
	handler := CSRFProtection(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(w, r)

	cookies := w.Header().Values("Set-Cookie")
	found := false
	for _, c := range cookies {
		if strings.Contains(c, "beamdrop_csrf") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected CSRF cookie to be set on GET")
	}
}

func TestCSRFProtection_SafeMethodsReusesCookie(t *testing.T) {
	handler := CSRFProtection(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	r.AddCookie(&http.Cookie{Name: "beamdrop_csrf", Value: "existing-token"})
	handler.ServeHTTP(w, r)

	if w.Header().Get("Set-Cookie") != "" {
		t.Fatal("should not set cookie if already present")
	}
}

func TestCSRFProtection_AuthorizationExempt(t *testing.T) {
	handler := CSRFProtection(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	r.Header.Set("Authorization", "Bearer token123")
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with auth header, got %d", w.Code)
	}
}

func TestCSRFProtection_NoSessionCookieExempt(t *testing.T) {
	handler := CSRFProtection(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 without session cookie, got %d", w.Code)
	}
}

func TestCSRFProtection_ShareAccessExempt(t *testing.T) {
	handler := CSRFProtection(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/shares/access/token123", nil)
	r.AddCookie(&http.Cookie{Name: "beamdrop_token", Value: "session"})
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for share access, got %d", w.Code)
	}
}

func TestCSRFProtection_MissingCookie(t *testing.T) {
	handler := CSRFProtection(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	r.AddCookie(&http.Cookie{Name: "beamdrop_token", Value: "session"})
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing CSRF cookie, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "missing CSRF token") {
		t.Fatal("expected missing CSRF token error")
	}
}

func TestCSRFProtection_InvalidToken(t *testing.T) {
	handler := CSRFProtection(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	r.AddCookie(&http.Cookie{Name: "beamdrop_token", Value: "session"})
	r.AddCookie(&http.Cookie{Name: "beamdrop_csrf", Value: "csrf-token"})
	r.Header.Set("X-CSRF-Token", "wrong-token")
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for invalid CSRF token, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid CSRF token") {
		t.Fatal("expected invalid CSRF token error")
	}
}

func TestCSRFProtection_ValidToken(t *testing.T) {
	handler := CSRFProtection(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	r.AddCookie(&http.Cookie{Name: "beamdrop_token", Value: "session"})
	r.AddCookie(&http.Cookie{Name: "beamdrop_csrf", Value: "valid-token"})
	r.Header.Set("X-CSRF-Token", "valid-token")
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid CSRF token, got %d", w.Code)
	}
}

func TestGenerateCSRFToken(t *testing.T) {
	token1 := generateCSRFToken()
	token2 := generateCSRFToken()

	if token1 == "" {
		t.Fatal("expected non-empty token")
	}
	if token1 == token2 {
		t.Fatal("tokens should be unique")
	}
	if len(token1) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(token1))
	}
}
