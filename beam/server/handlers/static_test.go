package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStaticHandler_RootServesHTML(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	StaticHandler(rec, req)

	res := rec.Result()
	ct := res.Header.Get("Content-Type")

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for /, got %d", res.StatusCode)
	}
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("expected text/html content-type for /, got %s", ct)
	}
}

func TestStaticHandler_SPAFallback_NoExtension(t *testing.T) {
	// Frontend routes like /shares, /api-keys should fallback to index.html
	routes := []string{"/shares", "/api-keys", "/share/some-token", "/some/deep/route"}

	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, route, nil)
			rec := httptest.NewRecorder()

			StaticHandler(rec, req)

			res := rec.Result()
			ct := res.Header.Get("Content-Type")

			if res.StatusCode != http.StatusOK {
				t.Errorf("expected 200 for %s, got %d", route, res.StatusCode)
			}
			if !strings.Contains(ct, "text/html") {
				t.Errorf("expected text/html for %s, got %s", route, ct)
			}
		})
	}
}

func TestStaticHandler_MissingAsset_Returns404(t *testing.T) {
	// Requests for actual files (with extension) that don't exist should 404
	assets := []string{"/nonexistent.js", "/missing.css", "/fake.png"}

	for _, asset := range assets {
		t.Run(asset, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, asset, nil)
			rec := httptest.NewRecorder()

			StaticHandler(rec, req)

			res := rec.Result()
			if res.StatusCode != http.StatusNotFound {
				t.Errorf("expected 404 for %s, got %d", asset, res.StatusCode)
			}
		})
	}
}
