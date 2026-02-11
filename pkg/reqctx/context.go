// Package reqctx provides request context utilities for timeout management,
// cancellation, and request ID tracking throughout the application.
package reqctx

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ContextKey is a type for context keys to avoid collisions
type ContextKey string

const (
	// RequestIDKey is the context key for request ID
	RequestIDKey ContextKey = "request_id"
	// StartTimeKey is the context key for request start time
	StartTimeKey ContextKey = "start_time"
	// UserAgentKey is the context key for user agent
	UserAgentKey ContextKey = "user_agent"
	// RemoteAddrKey is the context key for remote address
	RemoteAddrKey ContextKey = "remote_addr"
)

// TimeoutConfig holds configurable timeout durations
type TimeoutConfig struct {
	// UploadTimeout is the maximum duration for upload operations
	UploadTimeout time.Duration
	// DownloadTimeout is the maximum duration for download operations
	DownloadTimeout time.Duration
	// DefaultTimeout is the default timeout for general operations
	DefaultTimeout time.Duration
	// DatabaseTimeout is the timeout for database operations
	DatabaseTimeout time.Duration
	// StorageTimeout is the timeout for storage operations
	StorageTimeout time.Duration
}

// DefaultTimeoutConfig returns the default timeout configuration
func DefaultTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		UploadTimeout:   30 * time.Minute,  // Large files may take time
		DownloadTimeout: 30 * time.Minute,  // Large files may take time
		DefaultTimeout:  30 * time.Second,  // General API operations
		DatabaseTimeout: 10 * time.Second,  // DB operations
		StorageTimeout:  5 * time.Minute,   // Storage operations
	}
}

// global timeout config (can be overridden)
var globalConfig = DefaultTimeoutConfig()

// SetGlobalConfig sets the global timeout configuration
func SetGlobalConfig(cfg *TimeoutConfig) {
	if cfg != nil {
		globalConfig = cfg
	}
}

// GetGlobalConfig returns the current global timeout configuration
func GetGlobalConfig() *TimeoutConfig {
	return globalConfig
}

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context) context.Context {
	return context.WithValue(ctx, RequestIDKey, uuid.New().String())
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// WithStartTime adds the request start time to the context
func WithStartTime(ctx context.Context) context.Context {
	return context.WithValue(ctx, StartTimeKey, time.Now())
}

// GetStartTime retrieves the request start time from context
func GetStartTime(ctx context.Context) time.Time {
	if t, ok := ctx.Value(StartTimeKey).(time.Time); ok {
		return t
	}
	return time.Time{}
}

// GetElapsedTime returns the elapsed time since the request started
func GetElapsedTime(ctx context.Context) time.Duration {
	startTime := GetStartTime(ctx)
	if startTime.IsZero() {
		return 0
	}
	return time.Since(startTime)
}

// WithRemoteAddr adds the remote address to the context
func WithRemoteAddr(ctx context.Context, addr string) context.Context {
	return context.WithValue(ctx, RemoteAddrKey, addr)
}

// GetRemoteAddr retrieves the remote address from context
func GetRemoteAddr(ctx context.Context) string {
	if addr, ok := ctx.Value(RemoteAddrKey).(string); ok {
		return addr
	}
	return ""
}

// WithUserAgent adds the user agent to the context
func WithUserAgent(ctx context.Context, ua string) context.Context {
	return context.WithValue(ctx, UserAgentKey, ua)
}

// GetUserAgent retrieves the user agent from context
func GetUserAgent(ctx context.Context) string {
	if ua, ok := ctx.Value(UserAgentKey).(string); ok {
		return ua
	}
	return ""
}

// WithUploadTimeout returns a context with the upload timeout applied
func WithUploadTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, globalConfig.UploadTimeout)
}

// WithDownloadTimeout returns a context with the download timeout applied
func WithDownloadTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, globalConfig.DownloadTimeout)
}

// WithDefaultTimeout returns a context with the default timeout applied
func WithDefaultTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, globalConfig.DefaultTimeout)
}

// WithDatabaseTimeout returns a context with the database timeout applied
func WithDatabaseTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, globalConfig.DatabaseTimeout)
}

// WithStorageTimeout returns a context with the storage timeout applied
func WithStorageTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, globalConfig.StorageTimeout)
}

// WithCustomTimeout returns a context with a custom timeout
func WithCustomTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// EnrichContext enriches a context with request metadata from an HTTP request
func EnrichContext(ctx context.Context, r *http.Request) context.Context {
	ctx = WithRequestID(ctx)
	ctx = WithStartTime(ctx)
	ctx = WithRemoteAddr(ctx, r.RemoteAddr)
	ctx = WithUserAgent(ctx, r.UserAgent())
	return ctx
}

// IsContextCanceled returns true if the context has been canceled
func IsContextCanceled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// ContextError returns a descriptive error based on context state
func ContextError(ctx context.Context) error {
	return ctx.Err()
}

// CheckContext checks if context is still valid, returns error if canceled/timeout
func CheckContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
