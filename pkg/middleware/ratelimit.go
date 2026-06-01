// Package middleware provides HTTP middleware, including per-IP rate limiting.
package middleware

import (
	"log/slog"
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	apperrors "github.com/ekilie/beamdrop/pkg/errors"
)

// ---------------------------------------------------------------------------
// Token-bucket rate limiter (per-IP, in-memory, zero external deps)
// ---------------------------------------------------------------------------

// bucket is a token-bucket for a single client IP.
type bucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64   // tokens per second
	lastRefill time.Time // last time tokens were refilled
}

// allow consumes one token and returns whether the request is allowed.
func (b *bucket) allow(now time.Time) bool {
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens = math.Min(b.maxTokens, b.tokens+elapsed*b.refillRate)
	b.lastRefill = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// retryAfter returns how long until at least 1 token is available.
func (b *bucket) retryAfter() time.Duration {
	if b.tokens >= 1 {
		return 0
	}
	need := 1.0 - b.tokens
	secs := need / b.refillRate
	return time.Duration(math.Ceil(secs)) * time.Second
}

// ---------------------------------------------------------------------------
// RateLimiterConfig configures the rate limiting behaviour.
// ---------------------------------------------------------------------------

// RateLimiterConfig holds the rates for different endpoint tiers.
type RateLimiterConfig struct {
	// General rate: requests per minute for normal endpoints.
	GeneralRate int
	// AuthRate: requests per minute for auth endpoints (e.g. /auth/login).
	AuthRate int
	// UploadRate: requests per minute for upload endpoints.
	UploadRate int
	// Enabled can be set to false to skip rate limiting entirely.
	Enabled bool
	// TrustedProxies is a list of trusted proxy IPs/CIDRs.
	// Only trust X-Forwarded-For/X-Real-IP from these sources.
	TrustedProxies []*net.IPNet
}

// DefaultRateLimiterConfig returns sensible defaults.
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		GeneralRate: 100,
		AuthRate:    5,
		UploadRate:  10,
		Enabled:     true,
	}
}

// ---------------------------------------------------------------------------
// RateLimiter is an HTTP middleware that enforces per-IP token-bucket limits.
// ---------------------------------------------------------------------------

// RateLimiter enforces per-IP, per-tier rate limits.
type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientBuckets
	config  RateLimiterConfig
	stopCh  chan struct{}
}

// ParseTrustedProxies parses a comma-separated list of IPs/CIDRs into net.IPNet entries.
func ParseTrustedProxies(raw string) []*net.IPNet {
	if raw == "" {
		return nil
	}
	var result []*net.IPNet
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		// If it's a plain IP, convert to /32 or /128
		if !strings.Contains(entry, "/") {
			ip := net.ParseIP(entry)
			if ip == nil {
				slog.Warn("Invalid trusted proxy IP, skipping", "ip", entry)
				continue
			}
			if ip.To4() != nil {
				entry += "/32"
			} else {
				entry += "/128"
			}
		}
		_, cidr, err := net.ParseCIDR(entry)
		if err != nil {
			slog.Warn("Invalid trusted proxy CIDR, skipping", "cidr", entry, "error", err)
			continue
		}
		result = append(result, cidr)
	}
	return result
}

// clientBuckets holds the three tier buckets for a single IP.
type clientBuckets struct {
	general  bucket
	auth     bucket
	upload   bucket
	lastSeen time.Time
}

// NewRateLimiter creates a RateLimiter and starts a background goroutine
// that evicts stale entries every 5 minutes.
func NewRateLimiter(cfg RateLimiterConfig) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*clientBuckets),
		config:  cfg,
		stopCh:  make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

// Close stops the background cleanup goroutine.
func (rl *RateLimiter) Close() {
	close(rl.stopCh)
}

