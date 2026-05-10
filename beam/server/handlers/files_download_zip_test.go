package handlers

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ekilie/beamdrop/config"
	"github.com/ekilie/beamdrop/pkg/crypto"
	"github.com/ekilie/beamdrop/pkg/db"
)

var downloadZipTestDBOnce sync.Once

func setupDownloadZipTestDB(t *testing.T) {
	t.Helper()

	downloadZipTestDBOnce.Do(func() {
		tempDir, err := os.MkdirTemp("", "beamdrop-download-zip-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		config.SetDBPath(filepath.Join(tempDir, "beamdrop.db"))
		crypto.SetEncryptionKey(bytes.Repeat([]byte("z"), 32))
		db.Init()
		db.AutoMigrate()
	})
}

func TestDownload_DirectoryStreamsZIP(t *testing.T) {
	setupDownloadZipTestDB(t)

	sharedDir := t.TempDir()
	folder := filepath.Join(sharedDir, "folder")
	if err := os.MkdirAll(filepath.Join(folder, "nested"), 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(folder, "a.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(folder, "nested", "b.txt"), []byte("world"), 0644); err != nil {
		t.Fatalf("failed to create nested file: %v", err)
	}

	handler := NewFileHandler(sharedDir)
	req := httptest.NewRequest(http.MethodGet, "/download?file=folder", nil)
	rec := httptest.NewRecorder()

	handler.Download(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if got := rec.Header().Get("Content-Type"); got != "application/zip" {
		t.Fatalf("expected application/zip content type, got %q", got)
	}

	disposition := rec.Header().Get("Content-Disposition")
	if disposition != `attachment; filename="folder.zip"` {
		t.Fatalf("unexpected content disposition: %q", disposition)
	}

	reader, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("expected valid zip body, got error: %v", err)
	}

	entries := map[string]bool{}
	for _, file := range reader.File {
		entries[file.Name] = true
	}

	if !entries["a.txt"] {
		t.Fatalf("expected zip to contain a.txt, got entries: %v", entries)
	}
	if !entries["nested/b.txt"] {
		t.Fatalf("expected zip to contain nested/b.txt, got entries: %v", entries)
	}
}
