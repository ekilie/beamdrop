package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ekilie/beamdrop/config"
	"github.com/ekilie/beamdrop/pkg/db"
)

func TestIsPublicRoute_ExactMatch(t *testing.T) {
	m := NewAuthMiddleware(NewPasswordService(""))
	publicPaths := []string{
		"/",
		"/auth/login",
		"/auth/status",
		"/health",
		"/health/live",
		"/health/ready",
		"/health/startup",
		"/ready",
		"/metrics",
		"/api/v1/buckets",
	}
	for _, path := range publicPaths {
		if !m.IsPublicRoute(path) {
			t.Errorf("expected %q to be a public route", path)
		}
	}
}

func TestIsPublicRoute_StaticPrefix(t *testing.T) {
	m := NewAuthMiddleware(NewPasswordService(""))
	prefixPaths := []string{
		"/assets/logo.png",
		"/static/style.css",
		"/share/abc123",
		"/api/shares/access/token",
		"/api/v1/buckets/my-bucket/objects",
		"/api/v1/presign/token123",
		"/dl/something",
	}
	for _, path := range prefixPaths {
		if !m.IsPublicRoute(path) {
			t.Errorf("expected %q to be a public route", path)
		}
	}
}

func TestIsPublicRoute_ProtectedPath(t *testing.T) {
	m := NewAuthMiddleware(NewPasswordService(""))
	protectedPaths := []string{
		"/files",
		"/upload",
		"/download",
		"/admin",
		"/api/shares",
	}
	for _, path := range protectedPaths {
		if m.IsPublicRoute(path) {
			t.Errorf("expected %q to be protected", path)
		}
	}
}

func TestRequireAuth_Disabled(t *testing.T) {
	ps := NewPasswordService("")
	m := NewAuthMiddleware(ps)

	handler := m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/protected", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when auth disabled, got %d", w.Code)
	}
}

func TestRequireAuth_PublicRoute(t *testing.T) {
	tmpDir := t.TempDir()
	config.InitDataDir(tmpDir)
	config.DBPath = tmpDir + "/.beamdrop/test.db"
	db.Init()

	ps := NewPasswordService("pass")
	m := NewAuthMiddleware(ps)

	handler := m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for public route, got %d", w.Code)
	}
}

func TestRequireAuth_NoAuth(t *testing.T) {
	tmpDir := t.TempDir()
	config.InitDataDir(tmpDir)
	config.DBPath = tmpDir + "/.beamdrop/test.db"
	db.Init()

	ps := NewPasswordService("pass")
	m := NewAuthMiddleware(ps)

	handler := m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/protected", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated request, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "unauthorized") {
		t.Fatal("expected error message in body")
	}
}

func TestRequireAuth_BearerToken(t *testing.T) {
	tmpDir := t.TempDir()
	config.InitDataDir(tmpDir)
	config.DBPath = tmpDir + "/.beamdrop/test.db"
	db.Init()

	ps := NewPasswordService("pass")
	m := NewAuthMiddleware(ps)
	token, err := ps.GenerateToken()
	if err != nil {
		t.Fatal(err)
	}

	handler := m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/protected", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid bearer token, got %d", w.Code)
	}
}

func TestRequireAuth_InvalidBearerToken(t *testing.T) {
	tmpDir := t.TempDir()
	config.InitDataDir(tmpDir)
	config.DBPath = tmpDir + "/.beamdrop/test.db"
	db.Init()

	ps := NewPasswordService("pass")
	m := NewAuthMiddleware(ps)

	handler := m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/protected", nil)
	r.Header.Set("Authorization", "Bearer invalid-token")
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with invalid bearer token, got %d", w.Code)
	}
}

func TestRequireAuth_CookieToken(t *testing.T) {
	tmpDir := t.TempDir()
	config.InitDataDir(tmpDir)
	config.DBPath = tmpDir + "/.beamdrop/test.db"
	db.Init()

	ps := NewPasswordService("pass")
	m := NewAuthMiddleware(ps)
	token, err := ps.GenerateToken()
	if err != nil {
		t.Fatal(err)
	}

	handler := m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/protected", nil)
	r.AddCookie(&http.Cookie{Name: "beamdrop_token", Value: token})
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid cookie token, got %d", w.Code)
	}
}

func TestRequireAuth_ExpiredCookieToken(t *testing.T) {
	tmpDir := t.TempDir()
	config.InitDataDir(tmpDir)
	config.DBPath = tmpDir + "/.beamdrop/test.db"
	db.Init()

	ps := NewPasswordService("pass")
	m := NewAuthMiddleware(ps)

	handler := m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/protected", nil)
	r.AddCookie(&http.Cookie{Name: "beamdrop_token", Value: "expired-token"})
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with expired cookie token, got %d", w.Code)
	}
}

func TestMiddleware_ReturnsRequireAuth(t *testing.T) {
	tmpDir := t.TempDir()
	config.InitDataDir(tmpDir)
	config.DBPath = tmpDir + "/.beamdrop/test.db"
	db.Init()

	ps := NewPasswordService("pass")
	m := NewAuthMiddleware(ps)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/protected", nil)
	m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 from Middleware wrapper, got %d", w.Code)
	}
}
