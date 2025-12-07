package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// File represents a file or directory in the file system
type File struct {
	Name    string `json:"name"`
	Size    string `json:"size"`
	IsDir   bool   `json:"isDir"`
	ModTime string `json:"modTime"`
	Path    string `json:"path"`
}

// ResolvePath safely resolves a relative path within the shared directory
func ResolvePath(sharedDir, reqPath string) (string, error) {
	if reqPath == "" {
		return sharedDir, nil
	}

	clean := filepath.Clean(reqPath)
	target := filepath.Join(sharedDir, clean)

	absShared, err := filepath.Abs(sharedDir)
	if err != nil {
		return "", err
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(absTarget, absShared) {
		return "", fmt.Errorf("path traversal attempt")
	}

	return absTarget, nil
}

// IsFile checks if the given path is a file
func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// FormatFileSize formats file size in human-readable format
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatModTime formats modification time
func FormatModTime(modTime string) string {
	t, err := time.Parse(time.RFC3339, modTime)
	if err != nil {
		return modTime
	}
	return t.Format("2006-01-02 15:04:05")
}

