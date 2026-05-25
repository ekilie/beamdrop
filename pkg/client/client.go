// Package client provides a typed Go client for the Beamdrop S3-compatible API.
//
// The client handles HMAC-SHA256 request signing, bucket and object operations,
// and both client-side and server-side presigned URL generation.
//
// Example usage:
//
//	ctx := context.Background()
//	client, err := client.New(client.Config{
//		BaseURL:     "http://localhost:7777",
//		AccessKeyID: "BDK_abc123",
//		SecretKey:   "sk_secret",
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Create or reuse a bucket
//	_, err = client.CreateBucketIfNotExists(ctx, "my-bucket")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Upload an object
//	_, err = client.PutObject(ctx, "my-bucket", "path/to/file.txt", []byte("hello world"))
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Download an object
//	obj, err := client.GetObject(ctx, "my-bucket", "path/to/file.txt")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(string(obj.Body))
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ekilie/beamdrop/pkg/crypto"
)

const defaultUserAgent = "beamdrop-go-client/0.1"

// Config holds the configuration for creating a new Beamdrop API client.
type Config struct {
	// BaseURL is the base URL of the Beamdrop server (required).
	// Example: "http://localhost:7777" or "https://files.example.com".
	BaseURL string

	// AccessKeyID is the public API access key identifier (optional for anonymous access).
	// Format: "BDK_xxxx" where xxxx is the key ID.
	AccessKeyID string

	// SecretKey is the private API secret key (required if AccessKeyID is set).
	// Format: "sk_xxxx" where xxxx is the secret key material.
	SecretKey string

	// HTTPClient is the underlying HTTP client used for requests (optional).
	// If nil, a default client with 2-minute timeout is used.
	HTTPClient *http.Client

	// Now is a function that returns the current time in UTC (optional).
	// Used for generating request timestamps. If nil, time.Now().UTC() is used.
	// Primarily for testing and time mocking.
	Now func() time.Time

	// UserAgent is the User-Agent header sent with requests (optional).
	// If empty, defaults to "beamdrop-go-client/0.1".
	UserAgent string
}

// Client is a typed Beamdrop S3-compatible API client.
// It handles HMAC-SHA256 request signing, response decoding, and error handling.
// All methods accept a context for cancellation and timeouts.
type Client struct {
	baseURL     *url.URL
	httpClient  *http.Client
	accessKeyID string
	secretKey   string
	now         func() time.Time
	userAgent   string
}

// New creates and returns a new Beamdrop API client configured with the provided Config.
// It validates the base URL and initializes default values for HTTPClient, Now, and UserAgent if not provided.
//
// Returns an error if the base URL is invalid or empty.
func New(config Config) (*Client, error) {
	if strings.TrimSpace(config.BaseURL) == "" {
		return nil, ErrInvalidBaseURL
	}

	baseURL, err := url.Parse(config.BaseURL)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, ErrInvalidBaseURL
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 2 * time.Minute}
	}

	now := config.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}

	userAgent := config.UserAgent
	if userAgent == "" {
		userAgent = defaultUserAgent
	}

	return &Client{
		baseURL:     baseURL,
		httpClient:  httpClient,
		accessKeyID: config.AccessKeyID,
		secretKey:   config.SecretKey,
		now:         now,
		userAgent:   userAgent,
	}, nil
}

