package reqctx

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultTimeoutConfig(t *testing.T) {
	cfg := DefaultTimeoutConfig()
	if cfg.UploadTimeout != 30*time.Minute {
		t.Fatalf("expected 30m upload timeout, got %v", cfg.UploadTimeout)
	}
	if cfg.DefaultTimeout != 30*time.Second {
		t.Fatalf("expected 30s default timeout, got %v", cfg.DefaultTimeout)
	}
	if cfg.DatabaseTimeout != 10*time.Second {
		t.Fatalf("expected 10s database timeout, got %v", cfg.DatabaseTimeout)
	}
}

func TestSetGlobalConfig_Nil(t *testing.T) {
	original := GetGlobalConfig()
	SetGlobalConfig(nil)
	if GetGlobalConfig() != original {
		t.Fatal("setting nil config should not change global")
	}
}

func TestSetGlobalConfig_Override(t *testing.T) {
	cfg := &TimeoutConfig{DefaultTimeout: 5 * time.Second}
	SetGlobalConfig(cfg)
	if GetGlobalConfig().DefaultTimeout != 5*time.Second {
		t.Fatal("global config should be updated")
	}
	SetGlobalConfig(DefaultTimeoutConfig())
}

func TestWithRequestID(t *testing.T) {
	ctx := WithRequestID(context.Background())
	id := GetRequestID(ctx)
	if id == "" {
		t.Fatal("expected non-empty request ID")
	}
}

func TestWithExistingRequestID(t *testing.T) {
	ctx := WithExistingRequestID(context.Background(), "my-id")
	if GetRequestID(ctx) != "my-id" {
		t.Fatalf("expected 'my-id', got %q", GetRequestID(ctx))
	}
}

func TestWithExistingRequestID_Empty(t *testing.T) {
	ctx := WithExistingRequestID(context.Background(), "")
	id := GetRequestID(ctx)
	if id == "" {
		t.Fatal("expected non-empty request ID for empty input")
	}
}

func TestGetRequestID_Empty(t *testing.T) {
	if GetRequestID(context.Background()) != "" {
		t.Fatal("expected empty string for context without request ID")
	}
}

func TestStartTime(t *testing.T) {
	ctx := WithStartTime(context.Background())
	start := GetStartTime(ctx)
	if start.IsZero() {
		t.Fatal("expected non-zero start time")
	}
}

func TestGetStartTime_Empty(t *testing.T) {
	if !GetStartTime(context.Background()).IsZero() {
		t.Fatal("expected zero time for context without start time")
	}
}

func TestGetElapsedTime(t *testing.T) {
	ctx := WithStartTime(context.Background())
	elapsed := GetElapsedTime(ctx)
	if elapsed < 0 {
		t.Fatal("elapsed time should be non-negative")
	}
}

func TestGetElapsedTime_NoStartTime(t *testing.T) {
	if GetElapsedTime(context.Background()) != 0 {
		t.Fatal("expected 0 elapsed without start time")
	}
}

func TestRemoteAddr(t *testing.T) {
	ctx := WithRemoteAddr(context.Background(), "192.168.1.1:8080")
	if GetRemoteAddr(ctx) != "192.168.1.1:8080" {
		t.Fatalf("expected '192.168.1.1:8080', got %q", GetRemoteAddr(ctx))
	}
}

func TestRemoteAddr_Empty(t *testing.T) {
	if GetRemoteAddr(context.Background()) != "" {
		t.Fatal("expected empty string")
	}
}

func TestUserAgent(t *testing.T) {
	ctx := WithUserAgent(context.Background(), "test-agent/1.0")
	if GetUserAgent(ctx) != "test-agent/1.0" {
		t.Fatalf("expected 'test-agent/1.0', got %q", GetUserAgent(ctx))
	}
}

func TestUserAgent_Empty(t *testing.T) {
	if GetUserAgent(context.Background()) != "" {
		t.Fatal("expected empty string")
	}
}

func TestGetAccessKeyID(t *testing.T) {
	ctx := context.WithValue(context.Background(), AccessKeyIDKey, "BDK_abcd1234")
	if GetAccessKeyID(ctx) != "BDK_abcd1234" {
		t.Fatalf("expected 'BDK_abcd1234', got %q", GetAccessKeyID(ctx))
	}
}

func TestGetAccessKeyID_Empty(t *testing.T) {
	if GetAccessKeyID(context.Background()) != "" {
		t.Fatal("expected empty string")
	}
}

func TestTimeoutFunctions(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func(context.Context) (context.Context, context.CancelFunc)
	}{
		{"Upload", WithUploadTimeout},
		{"Download", WithDownloadTimeout},
		{"Default", WithDefaultTimeout},
		{"Database", WithDatabaseTimeout},
		{"Storage", WithStorageTimeout},
		{"Custom", func(ctx context.Context) (context.Context, context.CancelFunc) {
			return WithCustomTimeout(ctx, 1*time.Second)
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := tc.fn(ctx)
			defer cancel()
			if _, ok := ctx.Deadline(); !ok {
				t.Fatal("expected deadline to be set")
			}
		})
	}
}

func TestEnrichContext(t *testing.T) {
	r := httptest.NewRequest("GET", "/test", nil)
	r.RemoteAddr = "10.0.0.1:5555"
	r.Header.Set("User-Agent", "test-agent")
	r.Header.Set("X-Request-ID", "req-123")

	ctx := EnrichContext(context.Background(), r)

	if GetRequestID(ctx) != "req-123" {
		t.Fatalf("expected 'req-123', got %q", GetRequestID(ctx))
	}
	if GetRemoteAddr(ctx) != "10.0.0.1:5555" {
		t.Fatalf("expected '10.0.0.1:5555', got %q", GetRemoteAddr(ctx))
	}
	if GetUserAgent(ctx) != "test-agent" {
		t.Fatalf("expected 'test-agent', got %q", GetUserAgent(ctx))
	}
	if GetStartTime(ctx).IsZero() {
		t.Fatal("expected non-zero start time")
	}
}

func TestIsContextCanceled(t *testing.T) {
	if IsContextCanceled(context.Background()) {
		t.Fatal("background context should not be canceled")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !IsContextCanceled(ctx) {
		t.Fatal("canceled context should be detected")
	}
}

func TestContextError(t *testing.T) {
	if ContextError(context.Background()) != nil {
		t.Fatal("background context should have no error")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if ContextError(ctx) == nil {
		t.Fatal("canceled context should have error")
	}
}

func TestCheckContext(t *testing.T) {
	if CheckContext(context.Background()) != nil {
		t.Fatal("background context should be valid")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if CheckContext(ctx) == nil {
		t.Fatal("canceled context should return error")
	}
}
