package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes uint64
		want  string
	}{
		{0, "0.0 B"},
		{500, "500.0 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
		{1125899906842624, "1.0 PB"},
	}

	for _, tc := range tests {
		got := FormatBytes(tc.bytes)
		if got != tc.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tc.bytes, got, tc.want)
		}
	}
}

func TestGetSystemStats(t *testing.T) {
	tmpDir := t.TempDir()
	stats := GetSystemStats(tmpDir)

	if stats.CPU.Cores <= 0 {
		t.Error("expected positive CPU cores")
	}
	if stats.CPU.Goroutines <= 0 {
		t.Error("expected positive goroutine count")
	}
	if stats.Memory.Total == 0 {
		t.Error("expected non-zero memory total")
	}
}

func TestGetSystemStats_EmptyDir(t *testing.T) {
	stats := GetSystemStats("")
	if stats.Disk.Total != 0 {
		t.Fatal("expected zero disk stats for empty dir")
	}
}

func TestGetDiskUsage(t *testing.T) {
	tmpDir := t.TempDir()
	usage := GetDiskUsage(tmpDir)

	if usage.Total == 0 {
		t.Error("expected non-zero total disk")
	}
	if usage.Free == 0 {
		t.Error("expected non-zero free disk")
	}
}

func TestGetDiskUsage_InvalidPath(t *testing.T) {
	usage := GetDiskUsage("/nonexistent/path/12345")
	if usage.Total != 0 {
		t.Fatal("expected zero usage for invalid path")
	}
}

func TestFormatFloat(t *testing.T) {
	if formatFloat(3.14159) != "3.1" {
		t.Fatalf("expected '3.1', got %q", formatFloat(3.14159))
	}
	if formatFloat(10.0) != "10.0" {
		t.Fatalf("expected '10.0', got %q", formatFloat(10.0))
	}
}

func TestDiskStats(t *testing.T) {
	tmpDir := t.TempDir()
	stats := getDiskStats(tmpDir)

	if stats.Total == 0 {
		t.Error("expected non-zero total")
	}
}

func TestMemoryStats(t *testing.T) {
	stats := getMemoryStats()

	if stats.Total == 0 {
		t.Error("expected non-zero total memory")
	}
	if stats.Used == 0 {
		t.Error("expected non-zero used memory")
	}
	if stats.Percent < 0 || stats.Percent > 100 {
		t.Errorf("expected percent between 0-100, got %f", stats.Percent)
	}
}

func TestCPUStats(t *testing.T) {
	stats := getCPUStats()

	if stats.Cores <= 0 {
		t.Error("expected at least 1 CPU core")
	}
	if stats.Goroutines <= 0 {
		t.Error("expected at least 1 goroutine")
	}
}

func TestGetDiskStats_EmptyDir(t *testing.T) {
	stats := getDiskStats("")
	if stats.Total != 0 {
		t.Fatal("expected zero disk stats for empty dir")
	}
}

func TestRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file to ensure dir exists on disk
	f, err := os.Create(filepath.Join(tmpDir, "test.txt"))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	disk := getDiskStats(tmpDir)
	if disk.Free == 0 {
		t.Error("expected non-zero free space")
	}
	if disk.Total == 0 {
		t.Error("expected non-zero total space")
	}
}
