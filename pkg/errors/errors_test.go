package errors

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	err := New(CodeInvalidRequest, CategoryValidation, "bad request", http.StatusBadRequest)
	if err.Code != CodeInvalidRequest {
		t.Fatalf("expected code %q, got %q", CodeInvalidRequest, err.Code)
	}
	if err.Category != CategoryValidation {
		t.Fatalf("expected category %q, got %q", CategoryValidation, err.Category)
	}
	if err.Message != "bad request" {
		t.Fatalf("expected message %q, got %q", "bad request", err.Message)
	}
	if err.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, err.HTTPStatus)
	}
	if err.Retryable {
		t.Fatal("validation errors should not be retryable")
	}
}

func TestNew_RetryableCategories(t *testing.T) {
	tests := []struct {
		category Category
		expected bool
	}{
		{CategoryRateLimit, true},
		{CategoryUnavailable, true},
		{CategoryValidation, false},
		{CategoryStorage, false},
		{CategoryAuth, false},
	}

	for _, tc := range tests {
		err := New("CODE", tc.category, "msg", 500)
		if err.Retryable != tc.expected {
			t.Errorf("category %q: expected retryable=%v, got %v", tc.category, tc.expected, err.Retryable)
		}
	}
}

func TestError_Error(t *testing.T) {
	err := New(CodeInvalidRequest, CategoryValidation, "bad input", 400)
	msg := err.Error()
	if msg != "INVALID_REQUEST: bad input" {
		t.Fatalf("unexpected error string: %q", msg)
	}
}

func TestError_ErrorWithCause(t *testing.T) {
	cause := errors.New("root cause")
	err := New(CodeDatabaseError, CategoryInternal, "db failed", 500).WithCause(cause)
	msg := err.Error()
	if msg != "DATABASE_ERROR: db failed: root cause" {
		t.Fatalf("unexpected error string: %q", msg)
	}
}

func TestError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := New(CodeDatabaseError, CategoryInternal, "db failed", 500).WithCause(cause)
	if !errors.Is(err, cause) {
		t.Fatal("errors.Is should find the wrapped cause")
	}
	if !errors.Is(err, err) {
		t.Fatal("errors.Is should find itself")
	}
}

func TestError_Is(t *testing.T) {
	e1 := New(CodeInvalidRequest, CategoryValidation, "msg", 400)
	e2 := New(CodeInvalidRequest, CategoryStorage, "other msg", 400)
	e3 := New(CodeFileTooLarge, CategoryValidation, "msg", 413)
	if !errors.Is(e1, e2) {
		t.Fatal("same code should match")
	}
	if errors.Is(e1, e3) {
		t.Fatal("different codes should not match")
	}
	if errors.Is(e1, errors.New("std")) {
		t.Fatal("standard errors should not match")
	}
}

func TestWithDetails(t *testing.T) {
	err := New(CodeInvalidRequest, CategoryValidation, "msg", 400).WithDetails(map[string]any{"field": "name"})
	if err.Details["field"] != "name" {
		t.Fatalf("expected detail field=name, got %v", err.Details)
	}
}

func TestWithDetail(t *testing.T) {
	err := New(CodeInvalidRequest, CategoryValidation, "msg", 400).WithDetail("field", "email")
	if err.Details["field"] != "email" {
		t.Fatalf("expected detail field=email, got %v", err.Details)
	}
}

