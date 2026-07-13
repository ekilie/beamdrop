package server

import (
	"net/http"
	"testing"

	"github.com/ekilie/beamdrop/config"
)

func TestGetCORSConfig_EmptyOrigins(t *testing.T) {
	s := &Server{
		flags: config.Flags{AllowedOrigins: ""},
	}
	cfg := s.getCORSConfig()
	if len(cfg.AllowedOrigins) != 0 {
		t.Fatal("expected default CORS config with empty origins")
	}
}

func TestGetCORSConfig_CustomOrigins(t *testing.T) {
	s := &Server{
		flags: config.Flags{AllowedOrigins: "https://example.com, https://other.com"},
	}
	cfg := s.getCORSConfig()
	if len(cfg.AllowedOrigins) != 2 {
		t.Fatalf("expected 2 origins, got %d", len(cfg.AllowedOrigins))
	}
	if cfg.AllowedOrigins[0] != "https://example.com" {
		t.Fatalf("expected first origin, got %q", cfg.AllowedOrigins[0])
	}
	if cfg.AllowedOrigins[1] != "https://other.com" {
		t.Fatalf("expected second origin, got %q", cfg.AllowedOrigins[1])
	}
}

func TestGetAllowedOrigins_NilWhenEmpty(t *testing.T) {
	s := &Server{
		flags: config.Flags{AllowedOrigins: ""},
	}
	origins := s.getAllowedOrigins()
	if origins != nil {
		t.Fatal("expected nil when empty")
	}
}

func TestGetAllowedOrigins_Parses(t *testing.T) {
	s := &Server{
		flags: config.Flags{AllowedOrigins: "http://a.com, http://b.com"},
	}
	origins := s.getAllowedOrigins()
	if len(origins) != 2 {
		t.Fatalf("expected 2 origins, got %d", len(origins))
	}
	if origins[0] != "http://a.com" {
		t.Fatalf("expected first origin, got %q", origins[0])
	}
}

func TestNewUpgrader_NoOrigins_AllowsSameOrigin(t *testing.T) {
	upgrader := newUpgrader(nil)
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "http://example.com")
	r.Host = "example.com"
	if !upgrader.CheckOrigin(r) {
		t.Fatal("expected same-origin request to be allowed")
	}
}

func TestNewUpgrader_NoOrigins_RejectsCrossOrigin(t *testing.T) {
	upgrader := newUpgrader(nil)
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "http://evil.com")
	r.Host = "example.com"
	if upgrader.CheckOrigin(r) {
		t.Fatal("expected cross-origin request to be rejected")
	}
}

func TestNewUpgrader_NoOrigin_Allows(t *testing.T) {
	upgrader := newUpgrader(nil)
	r, _ := http.NewRequest("GET", "/", nil)
	if !upgrader.CheckOrigin(r) {
		t.Fatal("expected request without origin to be allowed")
	}
}

func TestNewUpgrader_WildcardAllowsAll(t *testing.T) {
	upgrader := newUpgrader([]string{"*"})
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "http://evil.com")
	r.Host = "example.com"
	if !upgrader.CheckOrigin(r) {
		t.Fatal("expected wildcard to allow all origins")
	}
}

func TestNewUpgrader_ExplicitOrigins_Allows(t *testing.T) {
	upgrader := newUpgrader([]string{"http://allowed.com"})
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "http://allowed.com")
	r.Host = "example.com"
	if !upgrader.CheckOrigin(r) {
		t.Fatal("expected allowed origin to pass")
	}
}

func TestNewUpgrader_ExplicitOrigins_Rejects(t *testing.T) {
	upgrader := newUpgrader([]string{"http://allowed.com"})
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "http://evil.com")
	r.Host = "example.com"
	if upgrader.CheckOrigin(r) {
		t.Fatal("expected disallowed origin to be rejected")
	}
}

func TestComputeStorageUsage_WithCap(t *testing.T) {
	s := computeStorageUsage(30, 100, 70, 100)
	if s.UsedBytes != 30 {
		t.Fatalf("expected 30 used, got %d", s.UsedBytes)
	}
	if s.AllocatedBytes != 100 {
		t.Fatalf("expected 100 allocated, got %d", s.AllocatedBytes)
	}
	if s.AvailableBytes != 70 {
		t.Fatalf("expected 70 available, got %d", s.AvailableBytes)
	}
	if s.UsagePercent != 30.0 {
		t.Fatalf("expected 30%% usage, got %f", s.UsagePercent)
	}
}

func TestComputeStorageUsage_AtCap(t *testing.T) {
	s := computeStorageUsage(100, 200, 100, 100)
	if s.AvailableBytes != 0 {
		t.Fatalf("expected 0 available, got %d", s.AvailableBytes)
	}
	if s.UsagePercent != 100.0 {
		t.Fatalf("expected 100%% usage, got %f", s.UsagePercent)
	}
}

func TestComputeStorageUsage_OverCap(t *testing.T) {
	s := computeStorageUsage(150, 200, 50, 100)
	if s.AvailableBytes != 0 {
		t.Fatalf("expected 0 available, got %d", s.AvailableBytes)
	}
	if s.UsagePercent != 100.0 {
		t.Fatalf("expected 100%% usage, got %f", s.UsagePercent)
	}
}

func TestComputeStorageUsage_NoCap(t *testing.T) {
	s := computeStorageUsage(30, 100, 70, 0)
	if s.AllocatedBytes != 0 {
		t.Fatalf("expected 0 allocated, got %d", s.AllocatedBytes)
	}
	if s.AvailableBytes != 70 {
		t.Fatalf("expected 70 available, got %d", s.AvailableBytes)
	}
	if s.UsagePercent != 30.0 {
		t.Fatalf("expected 30%% usage, got %f", s.UsagePercent)
	}
}

func TestComputeStorageUsage_NoCapZeroTotal(t *testing.T) {
	s := computeStorageUsage(0, 0, 0, 0)
	if s.UsagePercent != 0.0 {
		t.Fatalf("expected 0%% usage, got %f", s.UsagePercent)
	}
}