// ListBuckets returns a list of all buckets accessible with the configured credentials.
// Returns BucketList containing bucket names and creation timestamps.
func (c *Client) ListBuckets(ctx context.Context) (*BucketList, error) {
	var response BucketList
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/buckets", nil, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// CreateBucket creates a new bucket with the given name.
// Returns an error if the bucket already exists (status 409) or if the name is invalid.
// See CreateBucketIfNotExists for an idempotent variant.
func (c *Client) CreateBucket(ctx context.Context, name string) (*BucketCreated, error) {
	var response BucketCreated
	if err := c.doJSON(ctx, http.MethodPut, bucketPath(name), nil, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// CreateBucketIfNotExists creates a bucket if it does not already exist, returning an idempotent operation.
// Returns 201 Created if the bucket was newly created, or 200 OK with exists=true if it already existed.
// Recommended for initialization and bootstrap use cases.
func (c *Client) CreateBucketIfNotExists(ctx context.Context, name string) (*BucketCreated, error) {
	var response BucketCreated
	query := url.Values{"createIfNotExists": []string{"true"}}
	if err := c.doJSON(ctx, http.MethodPut, bucketPath(name), query, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// DeleteBucket deletes an empty bucket.
// Returns an error if the bucket is not found (404) or if it contains objects (409).
// Delete all objects in the bucket before calling this method.
func (c *Client) DeleteBucket(ctx context.Context, name string) error {
	return c.doNoContent(ctx, http.MethodDelete, bucketPath(name), nil)
}

// BucketExists checks whether a bucket exists using a HEAD request.
// Returns true if the bucket exists, false if it returns a 404, or an error for other failures.
func (c *Client) BucketExists(ctx context.Context, name string) (bool, error) {
	err := c.doNoContent(ctx, http.MethodHead, bucketPath(name), nil)
	if err == nil {
		return true, nil
	}
	if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, err
}

// ListObjects lists objects in a bucket with optional prefix, delimiter, and max key limit.
// Supports S3-style hierarchical listing with delimiters and common prefixes.
// MaxKeys defaults to 1000 if not specified.
func (c *Client) ListObjects(ctx context.Context, bucket string, options ListObjectsOptions) (*ObjectList, error) {
	query := url.Values{}
	if options.Prefix != "" {
		query.Set("prefix", options.Prefix)
	}
	if options.Delimiter != "" {
		query.Set("delimiter", options.Delimiter)
	}
	if options.MaxKeys > 0 {
		query.Set("max-keys", fmt.Sprintf("%d", options.MaxKeys))
	}

	var response ObjectList
	if err := c.doJSON(ctx, http.MethodGet, bucketPath(bucket), query, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// PutObject uploads an object (file) to a bucket with the given key (path).
// The body is provided as a byte slice. For streaming uploads, use PutObjectReader.
// Returns object metadata including ETag and size on success.
func (c *Client) PutObject(ctx context.Context, bucket, key string, body []byte) (*ObjectCreated, error) {
	return c.PutObjectReader(ctx, bucket, key, bytes.NewReader(body))
}

// PutObjectReader uploads an object from an io.Reader, supporting streaming for large files.
// The reader is consumed completely during upload. Provide a buffered or seekable reader for retries.
// Returns object metadata including ETag and size on success.
func (c *Client) PutObjectReader(ctx context.Context, bucket, key string, body io.Reader) (*ObjectCreated, error) {
	var response ObjectCreated
	if err := c.doJSON(ctx, http.MethodPut, objectPath(bucket, key), nil, body, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// GetObject downloads an object from a bucket with the given key.
// The entire object is read into memory and returned in ObjectBody.Body.
// For very large objects, consider using a presigned URL and a standard HTTP client instead.
func (c *Client) GetObject(ctx context.Context, bucket, key string) (*ObjectBody, error) {
	response, err := c.doRaw(ctx, http.MethodGet, objectPath(bucket, key), nil, nil)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// HeadObject retrieves metadata about an object without downloading the body.
// Returns content type, length, ETag, and last modified timestamp.
func (c *Client) HeadObject(ctx context.Context, bucket, key string) (*ObjectMetadata, error) {
	response, err := c.doRaw(ctx, http.MethodHead, objectPath(bucket, key), nil, nil)
	if err != nil {
		return nil, err
	}
	return &response.ObjectMetadata, nil
}

// DeleteObject deletes an object from a bucket.
// Returns an error if the object is not found (404) or if the deletion fails.
func (c *Client) DeleteObject(ctx context.Context, bucket, key string) error {
	return c.doNoContent(ctx, http.MethodDelete, objectPath(bucket, key), nil)
}

// ObjectExists checks whether an object exists using a HEAD request.
// Returns true if the object exists, false if it returns a 404, or an error for other failures.
func (c *Client) ObjectExists(ctx context.Context, bucket, key string) (bool, error) {
	err := c.doNoContent(ctx, http.MethodHead, objectPath(bucket, key), nil)
	if err == nil {
		return true, nil
	}
	if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, err
}

// PresignObjectURL generates a client-side presigned URL for direct access to an object.
// The URL is self-contained, signed using your secret key, and does not require server involvement.
// Method should be "GET" or "PUT". The expiresAt timestamp is in UTC.
// Note: key rotation will invalidate existing presigned URLs generated this way.
// For more control over expiration and revocation, use CreatePresignedURL (server-side).
func (c *Client) PresignObjectURL(method, bucket, key string, expiresAt time.Time) (string, error) {
	if c.accessKeyID == "" || c.secretKey == "" {
		return "", ErrMissingCredentials
	}

	method = strings.ToUpper(method)
	path := objectPath(bucket, key)
	relative := &url.URL{Path: path}
	presignedURL := c.baseURL.ResolveReference(relative)
	query := presignedURL.Query()
	query.Set("access_key", c.accessKeyID)
	query.Set("expires", expiresAt.UTC().Format(time.RFC3339))
	query.Set("token", crypto.GeneratePresignedToken(c.secretKey, method, bucket, key, expiresAt.UTC()))
	presignedURL.RawQuery = query.Encode()
	return presignedURL.String(), nil
}

// CreatePresignedURL creates a server-side presigned URL with optional download limits and revocation support.
// The URL is stored in Beamdrop's presigned URL registry and can be revoked at any time.
// ExpiresIn is specified in seconds; if nil, the URL never expires (depends on server policy).
// MaxDownloads limits the number of times the presigned URL can be used; if nil, unlimited.
// Returns a PresignedURL containing the token and the full URL.
func (c *Client) CreatePresignedURL(ctx context.Context, request CreatePresignedURLRequest) (*PresignedURL, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpRequest, err := c.newRequest(ctx, http.MethodPost, "/api/v1/presign", nil, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")

	httpResponse, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		apiErr := decodeAPIError(httpResponse)
		httpResponse.Body.Close()
		return nil, apiErr
	}
	defer httpResponse.Body.Close()

	var response PresignedURL
	if err := json.NewDecoder(httpResponse.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &response, nil
}

// ListPresignedURLs lists all server-side presigned URLs created with this API key.
// Returns a PresignedURLList containing all tokens, buckets, keys, and metadata.
func (c *Client) ListPresignedURLs(ctx context.Context) (*PresignedURLList, error) {
	var response PresignedURLList
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/presign", nil, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// GetPresignedURL retrieves a single server-side presigned URL by its token.
// Returns an error if the token is not found or has expired.
func (c *Client) GetPresignedURL(ctx context.Context, token string) (*PresignedURL, error) {
	var response PresignedURL
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/presign/"+url.PathEscape(token), nil, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// DeletePresignedURL revokes a server-side presigned URL by its token.
// The URL becomes invalid immediately and cannot be used for access.
func (c *Client) DeletePresignedURL(ctx context.Context, token string) error {
	return c.doNoContent(ctx, http.MethodDelete, "/api/v1/presign/"+url.PathEscape(token), nil)
}

func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, body io.Reader, dst any) error {
	response, err := c.do(ctx, method, path, query, body)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if dst == nil || response.StatusCode == http.StatusNoContent {
		io.Copy(io.Discard, response.Body)
		return nil
	}

	if err := json.NewDecoder(response.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) doNoContent(ctx context.Context, method, path string, query url.Values) error {
	response, err := c.do(ctx, method, path, query, nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	io.Copy(io.Discard, response.Body)
	return nil
}

func (c *Client) doRaw(ctx context.Context, method, path string, query url.Values, body io.Reader) (*ObjectBody, error) {
	response, err := c.do(ctx, method, path, query, body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	metadata := metadataFromHeaders(response.Header)
	result := &ObjectBody{ObjectMetadata: metadata}
	if method != http.MethodHead {
		result.Body, err = io.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf("read response body: %w", err)
		}
	}
	return result, nil
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Response, error) {
	request, err := c.newRequest(ctx, method, path, query, body)
	if err != nil {
		return nil, err
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		apiErr := decodeAPIError(response)
		response.Body.Close()
		return nil, apiErr
	}

	return response, nil
}

func (c *Client) newRequest(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Request, error) {
	if strings.TrimSpace(path) == "" {
		return nil, ErrInvalidPath
	}

	relative := &url.URL{Path: path}
	if len(query) > 0 {
		relative.RawQuery = query.Encode()
	}

	requestURL := c.baseURL.ResolveReference(relative)
	request, err := http.NewRequestWithContext(ctx, method, requestURL.String(), body)
	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", c.userAgent)
	request.Header.Set("Accept", "application/json")

	if method == http.MethodPut || method == http.MethodPost {
		if request.Header.Get("Content-Type") == "" {
			request.Header.Set("Content-Type", "application/octet-stream")
		}
	}

	if c.accessKeyID != "" || c.secretKey != "" {
		if c.accessKeyID == "" || c.secretKey == "" {
			return nil, ErrMissingCredentials
		}

		timestamp := c.now().UTC().Format(time.RFC3339)
		signature := crypto.GenerateSignature(c.secretKey, method, path, timestamp)
		request.Header.Set("Authorization", "Bearer "+c.accessKeyID+":"+signature)
		request.Header.Set("X-Beamdrop-Date", timestamp)
	}

	return request, nil
}

func bucketPath(bucket string) string {
	return "/api/v1/buckets/" + url.PathEscape(bucket)
}

func objectPath(bucket, key string) string {
	segments := strings.Split(key, "/")
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}
	return bucketPath(bucket) + "/" + strings.Join(segments, "/")
}
