package config

import "testing"

func TestDefaultPorts(t *testing.T) {
	if len(DefaultPorts) == 0 {
		t.Fatal("expected non-empty DefaultPorts list")
	}
	if DefaultPorts[0] != 7777 {
		t.Fatalf("expected first default port 7777, got %d", DefaultPorts[0])
	}
}

func TestIsPortAvailable_InvalidPort(t *testing.T) {
	if IsPortAvailable(-1) {
		t.Fatal("negative port should not be available")
	}
	if IsPortAvailable(99999) {
		t.Fatal("invalid port should not be available")
	}
}
