package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}

func TestRateLimiter_GeneralAllowed(t *testing.T) {
	cfg := DefaultRateLimiterConfig()
	cfg.GeneralRate = 5 // 5 req/min for test
	rl := NewRateLimiter(cfg)
	defer rl.Close()

	handler := rl.Middleware(okHandler())

	// Should allow 5 requests (bucket starts full)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/files", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimiter_GeneralExceeded(t *testing.T) {
	cfg := DefaultRateLimiterConfig()
	cfg.GeneralRate = 3
	rl := NewRateLimiter(cfg)
	defer rl.Close()

	handler := rl.Middleware(okHandler())

	// Exhaust all tokens
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/files", nil)
		req.RemoteAddr = "10.0.0.1:9999"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	// Next request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/files", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}

	// Should have Retry-After header
	ra := w.Header().Get("Retry-After")
	if ra == "" {
		t.Fatal("expected Retry-After header")
	}

	// Should have proper JSON body with error code
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Error.Code != "RATE_LIMIT_EXCEEDED" {
		t.Fatalf("expected RATE_LIMIT_EXCEEDED, got %s", body.Error.Code)
	}
}

func TestRateLimiter_AuthTier(t *testing.T) {
	cfg := DefaultRateLimiterConfig()
	cfg.AuthRate = 2
	rl := NewRateLimiter(cfg)
	defer rl.Close()

	handler := rl.Middleware(okHandler())

	// Exhaust auth tokens (2)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = "10.0.0.2:8080"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	// 3rd auth request should fail
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = "10.0.0.2:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

func TestRateLimiter_UploadTier(t *testing.T) {
	cfg := DefaultRateLimiterConfig()
	cfg.UploadRate = 2
	rl := NewRateLimiter(cfg)
	defer rl.Close()

	handler := rl.Middleware(okHandler())

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/upload", nil)
		req.RemoteAddr = "10.0.0.3:5555"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/upload", nil)
	req.RemoteAddr = "10.0.0.3:5555"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

func TestRateLimiter_PerIPIsolation(t *testing.T) {
	cfg := DefaultRateLimiterConfig()
	cfg.GeneralRate = 2
	rl := NewRateLimiter(cfg)
	defer rl.Close()

	handler := rl.Middleware(okHandler())

	// Exhaust IP1
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/files", nil)
		req.RemoteAddr = "1.1.1.1:1111"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// IP2 should still work
	req := httptest.NewRequest(http.MethodGet, "/files", nil)
	req.RemoteAddr = "2.2.2.2:2222"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("different IP should not be rate limited, got %d", w.Code)
	}
}

func TestRateLimiter_TokenRefill(t *testing.T) {
	cfg := DefaultRateLimiterConfig()
	cfg.GeneralRate = 60 // 60/min = 1/sec
	rl := NewRateLimiter(cfg)
	defer rl.Close()

	handler := rl.Middleware(okHandler())

	// Exhaust all tokens
	for i := 0; i < 60; i++ {
		req := httptest.NewRequest(http.MethodGet, "/files", nil)
		req.RemoteAddr = "10.0.0.5:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// Wait for 1 token to refill (1 token/sec)
	time.Sleep(1100 * time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/files", nil)
	req.RemoteAddr = "10.0.0.5:1234"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected token to refill, got %d", w.Code)
	}
}

func TestRateLimiter_Disabled(t *testing.T) {
	cfg := DefaultRateLimiterConfig()
	cfg.Enabled = false
	rl := NewRateLimiter(cfg)
	defer rl.Close()

	handler := rl.Middleware(okHandler())

	// Should never be rate limited
	for i := 0; i < 200; i++ {
		req := httptest.NewRequest(http.MethodGet, "/files", nil)
		req.RemoteAddr = "10.0.0.9:7777"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: disabled limiter should not block, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimiter_XForwardedFor(t *testing.T) {
	cfg := DefaultRateLimiterConfig()
	cfg.GeneralRate = 1
	rl := NewRateLimiter(cfg)
	defer rl.Close()

	handler := rl.Middleware(okHandler())

	// First request with X-Forwarded-For
	req := httptest.NewRequest(http.MethodGet, "/files", nil)
	req.RemoteAddr = "127.0.0.1:9999"
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Second request from same forwarded IP – should be limited
	req2 := httptest.NewRequest(http.MethodGet, "/files", nil)
	req2.RemoteAddr = "127.0.0.1:9999"
	req2.Header.Set("X-Forwarded-For", "203.0.113.50")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for same forwarded IP, got %d", w2.Code)
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name     string
		remote   string
		xff      string
		xri      string
		expected string
	}{
		{"plain RemoteAddr", "192.168.1.1:1234", "", "", "192.168.1.1"},
		{"X-Forwarded-For single", "127.0.0.1:80", "10.0.0.1", "", "10.0.0.1"},
		{"X-Forwarded-For multiple", "127.0.0.1:80", "10.0.0.1, 10.0.0.2", "", "10.0.0.1"},
		{"X-Real-IP", "127.0.0.1:80", "", "172.16.0.1", "172.16.0.1"},
		{"XFF takes priority over XRI", "127.0.0.1:80", "10.0.0.1", "172.16.0.1", "10.0.0.1"},
	}

	// Parse 127.0.0.1 as a trusted proxy for tests that use XFF/XRI headers
	trustedProxies := ParseTrustedProxies("127.0.0.1/32")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remote
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}
			var proxies []*net.IPNet
			if tt.xff != "" || tt.xri != "" {
				proxies = trustedProxies
			}
			got := extractIP(req, proxies)
			if got != tt.expected {
				t.Errorf("extractIP() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestClassifyRequest(t *testing.T) {
	tests := []struct {
		method   string
		path     string
		expected tier
	}{
		{http.MethodPost, "/auth/login", tierAuth},
		{http.MethodGet, "/auth/login", tierAuth},
		{http.MethodPost, "/upload", tierUpload},
		{http.MethodPut, "/upload", tierUpload},
		{http.MethodPut, "/api/v1/buckets/mybucket/myobject", tierUpload},
		{http.MethodGet, "/api/v1/buckets/mybucket/myobject", tierGeneral},
		{http.MethodGet, "/files", tierGeneral},
		{http.MethodGet, "/", tierGeneral},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			got := classifyRequest(req)
			if got != tt.expected {
				t.Errorf("classifyRequest() = %v, want %v", got, tt.expected)
			}
		})
	}
}