func TestWithRetryAfter(t *testing.T) {
	err := New(CodeRateLimitExceeded, CategoryRateLimit, "slow down", 429).WithRetryAfter(5 * time.Second)
	if !err.Retryable {
		t.Fatal("expected retryable")
	}
	if err.RetryAfter != 5*time.Second {
		t.Fatalf("expected 5s retry-after, got %v", err.RetryAfter)
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("original")
	err := Wrap(cause, CodeIOError, CategoryStorage, "i/o error", 500)
	if err.Cause != cause {
		t.Fatal("Wrap should preserve cause")
	}
}

func TestFromError(t *testing.T) {
	if got := FromError(nil); got != nil {
		t.Fatal("expected nil for nil input")
	}

	appErr := New(CodeBucketNotFound, CategoryNotFound, "gone", 404)
	if got := FromError(appErr); got != appErr {
		t.Fatal("should return same *Error")
	}

	stdErr := errors.New("standard error")
	converted := FromError(stdErr)
	if converted.Code != CodeInternalError {
		t.Fatalf("expected internal error code, got %q", converted.Code)
	}
	if converted.Cause != stdErr {
		t.Fatal("should wrap the original error")
	}
}

func TestPredefinedConstructors(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expCode  Code
		expCat   Category
		expHTTP  int
	}{
		{"InvalidRequest", InvalidRequest("msg"), CodeInvalidRequest, CategoryValidation, 400},
		{"InvalidBucketName", InvalidBucketName("msg"), CodeInvalidBucketName, CategoryValidation, 400},
		{"InvalidObjectKey", InvalidObjectKey("msg"), CodeInvalidObjectKey, CategoryValidation, 400},
		{"InvalidPath", InvalidPath("msg"), CodeInvalidPath, CategoryValidation, 400},
		{"InvalidMIMEType", InvalidMIMEType("image/gif"), CodeInvalidMIMEType, CategoryValidation, 415},
		{"FileTooLarge", FileTooLarge("10MB"), CodeFileTooLarge, CategoryValidation, 413},
		{"MissingField", MissingField("email"), CodeMissingField, CategoryValidation, 400},
		{"StorageFull", StorageFull(), CodeStorageFull, CategoryStorage, 507},
		{"QuotaExceeded", QuotaExceeded("full"), CodeQuotaExceeded, CategoryStorage, 403},
		{"ObjectLocked", ObjectLocked("obj"), CodeObjectLocked, CategoryStorage, 423},
		{"WriteFailed", WriteFailed("err"), CodeWriteFailed, CategoryStorage, 500},
		{"ReadFailed", ReadFailed("err"), CodeReadFailed, CategoryStorage, 500},
		{"IOError", IOError("err"), CodeIOError, CategoryStorage, 500},
		{"ChecksumMismatch", ChecksumMismatch(), CodeChecksumMismatch, CategoryStorage, 400},
		{"Unauthorized", Unauthorized(""), CodeUnauthorized, CategoryAuth, 401},
		{"Forbidden", Forbidden(""), CodeForbidden, CategoryAuth, 403},
		{"InvalidToken", InvalidToken(""), CodeInvalidToken, CategoryAuth, 401},
		{"TokenExpired", TokenExpired(), CodeTokenExpired, CategoryAuth, 401},
		{"InvalidAPIKey", InvalidAPIKey(), CodeInvalidAPIKey, CategoryAuth, 401},
		{"APIKeyExpired", APIKeyExpired(), CodeAPIKeyExpired, CategoryAuth, 401},
		{"InvalidPassword", InvalidPassword(), CodeInvalidPassword, CategoryAuth, 401},
		{"PermissionDenied", PermissionDenied("res"), CodePermissionDenied, CategoryAuth, 403},
		{"BucketNotFound", BucketNotFound("b"), CodeBucketNotFound, CategoryNotFound, 404},
		{"ObjectNotFound", ObjectNotFound("o"), CodeObjectNotFound, CategoryNotFound, 404},
		{"FileNotFound", FileNotFound("f"), CodeFileNotFound, CategoryNotFound, 404},
		{"PathNotFound", PathNotFound("p"), CodePathNotFound, CategoryNotFound, 404},
		{"LinkNotFound", LinkNotFound("l"), CodeLinkNotFound, CategoryNotFound, 404},
		{"BucketExists", BucketExists("b"), CodeBucketExists, CategoryConflict, 409},
		{"ObjectExists", ObjectExists("o"), CodeObjectExists, CategoryConflict, 409},
		{"FileExists", FileExists("f"), CodeFileExists, CategoryConflict, 409},
		{"BucketNotEmpty", BucketNotEmpty("b"), CodeBucketNotEmpty, CategoryConflict, 409},
		{"RateLimitExceeded", RateLimitExceeded(1 * time.Second), CodeRateLimitExceeded, CategoryRateLimit, 429},
		{"TooManyRequests", TooManyRequests(1 * time.Second), CodeTooManyRequests, CategoryRateLimit, 429},
		{"BandwidthExceeded", BandwidthExceeded(), CodeBandwidthExceeded, CategoryRateLimit, 429},
		{"InternalError", InternalError(""), CodeInternalError, CategoryInternal, 500},
		{"DatabaseError", DatabaseError("db"), CodeDatabaseError, CategoryInternal, 500},
		{"ServiceUnavailable", ServiceUnavailable(5 * time.Second), CodeServiceUnavailable, CategoryUnavailable, 503},
		{"MaintenanceMode", MaintenanceMode(5 * time.Second), CodeMaintenanceMode, CategoryUnavailable, 503},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.Code != tc.expCode {
				t.Errorf("expected code %q, got %q", tc.expCode, tc.err.Code)
			}
			if tc.err.Category != tc.expCat {
				t.Errorf("expected category %q, got %q", tc.expCat, tc.err.Category)
			}
			if tc.err.HTTPStatus != tc.expHTTP {
				t.Errorf("expected HTTP %d, got %d", tc.expHTTP, tc.err.HTTPStatus)
			}
		})
	}
}

