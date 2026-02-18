// Package middleware provides HTTP middleware, including per-IP rate limiting.
package middleware

import (
	"net/http"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	// AllowedOrigins is a list of origins allowed for CORS
	// If empty, CORS is disabled (most secure for local file sharing)
	AllowedOrigins []string
	// AllowCredentials indicates whether credentials are allowed
	AllowCredentials bool
}

// DefaultCORSConfig returns a secure default CORS configuration
// By default, CORS is disabled for security
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{}, // Empty = no CORS by default
		AllowCredentials: true,
	}
}

// CORS middleware handles Cross-Origin Resource Sharing
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// If no allowed origins configured, don't set CORS headers
			// This is the most secure option for a local file server
			if len(config.AllowedOrigins) == 0 {
				// For preflight requests, reject them
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range config.AllowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				// Set CORS headers
				if origin != "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
				
				if config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
				w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
				w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				if allowed {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(http.StatusForbidden)
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IsOriginAllowed checks if an origin is in the allowed list
func IsOriginAllowed(origin string, allowedOrigins []string) bool {
	if len(allowedOrigins) == 0 {
		return false
	}

	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		// Origins are case-sensitive per CORS spec
		if allowed == origin {
			return true
		}
	}
	return false
}
