package client

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

type BucketInfo struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type BucketList struct {
	Buckets []BucketInfo `json:"buckets"`
	Count   int          `json:"count"`
}

type BucketCreated struct {
	Bucket   string `json:"bucket"`
	Created  string `json:"created,omitempty"`
	Exists   bool   `json:"exists,omitempty"`
	Location string `json:"location"`
}

type ObjectInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModified"`
	ETag         string    `json:"etag"`
	ContentType  string    `json:"contentType,omitempty"`
}

type CommonPrefix struct {
	Prefix string `json:"prefix"`
}

type ObjectList struct {
	Bucket         string         `json:"bucket"`
	Prefix         string         `json:"prefix"`
	Delimiter      string         `json:"delimiter,omitempty"`
	MaxKeys        int            `json:"maxKeys"`
	IsTruncated    bool           `json:"isTruncated"`
	Contents       []ObjectInfo   `json:"contents"`
	CommonPrefixes []CommonPrefix `json:"commonPrefixes,omitempty"`
}

type ObjectCreated struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
	ETag   string `json:"etag"`
	Size   int64  `json:"size"`
	URL    string `json:"url"`
}

type ObjectMetadata struct {
	ContentType   string
	ContentLength int64
	ETag          string
	LastModified  string
}

type ObjectBody struct {
	ObjectMetadata
	Body []byte
}

type ListObjectsOptions struct {
	Prefix    string
	Delimiter string
	MaxKeys   int
}

type CreatePresignedURLRequest struct {
	Bucket       string `json:"bucket"`
	Key          string `json:"key"`
	Method       string `json:"method,omitempty"`
	ExpiresIn    *int64 `json:"expiresIn,omitempty"`
	MaxDownloads *int   `json:"maxDownloads,omitempty"`
}

type PresignedURL struct {
	ID            uint       `json:"id,omitempty"`
	Token         string     `json:"token"`
	URL           string     `json:"url,omitempty"`
	Bucket        string     `json:"bucket"`
	Key           string     `json:"key"`
	Method        string     `json:"method"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
	MaxDownloads  *int       `json:"maxDownloads,omitempty"`
	DownloadCount int        `json:"downloadCount,omitempty"`
	CreatedBy     string     `json:"createdBy,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	Message       string     `json:"message,omitempty"`
}

type PresignedURLList struct {
	URLs  []PresignedURL `json:"urls"`
	Count int            `json:"count"`
}

func metadataFromHeaders(headers http.Header) ObjectMetadata {
	length, _ := strconv.ParseInt(headers.Get("Content-Length"), 10, 64)
	return ObjectMetadata{
		ContentType:   headers.Get("Content-Type"),
		ContentLength: length,
		ETag:          strings.Trim(headers.Get("ETag"), `"`),
		LastModified:  headers.Get("Last-Modified"),
	}
}
