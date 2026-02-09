package storage

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ObjectManager handles object filesystem operations
type ObjectManager struct {
	bucketManager *BucketManager
}

// NewObjectManager creates a new object manager
func NewObjectManager(sharedDir string) *ObjectManager {
	return &ObjectManager{
		bucketManager: NewBucketManager(sharedDir),
	}
}

// ObjectInfo contains object metadata
type ObjectInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModified"`
	ETag         string    `json:"etag"`
	ContentType  string    `json:"contentType,omitempty"`
}

// PutObject writes an object to storage
func (om *ObjectManager) PutObject(bucket, key string, reader io.Reader) (*ObjectInfo, error) {
	if err := ValidateBucketName(bucket); err != nil {
		return nil, err
	}
	if err := ValidateObjectKey(key); err != nil {
		return nil, err
	}

	if !om.bucketManager.BucketExists(bucket) {
		return nil, ErrBucketNotFound
	}

	bucketPath, _ := om.bucketManager.GetBucketPath(bucket)
	objectPath := filepath.Join(bucketPath, filepath.FromSlash(key))

	// Create parent directories if needed
	if err := os.MkdirAll(filepath.Dir(objectPath), 0755); err != nil {
		return nil, err
	}

	// Create temporary file for atomic write
	tmpFile, err := os.CreateTemp(filepath.Dir(objectPath), ".tmp-")
	if err != nil {
		return nil, err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Clean up temp file on error

	// Write content and calculate ETag (MD5)
	hash := md5.New()
	multiWriter := io.MultiWriter(tmpFile, hash)

	size, err := io.Copy(multiWriter, reader)
	if err != nil {
		tmpFile.Close()
		return nil, err
	}
	tmpFile.Close()

	// Atomic rename
	if err := os.Rename(tmpPath, objectPath); err != nil {
		return nil, err
	}

	etag := hex.EncodeToString(hash.Sum(nil))

	info, err := os.Stat(objectPath)
	if err != nil {
		return nil, err
	}

	return &ObjectInfo{
		Key:          key,
		Size:         size,
		LastModified: info.ModTime(),
		ETag:         etag,
	}, nil
}

// GetObject retrieves an object from storage
func (om *ObjectManager) GetObject(bucket, key string) (*os.File, *ObjectInfo, error) {
	if err := ValidateBucketName(bucket); err != nil {
		return nil, nil, err
	}
	if err := ValidateObjectKey(key); err != nil {
		return nil, nil, err
	}

	if !om.bucketManager.BucketExists(bucket) {
		return nil, nil, ErrBucketNotFound
	}

	bucketPath, _ := om.bucketManager.GetBucketPath(bucket)
	objectPath := filepath.Join(bucketPath, filepath.FromSlash(key))

	file, err := os.Open(objectPath)
	if os.IsNotExist(err) {
		return nil, nil, ErrObjectNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, err
	}

	if info.IsDir() {
		file.Close()
		return nil, nil, ErrObjectNotFound
	}

	return file, &ObjectInfo{
		Key:          key,
		Size:         info.Size(),
		LastModified: info.ModTime(),
	}, nil
}

// DeleteObject removes an object from storage
func (om *ObjectManager) DeleteObject(bucket, key string) error {
	if err := ValidateBucketName(bucket); err != nil {
		return err
	}
	if err := ValidateObjectKey(key); err != nil {
		return err
	}

	if !om.bucketManager.BucketExists(bucket) {
		return ErrBucketNotFound
	}

	bucketPath, _ := om.bucketManager.GetBucketPath(bucket)
	objectPath := filepath.Join(bucketPath, filepath.FromSlash(key))

	err := os.Remove(objectPath)
	if os.IsNotExist(err) {
		return ErrObjectNotFound
	}
	return err
}

// HeadObject returns object metadata without the content
func (om *ObjectManager) HeadObject(bucket, key string) (*ObjectInfo, error) {
	if err := ValidateBucketName(bucket); err != nil {
		return nil, err
	}
	if err := ValidateObjectKey(key); err != nil {
		return nil, err
	}

	if !om.bucketManager.BucketExists(bucket) {
		return nil, ErrBucketNotFound
	}

	bucketPath, _ := om.bucketManager.GetBucketPath(bucket)
	objectPath := filepath.Join(bucketPath, filepath.FromSlash(key))

	info, err := os.Stat(objectPath)
	if os.IsNotExist(err) {
		return nil, ErrObjectNotFound
	}
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return nil, ErrObjectNotFound
	}

	return &ObjectInfo{
		Key:          key,
		Size:         info.Size(),
		LastModified: info.ModTime(),
	}, nil
}

// ListObjects lists objects in a bucket with optional prefix filtering
func (om *ObjectManager) ListObjects(bucket, prefix, delimiter string, maxKeys int) (*ListObjectsResult, error) {
	if err := ValidateBucketName(bucket); err != nil {
		return nil, err
	}

	if !om.bucketManager.BucketExists(bucket) {
		return nil, ErrBucketNotFound
	}

	bucketPath, _ := om.bucketManager.GetBucketPath(bucket)
	searchPath := bucketPath
	if prefix != "" {
		searchPath = filepath.Join(bucketPath, filepath.FromSlash(prefix))
	}

	result := &ListObjectsResult{
		Prefix:         prefix,
		Delimiter:      delimiter,
		MaxKeys:        maxKeys,
		CommonPrefixes: []string{},
		Contents:       []ObjectInfo{},
	}

	if maxKeys <= 0 {
		maxKeys = 1000
	}

	// Track common prefixes for delimiter handling
	seenPrefixes := make(map[string]bool)

	err := filepath.Walk(bucketPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip the bucket root
		if path == bucketPath {
			return nil
		}

		// Get relative key
		relPath, _ := filepath.Rel(bucketPath, path)
		key := filepath.ToSlash(relPath)

		// Apply prefix filter
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			if info.IsDir() && !strings.HasPrefix(prefix, key+"/") {
				return filepath.SkipDir
			}
			return nil
		}

		// Handle delimiter (virtual directories)
		if delimiter != "" {
			afterPrefix := key
			if prefix != "" {
				afterPrefix = strings.TrimPrefix(key, prefix)
			}

			if idx := strings.Index(afterPrefix, delimiter); idx >= 0 {
				commonPrefix := prefix + afterPrefix[:idx+len(delimiter)]
				if !seenPrefixes[commonPrefix] {
					seenPrefixes[commonPrefix] = true
					result.CommonPrefixes = append(result.CommonPrefixes, commonPrefix)
				}
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Skip directories in the contents list
		if info.IsDir() {
			return nil
		}

		// Check max keys limit
		if len(result.Contents) >= maxKeys {
			result.IsTruncated = true
			return filepath.SkipAll
		}

		result.Contents = append(result.Contents, ObjectInfo{
			Key:          key,
			Size:         info.Size(),
			LastModified: info.ModTime(),
		})

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, err
	}

	return result, nil
}

// ListObjectsResult contains the result of a list objects operation
type ListObjectsResult struct {
	Prefix         string       `json:"prefix"`
	Delimiter      string       `json:"delimiter,omitempty"`
	MaxKeys        int          `json:"maxKeys"`
	IsTruncated    bool         `json:"isTruncated"`
	Contents       []ObjectInfo `json:"contents"`
	CommonPrefixes []string     `json:"commonPrefixes,omitempty"`
}
