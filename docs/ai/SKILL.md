---
name: beamdrop
description: Interact with a Beamdrop file storage server — upload, download, and manage files via the S3-compatible API. Use when the user wants to store files, create buckets, generate presigned/shareable URLs, manage API keys, or integrate Beamdrop into their project. Covers Go SDK, HTTP API, presigned URL strategies, and error handling.
---

# Beamdrop File Storage Skill

## When to Use This Skill

Use this skill when the user asks to:

- Upload, download, or manage files on a Beamdrop server
- Create, list, or delete storage buckets
- Generate presigned or shareable download URLs
- Manage Beamdrop API keys (create, list, delete, scope permissions)
- Store AI-generated artifacts (code, images, documents, build outputs) on Beamdrop
- Set up Beamdrop integration in their Go, Python, JavaScript, PHP, or any project
- Compare presigned URL types or decide on a file-sharing strategy
- Debug Beamdrop API errors (401, 404, 409, 429)
- Configure Beamdrop for production deployment

## Prerequisites

The user needs a running Beamdrop instance with API authentication enabled:

```bash
beamdrop -dir /path/to/share -api-auth
```

They need these environment variables or config values:

- `BEAMDROP_BASE_URL` — The server URL (e.g., `http://localhost:7777`)
- `BEAMDROP_ACCESS_KEY_ID` — API access key (format: `BDK_` + 16 hex chars)
- `BEAMDROP_SECRET_KEY` — API secret key (format: `sk_` + 40 hex chars)

If the user doesn't have API keys yet, they can create them:

```bash
curl -X POST http://localhost:7777/api/v1/keys
# Response includes accessKeyId and secretKey (secret shown ONCE — save immediately)
```

## Authentication Details

All S3 API requests at `/api/v1/` use HMAC-SHA256 signing:

```
StringToSign = METHOD + "\n" + PATH + "\n" + RFC3339_TIMESTAMP
Signature = Base64(HMAC-SHA256(StringToSign, SecretKey))
```

- PATH is the URL path only — no query string. Example: `/api/v1/buckets/my-bucket`
- TIMESTAMP is RFC3339 UTC (e.g., `2024-01-15T10:30:00Z`). Must be within ±15 minutes of server time.
- Base64 is standard encoding (not URL-safe)

Headers required on every request:

```
Authorization: Bearer BDK_xxxx:SIGNATURE
X-Beamdrop-Date: 2024-01-15T10:30:00Z
```

API key properties:

- `permissions`: `"read"`, `"write"`, or `"read,write"`
- `bucketScope`: Optional — restricts key to a single bucket
- `expiresAt`: Optional — key auto-expires after this time
- `disabled`: Can be set to temporarily disable without deleting

## Go Client SDK

When generating Go code, always use the official client SDK at `github.com/ekilie/beamdrop/pkg/client`:

```go
import "github.com/ekilie/beamdrop/pkg/client"

// Initialize client (validates BaseURL, creates HTTP client with 2-min timeout)
c, err := client.New(client.Config{
    BaseURL:     os.Getenv("BEAMDROP_BASE_URL"),   // Required, must have scheme+host
    AccessKeyID: os.Getenv("BEAMDROP_ACCESS_KEY_ID"),
    SecretKey:   os.Getenv("BEAMDROP_SECRET_KEY"),
    // HTTPClient: &http.Client{Timeout: 5*time.Minute},  // Optional custom client
    // UserAgent:  "my-app/1.0",                           // Optional, default "beamdrop-go-client/0.1"
})
if err != nil {
    log.Fatal(err)  // ErrInvalidBaseURL or ErrMissingCredentials
}
```

### Bucket Operations

