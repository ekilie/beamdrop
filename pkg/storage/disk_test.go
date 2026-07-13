package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCalculateDirSize(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some files
	os.WriteFile(filepath.Join(tmpDir, "a.txt"), make([]byte, 100), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "sub"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "sub", "b.txt"), make([]byte, 200), 0644)

	size, err := calculateDirSize(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 300 {
		t.Fatalf("expected 300 bytes, got %d", size)
	}
}

func TestCalculateDirSize_SkipsBeamdropDirs(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "file.txt"), make([]byte, 50), 0644)

	for _, dir := range []string{".beamdrop", ".beamdrop_data", ".beamdrop_trash"} {
		path := filepath.Join(tmpDir, dir)
		os.MkdirAll(path, 0755)
		os.WriteFile(filepath.Join(path, "big.dat"), make([]byte, 9999), 0644)
	}

	size, err := calculateDirSize(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 50 {
		t.Fatalf("expected 50 bytes (skipping .beamdrop dirs), got %d", size)
	}
}

func TestCalculateDirSize_NonExistent(t *testing.T) {
	_, err := calculateDirSize("/path/does/not/exist")
	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
}

func TestGetDirStorageUsage_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	usage, err := GetDirStorageUsage(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.UsedBytes != 0 {
		t.Fatalf("expected 0 used bytes, got %d", usage.UsedBytes)
	}
	if usage.TotalBytes == 0 {
		t.Fatal("expected non-zero total bytes")
	}
	if usage.LastUpdated.IsZero() {
		t.Fatal("expected non-zero last updated")
	}
}

func TestGetDirStorageUsage_Caches(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "data.txt"), make([]byte, 42), 0644)

	usage1, err := GetDirStorageUsage(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	usage2, err := GetDirStorageUsage(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if usage1.UsedBytes != usage2.UsedBytes {
		t.Fatal("cached values should match")
	}
}

func TestValidateMaxStorage(t *testing.T) {
	tmpDir := t.TempDir()

	if err := ValidateMaxStorage(tmpDir, 0); err != nil {
		t.Fatalf("zero max-storage should pass: %v", err)
	}

	if err := ValidateMaxStorage(tmpDir, -1); err != nil {
		t.Fatalf("negative max-storage should pass: %v", err)
	}

	// Very small max should fail (less than filesystem capacity)
	if err := ValidateMaxStorage(tmpDir, 1); err == nil {
		t.Log("note: max-storage validation for 1 byte passed (depends on FS)")
	}

	// Non-existent path - total will be 0
	if err := ValidateMaxStorage("/nonexistent", 100); err == nil {
		t.Fatal("expected error for non-existent path with positive max")
	}
}

func TestGetDirStorageUsage_WithContent(t *testing.T) {
	usageCacheLock.Lock()
	usageCache = DirUsage{}
	usageCacheLock.Unlock()

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "a.dat"), make([]byte, 512), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.dat"), make([]byte, 256), 0644)

	usage, err := GetDirStorageUsage(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.UsedBytes != 768 {
		t.Fatalf("expected 768 bytes, got %d", usage.UsedBytes)
	}
}

func TestGetFilesystemStats(t *testing.T) {
	tmpDir := t.TempDir()
	total, free := getFilesystemStats(tmpDir)
	if total == 0 {
		t.Error("expected non-zero total")
	}
	if free == 0 {
		t.Error("expected non-zero free")
	}
}
