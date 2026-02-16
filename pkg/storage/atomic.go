package storage

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	// TempFilePrefix is used to identify temporary files during atomic writes
	TempFilePrefix = ".beamdrop_tmp_"
)

// AtomicWriter provides atomic file write operations using temp file + fsync + rename
type AtomicWriter struct {
	targetPath string
	tempFile   *os.File
	closed     bool
}

// NewAtomicWriter creates a new atomic writer for the target path.
// The file is written to a temporary location first, then atomically renamed.
func NewAtomicWriter(targetPath string) (*AtomicWriter, error) {
	dir := filepath.Dir(targetPath)

	// Ensure target directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Create temp file in the same directory (required for atomic rename)
	tempFile, err := os.CreateTemp(dir, TempFilePrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	return &AtomicWriter{
		targetPath: targetPath,
		tempFile:   tempFile,
		closed:     false,
	}, nil
}

// Write implements io.Writer
func (aw *AtomicWriter) Write(p []byte) (n int, err error) {
	if aw.closed {
		return 0, fmt.Errorf("writer is closed")
	}
	return aw.tempFile.Write(p)
}

// ReadFrom implements io.ReaderFrom for efficient copying
func (aw *AtomicWriter) ReadFrom(r io.Reader) (n int64, err error) {
	if aw.closed {
		return 0, fmt.Errorf("writer is closed")
	}
	return io.Copy(aw.tempFile, r)
}

// Commit finalizes the atomic write: fsync + rename
// This is the critical method that ensures atomicity
func (aw *AtomicWriter) Commit() error {
	if aw.closed {
		return fmt.Errorf("writer already closed")
	}
	aw.closed = true

	tempPath := aw.tempFile.Name()

	// Step 1: Sync file data to disk (crucial for crash safety)
	if err := aw.tempFile.Sync(); err != nil {
		aw.tempFile.Close()
		os.Remove(tempPath)
		return fmt.Errorf("failed to sync file: %w", err)
	}

	// Step 2: Close the file
	if err := aw.tempFile.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Step 3: Atomic rename (this is atomic on POSIX systems)
	if err := os.Rename(tempPath, aw.targetPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Step 4: Sync the directory to ensure the rename is persisted
	if err := syncDir(filepath.Dir(aw.targetPath)); err != nil {
		// We Log but don't fail - the file is already in place
		slog.Warn("Failed to sync directory", "error", err)
	}

	return nil
}

// Abort cancels the write and cleans up the temp file
func (aw *AtomicWriter) Abort() error {
	if aw.closed {
		return nil
	}
	aw.closed = true

	tempPath := aw.tempFile.Name()
	aw.tempFile.Close()
	return os.Remove(tempPath)
}

// TempPath returns the path to the temporary file (useful for progress tracking)
func (aw *AtomicWriter) TempPath() string {
	return aw.tempFile.Name()
}

// syncDir syncs a directory to ensure metadata changes are persisted
func syncDir(dirPath string) error {
	dir, err := os.Open(dirPath)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

// CleanupOrphanedTempFiles removes any temporary files left from interrupted writes.
// Call this on application startup.
func CleanupOrphanedTempFiles(rootDir string) error {
	var cleaned int
	var errors []error

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if this is an orphaned temp file
		if strings.HasPrefix(info.Name(), TempFilePrefix) {
			if err := os.Remove(path); err != nil {
				errors = append(errors, fmt.Errorf("failed to remove %s: %w", path, err))
			} else {
				cleaned++
				slog.Info("Cleaned up orphaned temp file", "path", path)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	if cleaned > 0 {
		slog.Info("Cleaned up orphaned temporary files", "count", cleaned)
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to clean some temp files: %v", errors)
	}

	return nil
}

// AtomicWriteFile is a convenience function for simple atomic writes
func AtomicWriteFile(targetPath string, r io.Reader) (int64, error) {
	writer, err := NewAtomicWriter(targetPath)
	if err != nil {
		return 0, err
	}

	n, err := writer.ReadFrom(r)
	if err != nil {
		writer.Abort()
		return 0, err
	}

	if err := writer.Commit(); err != nil {
		return 0, err
	}

	return n, nil
}
