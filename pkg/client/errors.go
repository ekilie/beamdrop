package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

var (
	// ErrInvalidBaseURL indicates that the provided BaseURL is empty, malformed, or missing scheme/host.
	ErrInvalidBaseURL = errors.New("invalid base URL")

	// ErrMissingCredentials indicates that AccessKeyID and SecretKey are required but not both provided.
	// If one is provided, both must be provided.
	ErrMissingCredentials = errors.New("missing beamdrop credentials")

	// ErrInvalidPath indicates that an internal request path is invalid or empty.
	ErrInvalidPath = errors.New("invalid path")
)

// APIError represents a structured error response from the Beamdrop API.
// It includes the HTTP status code, error code, category, human-readable message, and retry information.
type APIError struct {
	// StatusCode is the HTTP status code of the response (e.g., 404, 409, 429).
	StatusCode int `json:"-"`

	// Code is the machine-readable error code (e.g., "BUCKET_NOT_FOUND", "RATE_LIMIT_EXCEEDED").
	Code string `json:"code,omitempty"`

	// Category is the error category (e.g., "NOT_FOUND", "CONFLICT", "RATE_LIMIT").
	Category string `json:"category,omitempty"`

	// Message is the human-readable error message.
	Message string `json:"message,omitempty"`

	// Details is optional structured data providing additional context about the error.
	Details map[string]any `json:"details,omitempty"`

	// Retryable indicates whether the operation can be safely retried.
	// True for rate limiting and service unavailable errors.
	Retryable bool `json:"-"`

	// RetryAfter is the recommended number of seconds to wait before retrying.
	// Set from the Retry-After response header if present.
	RetryAfter int `json:"-"`

	// Body is the raw response body for debugging and inspection.
	Body []byte `json:"-"`
}

// Error implements the error interface, returning a formatted error message.
func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code != "" && e.Message != "" {
		return fmt.Sprintf("beamdrop API error (%d %s): %s", e.StatusCode, e.Code, e.Message)
	}
	if e.Message != "" {
		return fmt.Sprintf("beamdrop API error (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("beamdrop API error (%d)", e.StatusCode)
}

// decodeAPIError parses an HTTP error response into an APIError.
// Attempts to unmarshal the response body as JSON; falls back to plain text if that fails.
// Extracts retry information from response headers.
func decodeAPIError(response *http.Response) *APIError {
	apiErr := &APIError{
		StatusCode: response.StatusCode,
		Retryable:  strings.EqualFold(response.Header.Get("X-Retryable"), "true"),
	}

	if retryAfter := response.Header.Get("Retry-After"); retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			apiErr.RetryAfter = seconds
		}
	}

	body, err := io.ReadAll(response.Body)
	if err == nil {
		apiErr.Body = body
		if len(body) > 0 {
			if err := json.Unmarshal(body, apiErr); err != nil || apiErr.Message == "" {
				apiErr.Message = strings.TrimSpace(string(body))
			}
		}
	}

	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(response.StatusCode)
	}

	return apiErr
}
