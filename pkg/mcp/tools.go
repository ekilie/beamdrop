package mcp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/ekilie/beamdrop/pkg/db"
)

// registerAllTools registers all MCP tools.
func (s *Server) registerAllTools() {
	s.registerBucketTools()
	s.registerObjectTools()
	s.registerPresignTools()
	s.registerKeyTools()
}

// --- Bucket Tools ---

func (s *Server) registerBucketTools() {
	s.register(ToolDefinition{
		Name:        "list_buckets",
		Description: "List all storage buckets on this Beamdrop server",
		InputSchema: InputSchema{Type: "object"},
	}, s.toolListBuckets)

	s.register(ToolDefinition{
		Name:        "create_bucket",
		Description: "Create a new storage bucket. Use idempotent=true to avoid errors if it already exists.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"name":       {Type: "string", Description: "Bucket name (3-63 chars, lowercase alphanumeric + hyphens/dots)"},
				"idempotent": {Type: "boolean", Description: "If true, don't error when bucket already exists", Default: true},
			},
			Required: []string{"name"},
		},
	}, s.toolCreateBucket)

	s.register(ToolDefinition{
		Name:        "delete_bucket",
		Description: "Delete an empty storage bucket. All objects must be deleted first.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"name": {Type: "string", Description: "Bucket name to delete"},
			},
			Required: []string{"name"},
		},
	}, s.toolDeleteBucket)

	s.register(ToolDefinition{
		Name:        "bucket_exists",
		Description: "Check whether a storage bucket exists",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"name": {Type: "string", Description: "Bucket name to check"},
			},
			Required: []string{"name"},
		},
	}, s.toolBucketExists)
}

func (s *Server) toolListBuckets(_ map[string]any) (*CallToolResult, error) {
	buckets, err := s.bucketManager.ListBuckets()
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]any{
		"buckets": buckets,
		"count":   len(buckets),
	}), nil
}

func (s *Server) toolCreateBucket(args map[string]any) (*CallToolResult, error) {
	name := strArg(args, "name", "")
	if name == "" {
		return errorResult("Missing required argument: name"), nil
	}
	idempotent := boolArg(args, "idempotent", true)

	if idempotent {
		created, err := s.bucketManager.CreateBucketIfNotExists(name)
		if err != nil {
			return storageError(err), nil
		}
		return jsonResult(map[string]any{
			"bucket":  name,
			"created": created,
			"exists":  !created,
		}), nil
	}

	if err := s.bucketManager.CreateBucket(name); err != nil {
		return storageError(err), nil
	}
	return jsonResult(map[string]any{
		"bucket":  name,
		"created": true,
	}), nil
}

func (s *Server) toolDeleteBucket(args map[string]any) (*CallToolResult, error) {
	name := strArg(args, "name", "")
	if name == "" {
		return errorResult("Missing required argument: name"), nil
	}
	if err := s.bucketManager.DeleteBucket(name); err != nil {
		return storageError(err), nil
	}
	return textResult(fmt.Sprintf("Bucket %q deleted successfully.", name)), nil
}

func (s *Server) toolBucketExists(args map[string]any) (*CallToolResult, error) {
	name := strArg(args, "name", "")
	if name == "" {
		return errorResult("Missing required argument: name"), nil
	}
	if s.bucketManager.BucketExists(name) {
		return textResult(fmt.Sprintf("Bucket %q exists.", name)), nil
	}
	return textResult(fmt.Sprintf("Bucket %q does not exist.", name)), nil
}

// --- Object Tools ---

