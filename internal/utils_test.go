package beam

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1023, "1023 B"},
		{1024, "1 KB"},
		{1536, "1.50 KB"},
		{1048576, "1 MB"},
		{1073741824, "1 GB"},
		{1099511627776, "1 TB"},
		{1125899906842624, "1 PB"},
		{1500000, "1.43 MB"},
	}

	for _, tc := range tests {
		got := FormatFileSize(tc.size)
		if got != tc.want {
			t.Errorf("FormatFileSize(%d) = %q, want %q", tc.size, got, tc.want)
		}
	}
}

func TestFormatModTime(t *testing.T) {
	result := FormatModTime("2024-01-15T10:30:00Z")
	if result != "2024-01-15 10:30:00" {
		t.Fatalf("expected '2024-01-15 10:30:00', got %q", result)
	}

	result = FormatModTime("invalid-date")
	if result != "invalid-date" {
		t.Fatalf("expected original string for invalid date, got %q", result)
	}
}

func TestResolvePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Valid path
	path, err := ResolvePath(tmpDir, "test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != filepath.Join(tmpDir, "test.txt") {
		t.Fatalf("unexpected path: %q", path)
	}

	// Valid nested path
	path, err = ResolvePath(tmpDir, "dir/sub/file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != filepath.Join(tmpDir, "dir/sub/file.txt") {
		t.Fatalf("unexpected path: %q", path)
	}
}

func TestResolvePath_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := ResolvePath(tmpDir, "../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}

	_, err = ResolvePath(tmpDir, "../../../etc")
	if err == nil {
		t.Fatal("expected error for deep path traversal")
	}
}

func TestResolvePath_AbsolutePath(t *testing.T) {
	sharedDir := t.TempDir()
	// In Go, filepath.Join appends absolute paths; they become sub-paths
	path, err := ResolvePath(sharedDir, "/etc/passwd")
	if err != nil {
		t.Fatalf("absolute paths are treated as sub-paths: %v", err)
	}
	if filepath.IsAbs(path) && !strings.HasPrefix(path, sharedDir) {
		t.Fatal("result should be within sharedDir")
	}
}

func TestIsDir(t *testing.T) {
	tmpDir := t.TempDir()

	if !IsDir(tmpDir) {
		t.Fatal("expected directory check to be true")
	}

	filePath := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(filePath, []byte("data"), 0644)

	if IsDir(filePath) {
		t.Fatal("file should not be a directory")
	}

	if IsDir("/nonexistent") {
		t.Fatal("nonexistent path should not be a directory")
	}
}

func TestIsFile(t *testing.T) {
	tmpDir := t.TempDir()

	filePath := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(filePath, []byte("data"), 0644)

	if !IsFile(filePath) {
		t.Fatal("expected file check to be true")
	}

	if IsFile(tmpDir) {
		t.Fatal("directory should not be a file")
	}

	if IsFile("/nonexistent") {
		t.Fatal("nonexistent path should not be a file")
	}
}

func TestGetLocalIP(t *testing.T) {
	ip := GetLocalIP()
	if ip == "" {
		t.Fatal("expected non-empty IP")
	}
}

func TestSearchFiles(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "hello.txt"), []byte("data"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "world.txt"), []byte("data"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "hello.md"), []byte("data"), 0644)

	var results []File
	err := searchFiles(tmpDir, "hello", "", &results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results matching 'hello', got %d", len(results))
	}
}

func TestSearchFiles_CaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "HELLO.txt"), []byte("data"), 0644)

	var results []File
	err := searchFiles(tmpDir, "hello", "", &results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for case-insensitive search, got %d", len(results))
	}
}

func TestSearchFiles_WithRelativePath(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "found.txt"), []byte("data"), 0644)

	var results []File
	err := searchFiles(subDir, "found", "sub", &results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(results) > 0 && results[0].Path != "sub/found.txt" {
		t.Fatalf("expected path 'sub/found.txt', got %q", results[0].Path)
	}
}
