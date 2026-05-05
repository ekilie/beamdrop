package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

// CSRFProtection implements the double-submit cookie pattern for CSRF protection.
// It sets a non-HttpOnly cookie with a random token, and requires state-changing
// requests to include that token in the X-CSRF-Token header.
// Safe methods (GET, HEAD, OPTIONS) are exempt.
// Requests with non-browser content types or API auth headers are exempt.
func CSRFProtection() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Safe methods are exempt
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				ensureCSRFCookie(w, r)
				next.ServeHTTP(w, r)
				return
			}

			// API clients using Authorization header (HMAC/Bearer) are exempt —
			// they don't use cookies so CSRF doesn't apply
			if r.Header.Get("Authorization") != "" {
				next.ServeHTTP(w, r)
				return
			}

			// Requests without the session cookie are exempt (no cookie = no CSRF risk)
			if _, err := r.Cookie("beamdrop_token"); err != nil {
				next.ServeHTTP(w, r)
				return
			}

			// Public share access endpoints are exempt (have their own password protection)
			if strings.HasPrefix(r.URL.Path, "/api/shares/access/") {
				next.ServeHTTP(w, r)
				return
			}

			// Validate CSRF token
			cookie, err := r.Cookie("beamdrop_csrf")
			if err != nil {
				http.Error(w, `{"error":"missing CSRF token"}`, http.StatusForbidden)
				return
			}

			headerToken := r.Header.Get("X-CSRF-Token")
			if headerToken == "" || headerToken != cookie.Value {
				http.Error(w, `{"error":"invalid CSRF token"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ensureCSRFCookie sets the CSRF cookie if not already present.
func ensureCSRFCookie(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie("beamdrop_csrf"); err == nil {
		return // cookie already exists
	}
	token := generateCSRFToken()
	http.SetCookie(w, &http.Cookie{
		Name:     "beamdrop_csrf",
		Value:    token,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
		// NOT HttpOnly — JavaScript needs to read this to include in headers
		HttpOnly: false,
		Secure:   r.TLS != nil,
	})
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
