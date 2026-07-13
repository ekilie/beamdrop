package reqctx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware_SetsRequestID(t *testing.T) {
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id == "" {
			t.Error("expected request ID in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get(RequestIDHeader) == "" {
		t.Fatal("expected X-Request-ID header in response")
	}
}

func TestMiddleware_PropagatesExistingRequestID(t *testing.T) {
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id != "existing-id" {
			t.Errorf("expected 'existing-id', got %q", id)
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set(RequestIDHeader, "existing-id")
	handler.ServeHTTP(w, r)
}

func TestTimeoutMiddleware_Default(t *testing.T) {
	handler := TimeoutMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Deadline(); !ok {
			t.Error("expected deadline on context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/test", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestTimeoutMiddleware_Upload(t *testing.T) {
	handler := TimeoutMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := r.Context().Deadline()
		if !ok {
			t.Fatal("expected deadline")
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/upload", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestTimeoutMiddleware_Download(t *testing.T) {
	handler := TimeoutMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Deadline(); !ok {
			t.Error("expected deadline")
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/download", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestTimeoutMiddleware_APIUpload(t *testing.T) {
	handler := TimeoutMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Deadline(); !ok {
			t.Error("expected deadline")
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/api/v1/buckets/my-bucket/objects/key", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestTimeoutMiddleware_APIDownload(t *testing.T) {
	handler := TimeoutMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Deadline(); !ok {
			t.Error("expected deadline")
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/buckets/my-bucket/objects/key", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestIsUploadRequest(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   bool
	}{
		{"POST", "/upload", true},
		{"PUT", "/upload", true},
		{"PUT", "/api/v1/buckets/my-bucket/objects/key", true},
		{"GET", "/upload", false},
		{"POST", "/api/v1/buckets/obj", false},
		{"GET", "/health", false},
	}

	for _, tc := range tests {
		r := httptest.NewRequest(tc.method, tc.path, nil)
		got := isUploadRequest(r)
		if got != tc.want {
			t.Errorf("isUploadRequest(%s %s) = %v, want %v", tc.method, tc.path, got, tc.want)
		}
	}
}

func TestIsDownloadRequest(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   bool
	}{
		{"GET", "/download", true},
		{"GET", "/api/v1/buckets/my-bucket/objects/key", true},
		{"POST", "/download", false},
		{"GET", "/health", false},
	}

	for _, tc := range tests {
		r := httptest.NewRequest(tc.method, tc.path, nil)
		got := isDownloadRequest(r)
		if got != tc.want {
			t.Errorf("isDownloadRequest(%s %s) = %v, want %v", tc.method, tc.path, got, tc.want)
		}
	}
}

func TestResponseWriterWrapper_WriteHeader(t *testing.T) {
	inner := httptest.NewRecorder()
	w := &responseWriterWrapper{ResponseWriter: inner}

	w.WriteHeader(http.StatusNotFound)
	if w.status != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.status)
	}
	if !w.written {
		t.Fatal("expected written flag")
	}

	w.WriteHeader(http.StatusOK)
	if w.status != http.StatusNotFound {
		t.Fatalf("first WriteHeader should not be overwritten, got %d", w.status)
	}
}

func TestResponseWriterWrapper_Write(t *testing.T) {
	inner := httptest.NewRecorder()
	w := &responseWriterWrapper{ResponseWriter: inner}

	n, err := w.Write([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Fatalf("expected 5 bytes written, got %d", n)
	}
	if w.status != http.StatusOK {
		t.Fatalf("expected default status %d, got %d", http.StatusOK, w.status)
	}
	if inner.Body.String() != "hello" {
		t.Fatalf("expected body 'hello', got %q", inner.Body.String())
	}
}