```go
// List all buckets → *BucketList{Buckets []BucketInfo, Count int}
buckets, err := c.ListBuckets(ctx)

// Create bucket — errors with 409 BUCKET_EXISTS if already exists
created, err := c.CreateBucket(ctx, "my-bucket")   // → *BucketCreated{Bucket, Created, Location}

// Create bucket (IDEMPOTENT — preferred for automation)
// Returns 201 if new, 200 with Exists=true if already existed
created, err := c.CreateBucketIfNotExists(ctx, "my-bucket")

// Check existence via HEAD (returns false on 404, not an error)
exists, err := c.BucketExists(ctx, "my-bucket")   // → bool

// Delete (must be empty — 409 BUCKET_NOT_EMPTY if objects remain)
err = c.DeleteBucket(ctx, "my-bucket")
```

Bucket name rules: 3-63 chars, regex `^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`, no IP-like names.

### Object Operations

```go
// Upload from bytes → *ObjectCreated{Bucket, Key, ETag, Size, URL}
uploaded, err := c.PutObject(ctx, "bucket", "path/to/file.txt", []byte("content"))

// Upload from io.Reader (streaming — use for large files, avoids loading into memory)
f, _ := os.Open("large-file.bin")
defer f.Close()
uploaded, err = c.PutObjectReader(ctx, "bucket", "large-file.bin", f)

// Download → *ObjectBody{ObjectMetadata, Body []byte}
obj, err := c.GetObject(ctx, "bucket", "path/to/file.txt")
fmt.Println(string(obj.Body))                    // file content
fmt.Println(obj.ContentType)                     // "text/plain"
fmt.Println(obj.ETag)                            // MD5 hex hash (quotes trimmed)

// Metadata only (no body download) → *ObjectMetadata
meta, err := c.HeadObject(ctx, "bucket", "path/to/file.txt")
fmt.Printf("Size: %d, Type: %s\n", meta.ContentLength, meta.ContentType)

// Check existence (false on 404, not an error)
exists, err := c.ObjectExists(ctx, "bucket", "key")

// Delete
err = c.DeleteObject(ctx, "bucket", "path/to/file.txt")

// List objects with S3-style prefix/delimiter → *ObjectList
list, err := c.ListObjects(ctx, "bucket", client.ListObjectsOptions{
    Prefix:    "folder/",    // only keys starting with "folder/"
    Delimiter: "/",          // group into "directories" at "/"
    MaxKeys:   100,          // default 1000
})
// list.Contents → []ObjectInfo{Key, Size, LastModified, ETag, ContentType}
// list.CommonPrefixes → []CommonPrefix{Prefix} (virtual subdirectories)
// list.IsTruncated → bool (more results available)
```

Object key rules: max 1024 bytes, no `..`, no leading `/`. Forward slashes create virtual directory hierarchies.
Max upload size: 5GB. Writes are atomic (crash-safe). ETag = MD5 hex hash of content.

### Presigned URLs — Choosing the Right Type

Beamdrop has two presigned URL mechanisms. **Always choose deliberately** — they have different trade-offs:

#### Client-side HMAC presigned URLs

Generated locally using your secret key. No server API call needed. Self-contained URL.

```go
url, err := c.PresignObjectURL("GET", "bucket", "file.txt", time.Now().Add(24*time.Hour))
// Returns: "http://server/api/v1/buckets/bucket/file.txt?access_key=BDK_xxx&expires=...&token=..."
```

**Use when:**

- You need zero server overhead (URL computed locally)
- Generating many links in a batch
- Embedding in automated emails or notifications
- Temporary access with no tracking needed

**Limitations:**

- Cannot be revoked (valid until expiry)
- No download counting or limits
- Key rotation invalidates ALL outstanding client-side URLs
- URL is long (contains bucket/key path + query params)

#### Server-side presigned URLs (RECOMMENDED for most use cases)

Created via API call, stored in database with a short 32-char hex token. Clean `/dl/{token}` URLs.

