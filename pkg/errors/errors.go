// Package errors provides structured error handling with error codes,
// categories, and HTTP response helpers for the beamdrop application.
package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Category represents the category of an error
type Category string

const (
	// CategoryValidation for input validation errors
	CategoryValidation Category = "VALIDATION"
	// CategoryStorage for storage-related errors
	CategoryStorage Category = "STORAGE"
	// CategoryAuth for authentication/authorization errors
	CategoryAuth Category = "AUTH"
	// CategoryNotFound for resource not found errors
	CategoryNotFound Category = "NOT_FOUND"
	// CategoryConflict for resource conflict errors
	CategoryConflict Category = "CONFLICT"
	// CategoryRateLimit for rate limiting errors
	CategoryRateLimit Category = "RATE_LIMIT"
	// CategoryInternal for internal server errors
	CategoryInternal Category = "INTERNAL"
	// CategoryUnavailable for service unavailable errors
	CategoryUnavailable Category = "UNAVAILABLE"
)

// Code represents a specific error code
type Code string

// Validation error codes
const (
	CodeInvalidRequest    Code = "INVALID_REQUEST"
	CodeInvalidBucketName Code = "INVALID_BUCKET_NAME"
	CodeInvalidObjectKey  Code = "INVALID_OBJECT_KEY"
	CodeInvalidPath       Code = "INVALID_PATH"
	CodeInvalidMIMEType   Code = "INVALID_MIME_TYPE"
	CodeFileTooLarge      Code = "FILE_TOO_LARGE"
	CodeMissingField      Code = "MISSING_FIELD"
)

// Storage error codes
const (
	CodeStorageFull      Code = "STORAGE_FULL"
	CodeQuotaExceeded    Code = "QUOTA_EXCEEDED"
	CodeObjectLocked     Code = "OBJECT_LOCKED"
	CodeWriteFailed      Code = "WRITE_FAILED"
	CodeReadFailed       Code = "READ_FAILED"
	CodeDeleteFailed     Code = "DELETE_FAILED"
	CodeIOError          Code = "IO_ERROR"
	CodeChecksumMismatch Code = "CHECKSUM_MISMATCH"
)

// Auth error codes
const (
	CodeUnauthorized     Code = "UNAUTHORIZED"
	CodeForbidden        Code = "FORBIDDEN"
	CodeInvalidToken     Code = "INVALID_TOKEN"
	CodeTokenExpired     Code = "TOKEN_EXPIRED"
	CodeInvalidAPIKey    Code = "INVALID_API_KEY"
	CodeAPIKeyExpired    Code = "API_KEY_EXPIRED"
	CodeInvalidPassword  Code = "INVALID_PASSWORD"
	CodeSessionExpired   Code = "SESSION_EXPIRED"
	CodePermissionDenied Code = "PERMISSION_DENIED"
)

// Not found error codes
const (
	CodeBucketNotFound Code = "BUCKET_NOT_FOUND"
	CodeObjectNotFound Code = "OBJECT_NOT_FOUND"
	CodeFileNotFound   Code = "FILE_NOT_FOUND"
	CodePathNotFound   Code = "PATH_NOT_FOUND"
	CodeLinkNotFound   Code = "LINK_NOT_FOUND"
)

// Conflict error codes
const (
	CodeBucketExists    Code = "BUCKET_EXISTS"
	CodeObjectExists    Code = "OBJECT_EXISTS"
	CodeFileExists      Code = "FILE_EXISTS"
	CodeBucketNotEmpty  Code = "BUCKET_NOT_EMPTY"
	CodeVersionConflict Code = "VERSION_CONFLICT"
)

// Rate limit error codes
const (
	CodeRateLimitExceeded Code = "RATE_LIMIT_EXCEEDED"
	CodeTooManyRequests   Code = "TOO_MANY_REQUESTS"
	CodeBandwidthExceeded Code = "BANDWIDTH_EXCEEDED"
)

// Internal error codes
const (
	CodeInternalError   Code = "INTERNAL_ERROR"
	CodeDatabaseError   Code = "DATABASE_ERROR"
	CodeConfigError     Code = "CONFIG_ERROR"
	CodeUnexpectedError Code = "UNEXPECTED_ERROR"
)

// Unavailable error codes
const (
	CodeServiceUnavailable     Code = "SERVICE_UNAVAILABLE"
	CodeMaintenanceMode        Code = "MAINTENANCE_MODE"
	CodeTemporarilyUnavailable Code = "TEMPORARILY_UNAVAILABLE"
)

