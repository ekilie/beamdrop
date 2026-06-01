package mcp

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ekilie/beamdrop/config"
	"github.com/ekilie/beamdrop/pkg/storage"
)

// ToolHandler is a function that handles a tool call.
type ToolHandler func(args map[string]any) (*CallToolResult, error)

// Server is an MCP server that handles JSON-RPC requests.
type Server struct {
	tools         []ToolDefinition
	handlers      map[string]ToolHandler
	objectManager *storage.ObjectManager
	bucketManager *storage.BucketManager
	sharedDir     string
}

// NewServer creates a new MCP server with all tools registered.
func NewServer(sharedDir string) *Server {
	s := &Server{
		handlers:      make(map[string]ToolHandler),
		objectManager: storage.NewObjectManager(sharedDir),
		bucketManager: storage.NewBucketManager(sharedDir),
		sharedDir:     sharedDir,
	}
	s.registerAllTools()
	return s
}

// HandleRequest processes a single JSON-RPC request and returns a response.
func (s *Server) HandleRequest(raw []byte) *JSONRPCResponse {
	var req JSONRPCRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    ErrCodeParse,
				Message: "Parse error",
			},
		}
	}

	if req.JSONRPC != "2.0" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    ErrCodeInvalidRequest,
				Message: "Invalid JSON-RPC version",
			},
		}
	}

	// Notifications (no ID) don't get responses
	if req.ID == nil || string(req.ID) == "null" {
		s.handleNotification(req.Method)
		return nil
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "ping":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  PingResult{},
		}
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    ErrCodeMethodNotFound,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

func (s *Server) handleNotification(method string) {
	switch method {
	case "notifications/initialized":
		slog.Debug("MCP client initialized")
	default:
		slog.Debug("Unknown MCP notification", "method", method)
	}
}

func (s *Server) handleInitialize(req JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: ServerCaps{
				Tools: &ToolsCap{},
			},
			ServerInfo: ServerInfo{
				Name:    "beamdrop",
				Version: config.VERSION,
			},
			Instructions: "Beamdrop MCP server. Provides tools for managing buckets, objects, presigned URLs, and API keys on this Beamdrop instance.",
		},
	}
}

func (s *Server) handleToolsList(req JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  ListToolsResult{Tools: s.tools},
	}
}

func (s *Server) handleToolsCall(req JSONRPCRequest) *JSONRPCResponse {
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    ErrCodeInvalidParams,
				Message: "Invalid tool call params",
			},
		}
	}

	handler, ok := s.handlers[params.Name]
	if !ok {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    ErrCodeMethodNotFound,
				Message: fmt.Sprintf("Unknown tool: %s", params.Name),
			},
		}
	}

	result, err := handler(params.Arguments)
	if err != nil {
		slog.Error("MCP tool call failed", "tool", params.Name, "error", err)
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Internal error: %v", err)}},
				IsError: true,
			},
		}
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func (s *Server) register(def ToolDefinition, handler ToolHandler) {
	s.tools = append(s.tools, def)
	s.handlers[def.Name] = handler
}

// Helper to extract string arg with default
func strArg(args map[string]any, key, fallback string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return fallback
}

// Helper to extract int arg with default
func intArg(args map[string]any, key string, fallback int) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return fallback
}

// Helper to extract bool arg with default
func boolArg(args map[string]any, key string, fallback bool) bool {
	if v, ok := args[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return fallback
}

// Helper to extract string array arg
func strSliceArg(args map[string]any, key string) []string {
	if v, ok := args[key]; ok {
		if arr, ok := v.([]any); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

// textResult creates a successful text result
func textResult(text string) *CallToolResult {
	return &CallToolResult{
		Content: []ToolContent{{Type: "text", Text: text}},
	}
}

// jsonResult creates a successful JSON result
func jsonResult(v any) *CallToolResult {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to serialize result: %v", err))
	}
	return textResult(string(data))
}

// errorResult creates an error result
func errorResult(msg string) *CallToolResult {
	return &CallToolResult{
		Content: []ToolContent{{Type: "text", Text: msg}},
		IsError: true,
	}
}

// storageError maps storage errors to readable messages
func storageError(err error) *CallToolResult {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "invalid bucket name"):
		return errorResult("Invalid bucket name: must be 3-63 chars, lowercase alphanumeric + hyphens/dots")
	case strings.Contains(msg, "bucket not found"):
		return errorResult("Bucket not found")
	case strings.Contains(msg, "bucket is not empty"):
		return errorResult("Bucket is not empty — delete all objects first")
	case strings.Contains(msg, "bucket already exists"):
		return errorResult("Bucket already exists")
	case strings.Contains(msg, "object not found"):
		return errorResult("Object not found")
	case strings.Contains(msg, "invalid object key"):
		return errorResult("Invalid object key")
	default:
		return errorResult(fmt.Sprintf("Error: %v", err))
	}
}
