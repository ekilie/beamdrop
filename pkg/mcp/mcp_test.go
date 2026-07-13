package mcp

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestJSONRPCVersionCheck(t *testing.T) {
	s := NewServer(t.TempDir())

	raw := []byte(`{"jsonrpc":"1.0","method":"ping","id":1}`)
	resp := s.HandleRequest(raw)

	if resp.Error == nil {
		t.Fatal("expected error for wrong JSON-RPC version")
	}
	if resp.Error.Code != ErrCodeInvalidRequest {
		t.Fatalf("expected code %d, got %d", ErrCodeInvalidRequest, resp.Error.Code)
	}
}

func TestHandleParseError(t *testing.T) {
	s := NewServer(t.TempDir())

	resp := s.HandleRequest([]byte(`not json`))
	if resp.Error == nil {
		t.Fatal("expected parse error")
	}
	if resp.Error.Code != ErrCodeParse {
		t.Fatalf("expected code %d, got %d", ErrCodeParse, resp.Error.Code)
	}
}

func TestHandlePing(t *testing.T) {
	s := NewServer(t.TempDir())

	raw := []byte(`{"jsonrpc":"2.0","method":"ping","id":1}`)
	resp := s.HandleRequest(raw)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if resp.Result == nil {
		t.Fatal("expected result")
	}
}

func TestHandleInitialize(t *testing.T) {
	s := NewServer(t.TempDir())

	raw := []byte(`{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}`)
	resp := s.HandleRequest(raw)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	result, ok := resp.Result.(InitializeResult)
	if !ok {
		t.Fatalf("expected InitializeResult, got %T", resp.Result)
	}
	if result.ProtocolVersion != "2024-11-05" {
		t.Fatalf("expected protocol version 2024-11-05, got %q", result.ProtocolVersion)
	}
	if result.ServerInfo.Name != "beamdrop" {
		t.Fatalf("expected server name 'beamdrop', got %q", result.ServerInfo.Name)
	}
}

func TestHandleToolsList(t *testing.T) {
	s := NewServer(t.TempDir())

	raw := []byte(`{"jsonrpc":"2.0","method":"tools/list","id":1}`)
	resp := s.HandleRequest(raw)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	result, ok := resp.Result.(ListToolsResult)
	if !ok {
		t.Fatalf("expected ListToolsResult, got %T", resp.Result)
	}
	if len(result.Tools) == 0 {
		t.Fatal("expected at least one tool")
	}
}

func TestHandleUnknownMethod(t *testing.T) {
	s := NewServer(t.TempDir())

	raw := []byte(`{"jsonrpc":"2.0","method":"unknown","id":1}`)
	resp := s.HandleRequest(raw)

	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != ErrCodeMethodNotFound {
		t.Fatalf("expected code %d, got %d", ErrCodeMethodNotFound, resp.Error.Code)
	}
}

func TestHandleNotification(t *testing.T) {
	s := NewServer(t.TempDir())

	// Notifications have no "id" field and should return nil
	raw := []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	resp := s.HandleRequest(raw)
	if resp != nil {
		t.Fatal("notifications should return nil response")
	}
}

func TestHandleToolsCall_UnknownTool(t *testing.T) {
	s := NewServer(t.TempDir())

	raw := []byte(`{"jsonrpc":"2.0","method":"tools/call","id":1,"params":{"name":"unknown_tool"}}`)
	resp := s.HandleRequest(raw)

	if resp.Error == nil {
		t.Fatal("expected error for unknown tool")
	}
	if resp.Error.Code != ErrCodeMethodNotFound {
		t.Fatalf("expected code %d, got %d", ErrCodeMethodNotFound, resp.Error.Code)
	}
}

func TestHandleToolsCall_InvalidParams(t *testing.T) {
	s := NewServer(t.TempDir())

	raw := []byte(`{"jsonrpc":"2.0","method":"tools/call","id":1,"params":"not-object"}`)
	resp := s.HandleRequest(raw)

	if resp.Error == nil {
		t.Fatal("expected error for invalid params")
	}
}

func TestHandleToolsCall_BucketExists(t *testing.T) {
	s := NewServer(t.TempDir())

	raw := []byte(`{"jsonrpc":"2.0","method":"tools/call","id":1,"params":{"name":"bucket_exists","arguments":{"name":"nonexistent"}}}`)
	resp := s.HandleRequest(raw)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestStrArg(t *testing.T) {
	args := map[string]any{"name": "test"}
	if v := strArg(args, "name", "default"); v != "test" {
		t.Fatalf("expected 'test', got %q", v)
	}
	if v := strArg(args, "missing", "default"); v != "default" {
		t.Fatalf("expected 'default', got %q", v)
	}
	if v := strArg(args, "name", ""); v != "test" {
		t.Fatalf("expected 'test' with empty default, got %q", v)
	}
}

func TestIntArg(t *testing.T) {
	args := map[string]any{"count": float64(42)}
	if v := intArg(args, "count", 0); v != 42 {
		t.Fatalf("expected 42, got %d", v)
	}
	if v := intArg(args, "missing", 10); v != 10 {
		t.Fatalf("expected 10, got %d", v)
	}
}

func TestBoolArg(t *testing.T) {
	args := map[string]any{"flag": true}
	if !boolArg(args, "flag", false) {
		t.Fatal("expected true")
	}
	if boolArg(args, "missing", false) {
		t.Fatal("expected false")
	}
	if !boolArg(args, "missing", true) {
		t.Fatal("expected true default")
	}
}

func TestStrSliceArg(t *testing.T) {
	args := map[string]any{"items": []any{"a", "b", "c"}}
	result := strSliceArg(args, "items")
	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}

	result = strSliceArg(args, "missing")
	if result != nil {
		t.Fatal("expected nil for missing key")
	}
}