// Middleware returns an http.Handler middleware that enforces rate limits.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	if !rl.config.Enabled {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for static assets — a single page load can
		// trigger dozens of JS/CSS chunk requests simultaneously.
		if isStaticAsset(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		ip := extractIP(r, rl.config.TrustedProxies)
		tier := classifyRequest(r)

		rl.mu.Lock()
		cb := rl.getOrCreate(ip)
		now := time.Now()
		cb.lastSeen = now

		var b *bucket
		switch tier {
		case tierAuth:
			b = &cb.auth
		case tierUpload:
			b = &cb.upload
		default:
			b = &cb.general
		}

		allowed := b.allow(now)
		retry := b.retryAfter()
		rl.mu.Unlock()

		if !allowed {
			slog.Warn("Rate limit exceeded",
				"ip", ip,
				"tier", tier.String(),
				"path", r.URL.Path,
				"retry_after", retry,
			)

			appErr := apperrors.RateLimitExceeded(retry)
			appErr.WriteHTTPResponse(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ---------------------------------------------------------------------------
// Internals
// ---------------------------------------------------------------------------

type tier int

const (
	tierGeneral tier = iota
	tierAuth
	tierUpload
)

func (t tier) String() string {
	switch t {
	case tierAuth:
		return "auth"
	case tierUpload:
		return "upload"
	default:
		return "general"
	}
}

// classifyRequest maps a request to a rate-limit tier.
func classifyRequest(r *http.Request) tier {
	path := r.URL.Path

	// Auth endpoints – strictest limits
	if strings.HasPrefix(path, "/auth/login") {
		return tierAuth
	}

	// Upload endpoints
	if path == "/upload" && (r.Method == http.MethodPost || r.Method == http.MethodPut) {
		return tierUpload
	}
	// S3-like PUT object
	if strings.HasPrefix(path, "/api/v1/buckets/") && r.Method == http.MethodPut {
		return tierUpload
	}

	return tierGeneral
}

// getOrCreate returns the clientBuckets for ip, creating one if needed.
// Caller MUST hold rl.mu.
func (rl *RateLimiter) getOrCreate(ip string) *clientBuckets {
	cb, ok := rl.clients[ip]
	if ok {
		return cb
	}

	now := time.Now()
	cb = &clientBuckets{
		general:  newBucket(rl.config.GeneralRate),
		auth:     newBucket(rl.config.AuthRate),
		upload:   newBucket(rl.config.UploadRate),
		lastSeen: now,
	}
	rl.clients[ip] = cb
	return cb
}

// newBucket creates a bucket for the given requests-per-minute rate.
func newBucket(rpm int) bucket {
	rps := float64(rpm) / 60.0
	return bucket{
		tokens:     float64(rpm), // start full
		maxTokens:  float64(rpm), // burst = rate (1 minute's worth)
		refillRate: rps,
		lastRefill: time.Now(),
	}
}

// extractIP returns the client IP from the request.
// Only trusts X-Forwarded-For and X-Real-IP headers when the direct connection
// comes from a trusted proxy IP. Otherwise, uses RemoteAddr.
func extractIP(r *http.Request, trustedProxies []*net.IPNet) string {
	// Get the direct connection IP
	directIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		directIP = r.RemoteAddr
	}

	// Only trust forwarded headers if the direct connection is from a trusted proxy
	if isTrustedProxy(directIP, trustedProxies) {
		// Try X-Forwarded-For first (may contain comma-separated list)
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.SplitN(xff, ",", 2)
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}

		// Try X-Real-IP
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return strings.TrimSpace(xri)
		}
	}

	return directIP
}

// isTrustedProxy checks if an IP is in the trusted proxies list.
func isTrustedProxy(ipStr string, trustedProxies []*net.IPNet) bool {
	if len(trustedProxies) == 0 {
		return false
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, cidr := range trustedProxies {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// isStaticAsset returns true for paths that serve embedded frontend assets
// and should not count against API rate limits.
func isStaticAsset(path string) bool {
	return strings.HasPrefix(path, "/assets/") ||
		path == "/favicon.ico" ||
		path == "/index.html"
}

// cleanup periodically removes stale client entries (unseen for >10 min).
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stopCh:
			return
		case <-ticker.C:
			rl.mu.Lock()
			cutoff := time.Now().Add(-10 * time.Minute)
			for ip, cb := range rl.clients {
				if cb.lastSeen.Before(cutoff) {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}
