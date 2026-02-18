// Package reqctx provides request context utilities.
// This file contains HTTP middleware for context propagation.
package reqctx

import (
	"context"
	"log/slog"
	"net/http"
)

// Middleware returns an HTTP middleware that enriches the request context
// with request metadata and handles client disconnection gracefully.
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Enrich the context with request metadata
			ctx := EnrichContext(r.Context(), r)

			// Create a new request with the enriched context
			r = r.WithContext(ctx)

			// Propagate the request ID back in the response header
			requestID := GetRequestID(ctx)
			w.Header().Set(RequestIDHeader, requestID)

			// Log request start
			slog.Debug("Request started", "requestID", requestID, "method", r.Method, "path", r.URL.Path)

			// Wrap response writer to detect when client disconnects
			wrapped := &responseWriterWrapper{
				ResponseWriter: w,
				written:        false,
			}

			// Serve the request
			next.ServeHTTP(wrapped, r)

			// Log request completion
			elapsed := GetElapsedTime(ctx)
			slog.Debug("Request completed", "requestID", requestID, "method", r.Method, "path", r.URL.Path, "elapsed", elapsed)
		})
	}
}

// TimeoutMiddleware returns an HTTP middleware that applies a timeout to the request context
func TimeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Apply default timeout unless it's an upload/download
		var ctx = r.Context()
		var cancel context.CancelFunc

		// Check if this is a file operation that needs extended timeout
		if isUploadRequest(r) {
			ctx, cancel = WithUploadTimeout(ctx)
		} else if isDownloadRequest(r) {
			ctx, cancel = WithDownloadTimeout(ctx)
		} else {
			ctx, cancel = WithDefaultTimeout(ctx)
		}
		defer cancel()

		// Create a new request with the timeout context
		r = r.WithContext(ctx)

		// Serve the request
		next.ServeHTTP(w, r)
	})
}

// isUploadRequest checks if the request is an upload operation
func isUploadRequest(r *http.Request) bool {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		return false
	}

	path := r.URL.Path
	return path == "/upload" ||
		(len(path) > 14 && path[:14] == "/api/v1/buckets" && r.Method == http.MethodPut)
}

// isDownloadRequest checks if the request is a download operation
func isDownloadRequest(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}

	path := r.URL.Path
	return path == "/download" ||
		(len(path) > 14 && path[:14] == "/api/v1/buckets")
}

// responseWriterWrapper wraps http.ResponseWriter to track if response was written
type responseWriterWrapper struct {
	http.ResponseWriter
	written bool
	status  int
}

func (w *responseWriterWrapper) WriteHeader(status int) {
	w.status = status
	w.written = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	if !w.written {
		w.status = http.StatusOK
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}

// Flush implements http.Flusher if the underlying writer supports it
func (w *responseWriterWrapper) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