// Error represents a structured application error
type Error struct {
	// Code is the machine-readable error code
	Code Code `json:"code"`
	// Category is the error category
	Category Category `json:"category"`
	// Message is the human-readable error message
	Message string `json:"message"`
	// Details contains additional error details (optional)
	Details map[string]any `json:"details,omitempty"`
	// HTTPStatus is the HTTP status code to return
	HTTPStatus int `json:"-"`
	// Retryable indicates if the operation can be retried
	Retryable bool `json:"-"`
	// RetryAfter indicates when the operation can be retried (for rate limiting)
	RetryAfter time.Duration `json:"-"`
	// Cause is the underlying error (for wrapping)
	Cause error `json:"-"`
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for errors.Is/As support
func (e *Error) Unwrap() error {
	return e.Cause
}

// Is implements error comparison
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// WithDetails adds details to the error
func (e *Error) WithDetails(details map[string]any) *Error {
	e.Details = details
	return e
}

// WithDetail adds a single detail to the error
func (e *Error) WithDetail(key string, value any) *Error {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}
	e.Details[key] = value
	return e
}

// WithCause wraps an underlying error
func (e *Error) WithCause(cause error) *Error {
	e.Cause = cause
	return e
}

// WithRetryAfter sets the retry-after duration
func (e *Error) WithRetryAfter(d time.Duration) *Error {
	e.RetryAfter = d
	e.Retryable = true
	return e
}

// New creates a new Error with the given parameters
func New(code Code, category Category, message string, httpStatus int) *Error {
	return &Error{
		Code:       code,
		Category:   category,
		Message:    message,
		HTTPStatus: httpStatus,
		Retryable:  isRetryableCategory(category),
	}
}

// isRetryableCategory returns true if errors in this category are typically retryable
func isRetryableCategory(category Category) bool {
	switch category {
	case CategoryRateLimit, CategoryUnavailable:
		return true
	default:
		return false
	}
}

// Wrap wraps an existing error with a structured Error
func Wrap(err error, code Code, category Category, message string, httpStatus int) *Error {
	return &Error{
		Code:       code,
		Category:   category,
		Message:    message,
		HTTPStatus: httpStatus,
		Retryable:  isRetryableCategory(category),
		Cause:      err,
	}
}

// FromError attempts to convert a standard error to a structured Error
// If the error is already a structured Error, it returns it as-is
// Otherwise, it wraps it as an internal error
func FromError(err error) *Error {
	if err == nil {
		return nil
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}

	return InternalError("An unexpected error occurred").WithCause(err)
}

// ============================================================================
// Pre-defined error constructors
// ============================================================================

// Validation errors

func InvalidRequest(message string) *Error {
	return New(CodeInvalidRequest, CategoryValidation, message, http.StatusBadRequest)
}

func InvalidBucketName(message string) *Error {
	return New(CodeInvalidBucketName, CategoryValidation, message, http.StatusBadRequest)
}

func InvalidObjectKey(message string) *Error {
	return New(CodeInvalidObjectKey, CategoryValidation, message, http.StatusBadRequest)
}

func InvalidPath(message string) *Error {
	return New(CodeInvalidPath, CategoryValidation, message, http.StatusBadRequest)
}

func InvalidMIMEType(mimeType string) *Error {
	return New(CodeInvalidMIMEType, CategoryValidation, fmt.Sprintf("File type '%s' is not allowed", mimeType), http.StatusUnsupportedMediaType)
}

func FileTooLarge(maxSize string) *Error {
	return New(CodeFileTooLarge, CategoryValidation, fmt.Sprintf("File too large. Maximum size is %s", maxSize), http.StatusRequestEntityTooLarge)
}

func MissingField(field string) *Error {
	return New(CodeMissingField, CategoryValidation, fmt.Sprintf("Required field '%s' is missing", field), http.StatusBadRequest)
}

// Storage errors

func StorageFull() *Error {
	e := New(CodeStorageFull, CategoryStorage, "Storage is full", http.StatusInsufficientStorage)
	e.Retryable = true
	return e
}

func QuotaExceeded(message string) *Error {
	return New(CodeQuotaExceeded, CategoryStorage, message, http.StatusForbidden)
}

func ObjectLocked(key string) *Error {
	e := New(CodeObjectLocked, CategoryStorage, fmt.Sprintf("Object '%s' is locked", key), http.StatusLocked)
	e.Retryable = true
	return e
}