```go
presigned, err := c.CreatePresignedURL(ctx, client.CreatePresignedURLRequest{
    Bucket:       "bucket",
    Key:          "file.txt",
    Method:       "GET",
    ExpiresIn:    int64Ptr(3600),    // 1 hour (seconds). Omit for no expiry.
    MaxDownloads: intPtr(10),         // Max 10 downloads. Omit for unlimited.
})
// presigned.URL = "https://server/dl/a1b2c3d4..."  (short, clean)
// presigned.Token = "a1b2c3d4..."
// presigned.DownloadCount = 0

// List all presigned URLs
urls, err := c.ListPresignedURLs(ctx)   // → *PresignedURLList{URLs, Count}

// Check download stats
details, err := c.GetPresignedURL(ctx, "token")
fmt.Printf("Downloaded %d/%d times\n", details.DownloadCount, *details.MaxDownloads)

// Revoke — /dl/token immediately returns 404
err = c.DeletePresignedURL(ctx, "token")
```

**Use when:**

- You need to revoke access after sharing
- You want download counting or limits
- You need an audit trail (createdBy, createdAt tracked)
- Link must survive API key rotation
- You want short, clean URLs for users
- Sharing sensitive files

**Decision matrix:**

| Scenario                          | Client-side | Server-side |
| --------------------------------- | :---------: | :---------: |
| Quick temporary link, no tracking |     ✅      |             |
| Need to revoke after sharing      |             |     ✅      |
| Limit number of downloads         |             |     ✅      |
| Track download count              |             |     ✅      |
| Batch-generate 1000 links         |     ✅      |             |
| Share in download portal          |             |     ✅      |
| Must survive key rotation         |             |     ✅      |
| Sensitive files with audit trail  |             |     ✅      |
| Embed in automated emails         |     ✅      |             |

## HTTP API Quick Reference

When generating code in other languages, use the HTTP API directly with HMAC signing:

### Buckets

- `GET /api/v1/buckets` → `{"buckets":[{name, createdAt}], "count":N}`
- `PUT /api/v1/buckets/{name}` → 201 `{bucket, created, location}` | 409 BUCKET_EXISTS
- `PUT /api/v1/buckets/{name}?createIfNotExists=true` → 201 (new) or 200 `{exists:true}` (existed)
- `HEAD /api/v1/buckets/{name}` → 200 or 404
- `DELETE /api/v1/buckets/{name}` → 204 | 409 BUCKET_NOT_EMPTY

### Objects

- `PUT /api/v1/buckets/{bucket}/{key}` (body = raw bytes) → 200 `{bucket, key, etag, size, url}`
- `GET /api/v1/buckets/{bucket}/{key}` → raw file (headers: Content-Type, Content-Length, ETag, Last-Modified). Supports Range header
- `HEAD /api/v1/buckets/{bucket}/{key}` → headers only, no body
- `DELETE /api/v1/buckets/{bucket}/{key}` → 204
- `GET /api/v1/buckets/{bucket}?list=true&prefix=X&delimiter=/&max-keys=N` → `{contents, commonPrefixes, isTruncated}`

### Presigned URLs

- `POST /api/v1/presign` (JSON: `{bucket, key, method, expires_in, max_downloads}`) → 201 `{token, url, ...}`
- `GET /api/v1/presign` → `{urls, count}`
- `GET /api/v1/presign/{token}` → presigned URL details with current downloadCount
- `DELETE /api/v1/presign/{token}` → 200 (immediate revocation)
- `GET /dl/{token}` → public download (no auth needed). 404 if expired/revoked/max-reached

### API Keys

- `POST /api/v1/keys` (JSON: `{name, permissions, bucket_scope}`) → 201 `{accessKeyId, secretKey, ...}` — secret shown ONCE
- `GET /api/v1/keys` → `{keys, count}` — no secrets
- `DELETE /api/v1/keys?accessKeyId=BDK_xxxx` → 204

## Error Handling

All errors return consistent JSON: `{"error":{"code":"CODE","category":"CATEGORY","message":"...","details":{}}}`

Handle these errors in generated code:

