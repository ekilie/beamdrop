package config

import (
	"log"
	"os"
	"path/filepath"
)

func SetDBPath(path string) {
	if path == "" {
		return
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("failed to create database directory %s: %v", dir, err)
	}
	DBPath = path
}
