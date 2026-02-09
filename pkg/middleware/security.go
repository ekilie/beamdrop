package middleware

import (
	"net/http"
)

// SecurityHeaders adds security-related HTTP headers to all responses
func SecurityHeaders(enableHSTS bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent clickjacking attacks
			w.Header().Set("X-Frame-Options", "DENY")
			
			// Prevent MIME type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")
			
			// Enable browser XSS protection
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			
			// Referrer policy for privacy
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			
			// Content Security Policy
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src 'self' ws: wss:")
			
			// Only add HSTS if TLS is enabled
			if enableHSTS && r.TLS != nil {
				// Enforce HTTPS for 1 year
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			
			next.ServeHTTP(w, r)
		})
	}
}
