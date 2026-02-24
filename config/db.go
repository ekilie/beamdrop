package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

func SetDBPath(path string) {
	resolvedPath := resolveDBPath(path)
	if resolvedPath == "" {
		return
	}

	// Ensure parent directory exists
	dir := filepath.Dir(resolvedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("failed to create database directory %s: %v", dir, err)
	}
	DBPath = resolvedPath
}

func resolveDBPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}

	if strings.HasSuffix(trimmed, string(os.PathSeparator)) {
		return filepath.Join(trimmed, DBName)
	}

	if info, err := os.Stat(trimmed); err == nil && info.IsDir() {
		return filepath.Join(trimmed, DBName)
	}

	cleaned := filepath.Clean(trimmed)
	if cleaned == "." {
		return filepath.Join(cleaned, DBName)
	}

	return trimmed
}
