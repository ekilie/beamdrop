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
	ErrInvalidBaseURL     = errors.New("invalid base URL")
	ErrMissingCredentials = errors.New("missing beamdrop credentials")
	ErrInvalidPath        = errors.New("invalid path")
)

type APIError struct {
	StatusCode int            `json:"-"`
	Code       string         `json:"code,omitempty"`
	Category   string         `json:"category,omitempty"`
	Message    string         `json:"message,omitempty"`
	Details    map[string]any `json:"details,omitempty"`
	Retryable  bool           `json:"-"`
	RetryAfter int            `json:"-"`
	Body       []byte         `json:"-"`
}

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
