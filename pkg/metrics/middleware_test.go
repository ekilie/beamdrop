package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/", "/"},
		{"/health", "/health"},
		{"/health/live", "/health/live"},
		{"/upload", "/upload"},
		{"/files", "/files"},
		{"/metrics", "/metrics"},
		{"/assets/main.js", "/assets/{file}"},
		{"/assets/css/style.css", "/assets/{file}"},
		{"/static/logo.png", "/static/{file}"},
		{"/share/abc123", "/share/{file}"},
		{"/api/shares/access/some-uuid-here", "/api/shares/access/{id}"},
		{"/api/v1/buckets/mybucket", "/api/v1/buckets/{bucket}"},
		{"/api/v1/buckets/mybucket/key.txt", "/api/v1/buckets/{bucket}"},
		{"/unknown/path", "/unknown/path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizePath(tt.input)
			if got != tt.want {
				t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMiddlewareRecordsMetrics(t *testing.T) {
	// Create a fresh registry for isolation
	reg := prometheus.NewRegistry()

	reqCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "test",
		Name:      "requests_total",
	}, []string{"method", "path", "status"})
	reg.MustRegister(reqCounter)

	// Simple handler that returns 200
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Wrap with our middleware
	wrapped := Middleware(inner)

	// Make a request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Verify that the global RequestsTotal counter was incremented
	m := &dto.Metric{}
	err := RequestsTotal.WithLabelValues("GET", "/health", "200").(prometheus.Metric).Write(m)
	if err != nil {
		t.Fatalf("failed to read metric: %v", err)
	}
	if got := m.GetCounter().GetValue(); got < 1 {
		t.Errorf("expected requests_total >= 1, got %v", got)
	}
}

func TestMiddlewareTracksActiveConnections(t *testing.T) {
	connGauge := &dto.Metric{}

	// Handler that checks active connections while request is in-flight
	var inFlightValue float64
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the gauge while we're inside the handler
		err := ActiveConnections.(prometheus.Metric).Write(connGauge)
		if err == nil {
			inFlightValue = connGauge.GetGauge().GetValue()
		}
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Middleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// While request was in-flight, active connections should have been >= 1
	if inFlightValue < 1 {
		t.Errorf("expected active_connections >= 1 during request, got %v", inFlightValue)
	}
}

func TestResponseCaptureDefaultStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	rc := &responseCapture{ResponseWriter: rec, statusCode: http.StatusOK}

	// Write without calling WriteHeader should default to 200
	rc.Write([]byte("hello"))

	if rc.statusCode != http.StatusOK {
		t.Errorf("expected default status 200, got %d", rc.statusCode)
	}
}

func TestResponseCaptureExplicitStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	rc := &responseCapture{ResponseWriter: rec, statusCode: http.StatusOK}

	rc.WriteHeader(http.StatusNotFound)

	if rc.statusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rc.statusCode)
	}

	// Second call should not override
	rc.WriteHeader(http.StatusInternalServerError)
	if rc.statusCode != http.StatusNotFound {
		t.Errorf("expected status to stay 404, got %d", rc.statusCode)
	}
}