func (s *Server) registerObjectTools() {
	s.register(ToolDefinition{
		Name:        "list_objects",
		Description: "List objects in a bucket with optional prefix and delimiter filtering. Use delimiter='/' for directory-like listing.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"bucket":    {Type: "string", Description: "Bucket name"},
				"prefix":    {Type: "string", Description: "Filter objects by key prefix (e.g., 'folder/')"},
				"delimiter": {Type: "string", Description: "Group objects by delimiter (use '/' for directory listing)"},
				"maxKeys":   {Type: "number", Description: "Maximum number of results (default 1000)"},
			},
			Required: []string{"bucket"},
		},
	}, s.toolListObjects)

	s.register(ToolDefinition{
		Name:        "put_object",
		Description: "Upload content to a bucket. For text, provide content directly. For binary, set isBase64=true and provide base64-encoded content.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"bucket":   {Type: "string", Description: "Bucket name"},
				"key":      {Type: "string", Description: "Object key/path (e.g., 'folder/file.txt')"},
				"content":  {Type: "string", Description: "File content (text or base64-encoded binary)"},
				"isBase64": {Type: "boolean", Description: "True if content is base64-encoded binary", Default: false},
			},
			Required: []string{"bucket", "key", "content"},
		},
	}, s.toolPutObject)

	s.register(ToolDefinition{
		Name:        "get_object",
		Description: "Download an object. Returns text directly, or base64-encoded content for binary files.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"bucket": {Type: "string", Description: "Bucket name"},
				"key":    {Type: "string", Description: "Object key/path"},
			},
			Required: []string{"bucket", "key"},
		},
	}, s.toolGetObject)

	s.register(ToolDefinition{
		Name:        "head_object",
		Description: "Get metadata about an object without downloading the body. Returns size, last modified, and ETag.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"bucket": {Type: "string", Description: "Bucket name"},
				"key":    {Type: "string", Description: "Object key/path"},
			},
			Required: []string{"bucket", "key"},
		},
	}, s.toolHeadObject)

	s.register(ToolDefinition{
		Name:        "delete_object",
		Description: "Delete an object from a bucket",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"bucket": {Type: "string", Description: "Bucket name"},
				"key":    {Type: "string", Description: "Object key/path to delete"},
			},
			Required: []string{"bucket", "key"},
		},
	}, s.toolDeleteObject)
}

func (s *Server) toolListObjects(args map[string]any) (*CallToolResult, error) {
	bucket := strArg(args, "bucket", "")
	if bucket == "" {
		return errorResult("Missing required argument: bucket"), nil
	}

	prefix := strArg(args, "prefix", "")
	delimiter := strArg(args, "delimiter", "")
	maxKeys := intArg(args, "maxKeys", 1000)

	result, err := s.objectManager.ListObjects(bucket, prefix, delimiter, maxKeys)
	if err != nil {
		return storageError(err), nil
	}

	return jsonResult(map[string]any{
		"bucket":          bucket,
		"prefix":          result.Prefix,
		"delimiter":       result.Delimiter,
		"max_keys":        result.MaxKeys,
		"is_truncated":    result.IsTruncated,
		"contents":        result.Contents,
		"common_prefixes": result.CommonPrefixes,
	}), nil
}

func (s *Server) toolPutObject(args map[string]any) (*CallToolResult, error) {
	bucket := strArg(args, "bucket", "")
	key := strArg(args, "key", "")
	content := strArg(args, "content", "")
	isBase64 := boolArg(args, "isBase64", false)

	if bucket == "" || key == "" {
		return errorResult("Missing required arguments: bucket, key"), nil
	}

	var reader io.Reader
	if isBase64 {
		decoded, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			return errorResult(fmt.Sprintf("Invalid base64 content: %v", err)), nil
		}
		reader = strings.NewReader(string(decoded))
	} else {
		reader = strings.NewReader(content)
	}

	info, err := s.objectManager.PutObject(bucket, key, reader)
	if err != nil {
		return storageError(err), nil
	}

	return jsonResult(map[string]any{
		"bucket":        bucket,
		"key":           info.Key,
		"size":          info.Size,
		"etag":          info.ETag,
		"last_modified": info.LastModified.UTC().Format(time.RFC3339),
	}), nil
}