func WriteFailed(message string) *Error {
	return New(CodeWriteFailed, CategoryStorage, message, http.StatusInternalServerError)
}

func ReadFailed(message string) *Error {
	return New(CodeReadFailed, CategoryStorage, message, http.StatusInternalServerError)
}

func IOError(message string) *Error {
	return New(CodeIOError, CategoryStorage, message, http.StatusInternalServerError)
}

func ChecksumMismatch() *Error {
	return New(CodeChecksumMismatch, CategoryStorage, "Checksum verification failed", http.StatusBadRequest)
}

// Auth errors

func Unauthorized(message string) *Error {
	if message == "" {
		message = "Authentication required"
	}
	return New(CodeUnauthorized, CategoryAuth, message, http.StatusUnauthorized)
}

func Forbidden(message string) *Error {
	if message == "" {
		message = "Access denied"
	}
	return New(CodeForbidden, CategoryAuth, message, http.StatusForbidden)
}

func InvalidToken(message string) *Error {
	if message == "" {
		message = "Invalid or malformed token"
	}
	return New(CodeInvalidToken, CategoryAuth, message, http.StatusUnauthorized)
}

func TokenExpired() *Error {
	return New(CodeTokenExpired, CategoryAuth, "Token has expired", http.StatusUnauthorized)
}

func InvalidAPIKey() *Error {
	return New(CodeInvalidAPIKey, CategoryAuth, "Invalid API key", http.StatusUnauthorized)
}

func APIKeyExpired() *Error {
	return New(CodeAPIKeyExpired, CategoryAuth, "API key has expired", http.StatusUnauthorized)
}

func InvalidPassword() *Error {
	return New(CodeInvalidPassword, CategoryAuth, "Invalid password", http.StatusUnauthorized)
}

func PermissionDenied(resource string) *Error {
	return New(CodePermissionDenied, CategoryAuth, fmt.Sprintf("Permission denied for resource: %s", resource), http.StatusForbidden)
}

// Not found errors

func BucketNotFound(bucket string) *Error {
	return New(CodeBucketNotFound, CategoryNotFound, fmt.Sprintf("Bucket '%s' not found", bucket), http.StatusNotFound)
}

func ObjectNotFound(key string) *Error {
	return New(CodeObjectNotFound, CategoryNotFound, fmt.Sprintf("Object '%s' not found", key), http.StatusNotFound)
}

func FileNotFound(path string) *Error {
	return New(CodeFileNotFound, CategoryNotFound, fmt.Sprintf("File '%s' not found", path), http.StatusNotFound)
}

func PathNotFound(path string) *Error {
	return New(CodePathNotFound, CategoryNotFound, fmt.Sprintf("Path '%s' not found", path), http.StatusNotFound)
}

func LinkNotFound(id string) *Error {
	return New(CodeLinkNotFound, CategoryNotFound, fmt.Sprintf("Link '%s' not found", id), http.StatusNotFound)
}

// Conflict errors

func BucketExists(bucket string) *Error {
	return New(CodeBucketExists, CategoryConflict, fmt.Sprintf("Bucket '%s' already exists", bucket), http.StatusConflict)
}

func ObjectExists(key string) *Error {
	return New(CodeObjectExists, CategoryConflict, fmt.Sprintf("Object '%s' already exists", key), http.StatusConflict)
}

func FileExists(path string) *Error {
	return New(CodeFileExists, CategoryConflict, fmt.Sprintf("File '%s' already exists", path), http.StatusConflict)
}

func BucketNotEmpty(bucket string) *Error {
	return New(CodeBucketNotEmpty, CategoryConflict, fmt.Sprintf("Bucket '%s' is not empty", bucket), http.StatusConflict)
}

// Rate limit errors

func RateLimitExceeded(retryAfter time.Duration) *Error {
	e := New(CodeRateLimitExceeded, CategoryRateLimit, "Rate limit exceeded", http.StatusTooManyRequests)
	e.RetryAfter = retryAfter
	e.Retryable = true
	return e
}

func TooManyRequests(retryAfter time.Duration) *Error {
	e := New(CodeTooManyRequests, CategoryRateLimit, "Too many requests", http.StatusTooManyRequests)
	e.RetryAfter = retryAfter
	e.Retryable = true
	return e
}

func BandwidthExceeded() *Error {
	e := New(CodeBandwidthExceeded, CategoryRateLimit, "Bandwidth limit exceeded", http.StatusTooManyRequests)
	e.Retryable = true
	return e
}

