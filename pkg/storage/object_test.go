package storage

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupObjectTest(t *testing.T) (*ObjectManager, string) {
	t.Helper()
	tmpDir := t.TempDir()
	om := NewObjectManager(tmpDir)

	bm := NewBucketManager(tmpDir)
	bm.EnsureBucketsDir()
	bm.CreateBucket("test-bucket")

	return om, tmpDir
}

func TestNewObjectManager(t *testing.T) {
	tmpDir := t.TempDir()
	om := NewObjectManager(tmpDir)
	if om.bucketManager == nil {
		t.Fatal("expected non-nil BucketManager")
	}
	if om.LockManager == nil {
		t.Fatal("expected non-nil LockManager")
	}
}

func TestPutObject(t *testing.T) {
	om, _ := setupObjectTest(t)
	content := "hello world"

	info, err := om.PutObject("test-bucket", "test.txt", strings.NewReader(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Key != "test.txt" {
		t.Fatalf("expected key 'test.txt', got %q", info.Key)
	}
	if info.Size != int64(len(content)) {
		t.Fatalf("expected size %d, got %d", len(content), info.Size)
	}
	if info.ETag == "" {
		t.Fatal("expected non-empty ETag")
	}
	if info.LastModified.IsZero() {
		t.Fatal("expected non-zero LastModified")
	}
}

func TestPutObject_BucketNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	om := NewObjectManager(tmpDir)

	_, err := om.PutObject("nonexistent", "key", strings.NewReader("data"))
	if err != ErrBucketNotFound {
		t.Fatalf("expected ErrBucketNotFound, got %v", err)
	}
}

func TestPutObject_InvalidBucket(t *testing.T) {
	tmpDir := t.TempDir()
	om := NewObjectManager(tmpDir)

	_, err := om.PutObject("INVALID", "key", strings.NewReader("data"))
	if err == nil {
		t.Fatal("expected error for invalid bucket")
	}
}

func TestPutObject_InvalidKey(t *testing.T) {
	om, _ := setupObjectTest(t)

	_, err := om.PutObject("test-bucket", "../bad", strings.NewReader("data"))
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestGetObject(t *testing.T) {
	om, _ := setupObjectTest(t)
	content := "hello world"
	om.PutObject("test-bucket", "test.txt", strings.NewReader(content))

	file, info, unlock, err := om.GetObject("test-bucket", "test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer file.Close()
	defer unlock()

	if info.Key != "test.txt" {
		t.Fatalf("expected key 'test.txt', got %q", info.Key)
	}
	if info.Size != int64(len(content)) {
		t.Fatalf("expected size %d, got %d", len(content), info.Size)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != content {
		t.Fatalf("expected %q, got %q", content, string(data))
	}
}

func TestGetObject_NotFound(t *testing.T) {
	om, _ := setupObjectTest(t)

	_, _, _, err := om.GetObject("test-bucket", "nonexistent.txt")
	if err != ErrObjectNotFound {
		t.Fatalf("expected ErrObjectNotFound, got %v", err)
	}
}

func TestGetObject_BucketNotFound(t *testing.T) {
	om, _ := setupObjectTest(t)

	_, _, _, err := om.GetObject("nonexistent", "key.txt")
	if err != ErrBucketNotFound {
		t.Fatalf("expected ErrBucketNotFound, got %v", err)
	}
}

func TestDeleteObject(t *testing.T) {
	om, _ := setupObjectTest(t)
	om.PutObject("test-bucket", "test.txt", strings.NewReader("data"))

	if err := om.DeleteObject("test-bucket", "test.txt"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, _, _, err := om.GetObject("test-bucket", "test.txt")
	if err != ErrObjectNotFound {
		t.Fatalf("expected ErrObjectNotFound after delete, got %v", err)
	}
}

func TestDeleteObject_NotFound(t *testing.T) {
	om, _ := setupObjectTest(t)

	err := om.DeleteObject("test-bucket", "nonexistent.txt")
	if err != ErrObjectNotFound {
		t.Fatalf("expected ErrObjectNotFound, got %v", err)
	}
}

func TestHeadObject(t *testing.T) {
	om, _ := setupObjectTest(t)
	content := "hello world"
	om.PutObject("test-bucket", "test.txt", strings.NewReader(content))

	info, err := om.HeadObject("test-bucket", "test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Size != int64(len(content)) {
		t.Fatalf("expected size %d, got %d", len(content), info.Size)
	}
}

func TestHeadObject_NotFound(t *testing.T) {
	om, _ := setupObjectTest(t)

	_, err := om.HeadObject("test-bucket", "nonexistent.txt")
	if err != ErrObjectNotFound {
		t.Fatalf("expected ErrObjectNotFound, got %v", err)
	}
}

func TestListObjects_Empty(t *testing.T) {
	om, _ := setupObjectTest(t)

	result, err := om.ListObjects("test-bucket", "", "", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Contents) != 0 {
		t.Fatalf("expected 0 objects, got %d", len(result.Contents))
	}
}

func TestListObjects(t *testing.T) {
	om, _ := setupObjectTest(t)

	om.PutObject("test-bucket", "a.txt", strings.NewReader("aaa"))
	om.PutObject("test-bucket", "b.txt", strings.NewReader("bbb"))
	om.PutObject("test-bucket", "dir/c.txt", strings.NewReader("ccc"))

	result, err := om.ListObjects("test-bucket", "", "", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Contents) != 3 {
		t.Fatalf("expected 3 objects, got %d", len(result.Contents))
	}
}

func TestListObjects_WithPrefix(t *testing.T) {
	om, _ := setupObjectTest(t)

	om.PutObject("test-bucket", "images/a.png", strings.NewReader("png"))
	om.PutObject("test-bucket", "images/b.png", strings.NewReader("png"))
	om.PutObject("test-bucket", "docs/a.txt", strings.NewReader("txt"))

	result, err := om.ListObjects("test-bucket", "images/", "", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Contents) != 2 {
		t.Fatalf("expected 2 objects with prefix 'images/', got %d", len(result.Contents))
	}
}

func TestListObjects_WithDelimiter(t *testing.T) {
	om, _ := setupObjectTest(t)

	om.PutObject("test-bucket", "dir1/a.txt", strings.NewReader("a"))
	om.PutObject("test-bucket", "dir1/b.txt", strings.NewReader("b"))
	om.PutObject("test-bucket", "dir2/c.txt", strings.NewReader("c"))
	om.PutObject("test-bucket", "root.txt", strings.NewReader("d"))

	result, err := om.ListObjects("test-bucket", "", "/", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Contents) != 1 {
		t.Fatalf("expected 1 root object, got %d", len(result.Contents))
	}
	if len(result.CommonPrefixes) != 2 {
		t.Fatalf("expected 2 common prefixes, got %d: %v", len(result.CommonPrefixes), result.CommonPrefixes)
	}
}

func TestListObjects_MaxKeys(t *testing.T) {
	om, _ := setupObjectTest(t)

	for i := 0; i < 5; i++ {
		name := string(rune('a' + i))
		om.PutObject("test-bucket", name+".txt", strings.NewReader("data"))
	}

	result, err := om.ListObjects("test-bucket", "", "", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Contents) != 3 {
		t.Fatalf("expected 3 objects (truncated), got %d", len(result.Contents))
	}
	if !result.IsTruncated {
		t.Fatal("expected results to be truncated")
	}
}

func TestListObjects_BucketNotFound(t *testing.T) {
	om, _ := setupObjectTest(t)

	_, err := om.ListObjects("nonexistent", "", "", 100)
	if err != ErrBucketNotFound {
		t.Fatalf("expected ErrBucketNotFound, got %v", err)
	}
}

func TestObjectRoundTrip_LargeContent(t *testing.T) {
	om, _ := setupObjectTest(t)
	content := bytes.Repeat([]byte("A"), 10000)

	info, err := om.PutObject("test-bucket", "large.bin", bytes.NewReader(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Size != 10000 {
		t.Fatalf("expected size 10000, got %d", info.Size)
	}

	file, _, unlock, err := om.GetObject("test-bucket", "large.bin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer file.Close()
	defer unlock()

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Fatal("content mismatch")
	}
}

func TestObjectManager_NestedKeys(t *testing.T) {
	om, _ := setupObjectTest(t)

	om.PutObject("test-bucket", "a/b/c/d.txt", strings.NewReader("nested"))

	info, err := om.HeadObject("test-bucket", "a/b/c/d.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Size != 6 {
		t.Fatalf("expected size 6, got %d", info.Size)
	}

	// Verify parent directory doesn't show up as an object
	_, err = om.HeadObject("test-bucket", "a")
	if err != ErrObjectNotFound {
		t.Fatalf("expected ErrObjectNotFound for directory, got %v", err)
	}
}

func TestHeadObject_Directory(t *testing.T) {
	om, _ := setupObjectTest(t)
	om.PutObject("test-bucket", "dir/file.txt", strings.NewReader("data"))

	_, err := om.HeadObject("test-bucket", "dir")
	if err != ErrObjectNotFound {
		t.Fatalf("expected ErrObjectNotFound for directory, got %v", err)
	}
}

func TestPutObject_CreatesParentDirs(t *testing.T) {
	om, _ := setupObjectTest(t)

	_, err := om.PutObject("test-bucket", "deeply/nested/path/file.txt", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the file exists on disk
	bucketPath := filepath.Join(om.bucketManager.basePath, "test-bucket")
	diskPath := filepath.Join(bucketPath, "deeply", "nested", "path", "file.txt")
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		t.Fatal("file should exist on disk")
	}
}