func (s *Server) toolGetObject(args map[string]any) (*CallToolResult, error) {
	bucket := strArg(args, "bucket", "")
	key := strArg(args, "key", "")
	if bucket == "" || key == "" {
		return errorResult("Missing required arguments: bucket, key"), nil
	}

	file, info, unlock, err := s.objectManager.GetObject(bucket, key)
	if err != nil {
		return storageError(err), nil
	}
	defer file.Close()
	defer unlock()

	data, err := io.ReadAll(io.LimitReader(file, 10*1024*1024)) // 10MB limit for MCP
	if err != nil {
		return nil, fmt.Errorf("read object: %w", err)
	}

	// Detect if content is text
	isText := isTextContent(key, data)
	if isText {
		return textResult(string(data)), nil
	}

	return textResult(fmt.Sprintf("[Binary file: %d bytes]\nBase64: %s",
		info.Size, base64.StdEncoding.EncodeToString(data))), nil
}

func (s *Server) toolHeadObject(args map[string]any) (*CallToolResult, error) {
	bucket := strArg(args, "bucket", "")
	key := strArg(args, "key", "")
	if bucket == "" || key == "" {
		return errorResult("Missing required arguments: bucket, key"), nil
	}

	info, err := s.objectManager.HeadObject(bucket, key)
	if err != nil {
		return storageError(err), nil
	}

	return jsonResult(map[string]any{
		"key":           info.Key,
		"size":          info.Size,
		"last_modified": info.LastModified.UTC().Format(time.RFC3339),
		"etag":          info.ETag,
	}), nil
}

func (s *Server) toolDeleteObject(args map[string]any) (*CallToolResult, error) {
	bucket := strArg(args, "bucket", "")
	key := strArg(args, "key", "")
	if bucket == "" || key == "" {
		return errorResult("Missing required arguments: bucket, key"), nil
	}

	if err := s.objectManager.DeleteObject(bucket, key); err != nil {
		return storageError(err), nil
	}
	return textResult(fmt.Sprintf("Object %q deleted from bucket %q.", key, bucket)), nil
}

// --- Presigned URL Tools ---

func (s *Server) registerPresignTools() {
	s.register(ToolDefinition{
		Name:        "create_presigned_url",
		Description: "Create a presigned URL for downloading a file. The URL is shareable and requires no authentication.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"bucket":       {Type: "string", Description: "Bucket name"},
				"key":          {Type: "string", Description: "Object key/path"},
				"method":       {Type: "string", Description: "HTTP method (GET for download)", Default: "GET"},
				"expiresIn":    {Type: "number", Description: "Expiry in seconds (e.g., 3600 for 1 hour). Omit for no expiry."},
				"maxDownloads": {Type: "number", Description: "Maximum downloads allowed. Omit for unlimited."},
			},
			Required: []string{"bucket", "key"},
		},
	}, s.toolCreatePresignedURL)

	s.register(ToolDefinition{
		Name:        "list_presigned_urls",
		Description: "List all presigned URLs",
		InputSchema: InputSchema{Type: "object"},
	}, s.toolListPresignedURLs)

	s.register(ToolDefinition{
		Name:        "get_presigned_url",
		Description: "Get details about a specific presigned URL by token",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"token": {Type: "string", Description: "Presigned URL token"},
			},
			Required: []string{"token"},
		},
	}, s.toolGetPresignedURL)

	s.register(ToolDefinition{
		Name:        "delete_presigned_url",
		Description: "Revoke a presigned URL. The URL will immediately return 404.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"token": {Type: "string", Description: "Presigned URL token to revoke"},
			},
			Required: []string{"token"},
		},
	}, s.toolDeletePresignedURL)
}

