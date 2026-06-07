package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestIsSensitiveName(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"config dir", ".beamdrop", true},
		{"trash dir", ".beamdrop_trash", true},
		{"legacy data dir", ".beamdrop_data", true},
		{"db file", "beamdrop.db", true},
		{"db journal", "beamdrop.db-journal", true},
		{"db shm", "beamdrop.db-shm", true},
		{"db wal", "beamdrop.db-wal", true},
		{"regular file", "photo.jpg", false},
		{"user folder", "documents", false},
		{"hidden file user-made", ".gitignore", false},
		{"empty", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsSensitiveName(tc.input); got != tc.expected {
				t.Errorf("IsSensitiveName(%q) = %v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestIsSensitiveRequestPath(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty", "", false},
		{"root", "/", false},
		{"clean root", ".", false},
		{"plain file", "documents/report.pdf", false},
		{"nested normal", "projects/2026/q1.pdf", false},
		{"config dir top", ".beamdrop", true},
		{"config dir nested path", ".beamdrop/jwt_secret", true},
		{"config dir with slash prefix", "/.beamdrop", true},
		{"config dir nested deeper", ".beamdrop/sub/inner", true},
		{"trash dir", ".beamdrop_trash", true},
		{"trash nested", ".beamdrop_trash/file.txt", true},
		{"db file in root", "beamdrop.db", true},
		{"db journal", "beamdrop.db-wal", true},
		{"path traversal attempt", "../.beamdrop/jwt_secret", true},
		{"path traversal normalised", "foo/../../.beamdrop", true},
		{"user folder named beamdrop", "mybeamdrop", false},
		{"user folder named beamdrop2", "beamdrop_backup", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsSensitiveRequestPath(tc.input); got != tc.expected {
				t.Errorf("IsSensitiveRequestPath(%q) = %v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestIsSensitivePath(t *testing.T) {
	sharedDir := t.TempDir()
	configDir := filepath.Join(sharedDir, ".beamdrop")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	jwtSecret := filepath.Join(configDir, "jwt_secret")
	if err := os.WriteFile(jwtSecret, []byte("secret"), 0600); err != nil {
		t.Fatalf("failed to write jwt secret: %v", err)
	}
	trashDir := filepath.Join(sharedDir, ".beamdrop_trash")
	if err := os.MkdirAll(trashDir, 0755); err != nil {
		t.Fatalf("failed to create trash dir: %v", err)
	}
	userFile := filepath.Join(sharedDir, "hello.txt")
	if err := os.WriteFile(userFile, []byte("hi"), 0644); err != nil {
		t.Fatalf("failed to write user file: %v", err)
	}

	cases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"config dir itself", configDir, true},
		{"jwt secret file", jwtSecret, true},
		{"trash dir", trashDir, true},
		{"user file", userFile, false},
		{"shared root", sharedDir, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsSensitivePath(sharedDir, tc.input); got != tc.expected {
				t.Errorf("IsSensitivePath(%q) = %v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestListFiles_HidesSensitiveEntries(t *testing.T) {
	setupDownloadZipTestDB(t)

	sharedDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(sharedDir, "report.pdf"), []byte("data"), 0644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(sharedDir, "documents"), 0755); err != nil {
		t.Fatalf("mkdir documents: %v", err)
	}

	configDir := filepath.Join(sharedDir, ".beamdrop")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("mkdir .beamdrop: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "jwt_secret"), []byte("secret"), 0600); err != nil {
		t.Fatalf("write jwt_secret: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "beamdrop.db"), []byte("db"), 0600); err != nil {
		t.Fatalf("write beamdrop.db: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(sharedDir, ".beamdrop_trash"), 0755); err != nil {
		t.Fatalf("mkdir .beamdrop_trash: %v", err)
	}

	handler := NewFileHandler(sharedDir)
	req := httptest.NewRequest(http.MethodGet, "/files?path=", nil)
	rec := httptest.NewRecorder()
	handler.ListFiles(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var files []File
	if err := json.NewDecoder(rec.Body).Decode(&files); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	for _, f := range files {
		if IsSensitiveName(f.Name) {
			t.Errorf("sensitive entry %q leaked into the file list", f.Name)
		}
	}

	found := false
	for _, f := range files {
		if f.Name == "report.pdf" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected report.pdf to be visible, got %d entries", len(files))
	}
}

func TestDownload_RejectsSensitiveFile(t *testing.T) {
	setupDownloadZipTestDB(t)

	sharedDir := t.TempDir()
	configDir := filepath.Join(sharedDir, ".beamdrop")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("mkdir .beamdrop: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "jwt_secret"), []byte("secret"), 0600); err != nil {
		t.Fatalf("write jwt_secret: %v", err)
	}

	handler := NewFileHandler(sharedDir)

	cases := []string{
		".beamdrop/jwt_secret",
		"/.beamdrop/jwt_secret",
		".beamdrop",
		"beamdrop.db",
	}

	for _, target := range cases {
		t.Run(target, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/download?file="+url.QueryEscape(target), nil)
			rec := httptest.NewRecorder()
			handler.Download(rec, req)
			if rec.Code != http.StatusForbidden {
				t.Errorf("expected 403 for %q, got %d (body=%s)", target, rec.Code, rec.Body.String())
			}
		})
	}
}
