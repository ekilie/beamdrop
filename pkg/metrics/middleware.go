package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// responseCapture wraps http.ResponseWriter to capture the status code.
type responseCapture struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rc *responseCapture) WriteHeader(code int) {
	if !rc.written {
		rc.statusCode = code
		rc.written = true
	}
	rc.ResponseWriter.WriteHeader(code)
}

func (rc *responseCapture) Write(b []byte) (int, error) {
	if !rc.written {
		rc.statusCode = http.StatusOK
		rc.written = true
	}
	return rc.ResponseWriter.Write(b)
}

// NormalizePath reduces high-cardinality URL paths to a stable label
// value so the metrics don't explode with unique paths.
func NormalizePath(path string) string {
	// Keep well-known routes as-is
	switch path {
	case "/", "/health", "/health/live", "/health/ready", "/health/startup",
		"/ready", "/stats", "/ws/stats", "/files", "/download", "/upload",
		"/move", "/trash", "/copy", "/mkdir", "/rename", "/write",
		"/search", "/star", "/starred", "/auth/login", "/auth/logout",
		"/auth/status", "/api/logs", "/api/shares", "/api/shares/list",
		"/api/shares/delete", "/api/v1/buckets", "/api/v1/keys",
		"/metrics":
		return path
	}

	// Collapse /assets/*, /static/*, /share/* prefixes
	for _, prefix := range []string{"/assets/", "/static/", "/share/"} {
		if strings.HasPrefix(path, prefix) {
			return prefix + "{file}"
		}
	}

	// Collapse /api/shares/access/<id>
	if strings.HasPrefix(path, "/api/shares/access/") {
		return "/api/shares/access/{id}"
	}

	// Collapse /api/v1/buckets/<name>/...
	if strings.HasPrefix(path, "/api/v1/buckets/") {
		return "/api/v1/buckets/{bucket}"
	}

	return path
}

// Middleware returns an http.Handler middleware that records per-request
// Prometheus metrics: requests_total, request_duration_seconds, and
// active_connections.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ActiveConnections.Inc()
		defer ActiveConnections.Dec()

		start := time.Now()
		rc := &responseCapture{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rc, r)

		elapsed := time.Since(start).Seconds()
		status := strconv.Itoa(rc.statusCode)
		path := NormalizePath(r.URL.Path)

		RequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		RequestDurationSeconds.WithLabelValues(r.Method, path, status).Observe(elapsed)
	})
}
