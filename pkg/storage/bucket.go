package storage

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// BucketInfo contains bucket metadata
type BucketInfo struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

var (
	ErrInvalidBucketName = errors.New("invalid bucket name")
	ErrBucketNotFound    = errors.New("bucket not found")
	ErrBucketNotEmpty    = errors.New("bucket is not empty")
	ErrBucketExists      = errors.New("bucket already exists")
	ErrObjectNotFound    = errors.New("object not found")
	ErrInvalidKey        = errors.New("invalid object key")
)

// bucketNameRegex validates bucket names (similar to S3 rules)
// - 3-63 characters
// - lowercase letters, numbers, hyphens, dots
// - must start and end with letter or number
var bucketNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)

// BucketManager handles bucket filesystem operations
type BucketManager struct {
	basePath string // Path to buckets directory
}

// NewBucketManager creates a new bucket manager
func NewBucketManager(sharedDir string) *BucketManager {
	bucketsPath := filepath.Join(sharedDir, "buckets")
	return &BucketManager{basePath: bucketsPath}
}

// EnsureBucketsDir ensures the buckets directory exists
func (bm *BucketManager) EnsureBucketsDir() error {
	return os.MkdirAll(bm.basePath, 0755)
}

// ValidateBucketName checks if a bucket name is valid
func ValidateBucketName(name string) error {
	if len(name) < 3 || len(name) > 63 {
		return ErrInvalidBucketName
	}
	if !bucketNameRegex.MatchString(name) {
		return ErrInvalidBucketName
	}
	// Prevent IP-like names
	if regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`).MatchString(name) {
		return ErrInvalidBucketName
	}
	return nil
}

// ValidateObjectKey checks if an object key is valid
func ValidateObjectKey(key string) error {
	if key == "" {
		return ErrInvalidKey
	}
	// Prevent path traversal
	if strings.Contains(key, "..") {
		return ErrInvalidKey
	}
	// Prevent absolute paths
	if strings.HasPrefix(key, "/") {
		return ErrInvalidKey
	}
	// Key length limit (S3 allows 1024 bytes)
	if len(key) > 1024 {
		return ErrInvalidKey
	}
	return nil
}

// CreateBucket creates a new bucket directory
func (bm *BucketManager) CreateBucket(name string) error {
	if err := ValidateBucketName(name); err != nil {
		return err
	}

	bucketPath := filepath.Join(bm.basePath, name)

	// Check if bucket already exists
	if _, err := os.Stat(bucketPath); err == nil {
		return ErrBucketExists
	}

	return os.MkdirAll(bucketPath, 0755)
}

// CreateBucketIfNotExists creates a new bucket directory if it doesn't already exist
func (bm *BucketManager) CreateBucketIfNotExists(name string) (bool, error) {
	if err := ValidateBucketName(name); err != nil {
		return false, err
	}

	bucketPath := filepath.Join(bm.basePath, name)

	// if bucket already exists
	// we return true without error, since the bucket is already there
	if _, err := os.Stat(bucketPath); err == nil {
		return true, nil
	}

	if err := os.MkdirAll(bucketPath, 0755); err != nil {
		return false, err
	}

	return true, nil
}

// DeleteBucket deletes a bucket if it's empty
func (bm *BucketManager) DeleteBucket(name string) error {
	if err := ValidateBucketName(name); err != nil {
		return err
	}

	bucketPath := filepath.Join(bm.basePath, name)

	// Check if bucket exists
	info, err := os.Stat(bucketPath)
	if os.IsNotExist(err) {
		return ErrBucketNotFound
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return ErrBucketNotFound
	}

	// Check if bucket is empty
	entries, err := os.ReadDir(bucketPath)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return ErrBucketNotEmpty
	}

	return os.Remove(bucketPath)
}

// BucketExists checks if a bucket exists
func (bm *BucketManager) BucketExists(name string) bool {
	if err := ValidateBucketName(name); err != nil {
		return false
	}
	bucketPath := filepath.Join(bm.basePath, name)
	info, err := os.Stat(bucketPath)
	return err == nil && info.IsDir()
}

// ListBuckets returns all bucket names
func (bm *BucketManager) ListBuckets() ([]BucketInfo, error) {
	entries, err := os.ReadDir(bm.basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []BucketInfo{}, nil
		}
		return nil, err
	}

	var buckets []BucketInfo
	for _, entry := range entries {
		if entry.IsDir() && ValidateBucketName(entry.Name()) == nil {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			buckets = append(buckets, BucketInfo{
				Name:      entry.Name(),
				CreatedAt: info.ModTime(),
			})
		}
	}
	return buckets, nil
}

// GetBucketPath returns the filesystem path for a bucket
func (bm *BucketManager) GetBucketPath(name string) (string, error) {
	if err := ValidateBucketName(name); err != nil {
		return "", err
	}
	return filepath.Join(bm.basePath, name), nil
}