func TestWriteHTTPResponse(t *testing.T) {
	w := httptest.NewRecorder()
	err := New(CodeInvalidRequest, CategoryValidation, "bad", 400)
	err.WriteHTTPResponse(w)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Fatal("expected json content type")
	}
	body := w.Body.String()
	if body == "" {
		t.Fatal("expected non-empty body")
	}
}

func TestWriteHTTPResponse_Retryable(t *testing.T) {
	w := httptest.NewRecorder()
	err := RateLimitExceeded(5 * time.Second)
	err.WriteHTTPResponse(w)

	if w.Header().Get("X-Retryable") != "true" {
		t.Fatal("expected X-Retryable header")
	}
	if w.Header().Get("Retry-After") != "5" {
		t.Fatalf("expected Retry-After: 5, got %q", w.Header().Get("Retry-After"))
	}
}

func TestWriteHTTPResponseWithRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	err := InternalError("oops")
	err.WriteHTTPResponseWithRequestID(w, "req-123")

	if w.Header().Get("X-Request-ID") != "req-123" {
		t.Fatal("expected X-Request-ID header")
	}
}

func TestSendError(t *testing.T) {
	w := httptest.NewRecorder()
	SendError(w, errors.New("test"))
	if w.Code != 500 {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestSendErrorWithRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	SendErrorWithRequestID(w, errors.New("test"), "req-1")
	if w.Header().Get("X-Request-ID") != "req-1" {
		t.Fatal("expected X-Request-ID header")
	}
}

func TestHelperFunctions(t *testing.T) {
	notFound := BucketNotFound("b")
	validation := InvalidRequest("bad")
	auth := Unauthorized("")
	conflict := BucketExists("b")
	storage := StorageFull()
	internal := InternalError("")
	rateLimit := RateLimitExceeded(1 * time.Second)

	if !IsNotFound(notFound) {
		t.Error("IsNotFound should be true")
	}
	if !IsValidation(validation) {
		t.Error("IsValidation should be true")
	}
	if !IsAuth(auth) {
		t.Error("IsAuth should be true")
	}
	if !IsConflict(conflict) {
		t.Error("IsConflict should be true")
	}
	if !IsStorage(storage) {
		t.Error("IsStorage should be true")
	}
	if !IsRetryable(internal) {
		if GetHTTPStatus(internal) != 500 {
			t.Error("internal error should have 500 status")
		}
	}
	if !IsRetryable(rateLimit) {
		t.Error("IsRetryable should be true for rate limit")
	}
	if IsRetryable(validation) {
		t.Error("IsRetryable should be false for validation")
	}

	if GetHTTPStatus(notFound) != 404 {
		t.Errorf("expected 404, got %d", GetHTTPStatus(notFound))
	}
	if GetHTTPStatus(errors.New("std")) != 500 {
		t.Errorf("expected 500 for std error, got %d", GetHTTPStatus(errors.New("std")))
	}

	if GetCode(validation) != CodeInvalidRequest {
		t.Errorf("expected INVALID_REQUEST, got %q", GetCode(validation))
	}
	if GetCode(errors.New("std")) != CodeUnexpectedError {
		t.Errorf("expected UNEXPECTED_ERROR, got %q", GetCode(errors.New("std")))
	}

	if !HasCode(notFound, CodeBucketNotFound) {
		t.Error("HasCode should match")
	}
	if HasCode(notFound, CodeObjectNotFound) {
		t.Error("HasCode should not match wrong code")
	}
}

func TestUnauthorized_DefaultMessage(t *testing.T) {
	err := Unauthorized("")
	if err.Message != "Authentication required" {
		t.Fatalf("expected default message, got %q", err.Message)
	}
	err2 := Unauthorized("custom")
	if err2.Message != "custom" {
		t.Fatalf("expected custom message, got %q", err2.Message)
	}
}

func TestInternalError_DefaultMessage(t *testing.T) {
	err := InternalError("")
	if err.Message != "An internal error occurred" {
		t.Fatalf("expected default message, got %q", err.Message)
	}
}
