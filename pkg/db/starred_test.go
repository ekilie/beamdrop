package db

import (
	"testing"
)

func TestStarFile(t *testing.T) {
	setupTestDB(t)

	err := StarFile("/path/to/file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !IsStarred("/path/to/file.txt") {
		t.Fatal("file should be starred")
	}
}

func TestStarFile_Duplicate(t *testing.T) {
	setupTestDB(t)

	if err := StarFile("/path.txt"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := StarFile("/path.txt"); err != nil {
		t.Fatalf("starring same file twice should not error: %v", err)
	}
}

func TestUnstarFile(t *testing.T) {
	setupTestDB(t)

	StarFile("/path/to/file.txt")

	if err := UnstarFile("/path/to/file.txt"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if IsStarred("/path/to/file.txt") {
		t.Fatal("file should not be starred after unstar")
	}
}

func TestUnstarFile_NotStarred(t *testing.T) {
	setupTestDB(t)

	err := UnstarFile("/never/starred.txt")
	if err != nil {
		t.Fatalf("unstarring non-starred file should not error: %v", err)
	}
}

func TestIsStarred_NotStarred(t *testing.T) {
	setupTestDB(t)

	if IsStarred("/nonexistent.txt") {
		t.Fatal("non-starred file should return false")
	}
}

func TestGetStarredFiles(t *testing.T) {
	setupTestDB(t)

	StarFile("/a.txt")
	StarFile("/b.txt")

	files, err := GetStarredFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 starred files, got %d", len(files))
	}
}

func TestGetStarredFiles_Empty(t *testing.T) {
	setupTestDB(t)

	files, err := GetStarredFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected 0 starred files, got %d", len(files))
	}
}
