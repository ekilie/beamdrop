package helpers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}
	SendJSON(w, data, http.StatusOK)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Fatal("expected Content-Type: application/json")
	}

	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if result["key"] != "value" {
		t.Fatalf("expected key=value, got %v", result)
	}
}

func TestSendJSON_CustomStatus(t *testing.T) {
	w := httptest.NewRecorder()
	SendJSON(w, nil, http.StatusNoContent)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestSendAPIError(t *testing.T) {
	w := httptest.NewRecorder()
	SendAPIError(w, "NOT_FOUND", "resource not found", http.StatusNotFound)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Fatal("expected Content-Type: application/json")
	}

	var body map[string]map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if body["error"]["code"] != "NOT_FOUND" {
		t.Fatalf("expected code NOT_FOUND, got %q", body["error"]["code"])
	}
	if body["error"]["message"] != "resource not found" {
		t.Fatalf("expected message 'resource not found', got %q", body["error"]["message"])
	}
}

func TestSendAPIError_EmptyMessage(t *testing.T) {
	w := httptest.NewRecorder()
	SendAPIError(w, "ERROR", "", http.StatusInternalServerError)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}
