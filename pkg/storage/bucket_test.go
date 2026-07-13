package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateBucketName(t *testing.T) {
	valid := []string{
		"my-bucket",
		"my.bucket",
		"mybucket123",
		"a1234567890123456789012345678901234567890123456789012345678901",
	}
	invalid := []string{
		"",
		"ab",       // too short
		"my-bucket-", // ends with hyphen
		"-my-bucket", // starts with hyphen
		".mybucket",  // starts with dot
		"my.bucket.", // ends with dot
		"MyBucket",   // uppercase
		"my bucket",  // space
		"192.168.1.1", // IP-like
		"a",
		"ab",
	}

	for _, name := range valid {
		if err := ValidateBucketName(name); err != nil {
			t.Errorf("expected valid: %q, got error: %v", name, err)
		}
	}
	for _, name := range invalid {
		if err := ValidateBucketName(name); err == nil {
			t.Errorf("expected invalid: %q", name)
		}
	}
}

func TestValidateObjectKey(t *testing.T) {
	valid := []string{
		"file.txt",
		"path/to/file.txt",
		"123",
		"a/b/c/d/file",
	}
	invalid := []string{
		"",
		"../file.txt",
		"foo/../../bar",
		"/absolute/path",
	}

	for _, key := range valid {
		if err := ValidateObjectKey(key); err != nil {
			t.Errorf("expected valid: %q, got error: %v", key, err)
		}
	}
	for _, key := range invalid {
		if err := ValidateObjectKey(key); err == nil {
			t.Errorf("expected invalid: %q", key)
		}
	}
}

func TestNewBucketManager(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBucketManager(tmpDir)

	if bm.basePath != filepath.Join(tmpDir, "buckets") {
		t.Fatalf("unexpected base path: %q", bm.basePath)
	}
}

func TestBucketManager_EnsureBucketsDir(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBucketManager(tmpDir)

	if err := bm.EnsureBucketsDir(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(bm.basePath); os.IsNotExist(err) {
		t.Fatal("buckets directory should exist")
	}
}

func TestBucketManager_CreateAndDeleteBucket(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBucketManager(tmpDir)
	bm.EnsureBucketsDir()

	if err := bm.CreateBucket("test-bucket"); err != nil {
		t.Fatalf("failed to create bucket: %v", err)
	}

	if !bm.BucketExists("test-bucket") {
		t.Fatal("bucket should exist")
	}

	if err := bm.CreateBucket("test-bucket"); err != ErrBucketExists {
		t.Fatalf("expected ErrBucketExists, got %v", err)
	}

	if err := bm.DeleteBucket("test-bucket"); err != nil {
		t.Fatalf("failed to delete bucket: %v", err)
	}

	if bm.BucketExists("test-bucket") {
		t.Fatal("bucket should not exist after deletion")
	}
}

func TestBucketManager_DeleteBucket_NotEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBucketManager(tmpDir)
	bm.EnsureBucketsDir()
	bm.CreateBucket("test-bucket")

	// Create a file inside the bucket
	bucketPath := filepath.Join(bm.basePath, "test-bucket")
	os.WriteFile(filepath.Join(bucketPath, "file.txt"), []byte("data"), 0644)

	if err := bm.DeleteBucket("test-bucket"); err != ErrBucketNotEmpty {
		t.Fatalf("expected ErrBucketNotEmpty, got %v", err)
	}
}

func TestBucketManager_DeleteBucket_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBucketManager(tmpDir)

	if err := bm.DeleteBucket("nonexistent"); err != ErrBucketNotFound {
		t.Fatalf("expected ErrBucketNotFound, got %v", err)
	}
}

func TestBucketManager_CreateBucketIfNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBucketManager(tmpDir)
	bm.EnsureBucketsDir()

	created, err := bm.CreateBucketIfNotExists("my-bucket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Fatal("expected bucket to be created")
	}

	created, err = bm.CreateBucketIfNotExists("my-bucket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Fatal("expected bucket to already exist")
	}
}

func TestBucketManager_ListBuckets(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBucketManager(tmpDir)
	bm.EnsureBucketsDir()

	buckets, err := bm.ListBuckets()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(buckets) != 0 {
		t.Fatalf("expected 0 buckets, got %d", len(buckets))
	}

	bm.CreateBucket("alpha")
	bm.CreateBucket("beta")

	buckets, err = bm.ListBuckets()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(buckets))
	}
}

func TestBucketManager_GetBucketPath(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBucketManager(tmpDir)

	path, err := bm.GetBucketPath("valid-bucket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != filepath.Join(tmpDir, "buckets", "valid-bucket") {
		t.Fatalf("unexpected path: %q", path)
	}

	_, err = bm.GetBucketPath("INVALID") // uppercase
	if err == nil {
		t.Fatal("expected error for invalid bucket name")
	}
}

func TestBucketManager_BucketExists(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBucketManager(tmpDir)
	bm.EnsureBucketsDir()

	if bm.BucketExists("ghost") {
		t.Fatal("bucket should not exist")
	}

	bm.CreateBucket("ghost")
	if !bm.BucketExists("ghost") {
		t.Fatal("bucket should exist")
	}

	if bm.BucketExists("INVALID") {
		t.Fatal("invalid name should not exist")
	}
}
