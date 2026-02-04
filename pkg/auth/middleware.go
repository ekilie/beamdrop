package auth

import (
	"net/http"
	"strings"
)

// PublicRoutes are routes that don't require authentication
var PublicRoutes = []string{
	"/",            // Landing page (will check auth status)
	"/auth/login",  // Login endpoint
	"/auth/status", // Auth status check
	"/health",      // Health check
	"/ready",       // Readiness check
}

// StaticPrefixes are static asset prefixes that don't require authentication
var StaticPrefixes = []string{
	"/assets/",
	"/static/",
}

// AuthMiddleware handles authentication for protected routes
type AuthMiddleware struct {
	passwordService *PasswordService
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(ps *PasswordService) *AuthMiddleware {
	return &AuthMiddleware{
		passwordService: ps,
	}
}

// IsPublicRoute checks if the given path is a public route
func (m *AuthMiddleware) IsPublicRoute(path string) bool {
	// Check exact matches
	for _, route := range PublicRoutes {
		if path == route {
			return true
		}
	}

	// Check static prefixes
	for _, prefix := range StaticPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// RequireAuth wraps a handler with authentication checking
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If auth is not enabled, pass through
		if !m.passwordService.IsEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		// Check if it's a public route
		if m.IsPublicRoute(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Check for Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			// Extract Bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				token := parts[1]
				if m.passwordService.ValidateToken(token) {
					next.ServeHTTP(w, r)
					return
				}
			}
		}

		// Check for token in cookie
		cookie, err := r.Cookie("beamdrop_token")
		if err == nil && cookie.Value != "" {
			if m.passwordService.ValidateToken(cookie.Value) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Unauthorized
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "unauthorized", "message": "Authentication required"}`))
	})
}

// Middleware returns the middleware function for use in the server
func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return m.RequireAuth(next)
}