// Internal errors

func InternalError(message string) *Error {
	if message == "" {
		message = "An internal error occurred"
	}
	return New(CodeInternalError, CategoryInternal, message, http.StatusInternalServerError)
}

func DatabaseError(message string) *Error {
	return New(CodeDatabaseError, CategoryInternal, message, http.StatusInternalServerError)
}

// Unavailable errors

func ServiceUnavailable(retryAfter time.Duration) *Error {
	e := New(CodeServiceUnavailable, CategoryUnavailable, "Service temporarily unavailable", http.StatusServiceUnavailable)
	e.RetryAfter = retryAfter
	e.Retryable = true
	return e
}

func MaintenanceMode(retryAfter time.Duration) *Error {
	e := New(CodeMaintenanceMode, CategoryUnavailable, "Service is under maintenance", http.StatusServiceUnavailable)
	e.RetryAfter = retryAfter
	e.Retryable = true
	return e
}

// ============================================================================
// HTTP Response helpers
// ============================================================================

// APIErrorResponse represents the JSON structure for API error responses
type APIErrorResponse struct {
	Error APIErrorBody `json:"error"`
}

// APIErrorBody represents the error body in API responses
type APIErrorBody struct {
	Code     Code           `json:"code"`
	Message  string         `json:"message"`
	Category Category       `json:"category,omitempty"`
	Details  map[string]any `json:"details,omitempty"`
	// RequestID for tracing (optional, set by middleware)
	RequestID string `json:"requestId,omitempty"`
}

// WriteHTTPResponse writes the error as an HTTP JSON response
func (e *Error) WriteHTTPResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")

	// Add retry headers for retryable errors
	if e.Retryable {
		w.Header().Set("X-Retryable", "true")
	}

	if e.RetryAfter > 0 {
		// Set Retry-After header in seconds
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(e.RetryAfter.Seconds())))
	}

	w.WriteHeader(e.HTTPStatus)

	response := APIErrorResponse{
		Error: APIErrorBody{
			Code:     e.Code,
			Message:  e.Message,
			Category: e.Category,
			Details:  e.Details,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// WriteHTTPResponseWithRequestID writes the error with a request ID
func (e *Error) WriteHTTPResponseWithRequestID(w http.ResponseWriter, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)

	// Add retry headers for retryable errors
	if e.Retryable {
		w.Header().Set("X-Retryable", "true")
	}

	if e.RetryAfter > 0 {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(e.RetryAfter.Seconds())))
	}

	w.WriteHeader(e.HTTPStatus)

	response := APIErrorResponse{
		Error: APIErrorBody{
			Code:      e.Code,
			Message:   e.Message,
			Category:  e.Category,
			Details:   e.Details,
			RequestID: requestID,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// SendError is a convenience function to send any error as HTTP response
// It converts standard errors to structured errors if needed
func SendError(w http.ResponseWriter, err error) {
	appErr := FromError(err)
	appErr.WriteHTTPResponse(w)
}

// SendErrorWithRequestID sends an error with request ID tracking
func SendErrorWithRequestID(w http.ResponseWriter, err error, requestID string) {
	appErr := FromError(err)
	appErr.WriteHTTPResponseWithRequestID(w, requestID)
}

// ============================================================================
// Helper functions for checking error types
// ============================================================================

// IsNotFound returns true if the error is a not-found error
func IsNotFound(err error) bool {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Category == CategoryNotFound
	}
	return false
}

// IsValidation returns true if the error is a validation error
func IsValidation(err error) bool {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Category == CategoryValidation
	}
	return false
}

// IsAuth returns true if the error is an authentication/authorization error
func IsAuth(err error) bool {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Category == CategoryAuth
	}
	return false
}

// IsConflict returns true if the error is a conflict error
func IsConflict(err error) bool {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Category == CategoryConflict
	}
	return false
}

// IsRetryable returns true if the error indicates the operation can be retried
func IsRetryable(err error) bool {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Retryable
	}
	return false
}

// IsStorage returns true if the error is a storage-related error
func IsStorage(err error) bool {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Category == CategoryStorage
	}
	return false
}

// GetHTTPStatus returns the HTTP status code for an error
func GetHTTPStatus(err error) int {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.HTTPStatus
	}
	return http.StatusInternalServerError
}

// GetCode returns the error code if available
func GetCode(err error) Code {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return CodeUnexpectedError
}

// HasCode checks if an error has a specific error code
func HasCode(err error, code Code) bool {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}