func (s *Server) toolCreatePresignedURL(args map[string]any) (*CallToolResult, error) {
	bucket := strArg(args, "bucket", "")
	key := strArg(args, "key", "")
	if bucket == "" || key == "" {
		return errorResult("Missing required arguments: bucket, key"), nil
	}

	method := strArg(args, "method", "GET")
	if method != "GET" && method != "PUT" {
		return errorResult("Method must be GET or PUT"), nil
	}

	var expiresIn *time.Duration
	if v, ok := args["expiresIn"]; ok {
		if n, ok := v.(float64); ok && n > 0 {
			d := time.Duration(n) * time.Second
			expiresIn = &d
		}
	}

	var maxDownloads *int
	if v, ok := args["maxDownloads"]; ok {
		if n, ok := v.(float64); ok && n > 0 {
			md := int(n)
			maxDownloads = &md
		}
	}

	p, err := db.CreatePresignedURL(bucket, key, method, "mcp", expiresIn, maxDownloads)
	if err != nil {
		return nil, fmt.Errorf("create presigned URL: %w", err)
	}

	return jsonResult(map[string]any{
		"id":             p.ID,
		"token":          p.Token,
		"url":            fmt.Sprintf("/dl/%s", p.Token),
		"bucket":         p.Bucket,
		"key":            p.Key,
		"method":         p.Method,
		"expires_at":     p.ExpiresAt,
		"max_downloads":  p.MaxDownloads,
		"download_count": p.DownloadCount,
		"created_at":     p.CreatedAt.UTC().Format(time.RFC3339),
	}), nil
}

func (s *Server) toolListPresignedURLs(_ map[string]any) (*CallToolResult, error) {
	urls, err := db.ListPresignedURLs()
	if err != nil {
		return nil, err
	}

	items := make([]map[string]any, len(urls))
	for i, u := range urls {
		items[i] = map[string]any{
			"id":             u.ID,
			"token":          u.Token,
			"url":            fmt.Sprintf("/dl/%s", u.Token),
			"bucket":         u.Bucket,
			"key":            u.Key,
			"method":         u.Method,
			"expires_at":     u.ExpiresAt,
			"max_downloads":  u.MaxDownloads,
			"download_count": u.DownloadCount,
			"created_at":     u.CreatedAt.UTC().Format(time.RFC3339),
		}
	}
	return jsonResult(map[string]any{"urls": items, "count": len(items)}), nil
}

func (s *Server) toolGetPresignedURL(args map[string]any) (*CallToolResult, error) {
	token := strArg(args, "token", "")
	if token == "" {
		return errorResult("Missing required argument: token"), nil
	}

	p, err := db.GetPresignedURLByToken(token)
	if err != nil {
		return errorResult(fmt.Sprintf("Error: %v", err)), nil
	}
	if p == nil {
		return errorResult("Presigned URL not found"), nil
	}

	return jsonResult(map[string]any{
		"id":             p.ID,
		"token":          p.Token,
		"url":            fmt.Sprintf("/dl/%s", p.Token),
		"bucket":         p.Bucket,
		"key":            p.Key,
		"method":         p.Method,
		"expires_at":     p.ExpiresAt,
		"max_downloads":  p.MaxDownloads,
		"download_count": p.DownloadCount,
		"created_at":     p.CreatedAt.UTC().Format(time.RFC3339),
	}), nil
}

func (s *Server) toolDeletePresignedURL(args map[string]any) (*CallToolResult, error) {
	token := strArg(args, "token", "")
	if token == "" {
		return errorResult("Missing required argument: token"), nil
	}

	if err := db.DeletePresignedURL(token); err != nil {
		return errorResult(fmt.Sprintf("Error: %v", err)), nil
	}
	return textResult(fmt.Sprintf("Presigned URL %q revoked.", token)), nil
}

// --- API Key Tools ---

func (s *Server) registerKeyTools() {
	s.register(ToolDefinition{
		Name:        "list_api_keys",
		Description: "List all API keys (secrets are not included)",
		InputSchema: InputSchema{Type: "object"},
	}, s.toolListAPIKeys)

	s.register(ToolDefinition{
		Name:        "create_api_key",
		Description: "Create a new API key. The secret is shown only once — save it immediately.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"name":        {Type: "string", Description: "Human-readable name for the key"},
				"bucketScope": {Type: "string", Description: "Restrict key to a specific bucket (omit for all)"},
				"permissions": {Type: "array", Description: "Allowed actions: read, write, or both", Items: &Property{Type: "string"}},
			},
		},
	}, s.toolCreateAPIKey)

	s.register(ToolDefinition{
		Name:        "delete_api_key",
		Description: "Delete an API key permanently. Requests using this key will immediately fail.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"accessKeyId": {Type: "string", Description: "Access key ID to delete (BDK_...)"},
			},
			Required: []string{"accessKeyId"},
		},
	}, s.toolDeleteAPIKey)
}

