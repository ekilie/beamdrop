package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpload_RespectsCurrentPath(t *testing.T) {
	setupDownloadZipTestDB(t)

	sharedDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(sharedDir, "docs"), 0755); err != nil {
		t.Fatalf("failed to create docs folder: %v", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "note.txt")
	if err != nil {
		t.Fatalf("failed to create file part: %v", err)
	}
	if _, err := part.Write([]byte("hello from folder")); err != nil {
		t.Fatalf("failed to write file part: %v", err)
	}
	if err := writer.WriteField("path", "docs"); err != nil {
		t.Fatalf("failed to write path field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	handler := NewFileHandler(sharedDir)
	handler.Upload(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	if _, err := os.Stat(filepath.Join(sharedDir, "docs", "note.txt")); err != nil {
		t.Fatalf("expected uploaded file in docs folder: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sharedDir, "note.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected no file at root, got err: %v", err)
	}
}

func TestMove_TargetDirectoryUsesSourceName(t *testing.T) {
	sharedDir := t.TempDir()
	sourcePath := filepath.Join(sharedDir, "report.txt")
	targetDir := filepath.Join(sharedDir, "archive")

	if err := os.WriteFile(sourcePath, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	payload := `{"sourcePath":"report.txt","targetPath":"archive"}`
	req := httptest.NewRequest(http.MethodPost, "/move", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler := NewFileOperationsHandler(sharedDir)
	handler.Move(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	if _, err := os.Stat(filepath.Join(sharedDir, "archive", "report.txt")); err != nil {
		t.Fatalf("expected moved file in archive dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sharedDir, "report.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected source file removed after move, got err: %v", err)
	}
}

func TestTrash_PreservesRelativePath(t *testing.T) {
	sharedDir := t.TempDir()
	sourceDir := filepath.Join(sharedDir, "nested")
	sourceFile := filepath.Join(sourceDir, "draft.txt")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.WriteFile(sourceFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	body, err := json.Marshal(map[string]string{"sourcePath": "nested/draft.txt"})
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/trash", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler := NewFileOperationsHandler(sharedDir)
	handler.Trash(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	trashPath := filepath.Join(sharedDir, ".beamdrop_trash", "nested", "draft.txt")
	if _, err := os.Stat(trashPath); err != nil {
		t.Fatalf("expected file in trash preserving relative path: %v", err)
	}
}
