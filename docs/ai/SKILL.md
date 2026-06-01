---
name: beamdrop
description: Interact with a Beamdrop file storage server — upload, download, and manage files via the S3-compatible API. Use when the user wants to store files, create buckets, generate presigned URLs, or manage API keys on a Beamdrop instance.
---

# Beamdrop File Storage Skill

Use this skill when the user asks to:

- Upload or download files to/from a Beamdrop server
- Create, list, or delete storage buckets
- Generate presigned/shareable download URLs
- Manage Beamdrop API keys
- Store AI-generated artifacts (code, images, documents) on Beamdrop
- Set up Beamdrop integration in their project

## Prerequisites

The user needs a running Beamdrop instance with API authentication enabled (`-api-auth` flag). They need these environment variables or config values:

- `BEAMDROP_BASE_URL` — The server URL (e.g., `http://localhost:7777`)
- `BEAMDROP_ACCESS_KEY_ID` — API access key (format: `BDK_*`)
- `BEAMDROP_SECRET_KEY` — API secret key (format: `sk_*`)

If the user doesn't have API keys yet, they can create them:

```bash
curl -X POST http://localhost:7777/api/v1/keys
```

## Authentication

All S3 API requests use HMAC-SHA256 signing:

```
StringToSign = METHOD + "\n" + PATH + "\n" + TIMESTAMP
Signature = Base64(HMAC-SHA256(StringToSign, SecretKey))
```

Headers required:

```
Authorization: Bearer ACCESS_KEY_ID:SIGNATURE
X-Beamdrop-Date: 2024-01-15T10:30:00Z
```

## Go Client Usage

When generating Go code, use the official client SDK:

```go
import "github.com/ekilie/beamdrop/pkg/client"

// Initialize client
c, err := client.New(client.Config{
    BaseURL:     os.Getenv("BEAMDROP_BASE_URL"),
    AccessKeyID: os.Getenv("BEAMDROP_ACCESS_KEY_ID"),
    SecretKey:   os.Getenv("BEAMDROP_SECRET_KEY"),
})
if err != nil {
    log.Fatal(err)
}

// Create bucket (idempotent)
_, err = c.CreateBucketIfNotExists(ctx, "my-bucket")

// Upload file
uploaded, err := c.PutObject(ctx, "my-bucket", "path/to/file.txt", []byte("content"))

// Upload from reader (streaming, for large files)
uploaded, err = c.PutObjectReader(ctx, "my-bucket", "large-file.bin", file)

// Download file
obj, err := c.GetObject(ctx, "my-bucket", "path/to/file.txt")
fmt.Println(string(obj.Body))

// List objects with prefix
list, err := c.ListObjects(ctx, "my-bucket", client.ListObjectsOptions{
    Prefix:    "folder/",
    Delimiter: "/",
})

// Generate presigned URL (client-side, self-contained)
url, err := c.PresignObjectURL("GET", "my-bucket", "file.txt", time.Now().Add(24*time.Hour))

// Generate presigned URL (server-side, revocable, with download tracking)
presigned, err := c.CreatePresignedURL(ctx, client.CreatePresignedURLRequest{
    Bucket:       "my-bucket",
    Key:          "file.txt",
    Method:       "GET",
    ExpiresIn:    intPtr(3600),    // 1 hour
    MaxDownloads: intPtr(10),       // limit to 10 downloads
})
// presigned.URL is something like "https://server/dl/abc123"
```

## HTTP API Quick Reference

When generating code in other languages, use the HTTP API directly:

### Buckets

- `GET /api/v1/buckets` — List buckets
- `PUT /api/v1/buckets/{name}` — Create bucket
- `PUT /api/v1/buckets/{name}?createIfNotExists=true` — Create or reuse bucket
- `HEAD /api/v1/buckets/{name}` — Check existence
- `DELETE /api/v1/buckets/{name}` — Delete (must be empty)

### Objects

- `PUT /api/v1/buckets/{bucket}/{key}` — Upload (body = file content)
- `GET /api/v1/buckets/{bucket}/{key}` — Download
- `HEAD /api/v1/buckets/{bucket}/{key}` — Metadata only
- `DELETE /api/v1/buckets/{bucket}/{key}` — Delete
- `GET /api/v1/buckets/{bucket}?list=true&prefix=...&delimiter=...` — List

### Presigned URLs

- `POST /api/v1/presign` — Create (JSON: bucket, key, method, expires_in, max_downloads)
- `GET /api/v1/presign` — List all
- `DELETE /api/v1/presign/{token}` — Revoke

## Error Handling

Handle these common errors:

- **429 Rate Limited**: Retry after `Retry-After` header seconds. Response includes `"retryable": true`.
- **404 Not Found**: Bucket or object doesn't exist. Check the name/key.
- **409 Conflict**: Bucket already exists (use `?createIfNotExists=true`) or bucket not empty (delete objects first).
- **401 Unauthorized**: Invalid or expired credentials. Check API key and timestamp (must be within 15 min of server time).

## Common Workflows

### Store AI-generated artifacts

```go
// Create a dedicated bucket for AI outputs
c.CreateBucketIfNotExists(ctx, "ai-artifacts")

// Store with structured key paths
c.PutObject(ctx, "ai-artifacts", fmt.Sprintf("session-%s/output.txt", sessionID), result)

// Generate a shareable link
url, _ := c.CreatePresignedURL(ctx, client.CreatePresignedURLRequest{
    Bucket:    "ai-artifacts",
    Key:       fmt.Sprintf("session-%s/output.txt", sessionID),
    Method:    "GET",
    ExpiresIn: intPtr(86400), // 24 hours
})
```

### Upload and share a file

```go
c.CreateBucketIfNotExists(ctx, "shared")
c.PutObject(ctx, "shared", "report.pdf", pdfBytes)
url, _ := c.PresignObjectURL("GET", "shared", "report.pdf", time.Now().Add(7*24*time.Hour))
// Share url with recipient
```

## Validation Rules

- **Bucket names**: 3-63 chars, lowercase alphanumeric + hyphens/dots, start/end with letter/number
- **Object keys**: Max 1024 bytes, no `..` or leading `/`, no path traversal
- **Max upload size**: 5GB per file
