package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfig(t *testing.T) {
	cfg := GetConfig()
	if cfg.PORT != PORT {
		t.Fatalf("expected PORT %d, got %d", PORT, cfg.PORT)
	}
}

func TestConstants(t *testing.T) {
	if PORT != 7777 {
		t.Fatalf("expected PORT 7777, got %d", PORT)
	}
	if ConfigDirName != ".beamdrop" {
		t.Fatalf("expected ConfigDirName '.beamdrop', got %q", ConfigDirName)
	}
	if MaxUploadSize != 5000*1024*1024 {
		t.Fatalf("expected MaxUploadSize 5GB, got %d", MaxUploadSize)
	}
	if MultipartFormMaxMemory != 10<<30 {
		t.Fatalf("expected MultipartFormMaxMemory 10GB, got %d", MultipartFormMaxMemory)
	}
}

func TestDefaultFlags(t *testing.T) {
	var f Flags
	if f.SharedDir != "" {
		t.Fatalf("expected empty SharedDir, got %q", f.SharedDir)
	}
	if f.Port != 0 {
		t.Fatalf("expected Port 0, got %d", f.Port)
	}
}

func TestInitDataDir(t *testing.T) {
	tmpDir := t.TempDir()
	InitDataDir(tmpDir)

	dataDir := filepath.Join(tmpDir, ConfigDirName)
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Fatal("data directory should exist")
	}

	expectedDBPath := filepath.Join(dataDir, DBName)
	if DBPath != expectedDBPath {
		t.Fatalf("expected DBPath %q, got %q", expectedDBPath, DBPath)
	}
}

func TestCreateTrashBin(t *testing.T) {
	tmpDir := t.TempDir()

	if err := CreateTrashBin(tmpDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	trashDir := filepath.Join(tmpDir, ".beamdrop_trash")
	if _, err := os.Stat(trashDir); os.IsNotExist(err) {
		t.Fatal("trash directory should exist")
	}
}

func TestAllowedMIMETypes(t *testing.T) {
	if len(AllowedMIMETypes) == 0 {
		t.Fatal("expected non-empty MIME types list")
	}
	found := false
	for _, mime := range AllowedMIMETypes {
		if mime == "image/jpeg" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected image/jpeg in allowed MIME types")
	}
}

func TestVersionBuildVars(t *testing.T) {
	if VERSION == "" {
		t.Fatal("expected VERSION to be non-empty (even if default)")
	}
}
