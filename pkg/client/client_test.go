package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	beamcrypto "github.com/ekilie/beamdrop/pkg/crypto"
)

func TestListBucketsSignsRequest(t *testing.T) {
	const (
		accessKey = "BDK_test"
		secretKey = "sk_test"
	)

	fixedTime := time.Date(2026, 4, 23, 10, 11, 12, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/buckets" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		expectedSignature := beamcrypto.GenerateSignature(secretKey, http.MethodGet, "/api/v1/buckets", fixedTime.Format(time.RFC3339))
		expectedAuthorization := "Bearer " + accessKey + ":" + expectedSignature

		if got := r.Header.Get("Authorization"); got != expectedAuthorization {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		if got := r.Header.Get("X-Beamdrop-Date"); got != fixedTime.Format(time.RFC3339) {
			t.Fatalf("unexpected timestamp header: %s", got)
		}

		writeJSON(t, w, http.StatusOK, BucketList{Buckets: []BucketInfo{{Name: "media"}}, Count: 1})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, accessKey, secretKey, fixedTime)

	response, err := client.ListBuckets(context.Background())
	if err != nil {
		t.Fatalf("ListBuckets returned error: %v", err)
	}
	if response.Count != 1 || len(response.Buckets) != 1 || response.Buckets[0].Name != "media" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestGetObjectReturnsBodyAndMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/buckets/media/folder/hello%20world.txt" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Length", "5")
		w.Header().Set("ETag", `"abc123"`)
		w.Header().Set("Last-Modified", time.Unix(0, 0).UTC().Format(http.TimeFormat))
		_, _ = io.WriteString(w, "hello")
	}))
	defer server.Close()

	client := newAnonymousTestClient(t, server.URL)

	response, err := client.GetObject(context.Background(), "media", "folder/hello world.txt")
	if err != nil {
		t.Fatalf("GetObject returned error: %v", err)
	}
	if string(response.Body) != "hello" {
		t.Fatalf("unexpected body: %q", string(response.Body))
	}
	if response.ETag != "abc123" || response.ContentType != "text/plain" || response.ContentLength != 5 {
		t.Fatalf("unexpected metadata: %+v", response.ObjectMetadata)
	}
}

func TestDeleteBucketReturnsStructuredError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusConflict, map[string]any{
			"code":     "BUCKET_NOT_EMPTY",
			"category": "CONFLICT",
			"message":  "Bucket is not empty",
		})
	}))
	defer server.Close()

	client := newAnonymousTestClient(t, server.URL)

	err := client.DeleteBucket(context.Background(), "media")
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusConflict || apiErr.Code != "BUCKET_NOT_EMPTY" {
		t.Fatalf("unexpected error: %+v", apiErr)
	}
}

func TestPresignObjectURL(t *testing.T) {
	expiresAt := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)
	client := newTestClient(t, "https://beamdrop.example.com", "BDK_test", "sk_test", expiresAt)

	presignedURL, err := client.PresignObjectURL(http.MethodGet, "media", "folder/file.txt", expiresAt)
	if err != nil {
		t.Fatalf("PresignObjectURL returned error: %v", err)
	}

	if !strings.Contains(presignedURL, "access_key=BDK_test") {
		t.Fatalf("missing access key in URL: %s", presignedURL)
	}
	if !strings.Contains(presignedURL, "expires=2026-04-23T12%3A00%3A00Z") {
		t.Fatalf("missing expiration in URL: %s", presignedURL)
	}
	if !strings.Contains(presignedURL, "token=") {
		t.Fatalf("missing token in URL: %s", presignedURL)
	}
	if !strings.Contains(presignedURL, "/api/v1/buckets/media/folder/file.txt") {
		t.Fatalf("missing object path in URL: %s", presignedURL)
	}
}

func TestCreatePresignedURLUsesJSONBody(t *testing.T) {
	expiresIn := int64(3600)
	maxDownloads := 2
	createdAt := time.Date(2026, 4, 23, 14, 0, 0, 0, time.UTC)
	var baseURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/presign" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("unexpected content type: %s", got)
		}

		var request CreatePresignedURLRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		expected := CreatePresignedURLRequest{
			Bucket:       "media",
			Key:          "folder/file.txt",
			Method:       http.MethodGet,
			ExpiresIn:    &expiresIn,
			MaxDownloads: &maxDownloads,
		}
		if !reflect.DeepEqual(request, expected) {
			t.Fatalf("unexpected request payload: %+v", request)
		}

		writeJSON(t, w, http.StatusCreated, PresignedURL{
			Token:        "abc123",
			URL:          baseURL + "/dl/abc123",
			Bucket:       "media",
			Key:          "folder/file.txt",
			Method:       http.MethodGet,
			MaxDownloads: &maxDownloads,
			CreatedAt:    createdAt,
		})
	}))
	defer server.Close()
	baseURL = server.URL

	client := newAnonymousTestClient(t, server.URL)
	response, err := client.CreatePresignedURL(context.Background(), CreatePresignedURLRequest{
		Bucket:       "media",
		Key:          "folder/file.txt",
		Method:       http.MethodGet,
		ExpiresIn:    &expiresIn,
		MaxDownloads: &maxDownloads,
	})
	if err != nil {
		t.Fatalf("CreatePresignedURL returned error: %v", err)
	}
	if response.Token != "abc123" || response.URL == "" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func newTestClient(t *testing.T, baseURL, accessKeyID, secretKey string, now time.Time) *Client {
	t.Helper()

	client, err := New(Config{
		BaseURL:     baseURL,
		AccessKeyID: accessKeyID,
		SecretKey:   secretKey,
		Now:         func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	return client
}

func newAnonymousTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()

	client, err := New(Config{BaseURL: baseURL})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	return client
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write JSON response: %v", err)
	}
}