func (s *Server) toolListAPIKeys(_ map[string]any) (*CallToolResult, error) {
	keys, err := db.ListAPIKeys()
	if err != nil {
		return nil, err
	}

	items := make([]map[string]any, len(keys))
	for i, k := range keys {
		items[i] = map[string]any{
			"id":            k.ID,
			"name":          k.Name,
			"access_key_id": k.AccessKeyID,
			"permissions":   k.Permissions,
			"bucket_scope":  k.BucketScope,
			"created_at":    k.CreatedAt.UTC().Format(time.RFC3339),
			"disabled":      k.Disabled,
		}
	}
	return jsonResult(map[string]any{"keys": items, "count": len(items)}), nil
}

func (s *Server) toolCreateAPIKey(args map[string]any) (*CallToolResult, error) {
	name := strArg(args, "name", "mcp-generated")
	bucketScope := strArg(args, "bucketScope", "")

	perms := strSliceArg(args, "permissions")
	permStr := ""
	if len(perms) > 0 {
		data, _ := json.Marshal(perms)
		permStr = string(data)
	}

	key, secret, err := db.CreateAPIKey(name, permStr, bucketScope, nil)
	if err != nil {
		return nil, fmt.Errorf("create API key: %w", err)
	}

	slog.Info("MCP: API key created", "name", name, "access_key_id", key.AccessKeyID)

	return jsonResult(map[string]any{
		"id":            key.ID,
		"name":          key.Name,
		"access_key_id": key.AccessKeyID,
		"secret_key":    secret,
		"bucket_scope":  key.BucketScope,
		"permissions":   key.Permissions,
		"created_at":    key.CreatedAt.UTC().Format(time.RFC3339),
		"warning":       "Save the secret_key now — it will not be shown again.",
	}), nil
}

func (s *Server) toolDeleteAPIKey(args map[string]any) (*CallToolResult, error) {
	accessKeyID := strArg(args, "accessKeyId", "")
	if accessKeyID == "" {
		return errorResult("Missing required argument: accessKeyId"), nil
	}

	if err := db.DeleteAPIKey(accessKeyID); err != nil {
		return errorResult(fmt.Sprintf("Error: %v", err)), nil
	}

	slog.Info("MCP: API key deleted", "access_key_id", accessKeyID)
	return textResult(fmt.Sprintf("API key %q deleted.", accessKeyID)), nil
}

// --- Helpers ---

// isTextContent checks if file content is likely text based on extension and content.
func isTextContent(key string, data []byte) bool {
	textExts := map[string]bool{
		".txt": true, ".md": true, ".json": true, ".xml": true, ".yaml": true,
		".yml": true, ".toml": true, ".csv": true, ".html": true, ".css": true,
		".js": true, ".ts": true, ".tsx": true, ".jsx": true, ".go": true,
		".py": true, ".rs": true, ".rb": true, ".java": true, ".c": true,
		".cpp": true, ".h": true, ".sh": true, ".bash": true, ".zsh": true,
		".fish": true, ".sql": true, ".env": true, ".conf": true, ".cfg": true,
		".ini": true, ".log": true, ".svg": true,
	}

	lower := strings.ToLower(key)
	for ext := range textExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}

	// Check content: if no null bytes in first 8KB, it's probably text
	checkLen := len(data)
	if checkLen > 8192 {
		checkLen = 8192
	}
	for i := 0; i < checkLen; i++ {
		if data[i] == 0 {
			return false
		}
	}
	return true
}
