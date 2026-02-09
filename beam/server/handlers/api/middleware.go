package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/tachRoutine/beamdrop-go/pkg/crypto"
	"github.com/tachRoutine/beamdrop-go/pkg/db"
	"github.com/tachRoutine/beamdrop-go/pkg/logger"
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
			sendAPIError(w, "AccessDenied", "Invalid or expired presigned URL", http.StatusForbidden)
			return
		}

		// Check Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			sendAPIError(w, "AccessDenied", "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Parse "Bearer <access_key_id>:<signature>" format
		if !strings.HasPrefix(authHeader, "Bearer ") {
			sendAPIError(w, "AccessDenied", "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		credentials := strings.TrimPrefix(authHeader, "Bearer ")
		parts := strings.SplitN(credentials, ":", 2)
		if len(parts) != 2 {
			sendAPIError(w, "AccessDenied", "Invalid credentials format", http.StatusUnauthorized)
			return
		}

		accessKeyID := parts[0]
		signature := parts[1]

		// Validate timestamp header
		timestamp := r.Header.Get("X-Beamdrop-Date")
		if timestamp == "" {
			sendAPIError(w, "AccessDenied", "Missing X-Beamdrop-Date header", http.StatusUnauthorized)
			return
		}

		if !crypto.IsTimestampValid(timestamp) {
			sendAPIError(w, "SignatureExpired", "Request timestamp is too old or in the future", http.StatusUnauthorized)
			return
		}

		// Look up API key
		apiKey, err := db.GetAPIKeyByAccessID(accessKeyID)
		if err != nil {
			logger.Error("Failed to look up API key: %v", err)
			sendAPIError(w, "InternalError", "Authentication error", http.StatusInternalServerError)
			return
		}

		if apiKey == nil {
			sendAPIError(w, "AccessDenied", "Invalid access key", http.StatusForbidden)
			return
		}

		// Verify signature
		// Note: TODO:  I will need to reconstruct the secret from a secure store
		// For now, we verify by regenerating the signature with the stored hash
		// This is a simplified approach - a full implementation would use proper HMAC verification
		expectedSignature := crypto.GenerateSignature(apiKey.SecretHash, r.Method, r.URL.Path, timestamp)
		if signature != expectedSignature {
			// For development/testing, also accept direct secret key verification
			if !crypto.VerifySignature(apiKey.SecretHash, r.Method, r.URL.Path, timestamp, signature) {
				sendAPIError(w, "SignatureMismatch", "Invalid signature", http.StatusForbidden)
				return
			}
		}

		// Update last used timestamp (async to not slow down request)
		go func() {
			if err := db.UpdateLastUsed(accessKeyID); err != nil {
				logger.Error("Failed to update last used: %v", err)
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
	return crypto.VerifyPresignedToken(apiKey.SecretHash, r.Method, bucket, key, expiresAt, token)
}
