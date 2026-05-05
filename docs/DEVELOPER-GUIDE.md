# Beamdrop Developer Guide

Welcome to the Beamdrop developer guide! This document will help you understand how the codebase is structured and how the S3-compatible API works.

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Project Structure](#project-structure)
4. [Understanding the S3 API](#understanding-the-s3-api)
5. [Request Flow](#request-flow)
6. [Storage Layer](#storage-layer)
7. [Authentication & Security](#authentication--security)
8. [Common Development Tasks](#common-development-tasks)
9. [Testing](#testing)

## Overview

Beamdrop is a self-hosted file sharing server built with Go and React. It provides:

- **Web Interface**: React-based UI for interactive file management
- **REST API**: Standard file operations (upload, download, list, etc.)
- **S3-Compatible API**: Amazon S3-like API for programmatic access with bucket/object storage

The backend is written in Go, serving both the web UI and APIs from a single binary.

## Architecture

### High-Level Components

```
┌─────────────────────────────────────────────────────────┐
│                    HTTP Server                          │
│                  (beam/server)                          │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │  Web Routes  │  │  API Routes  │  │  S3 Routes   │ │
│  │              │  │              │  │              │ │
│  │  /files      │  │  /api/shares │  │  /api/v1/    │ │
│  │  /upload     │  │  /api/logs   │  │  buckets/    │ │
│  │  /download   │  │              │  │              │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
│                                                         │
├─────────────────────────────────────────────────────────┤
│                   Middleware Layer                      │
│  • CORS • Rate Limiting • Auth • CSRF • Security Headers│
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │   Storage    │  │   Database   │  │    Crypto    │ │
│  │   Package    │  │   (SQLite)   │  │ (HMAC, AES)  │ │
│  │              │  │              │  │              │ │
│  │  Buckets &   │  │  API Keys    │  │  Signatures  │ │
│  │  Objects     │  │  Links       │  │  Encryption  │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
│                                                         │
└─────────────────────────────────────────────────────────┘
                          ↓
              ┌───────────────────────┐
              │   Filesystem          │
              │                       │
              │  /shared-directory/   │
              │    ├── buckets/       │
              │    ├── .beamdrop/     │
              │    └── .beamdrop_data/│
              └───────────────────────┘
```

## Project Structure

```
beamdrop/
├── cmd/beam/                 # Application entry point
│   ├── main.go              # CLI argument parsing & server startup
│   └── help.go              # Help text and version info
│
├── beam/                    # Core server implementation
│   └── server/
│       ├── server.go        # HTTP server setup
│       ├── routes.go        # Route registration
│       ├── websocket.go     # WebSocket handlers for real-time stats
│       └── handlers/        # HTTP request handlers
│           ├── files.go     # File browser operations
│           ├── health.go    # Health check endpoints
│           ├── shareable_links.go  # Shareable link management
│           └── api/         # S3-compatible API handlers
│               ├── buckets.go      # Bucket operations
│               ├── objects.go      # Object operations
│               ├── keys.go         # API key management
│               └── middleware.go   # API authentication
│
├── pkg/                     # Reusable packages
│   ├── storage/             # Storage abstraction layer
│   │   ├── bucket.go        # Bucket management
│   │   ├── object.go        # Object management
│   │   ├── atomic.go        # Atomic file writes
│   │   └── locks.go         # File locking for concurrency
│   │
│   ├── crypto/              # Cryptographic utilities
│   │   ├── signature.go     # HMAC-SHA256 signing/verification, AES-256-GCM encryption, bcrypt hashing
│   │   └── keystore.go      # Shared encryption key management
│   │
│   ├── db/                  # Database layer (SQLite)
│   │   ├── db.go           # Database connection
│   │   ├── api_keys.go     # API key CRUD operations
│   │   ├── shareable_links.go  # Shareable link storage
│   │   └── migrate.go      # Database migrations
│   │
│   ├── auth/                # Authentication middleware
│   │   ├── middleware.go   # Session-based auth for web UI
│   │   └── password.go     # Password hashing, JWT management, token revocation
│   │
│   ├── middleware/          # HTTP middleware
│   │   ├── cors.go         # CORS handling
│   │   ├── csrf.go         # CSRF double-submit cookie protection
│   │   ├── ratelimit.go    # Rate limiting with trusted proxy support
│   │   └── security.go     # Security headers (CSP, Permissions-Policy, etc.)
│   │
│   ├── errors/              # Structured error handling
│   │   └── errors.go       # Error types and HTTP responses
│   │
│   ├── metrics/             # Prometheus metrics
│   │   ├── metrics.go      # Metric definitions
│   │   └── collector.go    # Background metrics collector
│   │
│   └── logger/              # Structured logging
│       └── logger.go       # Dual-output logger (console + file)
│
├── config/                  # Configuration management
│   ├── config.go           # Flag definitions
│   └── db.go               # Database path configuration
│
├── static/frontend/         # React web application
│   └── src/                # Frontend source code
│
└── docs/                    # Documentation
    ├── s3-api-design.md    # S3 API specification
    ├── s3-api-walkthrough.md  # Detailed code walkthrough
    └── openapi.yaml        # OpenAPI specification
```

## Understanding the S3 API

The S3-compatible API allows applications to interact with Beamdrop using familiar S3-like operations. It's **not a full S3 implementation** but supports the most common operations.

### Key Concepts

#### Buckets

- A **bucket** is a top-level container for objects (like a folder)
- Bucket names must be 3-63 characters, lowercase alphanumeric, hyphens, or dots
- Stored as directories under `{shared-dir}/buckets/`

#### Objects

- An **object** is a file stored in a bucket
- Objects are identified by a **key** (path within the bucket)
- Keys can contain slashes to create hierarchical structures
- Stored as files: `{shared-dir}/buckets/{bucket}/{key}`

#### API Keys

- API keys provide programmatic access to the S3 API
- Each key has an **access key ID** (public) and **secret key** (private)
- Requests are signed using HMAC-SHA256 to prove ownership of the secret key

### Supported Operations

| Operation           | HTTP Method | Endpoint                              |
| ------------------- | ----------- | ------------------------------------- |
| List buckets        | GET         | `/api/v1/buckets`                     |
| Create bucket       | PUT         | `/api/v1/buckets/{bucket}`            |
| Delete bucket       | DELETE      | `/api/v1/buckets/{bucket}`            |
| Check bucket exists | HEAD        | `/api/v1/buckets/{bucket}`            |
| List objects        | GET         | `/api/v1/buckets/{bucket}?prefix=...` |
| Get object          | GET         | `/api/v1/buckets/{bucket}/{key}`      |
| Put object          | PUT         | `/api/v1/buckets/{bucket}/{key}`      |
| Delete object       | DELETE      | `/api/v1/buckets/{bucket}/{key}`      |
| Head object         | HEAD        | `/api/v1/buckets/{bucket}/{key}`      |

### Code Organization

The S3 API implementation is in `beam/server/handlers/api/`:

1. **buckets.go**: Handles bucket operations (create, list, delete)
2. **objects.go**: Handles object operations (put, get, delete, list)
3. **keys.go**: Manages API keys (create, list, delete)
4. **middleware.go**: Authenticates requests using HMAC signatures

## Request Flow

Let's trace a complete request through the system.

### Example: Uploading a File via S3 API

```
Client Request:
  PUT /api/v1/buckets/photos/vacation/beach.jpg
  Authorization: Bearer BDK_abc123:signature
  X-Beamdrop-Date: 2026-02-24T12:00:00Z
  Content-Type: image/jpeg
  Body: [file data]

     ↓

1. HTTP Server (server.go:ServeHTTP)
   • Logs request
   • Applies middleware (CORS, rate limiting, max storage check, security headers)

     ↓

2. Route Matching (routes.go:setupAPIRoutes)
   • Matches `/api/v1/buckets/` pattern
   • Routes to API handler wrapper

     ↓

3. API Auth Middleware (api/middleware.go:Middleware)
   • Checks if API auth is enabled
   • Extracts Authorization header
   • Parses: "Bearer {accessKeyId}:{signature}"
   • Validates X-Beamdrop-Date timestamp (15-min window)
   • Looks up API key in database
   • Computes expected signature:
       message = "PUT\n/api/v1/buckets/photos/vacation/beach.jpg\n2026-02-24T12:00:00Z"
       expected_sig = HMAC-SHA256(secret_key, message)
   • Compares signatures (constant-time)
   • Updates "last used" timestamp

     ↓

4. Object Handler (api/objects.go:Handle)
   • Parses path: bucket="photos", key="vacation/beach.jpg"
   • Routes to putObject() based on PUT method

     ↓

5. Put Object (api/objects.go:putObject)
   • Validates bucket name
   • Checks bucket exists
   • Calls ObjectManager.PutObject()

     ↓

6. Object Manager (storage/object.go:PutObject)
   • Validates object key
   • Acquires write lock for this object (prevents concurrent writes)
   • Creates parent directories
   • Uses AtomicWriter for crash-safe writes

     ↓

7. Atomic Writer (storage/atomic.go)
   • Writes to temporary file: beach.jpg.tmp.{uuid}
   • Streams data while computing MD5 hash (ETag)
   • Calls fsync() to flush to disk
   • Atomically renames temp file to final name
   • Releases lock

     ↓

8. Response
   • Returns 200 OK with JSON:
     {
       "bucket": "photos",
       "key": "vacation/beach.jpg",
       "etag": "d41d8cd98f00b204e9800998ecf8427e",
       "size": 1234567,
       "url": "/api/v1/buckets/photos/vacation/beach.jpg"
     }
```

### Request Flow Diagram

```
┌──────────┐
│  Client  │
└────┬─────┘
     │ PUT /api/v1/buckets/photos/beach.jpg
     │ Authorization: Bearer key:sig
     │ X-Beamdrop-Date: timestamp
     │
     ▼
┌─────────────────────────────────┐
│  HTTP Server (server.go)        │
│  • Logs request                 │
│  • Applies rate limiting        │
│  • Checks max storage limit     │
│  • Adds security headers        │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│  Routes (routes.go)             │
│  • Pattern match                │
│  • Route to handler             │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│  API Auth Middleware            │
│  • Extract credentials          │
│  • Look up API key in DB        │
│  • Verify HMAC signature        │
│  • Check timestamp validity     │
└────────────┬────────────────────┘
             │ Authenticated ✓
             ▼
┌─────────────────────────────────┐
│  Object Handler (objects.go)    │
│  • Parse bucket & key           │
│  • Route by HTTP method         │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│  Object Manager (object.go)     │
│  • Validate inputs              │
│  • Acquire file lock            │
│  • Write with AtomicWriter      │
│  • Calculate ETag (MD5)         │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│  Filesystem                     │
│  buckets/photos/beach.jpg       │
└─────────────────────────────────┘
```

## Storage Layer

The storage layer (`pkg/storage/`) provides safe, concurrent access to files.

### Bucket Manager (`bucket.go`)

Manages bucket directories:

```go
type BucketManager struct {
    basePath string  // Path to buckets/ directory
}

// Operations
CreateBucket(name)              // Create directory: buckets/{name}/
CreateBucketIfNotExists(name)   // Create if absent, returns (created bool, err)
DeleteBucket(name)              // Delete if empty
ListBuckets()                   // Read buckets/ directory
BucketExists(name)              // Check if directory exists
```

**Key validations:**

- Bucket names: 3-63 chars, lowercase, alphanumeric/hyphens/dots
- No IP addresses (prevents `192.168.1.1` as bucket name)
- Must start/end with letter or number

### Object Manager (`object.go`)

Manages files within buckets:

```go
type ObjectManager struct {
    bucketManager *BucketManager
    LockManager   *LockManager
}

// Operations
PutObject(bucket, key, reader)     // Upload file
GetObject(bucket, key)             // Download file
DeleteObject(bucket, key)          // Delete file
HeadObject(bucket, key)            // Get metadata only
ListObjects(bucket, prefix, delimiter, maxKeys)  // List files
```

**Key features:**

- **Path validation**: Prevents `..` (directory traversal) and leading `/`
- **File locking**: Prevents concurrent writes to same file
- **Atomic writes**: Uses temporary files + rename for crash safety
- **ETag calculation**: MD5 hash computed during upload

### Atomic Writer (`atomic.go`)

Ensures writes are crash-safe:

```go
type AtomicWriter struct {
    targetPath string      // Final file path
    tempPath   string      // Temporary file path
    file       *os.File
}

// Usage
writer := NewAtomicWriter("/buckets/photos/beach.jpg")
writer.Write(data)  // Writes to beach.jpg.tmp.{uuid}
writer.Commit()     // fsync() + rename to beach.jpg
```

**How it works:**

1. Create temp file: `beach.jpg.tmp.abc123`
2. Write all data to temp file
3. Call `fsync()` to flush to disk
4. Atomically rename temp → final (OS guarantees atomicity)
5. If crash occurs before step 4, temp file is orphaned (cleaned up later)

### Lock Manager (`locks.go`)

Prevents concurrent modifications:

```go
type LockManager struct {
    locks   map[string]*sync.RWMutex
    timeout time.Duration
}

// Usage
unlock, err := lockManager.Lock("photos", "beach.jpg")
defer unlock()
// ... perform write operation ...
```

**Why locking?**

- Multiple clients might upload to same key simultaneously
- Without locking, file could be corrupted
- Locks are per-object (bucket + key), not global

## Authentication & Security

### S3 API Authentication

Beamdrop uses **HMAC-SHA256 request signing**, similar to AWS Signature Version 4 (but simpler).

#### How It Works

1. **Client has:**
   - Access Key ID: `BDK_abc123` (public identifier)
   - Secret Key: `sk_xyz789` (private, never sent over network)

2. **Client creates signature:**

   ```
   message = "{METHOD}\n{PATH}\n{TIMESTAMP}"
   signature = Base64(HMAC-SHA256(secret_key, message))
   ```

3. **Client sends request:**

   ```http
   PUT /api/v1/buckets/photos/beach.jpg
   Authorization: Bearer BDK_abc123:{signature}
   X-Beamdrop-Date: 2026-02-24T12:00:00Z
   ```

4. **Server verifies:**
   - Looks up secret key for `BDK_abc123` in database
   - Computes expected signature using same algorithm
   - Compares signatures (constant-time to prevent timing attacks)

#### Code: Signature Generation (`pkg/crypto/signature.go`)

```go
func GenerateSignature(secretKey, method, path, timestamp string) string {
    // Create message to sign
    message := fmt.Sprintf("%s\n%s\n%s", method, path, timestamp)

    // Compute HMAC-SHA256
    h := hmac.New(sha256.New, []byte(secretKey))
    h.Write([]byte(message))

    // Return base64-encoded signature
    return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func VerifySignature(secretKey, method, path, timestamp, signature string) bool {
    expected := GenerateSignature(secretKey, method, path, timestamp)
    return hmac.Equal([]byte(expected), []byte(signature))  // Constant-time comparison
}
```

#### Why Timestamp?

The `X-Beamdrop-Date` header prevents **replay attacks**:

- Server only accepts requests within ±15 minutes of current time
- An attacker who intercepts a signed request can't replay it later
- Implemented in `crypto.IsTimestampValid()`

#### API Key Storage (`pkg/db/api_keys.go`)

```go
type APIKey struct {
    ID          uint
    Name        string        // Human-friendly name
    AccessKeyID string        // Public identifier (BDK_...)
    SecretKey   string        // HMAC key (AES-256-GCM encrypted in database)
    Permissions string        // Future: fine-grained permissions
    BucketScope string        // Future: restrict to specific bucket
    ExpiresAt   *time.Time    // Optional expiration
    LastUsedAt  *time.Time    // Track usage
    Disabled    bool          // Soft delete
}

// Creating a key
apiKey, secretKey, err := db.CreateAPIKey(name, permissions, bucketScope, expiresIn)
// The secretKey is encrypted with AES-256-GCM before storage
// Returns the encrypted version (stored) and plain secret (shown once to user)
```

### Web UI Authentication

The web UI uses **cookie-based JWT authentication**:

- Optional password protection (`-p` flag)
- JWT tokens stored in `HttpOnly`, `SameSite=Strict` cookies (not localStorage)
- Token revocation on logout via in-memory JTI blocklist
- CSRF protection via double-submit cookie pattern
- Implemented in `pkg/auth/middleware.go`

## Common Development Tasks

### Adding a New S3 API Endpoint

Let's add support for getting bucket metadata.

**1. Add handler method** (`beam/server/handlers/api/buckets.go`):

```go
func (h *BucketHandler) getBucketMetadata(w http.ResponseWriter, r *http.Request, name string) {
    if !h.bucketManager.BucketExists(name) {
        errors.BucketNotFound(name).WriteHTTPResponse(w)
        return
    }

    // Get bucket info
    info, err := h.bucketManager.GetBucketInfo(name)
    if err != nil {
        errors.InternalError("Failed to get bucket info").WriteHTTPResponse(w)
        return
    }

    sendJSON(w, info, http.StatusOK)
}
```

**2. Add storage method** (`pkg/storage/bucket.go`):

```go
func (bm *BucketManager) GetBucketInfo(name string) (*BucketInfo, error) {
    if err := ValidateBucketName(name); err != nil {
        return nil, err
    }

    bucketPath := filepath.Join(bm.basePath, name)
    stat, err := os.Stat(bucketPath)
    if err != nil {
        return nil, ErrBucketNotFound
    }

    return &BucketInfo{
        Name:      name,
        CreatedAt: stat.ModTime(),
    }, nil
}
```

**3. Wire up route** (`beam/server/handlers/api/buckets.go`):

```go
func (h *BucketHandler) Handle(w http.ResponseWriter, r *http.Request) {
    // ... existing code ...

    case http.MethodGet:
        if bucketName == "" {
            h.listBuckets(w, r)
        } else if r.URL.Query().Get("metadata") == "true" {
            h.getBucketMetadata(w, r, bucketName)  // New!
        } else {
            h.getBucketInfo(w, r, bucketName)
        }
}
```

### Adding Error Handling

Beamdrop uses structured errors (`pkg/errors/errors.go`):

```go
// Create custom error
err := errors.New(
    errors.CodeBucketNotFound,
    errors.CategoryNotFound,
    "The requested bucket does not exist",
    http.StatusNotFound,
)

// Add context
err = err.WithCause(originalErr)

// Send HTTP response
err.WriteHTTPResponse(w)
// Sends: {"error": "...", "code": "...", "status": 404}
```

### Adding Metrics

Prometheus metrics are defined in `pkg/metrics/metrics.go`:

```go
// Define metric
var bucketCreations = prometheus.NewCounter(prometheus.CounterOpts{
    Name: "beamdrop_bucket_creations_total",
    Help: "Total number of buckets created",
})

// Increment in handler
func (h *BucketHandler) createBucket(...) {
    // ... create bucket ...
    metrics.BucketCreations.Inc()
}

// Idempotent variant: PUT /api/v1/buckets/{name}?createIfNotExists=true
func (h *BucketHandler) createBucketIfNotExists(...) {
    // ... returns 201 if new, 200 if already exists ...
    metrics.BucketCreations.Inc()
}
```

### Enabling Debug Logging

```go
import "log/slog"

// Structured logging
slog.Debug("Processing request", "bucket", bucket, "key", key)
slog.Info("Bucket created", "name", name)
slog.Warn("Rate limit exceeded", "ip", ip)
slog.Error("Failed to write file", "error", err)
```

Logs go to:

- **Console**: Human-readable, colored output
- **File**: `{shared-dir}/.beamdrop/beamdrop.log` (structured JSON)

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run specific package
go test ./pkg/storage/...

# Run with coverage
go test -cover ./...

# Run with verbose output
go test -v ./pkg/storage/
```

### Example Test

```go
func TestBucketManager_CreateBucket(t *testing.T) {
    // Setup
    tmpDir := t.TempDir()
    bm := storage.NewBucketManager(tmpDir)
    bm.EnsureBucketsDir()

    // Test
    err := bm.CreateBucket("test-bucket")
    if err != nil {
        t.Fatalf("CreateBucket failed: %v", err)
    }

    // Verify
    if !bm.BucketExists("test-bucket") {
        t.Error("Bucket was not created")
    }
}
```

### Testing S3 API with curl

```bash
# 1. Start server with API auth
./beamdrop -dir ./test-data -api-auth

# 2. Create API key (via web UI or API)
curl -X POST http://localhost:7777/api/v1/keys \
  -H "Content-Type: application/json" \
  -d '{"name": "test-key"}'

# Save the response:
# {
#   "accessKeyId": "BDK_abc123",
#   "secretKey": "sk_xyz789",
#   ...
# }

# 3. Generate signature and upload
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
SECRET="sk_xyz789"
METHOD="PUT"
PATH="/api/v1/buckets/test/hello.txt"

# Compute signature
MESSAGE="${METHOD}\n${PATH}\n${TIMESTAMP}"
SIG=$(echo -n "$MESSAGE" | openssl dgst -sha256 -hmac "$SECRET" -binary | base64)

# Upload
curl -X PUT "http://localhost:7777${PATH}" \
  -H "Authorization: Bearer BDK_abc123:${SIG}" \
  -H "X-Beamdrop-Date: ${TIMESTAMP}" \
  -H "Content-Type: text/plain" \
  -d "Hello, World!"
```

## Summary

Key takeaways:

1. **Architecture**: Clean separation between server, handlers, and storage
2. **S3 API**: Simplified S3-compatible API with bucket/object model
3. **Storage**: Atomic writes + file locking for safety and concurrency
4. **Auth**: HMAC-SHA256 request signing with timestamp validation
5. **Code Organization**: Each component has a single responsibility

### Recommended Reading Order

1. Start: `cmd/beam/main.go` - Entry point
2. Server: `beam/server/server.go` - HTTP server setup
3. Routes: `beam/server/routes.go` - URL routing
4. S3 API: `beam/server/handlers/api/` - S3 handlers
5. Storage: `pkg/storage/` - File operations
6. Auth: `pkg/crypto/signature.go` - Request signing

### Further Documentation

- **S3 API Spec**: [docs/s3-api-design.md](s3-api-design.md)
- **Code Walkthrough**: [docs/s3-api-walkthrough.md](s3-api-walkthrough.md)
- **OpenAPI**: [docs/openapi.yaml](openapi.yaml)
- **Security**: [docs/SECURITY.md](SECURITY.md)

### Getting Help

- Review existing code - patterns are consistent
- Check error handling - structured errors guide you
- Use logging - `slog` statements show execution flow
- Read tests - they demonstrate usage

Happy coding! 🚀
