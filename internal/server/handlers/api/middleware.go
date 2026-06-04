package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ekilie/beamdrop/pkg/crypto"
	"github.com/ekilie/beamdrop/pkg/db"
	"github.com/ekilie/beamdrop/pkg/errors"
	"github.com/ekilie/beamdrop/pkg/reqctx"
)

// lastUsedCh is a bounded channel for async last-used updates to avoid
// an unbounded goroutine per authenticated request.
var lastUsedCh = make(chan string, 1000)

func init() {
	go lastUsedLoop()
}

func lastUsedLoop() {
	for id := range lastUsedCh {
		if err := db.UpdateLastUsed(id); err != nil {
			slog.Error("Failed to update last used", "error", err)
		}
	}
}

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

		// Verify signature using the stored secret key (decrypt first)
		secretKey, err := db.DecryptSecretKey(apiKey)
		if err != nil {
			slog.Error("Failed to decrypt API secret key", "error", err)
			errors.InternalError("Authentication error").WriteHTTPResponse(w)
			return
		}

		if !crypto.VerifySignature(secretKey, r.Method, r.URL.Path, timestamp, signature) {
			errors.Forbidden("Invalid signature").WriteHTTPResponse(w)
			return
		}

		if !isMethodAllowed(apiKey.Permissions, r.Method) {
			errors.Forbidden("API key does not have permission for this operation").WriteHTTPResponse(w)
			return
		}

		if apiKey.BucketScope != "" {
			bucket, ok, err := bucketForRequest(r)
			if err != nil {
				errors.InvalidRequest("Invalid request body").WriteHTTPResponse(w)
				return
			}
			if !ok {
				errors.Forbidden("API key is restricted to a specific bucket").WriteHTTPResponse(w)
				return
			}
			if bucket != apiKey.BucketScope {
				errors.Forbidden("API key is not allowed for this bucket").WriteHTTPResponse(w)
				return
			}
		}

		// Update last used timestamp (async to not slow down request)
		select {
		case lastUsedCh <- accessKeyID:
		default:
			slog.Warn("Last-used update dropped, queue full")
		}

		// Store the authenticated access key ID in request context
		ctx := context.WithValue(r.Context(), reqctx.AccessKeyIDKey, accessKeyID)
		next.ServeHTTP(w, r.WithContext(ctx))
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

	// Decrypt secret key for verification
	secretKey, err := db.DecryptSecretKey(apiKey)
	if err != nil {
		return false
	}

	// Verify token
	return crypto.VerifyPresignedToken(secretKey, r.Method, bucket, key, expiresAt, token)
}

func isMethodAllowed(permissions, method string) bool {
	canRead, canWrite := parsePermissions(permissions)
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return canRead
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return canWrite
	default:
		return false
	}
}

func parsePermissions(raw string) (bool, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return true, true
	}

	canRead := false
	canWrite := false
	for _, part := range strings.Split(trimmed, ",") {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "read":
			canRead = true
		case "write":
			canWrite = true
		}
	}

	return canRead, canWrite
}

func bucketForRequest(r *http.Request) (string, bool, error) {
	if strings.HasPrefix(r.URL.Path, "/api/v1/buckets/") {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/buckets/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			return parts[0], true, nil
		}
		return "", false, nil
	}

	if strings.HasPrefix(r.URL.Path, "/api/v1/presign") && r.Method == http.MethodPost {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return "", false, err
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		var req struct {
			Bucket string `json:"bucket"`
		}
		if len(bytes.TrimSpace(body)) == 0 {
			return "", false, nil
		}
		if err := json.Unmarshal(body, &req); err != nil {
			return "", false, err
		}
		if strings.TrimSpace(req.Bucket) == "" {
			return "", false, nil
		}
		return req.Bucket, true, nil
	}

	return "", false, nil
}
