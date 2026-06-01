package handlers

import (
	"encoding/json"
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
		h.handleInfo(w, r)
	case http.MethodOptions:
		w.Header().Set("Allow", "GET, POST, OPTIONS")
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, POST, OPTIONS")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePost processes a JSON-RPC request.
func (h *MCPHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("Content-Type")
	if ct != "" && ct != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	// 1MB limit to prevent abuse
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

	data, err := json.Marshal(resp)
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
	data, _ := json.Marshal(info)
	w.Write(data)
}
