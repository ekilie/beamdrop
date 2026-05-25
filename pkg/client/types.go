package client

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// BucketInfo represents metadata about a single bucket.
type BucketInfo struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// BucketList is the response from ListBuckets.
type BucketList struct {
	// Buckets is a slice of all accessible buckets.
	Buckets []BucketInfo `json:"buckets"`
	// Count is the total number of buckets.
	Count int `json:"count"`
}

// BucketCreated is the response from CreateBucket and CreateBucketIfNotExists.
type BucketCreated struct {
	// Bucket is the name of the bucket.
	Bucket string `json:"bucket"`
	// Created is the RFC3339 timestamp when the bucket was created (only if newly created).
	Created string `json:"created,omitempty"`
	// Exists indicates whether the bucket already existed (used in CreateBucketIfNotExists response).
	Exists bool `json:"exists,omitempty"`
	// Location is the API path to the bucket resource.
	Location string `json:"location"`
}

// ObjectInfo represents metadata about a single object in a bucket.
type ObjectInfo struct {
	// Key is the object's path/key within the bucket.
	Key string `json:"key"`
	// Size is the object's size in bytes.
	Size int64 `json:"size"`
	// LastModified is the RFC3339 timestamp of the last modification.
	LastModified time.Time `json:"lastModified"`
	// ETag is the MD5 hash of the object content, useful for detecting changes.
	ETag string `json:"etag"`
	// ContentType is the MIME type detected from the object key extension.
	ContentType string `json:"contentType,omitempty"`
}

// CommonPrefix represents a common prefix (directory-like grouping) in a ListObjects result.
type CommonPrefix struct {
	// Prefix is the shared prefix string for a group of objects.
	Prefix string `json:"prefix"`
}

// ObjectList is the response from ListObjects.
// It includes both object listings and common prefixes (for hierarchical S3-style listing).
type ObjectList struct {
	// Bucket is the bucket being listed.
	Bucket string `json:"bucket"`
	// Prefix is the prefix filter used in the request.
	Prefix string `json:"prefix"`
	// Delimiter is the delimiter used for grouping (typically "/").
	Delimiter string `json:"delimiter,omitempty"`
	// MaxKeys is the maximum number of keys requested.
	MaxKeys int `json:"maxKeys"`
	// IsTruncated indicates whether there are more results to fetch.
	IsTruncated bool `json:"isTruncated"`
	// Contents is the list of objects matching the prefix and delimiter.
	Contents []ObjectInfo `json:"contents"`
	// CommonPrefixes is a list of shared prefixes (for hierarchical listing with a delimiter).
	CommonPrefixes []CommonPrefix `json:"commonPrefixes,omitempty"`
}

// ObjectCreated is the response from PutObject and PutObjectReader.
type ObjectCreated struct {
	// Bucket is the bucket where the object was stored.
	Bucket string `json:"bucket"`
	// Key is the object's path/key.
	Key string `json:"key"`
	// ETag is the MD5 hash of the uploaded content.
	ETag string `json:"etag"`
	// Size is the total size of the uploaded object in bytes.
	Size int64 `json:"size"`
	// URL is the API path to the object resource.
	URL string `json:"url"`
}

// ObjectMetadata contains HTTP headers from a HEAD or GET request for an object.
type ObjectMetadata struct {
	// ContentType is the MIME type of the object.
	ContentType string
	// ContentLength is the size of the object in bytes.
	ContentLength int64
	// ETag is the MD5 hash of the object content.
	ETag string
	// LastModified is the RFC1123 formatted timestamp of the last modification.
	LastModified string
}

// ObjectBody is the response from GetObject, containing both metadata and the object body.
type ObjectBody struct {
	// ObjectMetadata contains headers like content type, length, and ETag.
	ObjectMetadata
	// Body is the complete object content as a byte slice.
	Body []byte
}

// ListObjectsOptions configures a ListObjects request.
type ListObjectsOptions struct {
	// Prefix filters the listing to objects whose keys start with this string.
	Prefix string
	// Delimiter groups objects by this separator (commonly "/" for hierarchical listing).
	Delimiter string
	// MaxKeys limits the number of objects returned (default 1000 if not specified).
	MaxKeys int
}

// CreatePresignedURLRequest configures a CreatePresignedURL request.
type CreatePresignedURLRequest struct {
	// Bucket is the target bucket name (required).
	Bucket string `json:"bucket"`
	// Key is the target object key/path (required).
	Key string `json:"key"`
	// Method is the HTTP method for the presigned URL ("GET" or "PUT"; defaults to "GET").
	Method string `json:"method,omitempty"`
	// ExpiresIn is the expiration time in seconds from now (optional).
	// If nil, the presigned URL never expires (subject to server policy).
	ExpiresIn *int64 `json:"expiresIn,omitempty"`
	// MaxDownloads limits the number of times the presigned URL can be used (optional).
	// If nil, the URL is unlimited in use.
	MaxDownloads *int `json:"maxDownloads,omitempty"`
}

// PresignedURL represents a server-side presigned URL record in Beamdrop's registry.
type PresignedURL struct {
	// ID is the unique database identifier (internal).
	ID uint `json:"id,omitempty"`
	// Token is the unique token for this presigned URL (used in /dl/{token}).
	Token string `json:"token"`
	// URL is the full presigned URL (including scheme, host, path, and parameters).
	URL string `json:"url,omitempty"`
	// Bucket is the target bucket name.
	Bucket string `json:"bucket"`
	// Key is the target object key/path.
	Key string `json:"key"`
	// Method is the HTTP method allowed for this presigned URL.
	Method string `json:"method"`
	// ExpiresAt is the expiration timestamp in RFC3339 format (nil if never expires).
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	// MaxDownloads is the maximum number of times this URL can be used (nil if unlimited).
	MaxDownloads *int `json:"maxDownloads,omitempty"`
	// DownloadCount is the current number of times the presigned URL has been used.
	DownloadCount int `json:"downloadCount,omitempty"`
	// CreatedBy is the access key ID that created this presigned URL.
	CreatedBy string `json:"createdBy,omitempty"`
	// CreatedAt is the RFC3339 timestamp when this presigned URL was created.
	CreatedAt time.Time `json:"createdAt"`
	// Message is an optional message from the server (e.g., error details).
	Message string `json:"message,omitempty"`
}

// PresignedURLList is the response from ListPresignedURLs.
type PresignedURLList struct {
	// URLs is the list of all presigned URL records.
	URLs []PresignedURL `json:"urls"`
	// Count is the total number of presigned URLs.
	Count int `json:"count"`
}

// metadataFromHeaders extracts ObjectMetadata from HTTP response headers.
func metadataFromHeaders(headers http.Header) ObjectMetadata {
	length, _ := strconv.ParseInt(headers.Get("Content-Length"), 10, 64)
	return ObjectMetadata{
		ContentType:   headers.Get("Content-Type"),
		ContentLength: length,
		ETag:          strings.Trim(headers.Get("ETag"), `"`),
		LastModified:  headers.Get("Last-Modified"),
	}
}