func TestTextResult(t *testing.T) {
	r := textResult("hello")
	if len(r.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(r.Content))
	}
	if r.Content[0].Text != "hello" {
		t.Fatalf("expected 'hello', got %q", r.Content[0].Text)
	}
	if r.IsError {
		t.Fatal("text result should not be error")
	}
}

func TestErrorResult(t *testing.T) {
	r := errorResult("something broke")
	if !r.IsError {
		t.Fatal("error result should be error")
	}
	if r.Content[0].Text != "something broke" {
		t.Fatalf("expected 'something broke', got %q", r.Content[0].Text)
	}
}

func TestJSONResult(t *testing.T) {
	r := jsonResult(map[string]string{"key": "value"})
	if r.IsError {
		t.Fatal("json result should not be error")
	}
	if r.Content[0].Text == "" {
		t.Fatal("expected non-empty text")
	}
}

func TestJSONResult_Error(t *testing.T) {
	r := jsonResult(func() {}) // function is not JSON-serializable
	if !r.IsError {
		t.Fatal("expected error for non-serializable value")
	}
}

func TestStorageError(t *testing.T) {
	tests := []struct {
		msg    string
		prefix string
	}{
		{"invalid bucket name", "Invalid bucket name"},
		{"bucket not found", "Bucket not found"},
		{"bucket is not empty", "Bucket is not empty"},
		{"bucket already exists", "Bucket already exists"},
		{"object not found", "Object not found"},
		{"invalid object key", "Invalid object key"},
		{"generic error", "Error:"},
	}

	for _, tc := range tests {
		r := storageError(fmt.Errorf("%s", tc.msg))
		if !r.IsError {
			t.Errorf("expected error for %q", tc.msg)
		}
	}
}

func TestIsTextContent(t *testing.T) {
	tests := []struct {
		key    string
		data   []byte
		isText bool
	}{
		{"file.txt", []byte("hello"), true},
		{"file.go", []byte("package main"), true},
		{"file.md", []byte("# Title"), true},
		{"file.json", []byte(`{"a":1}`), true},
		{"file.bin", []byte{0x00, 0x01, 0x02}, false},
		{"file.bin", []byte{0x48, 0x00, 0x65}, false},
		{"UNKNOWN", []byte("plain text"), true},
		{"binary.dat", []byte{0x00}, false},
	}

	for _, tc := range tests {
		got := isTextContent(tc.key, tc.data)
		if got != tc.isText {
			t.Errorf("isTextContent(%q, data) = %v, want %v", tc.key, got, tc.isText)
		}
	}
}

func TestRegister(t *testing.T) {
	s := NewServer(t.TempDir())

	def := ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: InputSchema{Type: "object"},
	}
	s.register(def, func(args map[string]any) (*CallToolResult, error) {
		return textResult("done"), nil
	})

	if _, ok := s.handlers["test_tool"]; !ok {
		t.Fatal("handler should be registered")
	}
}

func TestToolCreateBucket_InvalidName(t *testing.T) {
	s := NewServer(t.TempDir())

	args := map[string]any{"name": ""}
	result, err := s.toolCreateBucket(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for empty name")
	}
}

func TestToolDeleteBucket_InvalidName(t *testing.T) {
	s := NewServer(t.TempDir())

	result, err := s.toolDeleteBucket(map[string]any{"name": ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for empty name")
	}
}

func TestToolBucketExists_EmptyName(t *testing.T) {
	s := NewServer(t.TempDir())

	result, err := s.toolBucketExists(map[string]any{"name": ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for empty name")
	}
}

func TestToolListObjects_NoBucket(t *testing.T) {
	s := NewServer(t.TempDir())

	result, err := s.toolListObjects(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing bucket")
	}
}

func TestToolPutObject_MissingArgs(t *testing.T) {
	s := NewServer(t.TempDir())

	result, err := s.toolPutObject(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing args")
	}
}

func TestToolGetObject_MissingArgs(t *testing.T) {
	s := NewServer(t.TempDir())

	result, err := s.toolGetObject(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing args")
	}
}

func TestToolHeadObject_MissingArgs(t *testing.T) {
	s := NewServer(t.TempDir())

	result, err := s.toolHeadObject(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing args")
	}
}

func TestToolDeleteObject_MissingArgs(t *testing.T) {
	s := NewServer(t.TempDir())

	result, err := s.toolDeleteObject(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing args")
	}
}

func TestToolCreatePresignedURL_MissingArgs(t *testing.T) {
	s := NewServer(t.TempDir())

	result, err := s.toolCreatePresignedURL(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing args")
	}
}

func TestToolCreatePresignedURL_InvalidMethod(t *testing.T) {
	s := NewServer(t.TempDir())

	result, err := s.toolCreatePresignedURL(map[string]any{
		"bucket": "b", "key": "k", "method": "DELETE",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for invalid method")
	}
}

func TestToolGetPresignedURL_MissingToken(t *testing.T) {
	s := NewServer(t.TempDir())

	result, err := s.toolGetPresignedURL(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing token")
	}
}

func TestToolDeletePresignedURL_MissingToken(t *testing.T) {
	s := NewServer(t.TempDir())

	result, err := s.toolDeletePresignedURL(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing token")
	}
}

func TestToolDeleteAPIKey_MissingAccessKey(t *testing.T) {
	s := NewServer(t.TempDir())

	result, err := s.toolDeleteAPIKey(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing accessKeyId")
	}
}

func TestProtocolTypes_JSON(t *testing.T) {
	// Test JSON serialization/deserialization of protocol types
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "test",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded JSONRPCRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.JSONRPC != "2.0" {
		t.Fatalf("expected '2.0', got %q", decoded.JSONRPC)
	}
}