| HTTP | Code                | What to Do                                                                                                       |
| ---- | ------------------- | ---------------------------------------------------------------------------------------------------------------- |
| 400  | INVALID_BUCKET_NAME | Fix bucket name: 3-63 chars, lowercase a-z0-9 + hyphens/dots, start/end with letter/digit                        |
| 400  | INVALID_OBJECT_KEY  | Fix key: no `..`, no leading `/`, max 1024 bytes                                                                 |
| 401  | UNAUTHORIZED        | Check: (1) API key exists and not disabled, (2) timestamp within ±15 min, (3) signing path matches request path  |
| 403  | PERMISSION_DENIED   | API key lacks required permission or bucket scope doesn't match                                                  |
| 404  | BUCKET_NOT_FOUND    | Create bucket first with `CreateBucketIfNotExists`                                                               |
| 404  | OBJECT_NOT_FOUND    | Object doesn't exist — check key spelling, verify bucket                                                         |
| 409  | BUCKET_EXISTS       | Use `?createIfNotExists=true` to avoid this                                                                      |
| 409  | BUCKET_NOT_EMPTY    | Delete all objects before deleting bucket                                                                        |
| 413  | FILE_TOO_LARGE      | File exceeds 5GB limit — split or compress                                                                       |
| 423  | OBJECT_LOCKED       | Another operation holds the lock — **retry after brief delay** (lock timeout is 30s)                             |
| 429  | RATE_LIMIT_EXCEEDED | **Retry after `Retry-After` header seconds**. Response has `X-Retryable: true`. General: 100/min, upload: 10/min |
| 507  | STORAGE_FULL        | Server reached `-max-storage` limit — cannot upload                                                              |

Go client error handling pattern:

```go
result, err := c.GetObject(ctx, "bucket", "key")
if err != nil {
    var apiErr *client.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.Code {
        case "OBJECT_NOT_FOUND":
            // handle missing object
        case "RATE_LIMIT_EXCEEDED":
            time.Sleep(time.Duration(apiErr.RetryAfter) * time.Second)
            // retry
        }
    }
}
```

## Common Workflows

### Store AI-generated artifacts

```go
c.CreateBucketIfNotExists(ctx, "ai-artifacts")

// Use structured key paths: {purpose}/{session}/{filename}
key := fmt.Sprintf("generations/%s/%s", sessionID, "output.json")
c.PutObject(ctx, "ai-artifacts", key, resultBytes)

// Generate a shareable link (server-side, revocable, 24h expiry)
presigned, _ := c.CreatePresignedURL(ctx, client.CreatePresignedURLRequest{
    Bucket:    "ai-artifacts",
    Key:       key,
    Method:    "GET",
    ExpiresIn: int64Ptr(86400),
})
fmt.Println("Download:", presigned.URL)
```

### Upload and share with download limits

```go
c.CreateBucketIfNotExists(ctx, "shared")
c.PutObject(ctx, "shared", "report.pdf", pdfBytes)

// Server-side URL — max 5 downloads, expires in 7 days
presigned, _ := c.CreatePresignedURL(ctx, client.CreatePresignedURLRequest{
    Bucket:       "shared",
    Key:          "report.pdf",
    Method:       "GET",
    ExpiresIn:    int64Ptr(7 * 24 * 3600),
    MaxDownloads: intPtr(5),
})
```

### List and organize files

```go
// List top-level "directories" in a bucket
list, _ := c.ListObjects(ctx, "bucket", client.ListObjectsOptions{
    Delimiter: "/",
})
// list.CommonPrefixes → ["folder1/", "folder2/"]

// List files within a "directory"
list, _ = c.ListObjects(ctx, "bucket", client.ListObjectsOptions{
    Prefix:    "folder1/",
    Delimiter: "/",
})
// list.Contents → [{Key:"folder1/file.txt", ...}]
// list.CommonPrefixes → ["folder1/sub/"]
```

### Deduplicate with ETags

```go
// ETags are MD5 hashes — use for deduplication
meta, _ := c.HeadObject(ctx, "bucket", "file.txt")
if meta.ETag == computeMD5Hex(newContent) {
    fmt.Println("Content unchanged, skipping upload")
} else {
    c.PutObject(ctx, "bucket", "file.txt", newContent)
}
```

