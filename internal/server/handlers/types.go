package handlers

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// File represents a file or directory in the file system
type File struct {
	Name      string `json:"name"`
	Size      string `json:"size"`
	IsDir     bool   `json:"isDir"`
	ModTime   string `json:"modTime"`
	Path      string `json:"path"`
	IsStarred bool   `json:"isStarred"`
}

// SensitiveSystemNames are file/directory names that must never be exposed
// through the file manager UI. They contain secrets, internal state, or
// infrastructure that end users should not see or operate on.
var SensitiveSystemNames = []string{
	".beamdrop",            // config dir: contains jwt_secret, beamdrop.db, beamdrop.log
	".beamdrop_trash",      // internal trash bin
	".beamdrop_data",       // legacy internal data dir
	"beamdrop.db",          // SQLite database (in case it ends up at the root)
	"beamdrop.db-journal",  // SQLite journal file
	"beamdrop.db-shm",      // SQLite shared memory file
	"beamdrop.db-wal",      // SQLite write-ahead log
}

// IsSensitiveName reports whether a top-level entry name is a sensitive
// system file/directory that should be hidden from the file manager.
func IsSensitiveName(name string) bool {
	for _, n := range SensitiveSystemNames {
		if name == n {
			return true
		}
	}
	return false
}

// IsSensitivePath reports whether the given absolute path (already
// confirmed to live inside sharedDir) points to or inside any sensitive
// system directory. It walks each path segment from the shared dir root
// and checks against SensitiveSystemNames.
func IsSensitivePath(sharedDir, absTarget string) bool {
	absShared, err := filepath.Abs(sharedDir)
	if err != nil {
		return false
	}
	absChecked, err := filepath.Abs(absTarget)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absShared, absChecked)
	if err != nil {
		return false
	}
	// Walk each segment of the relative path.
	for _, seg := range strings.Split(filepath.ToSlash(rel), "/") {
		if seg == "" || seg == "." {
			continue
		}
		if IsSensitiveName(seg) {
			return true
		}
	}
	return false
}

// IsSensitiveRequestPath reports whether a request-supplied path
// (relative to the shared dir) points to a sensitive system file/dir.
// It checks both the cleaned form and each leading segment.
func IsSensitiveRequestPath(reqPath string) bool {
	cleaned := strings.TrimPrefix(path.Clean("/"+reqPath), "/")
	if cleaned == "" {
		return false
	}
	for _, seg := range strings.Split(filepath.ToSlash(cleaned), "/") {
		if seg == "" || seg == "." {
			continue
		}
		if IsSensitiveName(seg) {
			return true
		}
	}
	return false
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

