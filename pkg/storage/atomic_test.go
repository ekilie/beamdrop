package storage

import (
    "bytes"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func TestAtomicWriter_Success(t *testing.T) {
    tmpDir := t.TempDir()
    targetPath := filepath.Join(tmpDir, "test.txt")
    content := []byte("hello world")

    writer, err := NewAtomicWriter(targetPath)
    if err != nil {
        t.Fatalf("Failed to create atomic writer: %v", err)
    }

    if _, err := writer.Write(content); err != nil {
        t.Fatalf("Failed to write: %v", err)
    }

    if err := writer.Commit(); err != nil {
        t.Fatalf("Failed to commit: %v", err)
    }

    // Verify file contents
    data, err := os.ReadFile(targetPath)
    if err != nil {
        t.Fatalf("Failed to read file: %v", err)
    }

    if !bytes.Equal(data, content) {
        t.Errorf("Content mismatch: got %q, want %q", data, content)
    }
}

func TestAtomicWriter_Abort(t *testing.T) {
    tmpDir := t.TempDir()
    targetPath := filepath.Join(tmpDir, "test.txt")

    writer, err := NewAtomicWriter(targetPath)
    if err != nil {
        t.Fatalf("Failed to create atomic writer: %v", err)
    }

    tempPath := writer.TempPath()

    if _, err := writer.Write([]byte("partial data")); err != nil {
        t.Fatalf("Failed to write: %v", err)
    }

    // Simulate crash by aborting
    if err := writer.Abort(); err != nil {
        t.Fatalf("Failed to abort: %v", err)
    }

    // Verify temp file is cleaned up
    if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
        t.Error("Temp file should be deleted after abort")
    }

    // Verify target file does not exist
    if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
        t.Error("Target file should not exist after abort")
    }
}

func TestAtomicWriter_SimulatedCrash(t *testing.T) {
    tmpDir := t.TempDir()
    targetPath := filepath.Join(tmpDir, "test.txt")

    writer, err := NewAtomicWriter(targetPath)
    if err != nil {
        t.Fatalf("Failed to create atomic writer: %v", err)
    }

    tempPath := writer.TempPath()

    // Write some data
    if _, err := writer.Write([]byte("partial data")); err != nil {
        t.Fatalf("Failed to write: %v", err)
    }

    // Simulate crash: close file handle directly without commit
    writer.tempFile.Close()

    // Temp file should still exist (orphaned)
    if _, err := os.Stat(tempPath); os.IsNotExist(err) {
        t.Fatal("Temp file should exist after simulated crash")
    }

    // Target file should NOT exist
    if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
        t.Error("Target file should not exist after crash")
    }
}

func TestCleanupOrphanedTempFiles(t *testing.T) {
    tmpDir := t.TempDir()

    // Create some orphaned temp files
    orphaned1 := filepath.Join(tmpDir, TempFilePrefix+"123456")
    orphaned2 := filepath.Join(tmpDir, "subdir", TempFilePrefix+"789012")
    legitimate := filepath.Join(tmpDir, "legitimate.txt")

    os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
    os.WriteFile(orphaned1, []byte("orphaned"), 0644)
    os.WriteFile(orphaned2, []byte("orphaned"), 0644)
    os.WriteFile(legitimate, []byte("keep me"), 0644)

    // Run cleanup
    err := CleanupOrphanedTempFiles(tmpDir)
    if err != nil {
        t.Fatalf("Cleanup failed: %v", err)
    }

    // Verify orphaned files are removed
    if _, err := os.Stat(orphaned1); !os.IsNotExist(err) {
        t.Error("Orphaned file 1 should be deleted")
    }
    if _, err := os.Stat(orphaned2); !os.IsNotExist(err) {
        t.Error("Orphaned file 2 should be deleted")
    }

    // Verify legitimate file still exists
    if _, err := os.Stat(legitimate); err != nil {
        t.Error("Legitimate file should still exist")
    }
}

func TestAtomicWriteFile(t *testing.T) {
    tmpDir := t.TempDir()
    targetPath := filepath.Join(tmpDir, "test.txt")
    content := "test content for atomic write"

    n, err := AtomicWriteFile(targetPath, strings.NewReader(content))
    if err != nil {
        t.Fatalf("AtomicWriteFile failed: %v", err)
    }

    if n != int64(len(content)) {
        t.Errorf("Bytes written mismatch: got %d, want %d", n, len(content))
    }

    data, err := os.ReadFile(targetPath)
    if err != nil {
        t.Fatalf("Failed to read file: %v", err)
    }

    if string(data) != content {
        t.Errorf("Content mismatch: got %q, want %q", string(data), content)
    }
}

func TestAtomicWriter_CreatesParentDirs(t *testing.T) {
    tmpDir := t.TempDir()
    targetPath := filepath.Join(tmpDir, "deep", "nested", "path", "test.txt")

    writer, err := NewAtomicWriter(targetPath)
    if err != nil {
        t.Fatalf("Failed to create atomic writer: %v", err)
    }

    writer.Write([]byte("nested file"))
    if err := writer.Commit(); err != nil {
        t.Fatalf("Commit failed: %v", err)
    }

    if _, err := os.Stat(targetPath); err != nil {
        t.Errorf("File should exist at nested path: %v", err)
    }
}