### Scoped API keys for limited access

```bash
# Create a read-only key scoped to one bucket
curl -X POST http://localhost:7777/api/v1/keys \
  -H "Content-Type: application/json" \
  -d '{"name":"readonly-reports","permissions":"read","bucket_scope":"reports"}'
```

### Set up webhooks for event notifications

```bash
# Create a webhook for all object events
curl -X POST http://localhost:7777/api/v1/webhooks \
  -H "Authorization: Bearer BDK_xxx:signature" \
  -H "X-Beamdrop-Date: 2024-01-15T10:30:00Z" \
  -H "Content-Type: application/json" \
  -d '{"name":"my-hook","url":"https://example.com/webhook","event_types":["beamdrop.object.*"]}'
# Save the "secret" from the response — it's shown only once
```

### Use the built-in MCP server

The MCP server is built into Beamdrop at `/mcp`:

- `GET /mcp` — Public discovery (no auth)
- `POST /mcp` — JSON-RPC 2.0 requests (requires API key auth)

```json
// Claude Desktop / VS Code MCP config
{
  "mcpServers": {
    "beamdrop": {
      "url": "https://your-server.com/mcp",
      "headers": {
        "Authorization": "Bearer BDK_your_key:signature",
        "X-Beamdrop-Date": "ISO-8601-timestamp"
      }
    }
  }
}
```

16 tools: list_buckets, create_bucket, delete_bucket, bucket_exists, list_objects, put_object, get_object, head_object, delete_object, create_presigned_url, list_presigned_urls, get_presigned_url, delete_presigned_url, list_api_keys, create_api_key, delete_api_key

## Webhooks Reference

Beamdrop supports real-time event notifications via HMAC-SHA256 signed webhooks.

**API:** `POST /api/v1/webhooks` (create), `GET /api/v1/webhooks` (list), `PATCH /api/v1/webhooks/{id}` (update), `DELETE /api/v1/webhooks/{id}` (delete), `POST /api/v1/webhooks/{id}/test` (test), `GET /api/v1/webhooks/{id}/deliveries` (history)

**Event types:** `beamdrop.object.created`, `.updated`, `.deleted`, `beamdrop.bucket.created`, `.deleted`, `beamdrop.share.created`, `.deleted`, `beamdrop.presign.created`, `.deleted`. Wildcards: `beamdrop.object.*`, `beamdrop.bucket.*`, etc.

**Signing:** `v1=` + hex(HMAC-SHA256(timestamp + "\n" + delivery_id + "\n" + body, secret)). Headers: `X-Beamdrop-Signature`, `X-Beamdrop-Webhook-Id`, `X-Beamdrop-Event`, `X-Beamdrop-Delivery-Id`, `X-Beamdrop-Timestamp`

**Delivery:** Max 8 retries, exponential backoff (2s-15min), 10s timeout, retryable codes: 408/425/429/5xx

## Validation Rules Summary

| Rule                      | Constraint                               |
| ------------------------- | ---------------------------------------- |
| Bucket name length        | 3-63 characters                          |
| Bucket name regex         | `^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`     |
| Bucket name chars         | Lowercase a-z, digits 0-9, hyphens, dots |
| Bucket name start/end     | Must be letter or digit                  |
| IP-like bucket names      | Rejected (e.g., `192.168.1.1`)           |
| Object key max length     | 1024 bytes                               |
| Object key no-go patterns | Empty, contains `..`, starts with `/`    |
| Max upload size           | 5GB (5,242,880,000 bytes)                |
| HMAC clock skew tolerance | ±15 minutes                              |
| Object lock timeout       | 30 seconds                               |
| Rate limit — general      | 100 req/min per IP                       |
| Rate limit — upload       | 10 req/min per IP                        |
| Rate limit — auth         | 5 req/min per IP                         |
