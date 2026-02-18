package api

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ekilie/beamdrop/pkg/crypto"
	"github.com/ekilie/beamdrop/pkg/db"
	"github.com/ekilie/beamdrop/pkg/errors"
)

// APIAuthMiddleware handles API key authentication
type APIAuthMiddleware struct {
	enabled bool
}

// NewAPIAuthMiddleware creates a new API auth middleware
func NewAPIAuthMiddleware(enabled bool) *APIAuthMiddleware {
	return &APIAuthMiddleware{enabled: enabled}
}

// Middleware wraps an HTTP handler with API key authentication
func (m *APIAuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled {
			// If API auth is disabled, allow all requests (for development)
			next.ServeHTTP(w, r)
			return
		}

		// Check for presigned URL token
		if token := r.URL.Query().Get("token"); token != "" {
			if m.verifyPresignedToken(r, token) {
				next.ServeHTTP(w, r)
				return
			}
			errors.Forbidden("Invalid or expired presigned URL").WriteHTTPResponse(w)
			return
		}

		// Check Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			errors.Unauthorized("Missing Authorization header").WriteHTTPResponse(w)
			return
		}

		// Parse "Bearer <access_key_id>:<signature>" format
		if !strings.HasPrefix(authHeader, "Bearer ") {
			errors.Unauthorized("Invalid Authorization header format").WriteHTTPResponse(w)
			return
		}

		credentials := strings.TrimPrefix(authHeader, "Bearer ")
		parts := strings.SplitN(credentials, ":", 2)
		if len(parts) != 2 {
			errors.Unauthorized("Invalid credentials format").WriteHTTPResponse(w)
			return
		}

		accessKeyID := parts[0]
		signature := parts[1]

		// Validate timestamp header
		timestamp := r.Header.Get("X-Beamdrop-Date")
		if timestamp == "" {
			errors.Unauthorized("Missing X-Beamdrop-Date header").WriteHTTPResponse(w)
			return
		}

		if !crypto.IsTimestampValid(timestamp) {
			errors.New(errors.CodeTokenExpired, errors.CategoryAuth, "Request timestamp is too old or in the future", http.StatusUnauthorized).WriteHTTPResponse(w)
			return
		}

		// Look up API key
		apiKey, err := db.GetAPIKeyByAccessID(accessKeyID)
		if err != nil {
			slog.Error("Failed to look up API key", "error", err)
			errors.InternalError("Authentication error").WithCause(err).WriteHTTPResponse(w)
			return
		}

		if apiKey == nil {
			errors.InvalidAPIKey().WriteHTTPResponse(w)
			return
		}

		// Verify signature using the stored secret key
		if !crypto.VerifySignature(apiKey.SecretKey, r.Method, r.URL.Path, timestamp, signature) {
			errors.Forbidden("Invalid signature").WriteHTTPResponse(w)
			return
		}

		// Update last used timestamp (async to not slow down request)
		go func() {
			if err := db.UpdateLastUsed(accessKeyID); err != nil {
				slog.Error("Failed to update last used", "error", err)
			}
		}()

		// TODO: Check permissions against requested action

		next.ServeHTTP(w, r)
	})
}

func (m *APIAuthMiddleware) verifyPresignedToken(r *http.Request, token string) bool {
	expiresStr := r.URL.Query().Get("expires")
	if expiresStr == "" {
		return false
	}

	// Parse expiration timestamp
	var expiresAt time.Time
	if _, err := time.Parse(time.RFC3339, expiresStr); err != nil {
		// Try Unix timestamp
		if ts, err := time.Parse("20060102T150405Z", expiresStr); err == nil {
			expiresAt = ts
		} else {
			return false
		}
	} else {
		expiresAt, _ = time.Parse(time.RFC3339, expiresStr)
	}

	// Check if expired
	if time.Now().After(expiresAt) {
		return false
	}

	// Extract bucket and key from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/buckets/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return false
	}

	bucket := parts[0]
	key := parts[1]

	// Get the access key from query params
	accessKeyID := r.URL.Query().Get("access_key")
	if accessKeyID == "" {
		return false
	}

	// Look up API key
	apiKey, err := db.GetAPIKeyByAccessID(accessKeyID)
	if err != nil || apiKey == nil {
		return false
	}

	// Verify token
	return crypto.VerifyPresignedToken(apiKey.SecretKey, r.Method, bucket, key, expiresAt, token)
}
