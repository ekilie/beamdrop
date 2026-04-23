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

type Config struct {
	BaseURL     string
	AccessKeyID string
	SecretKey   string
	HTTPClient  *http.Client
	Now         func() time.Time
	UserAgent   string
}

type Client struct {
	baseURL     *url.URL
	httpClient  *http.Client
	accessKeyID string
	secretKey   string
	now         func() time.Time
	userAgent   string
}

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

func (c *Client) ListBuckets(ctx context.Context) (*BucketList, error) {
	var response BucketList
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/buckets", nil, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) CreateBucket(ctx context.Context, name string) (*BucketCreated, error) {
	var response BucketCreated
	if err := c.doJSON(ctx, http.MethodPut, bucketPath(name), nil, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) CreateBucketIfNotExists(ctx context.Context, name string) (*BucketCreated, error) {
	var response BucketCreated
	query := url.Values{"createIfNotExists": []string{"true"}}
	if err := c.doJSON(ctx, http.MethodPut, bucketPath(name), query, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) DeleteBucket(ctx context.Context, name string) error {
	return c.doNoContent(ctx, http.MethodDelete, bucketPath(name), nil)
}

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

func (c *Client) PutObject(ctx context.Context, bucket, key string, body []byte) (*ObjectCreated, error) {
	return c.PutObjectReader(ctx, bucket, key, bytes.NewReader(body))
}

func (c *Client) PutObjectReader(ctx context.Context, bucket, key string, body io.Reader) (*ObjectCreated, error) {
	var response ObjectCreated
	if err := c.doJSON(ctx, http.MethodPut, objectPath(bucket, key), nil, body, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GetObject(ctx context.Context, bucket, key string) (*ObjectBody, error) {
	response, err := c.doRaw(ctx, http.MethodGet, objectPath(bucket, key), nil, nil)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) HeadObject(ctx context.Context, bucket, key string) (*ObjectMetadata, error) {
	response, err := c.doRaw(ctx, http.MethodHead, objectPath(bucket, key), nil, nil)
	if err != nil {
		return nil, err
	}
	return &response.ObjectMetadata, nil
}

func (c *Client) DeleteObject(ctx context.Context, bucket, key string) error {
	return c.doNoContent(ctx, http.MethodDelete, objectPath(bucket, key), nil)
}

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

func (c *Client) CreatePresignedURL(ctx context.Context, request CreatePresignedURLRequest) (*PresignedURL, error) {
	var response PresignedURL
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/presign", nil, jsonBody(request), &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ListPresignedURLs(ctx context.Context) (*PresignedURLList, error) {
	var response PresignedURLList
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/presign", nil, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GetPresignedURL(ctx context.Context, token string) (*PresignedURL, error) {
	var response PresignedURL
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/presign/"+url.PathEscape(token), nil, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

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

func jsonBody(payload any) io.Reader {
	if payload == nil {
		return nil
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return bytes.NewReader(nil)
	}
	return bytes.NewReader(body)
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
