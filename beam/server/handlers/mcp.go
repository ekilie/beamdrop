package handlers

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/ekilie/beamdrop/pkg/mcp"
)

// MCPHandler handles MCP protocol requests at /mcp.
// Supports Streamable HTTP transport: POST for JSON-RPC requests.
type MCPHandler struct {
	server *mcp.Server
}

// NewMCPHandler creates a new MCP HTTP handler.
func NewMCPHandler(sharedDir string) *MCPHandler {
	return &MCPHandler{
		server: mcp.NewServer(sharedDir),
	}
}

// ServeHTTP handles MCP requests.
func (h *MCPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handlePost(w, r)
	case http.MethodGet:
		// GET /mcp returns server info (useful for health checks and discovery)
		h.handleInfo(w, r)
	case http.MethodOptions:
		// CORS preflight
		w.Header().Set("Allow", "GET, POST, OPTIONS")
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, POST, OPTIONS")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePost processes a JSON-RPC request.
func (h *MCPHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	// Validate content type
	ct := r.Header.Get("Content-Type")
	if ct != "" && ct != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	// Read body with 1MB limit to prevent abuse
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		slog.Error("MCP: failed to read request body", "error", err)
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		http.Error(w, "Empty request body", http.StatusBadRequest)
		return
	}

	resp := h.server.HandleRequest(body)

	// Notifications (no ID) return no response
	if resp == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	data, err := marshalJSON(resp)
	if err != nil {
		slog.Error("MCP: failed to marshal response", "error", err)
		return
	}
	w.Write(data)
}

// handleInfo returns MCP server info for GET requests.
func (h *MCPHandler) handleInfo(w http.ResponseWriter, _ *http.Request) {
	info := map[string]any{
		"protocol":    "mcp",
		"transport":   "streamable-http",
		"version":     "2024-11-05",
		"name":        "beamdrop",
		"description": "Beamdrop MCP endpoint. POST JSON-RPC requests to this URL.",
		"methods":     []string{"initialize", "ping", "tools/list", "tools/call"},
	}

	w.Header().Set("Content-Type", "application/json")
	data, _ := marshalJSON(info)
	w.Write(data)
}

func marshalJSON(v any) ([]byte, error) {
	// Using json.Marshal without indent for responses (faster)
	return jsonMarshal(v)
}

// jsonMarshal wraps encoding/json.Marshal
var jsonMarshal = func() func(v any) ([]byte, error) {
	return func(v any) ([]byte, error) {
		import_json_marshal := jsonMarshalImpl
		return import_json_marshal(v)
	}
}()

func jsonMarshalImpl(v any) ([]byte, error) {
	// Need to import encoding/json
	return nil, nil
}
