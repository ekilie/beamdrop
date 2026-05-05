# Beamdrop S3 API Deep Code Walkthrough

A plain-English, file-by-file explanation of how the S3-compatible API works inside beamdrop, so you can read the code with confidence.

---

## Table of Contents

1. [Big Picture How Everything Fits Together](#1-big-picture)
2. [How a Request Travels Through the Code](#2-request-lifecycle)
3. [Route Registration (`beam/server/routes.go`)](#3-route-registration)
4. [Authentication Middleware (`beam/server/handlers/api/middleware.go`)](#4-authentication-middleware)
5. [Rate Limiting Middleware (`pkg/middleware/ratelimit.go`)](#5-rate-limiting)
6. [Structured Logging (`pkg/logger/logger.go`)](#6-structured-logging)
7. [Bucket Handler (`beam/server/handlers/api/buckets.go`)](#7-bucket-handler)
8. [Object Handler (`beam/server/handlers/api/objects.go`)](#8-object-handler)
9. [Storage Layer BucketManager (`pkg/storage/bucket.go`)](#9-bucket-manager)
10. [Storage Layer ObjectManager (`pkg/storage/object.go`)](#10-object-manager)
11. [Atomic Writes (`pkg/storage/atomic.go`)](#11-atomic-writes)
12. [File-Level Locking (`pkg/storage/locks.go`)](#12-file-locking)
13. [Database Transaction Safety (`pkg/db/`)](#13-db-transaction-safety)
14. [API Key Management Database (`pkg/db/api_keys.go`)](#14-api-key-database)
15. [API Key Management Handler (`beam/server/handlers/api/keys.go`)](#15-keys-handler)
16. [Cryptography & Signatures (`pkg/crypto/signature.go`)](#16-cryptography)
17. [Error Handling (`pkg/errors/errors.go`)](#17-error-handling)
18. [How It All Connects Diagrams](#18-diagrams)
19. [Example Walkthrough: Uploading a File](#19-example-upload)
20. [Example Walkthrough: Downloading a File](#20-example-download)
21. [Glossary](#21-glossary)

---

<a name="1-big-picture"></a>

## 1. Big Picture How Everything Fits Together

Beamdrop's S3 API lets you manage files programmatically using **buckets** (like folders) and **objects** (files inside those folders). It mimics the concepts from Amazon S3 but is backed by your **local filesystem** instead of cloud storage.

### The Five Layers

```
┌─────────────────────────────────────────────────┐
│  HTTP Layer  (routes.go)                        │
│  Receives HTTP requests, routes to handlers     │
├─────────────────────────────────────────────────┤
│  Rate Limiting  (pkg/middleware/ratelimit.go)   │
│  Per-IP token-bucket throttling (3 tiers)       │
├─────────────────────────────────────────────────┤
│  Auth Layer  (api/middleware.go)                │
│  Validates API keys and request signatures      │
├─────────────────────────────────────────────────┤
│  Handler Layer  (api/buckets.go, objects.go)    │
│  Business logic   what to do with the request   │
├─────────────────────────────────────────────────┤
│  Storage Layer  (pkg/storage/*)                 │
│  Reads/writes actual files on disk              │
└─────────────────────────────────────────────────┘
```

### Where Files Live on Disk

```
your_shared_directory/
└── buckets/                    ← BucketManager manages this
    ├── my-app-uploads/         ← A "bucket" (just a directory)
    │   ├── images/
    │   │   └── photo.jpg       ← An "object" (just a file)
    │   └── report.pdf
    └── backups/
        └── db-dump.sql
```

Every "bucket" is a **directory**. Every "object" is a **file** inside that directory. The "key" is just the **relative path** of the file inside the bucket.

---

<a name="2-request-lifecycle"></a>

## 2. How a Request Travels Through the Code

Let's trace what happens when you send:

```
PUT /api/v1/buckets/photos/vacation/beach.jpg
```

```
Step 1  →  Server receives HTTP request
              (beam/server/server.go   ServeHTTP method)

Step 2  →  Security headers & CORS middleware run
              (pkg/middleware/security.go, cors.go)

Step 3  →  Rate limiter checks per-IP token bucket
              (pkg/middleware/ratelimit.go   classifies as
               general / auth / upload tier, rejects with 429
               if tokens exhausted)

Step 4  →  Route matching
              (beam/server/routes.go   matches "/api/v1/buckets/")

Step 5  →  API Auth Middleware runs
              (api/middleware.go   checks API key + signature)

Step 6  →  Router decides: bucket or object handler?
              Path has "photos/vacation/beach.jpg"
              → parts[0] = "photos" (bucket)
              → parts[1] = "vacation/beach.jpg" (key, non-empty)
              → Routes to ObjectHandler

Step 7  →  ObjectHandler.Handle() dispatches by HTTP method
              Method is PUT → calls putObject()

Step 8  →  putObject() uses ObjectManager.PutObject()
              (pkg/storage/object.go)

Step 9  →  ObjectManager acquires a write lock
              (pkg/storage/locks.go   per-object locking)

Step 10 →  ObjectManager validates names, creates dirs,
              uses AtomicWriter to write the file safely

Step 11 →  Write lock released, response sent back
              with ETag, size, URL
```

---

<a name="3-route-registration"></a>

## 3. Route Registration `beam/server/routes.go`

This is where the S3 API endpoints get wired up.

### The Key Function: `setupAPIRoutes()`

```go
func (s *Server) setupAPIRoutes() {
    bucketHandler := api.NewBucketHandler(s.sharedDir)
    objectHandler := api.NewObjectHandler(s.sharedDir)
    keysHandler   := api.NewKeysHandler()

    apiAuth := api.NewAPIAuthMiddleware(s.flags.APIAuth)
    // ...
}
```

**What it does:**

1. Creates three handlers one for buckets, one for objects, one for API keys
2. Creates the auth middleware (enabled/disabled by the `-api-auth` command-line flag)
3. Registers URL patterns on the HTTP mux

### URL Patterns

| Pattern            | What it matches                                                 |
| ------------------ | --------------------------------------------------------------- |
| `/api/v1/buckets`  | List all buckets (no trailing slash)                            |
| `/api/v1/buckets/` | Anything under buckets the router inspects the rest of the path |
| `/api/v1/keys`     | Manage API keys                                                 |

### The Smart Router Logic

Inside the `/api/v1/buckets/` handler, the code decides **who handles the request**:

```go
path := strings.TrimPrefix(r.URL.Path, "/api/v1/buckets/")
parts := strings.SplitN(path, "/", 2)

if len(parts) > 1 && parts[1] != "" {
    objectHandler.Handle(w, r)   // Has a key → object operation
    return
}
bucketHandler.Handle(w, r)       // No key → bucket operation
```

**Translation:**

- `/api/v1/buckets/my-bucket` → `parts = ["my-bucket"]` → BucketHandler
- `/api/v1/buckets/my-bucket/file.txt` → `parts = ["my-bucket", "file.txt"]` → ObjectHandler

---

<a name="4-authentication-middleware"></a>

## 4. Authentication Middleware `beam/server/handlers/api/middleware.go`

This file is responsible for making sure only authorized clients can use the S3 API.

### How It Works

The middleware supports **two** authentication methods:

#### Method 1: Signed Requests (Authorization Header)

```
Authorization: Bearer BDK_abc123:BASE64_SIGNATURE
X-Beamdrop-Date: 2026-02-12T10:30:00Z
```

**The flow:**

```go
// 1. Extract credentials from header
credentials := "BDK_abc123:BASE64_SIGNATURE"
accessKeyID := "BDK_abc123"     // Your public key ID
signature   := "BASE64_SIGNATURE"  // Proof you know the secret

// 2. Check the timestamp (must be within 15 minutes)
timestamp := r.Header.Get("X-Beamdrop-Date")
crypto.IsTimestampValid(timestamp)  // Prevents replay attacks

// 3. Look up the API key in the database
apiKey := db.GetAPIKeyByAccessID(accessKeyID)

// 4. Verify the signature matches
crypto.VerifySignature(apiKey.SecretKey, method, path, timestamp, signature)
```

The signature is an **HMAC-SHA256** hash of: `METHOD\nPATH\nTIMESTAMP`, signed with your secret key. This proves you know the secret without sending it over the network.

#### Method 2: Presigned URLs (Token in Query String)

```
GET /api/v1/buckets/photos/pic.jpg?token=ABC&expires=2026-02-12T12:00:00Z&access_key=BDK_abc123
```

Presigned URLs let you share a temporary download/upload link without revealing your API key. The URL itself contains all the authentication information anyone with the link can access the file until it expires.

**How the middleware verifies a presigned URL:**

1. Checks if the URL has expired (`time.Now().After(expiresAt)` → reject if past)
2. Extracts the bucket and key from the path
3. Looks up the API key by `access_key` query parameter
4. Recomputes the token: `HMAC-SHA256(secret_key, "METHOD\nBUCKET\nKEY\nUNIX_TIMESTAMP")`
5. Compares the recomputed token against the provided `token` constant-time comparison via `hmac.Equal()`

**How to generate a presigned URL (client-side):**

```
Step 1: Decide when it expires → expiresAt = now + duration (as Unix timestamp)
Step 2: Build the message    → "GET\nphotos\npic.jpg\n1707741600"
Step 3: Compute the token    → Base64URL(HMAC-SHA256(secret_key, message))
Step 4: Build the URL        → /api/v1/buckets/photos/pic.jpg?token=TOKEN&expires=ISO8601&access_key=BDK_xxx
```

**Expiration format:** The `expires` query parameter accepts both RFC 3339 (`2026-02-12T12:00:00Z`) and compact ISO (`20260212T120000Z`) formats, but the HMAC message always uses the **Unix timestamp** (integer seconds since epoch).

**Important behaviors and limitations:**

| Behavior            | Detail                                                                                                                      |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| **Always expires**  | There is no "permanent" presigned URL. The middleware always checks `time.Now().After(expiresAt)`.                          |
| **Tied to API key** | If the API key is deleted, disabled, or rotated, all presigned URLs generated with that key stop working immediately.       |
| **Method-specific** | A token generated for `GET` will not work for `PUT` the HTTP method is part of the signed message.                          |
| **Key-specific**    | A token for `photos/pic.jpg` will not work for `photos/other.jpg` the bucket and key are part of the signed message.        |
| **Maximum expiry**  | No server-side maximum. You can set `expiresAt` to year 2100, but the URL will break if the API key is rotated before then. |
| **No revocation**   | Individual presigned URLs cannot be revoked. To invalidate all URLs, delete or disable the API key.                         |

**Practical expiry guidelines:**

| Use case                                     | Suggested expiry                               |
| -------------------------------------------- | ---------------------------------------------- |
| One-time download link (email, chat)         | 1–24 hours                                     |
| Embedded in a web page (avatars, thumbnails) | 1–7 days                                       |
| Client portal / invoice download             | 7–30 days                                      |
| Semi-permanent static asset                  | 1–10 years (works, but breaks on key rotation) |

**For truly public / permanent files**, consider running Beamdrop without `-api-auth` (all reads are public), or serving files through your application as a proxy.

#### Method 3: Server-Side Pretty Presigned URLs (URL Registry)

```
POST /api/v1/presign → creates a short /dl/{token} URL
GET  /dl/{token}     → anyone can download (no auth required)
```

Instead of generating long HMAC-signed URLs client-side, you can create short, clean download links via the server-side presigned URL registry. This requires an authenticated API call to create the link, but the resulting `/dl/{token}` URL is short enough to share in emails, embed in pages, or send to clients.

**Advantages over client-side presigned URLs:**

| Feature                     | Client-Side (HMAC)                           | Server-Side (Pretty)                   |
| --------------------------- | -------------------------------------------- | -------------------------------------- |
| URL length                  | Long (token + expires + access_key in query) | Short (`/dl/a1b2c3d4...`)              |
| Server round-trip to create | None                                         | One POST request                       |
| Max download limit          | No                                           | Yes                                    |
| Individually revocable      | No                                           | Yes (`DELETE /api/v1/presign/{token}`) |
| Download tracking           | No                                           | Yes (server counts)                    |
| Survives API key rotation   | No HMAC is tied to the key                   | Yes token is in the database           |

**Create a pretty presigned URL:**

```bash
curl -X POST https://server/api/v1/presign \
  -H "Authorization: Bearer BDK_xxx:signature" \
  -H "X-Beamdrop-Date: 2026-02-24T12:00:00Z" \
  -H "Content-Type: application/json" \
  -d '{
    "bucket": "photos",
    "key": "vacation/beach.jpg",
    "expiresIn": 3600,
    "maxDownloads": 100
  }'
# → {"token": "a1b2c3d4...", "url": "https://server/dl/a1b2c3d4...", ...}
```

**Download (public no auth):**

```bash
curl https://server/dl/a1b2c3d4...
```

**Revoke:**

```bash
curl -X DELETE https://server/api/v1/presign/a1b2c3d4... \
  -H "Authorization: Bearer BDK_xxx:signature" \
  -H "X-Beamdrop-Date: ..."
```

**Both methods can be used together.** Use client-side HMAC URLs for quick, ephemeral use cases. Use server-side pretty URLs for user-facing links where you need clean URLs, download limits, or revocation.

#### What If Auth Is Disabled?

```go
if !m.enabled {
    next.ServeHTTP(w, r)  // Just pass the request through
    return
}
```

When you start beamdrop **without** the `-api-auth` flag, all API requests go straight through. This is handy for development.

---

<a name="5-rate-limiting"></a>

## 5. Rate Limiting Middleware `pkg/middleware/ratelimit.go`

Beamdrop includes per-IP rate limiting to prevent abuse. It uses a **token-bucket** algorithm with three tiers, runs entirely in-memory, and requires zero external dependencies.

### Where It Sits in the Chain

Rate limiting runs **before** authentication abusive IPs are blocked before they can even attempt auth:

```
Security Headers → CORS → Rate Limiter → Auth → Mux → Handlers
```

### The Three Tiers

Different endpoints have different limits because not all operations have the same cost:

| Tier        | Endpoints                                   | Default Rate | Why it's different                                |
| ----------- | ------------------------------------------- | ------------ | ------------------------------------------------- |
| **General** | Everything not listed below                 | 100 req/min  | Normal browsing/API usage                         |
| **Auth**    | `/auth/login`                               | 5 req/min    | Prevents brute-force password guessing            |
| **Upload**  | `POST/PUT /upload`, `PUT /api/v1/buckets/…` | 10 req/min   | Uploads are expensive (disk I/O, CPU for hashing) |

Rates are configurable via the `-rate-limit` CLI flag, which sets the general rate. Auth and upload tiers are derived automatically:

```
General rate  = -rate-limit value (e.g. 100)
Auth rate     = max(1, general / 20)   →  5
Upload rate   = max(1, general / 10)   →  10
```

### How Token Buckets Work

Each client IP gets **three buckets** (one per tier). A bucket starts full and drains as requests arrive:

```
Bucket capacity: 100 tokens (= general rate)
Refill rate:     100/60 ≈ 1.67 tokens/second

Request arrives → consume 1 token
                  tokens > 0? → ✅ Allow request
                  tokens = 0? → ❌ Return 429 Too Many Requests
```

The bucket refills continuously at a steady rate, so a client can burst up to the bucket capacity and then must slow down to the refill rate.

### Request Classification

The middleware inspects the request path and method to determine the tier:

```go
func classifyRequest(r *http.Request) tier {
    // /auth/login → tierAuth (strictest)
    // POST|PUT /upload, PUT /api/v1/buckets/... → tierUpload
    // Everything else → tierGeneral
}
```

### IP Extraction

Client IPs are resolved in priority order:

1. `X-Forwarded-For` header (first IP in the list the original client)
2. `X-Real-IP` header
3. `RemoteAddr` (direct connection, stripped of port)

This ensures correct per-IP tracking behind reverse proxies.

### What Happens When a Client Is Rate-Limited

```
Client → PUT /api/v1/buckets/photos/pic.jpg
           │
           ▼
     Rate limiter checks upload bucket for this IP
           │
     Tokens remaining = 0
           │
           ▼
     429 Too Many Requests
     Retry-After: 3
     X-Retryable: true
```

The response uses the structured error system (see [Section 17](#17-error-handling)):

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "category": "RATE_LIMIT",
    "message": "Rate limit exceeded. Try again in 3s",
    "status": 429,
    "retryable": true,
    "retryAfter": 3,
    "timestamp": "2026-02-12T10:30:00Z"
  }
}
```

### Background Cleanup

Client buckets are lazily created on first request. A background goroutine runs every **5 minutes** and evicts entries for IPs not seen in the last **10 minutes**, keeping memory bounded.

| Constant         | Value  | Meaning                                     |
| ---------------- | ------ | ------------------------------------------- |
| Cleanup interval | 5 min  | How often stale entries are scanned         |
| Stale threshold  | 10 min | How long since last request before eviction |

### Disabling Rate Limiting

Set `-rate-limit 0` (or omit the flag) to disable rate limiting entirely. When disabled, the middleware returns `next` directly zero overhead.

---

<a name="6-structured-logging"></a>

## 6. Structured Logging `pkg/logger/logger.go`

Beamdrop uses Go's standard `log/slog` package for structured logging with **dual output**:

1. **Terminal** colored, human-readable lines for interactive use
2. **JSON file** machine-parseable structured logs at `<sharedDir>/.beamdrop/beamdrop.log`

### Initialization

```go
logger.Init(level, sharedDir)
// level: "debug", "info", "warn", or "error"
// sharedDir: base directory   log file goes to <sharedDir>/.beamdrop/beamdrop.log
```

Called once at startup from `main.go`. All subsequent logging uses `slog` directly:

```go
slog.Info("Rate limiting enabled", "general", rate, "unit", "req/min")
slog.Warn("Orphan cleanup failed", "error", err)
slog.Debug("Lock acquired", "bucket", bucket, "key", key)
```

### Terminal Output

The terminal handler uses colored output for quick scanning:

```
14:30:05.123 INFO  Rate limiting enabled general=100 unit=req/min
14:30:05.124 WARN  Orphan cleanup failed error="disk full"
```

| Level   | Color      |
| ------- | ---------- |
| `DEBUG` | Cyan       |
| `INFO`  | Green      |
| `WARN`  | Yellow     |
| `ERROR` | Red (bold) |

Source file paths are **not** shown in terminal output they're only recorded in the JSON log file.

### JSON Log File

The file handler writes one JSON object per line with full detail:

```json
{
  "time": "2026-02-12T14:30:05.123Z",
  "level": "INFO",
  "source": { "function": "main.main", "file": "cmd/beam/main.go", "line": 42 },
  "msg": "Rate limiting enabled",
  "general": 100,
  "unit": "req/min"
}
```

This is useful for log aggregation tools, `jq` queries, and post-incident analysis.

### Log Levels

Configure with the `-log-level` CLI flag:

| Level   | What it captures                                             |
| ------- | ------------------------------------------------------------ |
| `debug` | Everything lock acquisition, request details, internal state |
| `info`  | Normal operations startup, connections, uploads (default)    |
| `warn`  | Recoverable issues failed cleanups, deprecated usage         |
| `error` | Failures requiring attention disk errors, DB corruption      |

### Fatal Logging

`slog` has no built-in fatal level. Beamdrop provides `logger.Fatal()` which logs at error level and calls `os.Exit(1)`. Used only in `main.go` and `server.go` for unrecoverable startup failures.

---

<a name="7-bucket-handler"></a>

## 7. Bucket Handler `beam/server/handlers/api/buckets.go`

Manages buckets (creating, listing, deleting directories).

### Structure

```go
type BucketHandler struct {
    bucketManager *storage.BucketManager  // Does the actual filesystem work
}
```

### The `Handle()` Method Request Router

```go
func (h *BucketHandler) Handle(w http.ResponseWriter, r *http.Request) {
    // Extract bucket name from URL
    bucketName := ... // e.g., "my-bucket" from "/api/v1/buckets/my-bucket"

    switch r.Method {
    case GET:    → listBuckets() or getBucketInfo()
    case PUT:    → createBucket() or createBucketIfNotExists() (with ?createIfNotExists=true)
    case DELETE: → deleteBucket()
    case HEAD:   → headBucket()
    }
}
```

### Operations Explained

| Operation                  | HTTP     | URL                                                | What it does                               |
| -------------------------- | -------- | -------------------------------------------------- | ------------------------------------------ |
| List all buckets           | `GET`    | `/api/v1/buckets`                                  | Returns all bucket names + count           |
| Create bucket              | `PUT`    | `/api/v1/buckets/my-bucket`                        | Creates directory on disk                  |
| Create bucket (idempotent) | `PUT`    | `/api/v1/buckets/my-bucket?createIfNotExists=true` | Creates directory or returns 200 if exists |
| Delete bucket              | `DELETE` | `/api/v1/buckets/my-bucket`                        | Removes directory (must be empty)          |
| Check bucket exists        | `HEAD`   | `/api/v1/buckets/my-bucket`                        | Returns 200 or 404 (no body)               |
| Get bucket info            | `GET`    | `/api/v1/buckets/my-bucket`                        | Returns `{bucket, exists}`                 |

### Example Response: List Buckets

```json
{
  "buckets": [
    { "name": "photos", "createdAt": "2026-02-10T08:00:00Z" },
    { "name": "backups", "createdAt": "2026-02-09T14:30:00Z" }
  ],
  "count": 2
}
```

### Example Response: Create Bucket

```json
{
  "bucket": "my-bucket",
  "created": "2026-02-12T10:00:00Z",
  "location": "/api/v1/buckets/my-bucket"
}
```

---

<a name="8-object-handler"></a>

## 8. Object Handler `beam/server/handlers/api/objects.go`

Manages objects (files) inside buckets. This is the core of the S3 API.

### Structure

```go
type ObjectHandler struct {
    objectManager *storage.ObjectManager  // Reads/writes files
    bucketManager *storage.BucketManager  // Checks bucket existence
}
```

### The `Handle()` Method

```go
func (h *ObjectHandler) Handle(w http.ResponseWriter, r *http.Request) {
    // Parse: /api/v1/buckets/{bucket}/{key...}
    bucket := "photos"
    key    := "vacation/beach.jpg"

    switch r.Method {
    case GET:    → getObject() or listObjects()
    case PUT:    → putObject()          // Raw body upload
    case POST:   → putObjectMultipart() // Form upload
    case DELETE: → deleteObject()
    case HEAD:   → headObject()
    }
}
```

### Operations Explained

| Operation         | HTTP     | URL                              | What it does                             |
| ----------------- | -------- | -------------------------------- | ---------------------------------------- |
| Download file     | `GET`    | `/api/v1/buckets/photos/pic.jpg` | Streams file content back                |
| Upload (raw body) | `PUT`    | `/api/v1/buckets/photos/pic.jpg` | Reads request body, saves to disk        |
| Upload (form)     | `POST`   | `/api/v1/buckets/photos/pic.jpg` | Reads multipart form data                |
| Delete file       | `DELETE` | `/api/v1/buckets/photos/pic.jpg` | Removes the file                         |
| Get file info     | `HEAD`   | `/api/v1/buckets/photos/pic.jpg` | Returns headers only (size, type)        |
| List files        | `GET`    | `/api/v1/buckets/photos`         | Lists objects, supports prefix/delimiter |

### How `getObject()` Works

```go
func (h *ObjectHandler) getObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
    // 1. Ask ObjectManager to open the file
    file, info, err := h.objectManager.GetObject(bucket, key)

    // 2. Set response headers
    w.Header().Set("Content-Length", info.Size)
    w.Header().Set("Content-Type", getContentType(ext))
    w.Header().Set("ETag", info.ETag)

    // 3. Stream the file (supports Range requests for partial downloads!)
    http.ServeContent(w, r, key, info.LastModified, file)
}
```

`http.ServeContent` is a Go standard library function that handles:

- Sending the file efficiently
- Supporting `Range` headers (resume downloads, video seeking)
- Setting `Last-Modified` headers

### How `putObject()` Works

```go
func (h *ObjectHandler) putObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
    // 1. Verify bucket exists
    if !h.bucketManager.BucketExists(bucket) { ... }

    // 2. Write the file using ObjectManager
    info, err := h.objectManager.PutObject(bucket, key, r.Body)
    //                                                  ^^^^^^
    //                        The raw HTTP request body IS the file data

    // 3. Return metadata
    sendJSON(w, { bucket, key, etag, size, url }, 200)
}
```

### How `putObjectMultipart()` Works

This is the alternative upload path using HTML-style form uploads:

```go
func (h *ObjectHandler) putObjectMultipart(...) {
    // 1. Parse the multipart form (max 10GB)
    r.ParseMultipartForm(10 << 30)

    // 2. Get the uploaded file from the "file" field
    file, header, err := r.FormFile("file")

    // 3. Save it using the same ObjectManager.PutObject()
    info, err := h.objectManager.PutObject(bucket, key, file)
}
```

### How `listObjects()` Works

Supports S3-style listing with `prefix`, `delimiter`, and `max-keys` query parameters:

```
GET /api/v1/buckets/photos?prefix=vacation/&delimiter=/&max-keys=100
```

```json
{
  "bucket": "photos",
  "prefix": "vacation/",
  "delimiter": "/",
  "maxKeys": 100,
  "isTruncated": false,
  "contents": [
    { "key": "vacation/beach.jpg", "size": 204800, "lastModified": "..." }
  ],
  "commonPrefixes": ["vacation/summer/", "vacation/winter/"]
}
```

- **`prefix`**: Only return objects whose key starts with this string
- **`delimiter`**: Group keys by this character (usually `/`) to simulate folders
- **`commonPrefixes`**: The "virtual folders" found when using a delimiter
- **`max-keys`**: Maximum number of objects to return (default 1000)
- **`isTruncated`**: `true` if there are more results than `max-keys`

### Content Type Detection

The handler includes a `getContentType()` helper that maps file extensions to MIME types:

```go
".png"  → "image/png"
".jpg"  → "image/jpeg"
".pdf"  → "application/pdf"
".json" → "application/json"
// ... and more
// Unknown extensions default to "application/octet-stream"
```

---

<a name="9-bucket-manager"></a>

## 9. Storage Layer BucketManager (`pkg/storage/bucket.go`)

This is the **lowest-level** code that touches the filesystem for bucket operations. The handlers never write files directly they always go through these managers.

### Structure

```go
type BucketManager struct {
    basePath string   // e.g., "/home/user/shared/buckets"
}
```

`basePath` always points to the `buckets/` subdirectory inside the shared directory.

### Bucket Name Validation

Bucket names follow S3-like rules, enforced by a regex:

```go
var bucketNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)
```

**Rules:**

- 3 to 63 characters long
- Only lowercase letters, numbers, hyphens (`-`), and dots (`.`)
- Must start and end with a letter or number
- Cannot look like an IP address (e.g., `192.168.1.1`)

**Examples:**

- `my-bucket` valid
- `My-Bucket` invalid (uppercase)
- `ab` invalid (too short)
- `192.168.1.1` invalid (looks like IP)

### Object Key Validation

```go
func ValidateObjectKey(key string) error {
    if key == "" { return ErrInvalidKey }
    if strings.Contains(key, "..") { return ErrInvalidKey }   // No path traversal
    if strings.HasPrefix(key, "/") { return ErrInvalidKey }   // No absolute paths
    if len(key) > 1024 { return ErrInvalidKey }               // Max 1024 bytes
    return nil
}
```

This prevents attackers from doing things like:

- `../../etc/passwd` path traversal attack
- `/root/.ssh/id_rsa` absolute path escape

### Key Methods

| Method                          | What it does                                                |
| ------------------------------- | ----------------------------------------------------------- |
| `CreateBucket(name)`            | Validates name, checks if exists, creates directory         |
| `CreateBucketIfNotExists(name)` | Creates directory if absent, returns `(created bool, err)`  |
| `DeleteBucket(name)`            | Validates name, checks if empty, removes directory          |
| `BucketExists(name)`            | Validates name, checks if directory exists                  |
| `ListBuckets()`                 | Reads all directories, returns `[]BucketInfo`               |
| `GetBucketPath(name)`           | Returns filesystem path (e.g., `/shared/buckets/my-bucket`) |
| `EnsureBucketsDir()`            | Creates the `buckets/` directory if it doesn't exist        |

### Error Sentinels

The package defines named errors that handlers check against:

```go
var (
    ErrInvalidBucketName = errors.New("invalid bucket name")
    ErrBucketNotFound    = errors.New("bucket not found")
    ErrBucketNotEmpty    = errors.New("bucket is not empty")
    ErrBucketExists      = errors.New("bucket already exists")
    ErrObjectNotFound    = errors.New("object not found")
    ErrInvalidKey        = errors.New("invalid object key")
)
```

There are also lock-related error sentinels (defined in `locks.go`):

```go
var (
    ErrLockTimeout = errors.New("lock acquisition timed out")
    ErrDeadlock    = errors.New("potential deadlock detected")
)
```

Handlers use `errors.Is(err, storage.ErrLockTimeout)` to translate these into HTTP 423 (Locked) responses.

---

<a name="10-object-manager"></a>

## 10. Storage Layer ObjectManager (`pkg/storage/object.go`)

Handles file read/write operations. Uses `BucketManager` internally to resolve paths and check bucket existence.

### Structure

```go
type ObjectManager struct {
    bucketManager *BucketManager
    LockManager   *LockManager   // Per-object read/write locking
}
```

The `LockManager` is created automatically in `NewObjectManager()` and provides file-level locking for all object operations (see [Section 12](#12-file-locking) for details).

### `PutObject()` Writing a File

This is the most important method. Here's what happens step by step:

```go
func (om *ObjectManager) PutObject(bucket, key string, reader io.Reader) (*ObjectInfo, error) {
    // 1. Validate the bucket name and object key
    ValidateBucketName(bucket)
    ValidateObjectKey(key)

    // 2. Check the bucket exists on disk
    if !om.bucketManager.BucketExists(bucket) { return ErrBucketNotFound }

    // 3. Acquire a WRITE lock for this object (with timeout)
    unlock, err := om.LockManager.Lock(bucket, key)
    // ...if timeout → return ErrLockTimeout
    defer unlock()

    // 4. Build the full filesystem path
    //    e.g., "/shared/buckets/photos/vacation/beach.jpg"
    objectPath := filepath.Join(bucketPath, filepath.FromSlash(key))

    // 5. Create parent directories if needed
    //    e.g., creates "vacation/" directory
    os.MkdirAll(filepath.Dir(objectPath), 0755)

    // 6. Create an AtomicWriter (crash-safe writing)
    writer := NewAtomicWriter(objectPath)

    // 7. Write data while simultaneously computing MD5 hash (ETag)
    hash := md5.New()
    multiWriter := io.MultiWriter(writer, hash)
    size := io.Copy(multiWriter, reader)

    // 8. Commit the atomic write (fsync + rename)
    writer.Commit()

    // 9. Return metadata (lock released by defer)
    return &ObjectInfo{ Key, Size, LastModified, ETag }
}
```

**Key insight:** Step 7 uses `io.MultiWriter` to write to the file AND compute the MD5 hash **at the same time**, in a single pass through the data. No need to read the file twice.

**Concurrency note:** Step 3 ensures that if two clients upload to the same key simultaneously, uploads are **serialized** one waits for the other to finish. Uploads to _different_ keys proceed in parallel without blocking.

### `GetObject()` Reading a File

```go
func (om *ObjectManager) GetObject(bucket, key string) (*os.File, *ObjectInfo, func(), error) {
    // 1. Validate names
    // 2. Check bucket exists
    // 3. Acquire a READ lock for this object (with timeout)
    unlock, err := om.LockManager.RLock(bucket, key)
    // 4. Build path and open the file
    file := os.Open(objectPath)
    // 5. Get file info (size, modification time)
    info := file.Stat()
    // 6. Return file handle, metadata, AND the unlock function
    return file, &ObjectInfo{...}, unlock, nil
}
```

The caller (the handler) is responsible for **closing the file** and **calling the unlock function** after streaming it to the client. Multiple readers can read the same file concurrently the read lock is shared.

### `ListObjects()` Listing Files

This uses `filepath.Walk` to recursively scan a bucket directory:

```go
filepath.Walk(bucketPath, func(path string, info os.FileInfo, err error) {
    // 1. Compute relative path (the "key")
    key := filepath.ToSlash(relPath)

    // 2. Apply prefix filter
    if !strings.HasPrefix(key, prefix) { skip }

    // 3. Handle delimiter (virtual directories)
    //    If key is "images/photo.jpg" and delimiter is "/",
    //    add "images/" to commonPrefixes instead of contents

    // 4. Add to results (up to maxKeys)
    result.Contents = append(result.Contents, ObjectInfo{...})
})
```

### ObjectInfo The Metadata Struct

```go
type ObjectInfo struct {
    Key          string    `json:"key"`           // "vacation/beach.jpg"
    Size         int64     `json:"size"`          // 204800 (bytes)
    LastModified time.Time `json:"lastModified"`  // File modification time
    ETag         string    `json:"etag"`          // MD5 hash of contents
    ContentType  string    `json:"contentType"`   // MIME type (optional)
}
```

---

<a name="11-atomic-writes"></a>

## 11. Atomic Writes `pkg/storage/atomic.go`

This is a clever safety mechanism. **Why not just write directly to the file?**

**Problem:** If the server crashes mid-write, you'd get a **corrupted, half-written file**. The old version is gone and the new version is incomplete.

**Solution:** Write to a **temporary file** first, then **atomically rename** it.

### How AtomicWriter Works

```
Step 1: Create temp file        .beamdrop_tmp_abc123
Step 2: Write all data   →     .beamdrop_tmp_abc123 (complete)
Step 3: fsync()          →     Data flushed to disk (not just OS cache)
Step 4: close()          →     File handle released
Step 5: rename()         →     .beamdrop_tmp_abc123 → beach.jpg  (ATOMIC!)
Step 6: fsync(directory) →     Directory entry persisted
```

**Why rename is atomic:** On POSIX systems (Linux, macOS), `os.Rename` within the same filesystem is a single atomic operation. At no point is the file in a half-written state. Either you see the old file or the new file never something in between.

### Key Methods

```go
// Create a writer
writer := NewAtomicWriter("/path/to/final/file.jpg")

// Write data (implements io.Writer)
writer.Write(data)

// Commit: fsync + close + rename
writer.Commit()

// OR abort: cancel and delete temp file
writer.Abort()
```

### Cleanup on Startup

If the server crashed with orphaned temp files, `CleanupOrphanedTempFiles()` removes them on next startup:

```go
func CleanupOrphanedTempFiles(rootDir string) error {
    filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) {
        if strings.HasPrefix(info.Name(), ".beamdrop_tmp_") {
            os.Remove(path)  // Clean up leftover temp files
        }
    })
}
```

---

<a name="12-file-locking"></a>

## 12. File-Level Locking `pkg/storage/locks.go`

This is the concurrency safety mechanism. **Why do we need per-file locking?**

**Problem:** If two clients upload to the same key at the same time, the two writes race against each other. Even with atomic writes, the final rename from one upload could clobber the other, and the losing client would never know their upload was silently overwritten.

**Solution:** A **LockManager** that tracks per-object read/write locks in memory. Writers get exclusive access; readers can proceed concurrently.

### How It Works

```
Client A: PUT photos/pic.jpg          Client B: PUT photos/pic.jpg
    │                                      │
    ▼                                      ▼
 Lock("photos", "pic.jpg")            Lock("photos", "pic.jpg")
    │                                      │
    ▼                                      ▼
 ✅ Lock acquired                      ⏳ Waiting (blocked)...
    │                                      │
    ▼                                      │
 Write file (atomic)                       │
    │                                      │
    ▼                                      │
 unlock()  ────────────────────────▶   ✅ Lock acquired
                                           │
                                           ▼
                                       Write file (atomic)
                                           │
                                           ▼
                                       unlock()
```

Uploads to **different** keys are fully independent `Lock("photos", "pic.jpg")` and `Lock("photos", "other.jpg")` never block each other.

### The Lock Entry

Each object key gets its own `lockEntry` when first accessed:

```go
type lockEntry struct {
    mu        sync.RWMutex  // The actual Go read/write mutex
    refCount  int           // Number of goroutines using this entry
    writeLock bool          // Is a writer currently holding the lock?
    readers   int           // Number of active readers
    lockedAt  time.Time     // When the write lock was acquired (for deadlock detection)
    key       string        // The object key (for diagnostic logs)
}
```

Entries are **lazily created** and **automatically cleaned up** when no goroutines reference them, so the map never grows unboundedly.

### Lock Types

| Operation      | Lock Type             | Behavior                                  |
| -------------- | --------------------- | ----------------------------------------- |
| `PutObject`    | **Write** (exclusive) | Only one writer at a time per key         |
| `DeleteObject` | **Write** (exclusive) | Only one writer at a time per key         |
| `GetObject`    | **Read** (shared)     | Multiple concurrent readers allowed       |
| `HeadObject`   | **Read** (shared)     | Multiple concurrent readers allowed       |
| `ListObjects`  | **None**              | Directory scan, no per-object lock needed |

### Timeout

Lock acquisition has a configurable **timeout** (default: 30 seconds). If a lock can't be acquired within this period, the operation fails with `ErrLockTimeout` and the handler returns HTTP **423 Locked**:

```go
func (lm *LockManager) Lock(bucket, key string) (unlock func(), err error) {
    // 1. Get or create the lock entry for this key
    entry := lm.getOrCreateEntry(objectKey(bucket, key))

    // 2. Try to acquire in a goroutine (RWMutex.Lock blocks until available)
    done := make(chan struct{})
    go func() {
        entry.mu.Lock()
        close(done)
    }()

    // 3. Wait for acquisition OR timeout
    select {
    case <-done:
        return unlockFunc, nil   // Success!
    case <-time.After(lm.timeout):
        return nil, ErrLockTimeout  // Timed out
    }
}
```

The same pattern is used for `RLock()` (read locks).

### Deadlock Detection

A background goroutine runs every **10 seconds**, scanning for write locks held longer than **5 minutes**. When it finds one, it logs a warning:

```
14:35:30.123 WARN  Potential deadlock detected key=photos/pic.jpg held=5m30s threshold=5m0s
```

This helps operators identify stuck uploads or crashes that left a lock in a bad state. The thresholds:

| Constant                    | Default | Meaning                                     |
| --------------------------- | ------- | ------------------------------------------- |
| `DefaultLockTimeout`        | 30s     | Max wait time to acquire a lock             |
| `DeadlockDetectionInterval` | 10s     | How often the detector scans                |
| `StaleLockerThreshold`      | 5min    | How long before a held lock is "suspicious" |

### Lock Stats

```go
stats := objectManager.LockManager.Stats()
// stats.ActiveLocks   number of keys with active locks
// stats.WriteLocks    number of write locks currently held
// stats.ReadLocks     number of read locks currently held
```

### Error Responses

When a lock times out, the handler returns:

```json
{
  "error": {
    "code": "OBJECT_LOCKED",
    "category": "STORAGE",
    "message": "Object 'pic.jpg' is locked",
    "status": 423,
    "timestamp": "2026-02-13T10:30:00Z"
  }
}
```

### Lifecycle

The `LockManager` is created inside `NewObjectManager()` and runs a deadlock detector goroutine. Call `LockManager.Close()` on server shutdown to stop it cleanly.

---

<a name="13-db-transaction-safety"></a>

## 13. Database Transaction Safety `pkg/db/`

Beamdrop uses SQLite via GORM for metadata storage (API keys, shareable links, starred files, server stats). Operations that touch **both the database and the filesystem** need special care if one succeeds and the other fails, the system ends up in an inconsistent state.

This section covers three mechanisms that keep data consistent.

### The Problem

Consider creating a shareable link:

```
Step 1: Verify file exists on disk         ✅
Step 2: Insert link record into SQLite     ✅
Step 3: ... later, file is deleted ...
Step 4: User clicks the link → 404!  💥  (orphaned DB record)
```

Or the reverse: a multi-step DB operation where the first insert succeeds but the second fails, leaving half-written data.

### 13a. Transaction Helper `pkg/db/transaction.go`

For operations that involve **multiple DB writes** that should be all-or-nothing:

```go
func WithTransaction(fn func(tx *gorm.DB) error) error {
    tx := db.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
            panic(r)  // re-panic after rollback
        }
    }()

    if err := fn(tx); err != nil {
        tx.Rollback()
        return err
    }
    return tx.Commit().Error
}
```

**Usage example:** Creating an API key and immediately disabling it atomically:

```go
err := db.WithTransaction(func(tx *gorm.DB) error {
    if err := tx.Create(&apiKey).Error; err != nil {
        return err  // → Rollback
    }
    return tx.Model(&APIKey{}).Where("access_key_id = ?", id).Update("disabled", true).Error
})
// If either step fails, neither is committed
```

**Key behaviors:**

- If `fn` returns an error → transaction is **rolled back**
- If `fn` panics → transaction is **rolled back**, then panic propagates
- If `fn` returns nil → transaction is **committed**

### 13b. Saga Pattern `pkg/db/saga.go`

Transactions only work within a single database. When you need to coordinate **DB writes + filesystem writes**, you need the **saga pattern** a sequence of steps where each step has a compensating "undo" action.

```go
type SagaStep struct {
    Name       string        // Human-readable label
    Action     func() error  // Forward action (DB insert, FS write, etc.)
    Compensate func() error  // Rollback action (DB delete, FS remove, etc.)
}
```

**How it works:**

```
Step 1: Insert DB record       ✅  (compensate = DELETE record)
Step 2: Write file to disk     ✅  (compensate = remove file)
Step 3: Update another record  💥  FAILS!
        → Compensate step 2: remove file
        → Compensate step 1: DELETE record
        → Return original error
```

Compensation always runs in **reverse order** most recent successful step first.

**Usage example:** Creating a shareable link with filesystem verification:

```go
saga := db.NewSaga("create-shareable-link")

saga.AddStep(db.SagaStep{
    Name: "verify-path-exists",
    Action: func() error {
        _, err := os.Stat(fullPath)
        return err
    },
    Compensate: nil, // Nothing to undo
})

saga.AddStep(db.SagaStep{
    Name: "insert-link-record",
    Action: func() error {
        return db.GetDB().Create(&link).Error
    },
    Compensate: func() error {
        return db.GetDB().Delete(&link).Error  // Undo the insert
    },
})

if err := saga.Execute(); err != nil {
    // All completed steps have been compensated
    return err
}
```

### 13c. Orphaned Records Cleanup `pkg/db/cleanup.go`

Even with sagas, records can become orphaned over time (e.g., files deleted outside beamdrop, manual filesystem cleanup). The `OrphanCleaner` runs a background job to find and remove them.

```go
type OrphanCleaner struct {
    sharedDir string       // Base directory to resolve paths against
    stopCh    chan struct{} // Shutdown signal
}
```

**What it cleans up:**

| Record Type     | Condition for Removal                               |
| --------------- | --------------------------------------------------- |
| Starred files   | `os.Stat(sharedDir + filePath)` returns "not found" |
| Shareable links | Target path no longer exists on disk                |
| Expired links   | `expiresAt` is in the past                          |

**How it runs:**

```go
cleaner := db.NewOrphanCleaner(sharedDir)
cleaner.Start()  // Runs every hour in a background goroutine
// ...
cleaner.Stop()   // On server shutdown
```

The cleaner runs once immediately on startup (to catch anything from while the server was down), then repeats every hour.

**Example log output:**

```
15:00:00.456 INFO  Orphan cleanup complete starred=3 links=1 expired=2
```

### When Each Mechanism Is Used

| Scenario                               | Mechanism           |
| -------------------------------------- | ------------------- |
| Multiple DB writes that must be atomic | `WithTransaction()` |
| DB write + filesystem write together   | `Saga`              |
| Records referencing deleted files      | `OrphanCleaner`     |
| Expired shareable links                | `OrphanCleaner`     |
| API key create + immediate disable     | `WithTransaction()` |

---

<a name="14-api-key-database"></a>

## 14. API Key Management Database (`pkg/db/api_keys.go`)

API keys are stored in a SQLite database using GORM (a Go ORM).

### The APIKey Model

```go
type APIKey struct {
    ID          uint       // Auto-incrementing primary key
    Name        string     // Human-readable name, e.g., "CI/CD Pipeline"
    AccessKeyID string     // Public identifier: "BDK_a1b2c3d4e5f6g7h8"
    SecretKey   string     // Secret for HMAC signing (AES-256-GCM encrypted in DB)
    Permissions string     // JSON string with permission rules (future use)
    BucketScope string     // Limit key to a specific bucket (optional)
    ExpiresAt   *time.Time // Optional expiration date
    LastUsedAt  *time.Time // Tracks when the key was last used
    CreatedAt   time.Time  // When the key was created
    Disabled    bool       // Soft-disable without deleting
}
```

### Key Generation

```go
func GenerateKeyPair() (accessKeyID, secretKey string, err error) {
    // Access Key: "BDK_" + 16 random hex chars
    //   e.g., "BDK_a1b2c3d4e5f6g7h8"
    accessKeyID = "BDK_" + hex.EncodeToString(randomBytes(8))

    // Secret Key: "sk_" + 40 random hex chars
    //   e.g., "sk_0123456789abcdef0123456789abcdef01234567"
    secretKey = "sk_" + hex.EncodeToString(randomBytes(20))
}
```

The access key ID is **public** it identifies which key is being used.
The secret key is **private** it's used to sign requests and is **only shown once** at creation time.

### Key Functions

| Function                                                  | What it does                                      |
| --------------------------------------------------------- | ------------------------------------------------- |
| `CreateAPIKey(name, permissions, bucketScope, expiresIn)` | Generates keys, saves to DB, returns key + secret |
| `GetAPIKeyByAccessID(id)`                                 | Looks up a key, checks if disabled/expired        |
| `UpdateLastUsed(id)`                                      | Updates the `last_used_at` timestamp              |
| `ListAPIKeys()`                                           | Returns all keys (ordered by creation date)       |
| `DeleteAPIKey(id)`                                        | Hard-deletes a key from the database              |
| `DisableAPIKey(id)`                                       | Soft-disables a key (sets `disabled = true`)      |

---

<a name="15-keys-handler"></a>

## 15. API Key Management Handler (`beam/server/handlers/api/keys.go`)

This handler provides the HTTP API for managing API keys from the web UI or curl.

**Important:** This endpoint does NOT use API key auth it's protected by the web UI's session authentication instead. This makes sense because you need to be able to create your first API key without already having one.

### Endpoints

| Method   | URL                                | What it does                       |
| -------- | ---------------------------------- | ---------------------------------- |
| `GET`    | `/api/v1/keys`                     | List all keys (secrets are hidden) |
| `POST`   | `/api/v1/keys`                     | Create a new key                   |
| `DELETE` | `/api/v1/keys?accessKeyId=BDK_...` | Delete a key                       |

### Creating a Key Request

```json
POST /api/v1/keys
{
  "name": "My CI Pipeline",
  "permissions": "read,write",
  "bucketScope": "artifacts",
  "expiresIn": 2592000
}
```

- `name` required, human-readable label
- `permissions` optional, comma-separated permission list
- `bucketScope` optional, restricts key to a specific bucket
- `expiresIn` optional, seconds until the key expires (e.g., 2592000 = 30 days)

### Creating a Key Response

```json
{
  "id": 1,
  "name": "My CI Pipeline",
  "accessKeyId": "BDK_a1b2c3d4e5f6g7h8",
  "secretKey": "sk_0123456789abcdef0123456789abcdef01234567",
  "permissions": "read,write",
  "bucketScope": "artifacts",
  "expiresAt": "2026-03-14T10:30:00Z",
  "createdAt": "2026-02-12T10:30:00Z",
  "warning": "Save the secret key now. It cannot be retrieved later."
}
```

**Critical:** The `secretKey` is only shown in this response. If you lose it, you must create a new key.

### Listing Keys Response

```json
{
  "keys": [
    {
      "id": 1,
      "name": "My CI Pipeline",
      "accessKeyId": "BDK_a1b2c3d4e5f6g7h8",
      "permissions": "read,write",
      "bucketScope": "artifacts",
      "expiresAt": "2026-03-14T10:30:00Z",
      "lastUsedAt": "2026-02-12T15:00:00Z",
      "createdAt": "2026-02-12T10:30:00Z",
      "disabled": false
    }
  ],
  "count": 1
}
```

Notice: **no `secretKey`** in the list response. It's intentionally excluded.

---

<a name="16-cryptography"></a>

## 16. Cryptography & Signatures `pkg/crypto/signature.go`

This package handles all the signing and verification logic.

### Request Signing (HMAC-SHA256)

When a client makes an API request, it must prove it knows the secret key **without sending the secret key**. This is done with HMAC:

```
Client side:
    message = "PUT\n/api/v1/buckets/photos/pic.jpg\n2026-02-12T10:30:00Z"
    signature = HMAC-SHA256(secret_key, message)
    signature = base64_encode(signature)

    Send: Authorization: Bearer BDK_abc123:<signature>
    Send: X-Beamdrop-Date: 2026-02-12T10:30:00Z

Server side:
    1. Extract access_key_id and signature from header
    2. Look up secret_key from database using access_key_id
    3. Recompute: expected = HMAC-SHA256(secret_key, "PUT\n/path\ntimestamp")
    4. Compare: does expected == received signature?
```

### Presigned URL Tokens

Similar concept, but the token includes the expiration time instead of the request timestamp:

```go
message = "GET\nphotos\npic.jpg\n1707741600"  // Unix timestamp of expiration
token   = HMAC-SHA256(secret_key, message)
token   = base64url_encode(token)
```

The four fields in the message are: **HTTP method**, **bucket name**, **object key**, and **expiration as Unix timestamp**. All four are separated by `\n`. Changing any field invalidates the token.

**Key differences from request signatures:**

|                | Request Signature                                     | Presigned URL Token                              |
| -------------- | ----------------------------------------------------- | ------------------------------------------------ |
| **Sent via**   | `Authorization` header                                | `token` query parameter                          |
| **Time field** | Current timestamp (15-min validity window)            | Expiration timestamp (arbitrary future date)     |
| **Path field** | Full URL path (e.g. `/api/v1/buckets/photos/pic.jpg`) | Bucket + key separately (`photos` and `pic.jpg`) |
| **Encoding**   | Standard Base64                                       | URL-safe Base64 (no `+`, `/`, or `=` padding)    |
| **Use case**   | Server-to-server API calls                            | Shareable links for end users                    |

**Important limitations:**

- Client-side presigned URLs **always expire** the server hard-checks `time.Now().After(expiresAt)`. There is no bypass.
- If the API key is **deleted or disabled**, all client-side presigned URLs generated with that key become invalid immediately.
- Individual client-side presigned URLs **cannot be revoked** without disabling the entire API key.
- For individually revocable URLs, use the server-side pretty presigned URL registry (`POST /api/v1/presign` → `/dl/{token}`).
- For truly permanent public access, run without `-api-auth` or proxy files through your application.

### Timestamp Validation

```go
func IsTimestampValid(timestamp string) bool {
    diff := now - parsedTimestamp
    return diff >= -15min && diff <= 15min
}
```

Requests must have a timestamp within **15 minutes** of the server's clock. This prevents **replay attacks** someone can't capture a valid request and send it again hours later.

---

<a name="17-error-handling"></a>

## 17. Error Handling `pkg/errors/errors.go`

Beamdrop has a structured error system with **codes**, **categories**, and **HTTP status mapping**.

### Error Categories

| Category     | Description                           |
| ------------ | ------------------------------------- |
| `VALIDATION` | Bad input from the client             |
| `STORAGE`    | File system problems                  |
| `AUTH`       | Authentication/authorization failures |
| `NOT_FOUND`  | Resource doesn't exist                |
| `CONFLICT`   | Resource already exists               |
| `RATE_LIMIT` | Client exceeded per-IP request rate   |
| `INTERNAL`   | Server-side bugs                      |

### How Errors Flow

```go
// In the handler:
err := h.bucketManager.CreateBucket(name)
switch err {
case storage.ErrInvalidBucketName:
    errors.InvalidBucketName("...").WriteHTTPResponse(w)
case storage.ErrBucketExists:
    errors.BucketExists(name).WriteHTTPResponse(w)
default:
    errors.InternalError("...").WithCause(err).WriteHTTPResponse(w)
}

// Idempotent variant (with ?createIfNotExists=true):
created, err := h.bucketManager.CreateBucketIfNotExists(name)
// created=true → 201, created=false → 200 (bucket already existed)
```

Each error factory function (like `errors.BucketNotFound()`) creates a structured error with the right HTTP status code, error code, and human-readable message. `WriteHTTPResponse()` serializes it to JSON and sends it.

### Example Error Response

```json
{
  "error": {
    "code": "BUCKET_NOT_FOUND",
    "category": "NOT_FOUND",
    "message": "Bucket 'my-bucket' not found",
    "status": 404,
    "timestamp": "2026-02-12T10:30:00Z"
  }
}
```

### Rate Limit Error Response

Rate limit errors include extra fields to help clients implement retry logic:

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "category": "RATE_LIMIT",
    "message": "Rate limit exceeded. Try again in 3s",
    "status": 429,
    "retryable": true,
    "retryAfter": 3,
    "timestamp": "2026-02-12T10:30:00Z"
  }
}
```

The `Retry-After` HTTP header is also set, and `X-Retryable: true` signals that the client should retry after the indicated delay.

---

<a name="18-diagrams"></a>

## 18. How It All Connects Diagrams

### File Dependency Graph

```
beam/server/server.go
    ├── pkg/middleware/ratelimit.go               (per-IP rate limiting)
    ├── pkg/middleware/cors.go                    (CORS headers)
    ├── pkg/middleware/security.go                (security headers)
    ├── pkg/auth/middleware.go                    (session auth)
    └── beam/server/routes.go
         ├── beam/server/handlers/api/middleware.go  (API key auth)
         ├── beam/server/handlers/api/buckets.go     (bucket HTTP logic)
         │       └── pkg/storage/bucket.go           (bucket filesystem)
         ├── beam/server/handlers/api/objects.go      (object HTTP logic)
         │       ├── pkg/storage/object.go            (object filesystem)
         │       │       ├── pkg/storage/bucket.go    (reused for path resolution)
         │       │       ├── pkg/storage/atomic.go    (crash-safe writes)
         │       │       └── pkg/storage/locks.go     (per-object read/write locking)
         │       └── pkg/storage/bucket.go            (bucket existence check)
         └── beam/server/handlers/api/keys.go         (API key management)
                 └── pkg/db/api_keys.go               (database CRUD)

beam/server/handlers/shareable_links.go
    └── pkg/db/shareable_links.go                (shareable link CRUD)

beam/server/handlers/file_operations.go
    └── pkg/db/starred.go                        (starred files CRUD)

pkg/db/transaction.go                            (DB transaction wrapper)
pkg/db/saga.go                                   (saga pattern for DB+FS coordination)
pkg/db/cleanup.go                                (orphaned record cleanup job)

beam/server/handlers/api/middleware.go
    ├── pkg/crypto/signature.go                  (HMAC signing/verification)
    └── pkg/db/api_keys.go                       (key lookup)

pkg/middleware/ratelimit.go
    └── pkg/errors/errors.go                     (429 rate limit responses)

pkg/logger/logger.go                             (dual-output slog: terminal + JSON file)
pkg/errors/errors.go                             (used by all handlers)
```

### Request Flow Diagram

```
                           ┌──────────────┐
                           │   Client     │
                           │  (curl/app)  │
                           └──────┬───────┘
                                  │
                    PUT /api/v1/buckets/photos/pic.jpg
                    Authorization: Bearer BDK_xxx:sig
                    X-Beamdrop-Date: <timestamp>
                    Body: <file data>
                                  │
                           ┌──────▼───────┐
                           │  server.go   │
                           │  ServeHTTP() │
                           └──────┬───────┘
                                  │
                       ┌──────────▼──────────┐
                       │  Security + CORS    │
                       │  headers middleware │
                       └──────────┬──────────┘
                                  │
                       ┌──────────▼──────────┐
                       │  ratelimit.go       │
                       │  Check IP bucket    │
                       │  (upload tier)      │
                       └──────────┬──────────┘
                                  │  ✓ Tokens available
                           ┌──────▼───────┐
                           │  routes.go   │
                           │  URL match   │
                           └──────┬───────┘
                                  │
                           ┌──────▼──────────┐
                           │  middleware.go   │
                           │  Verify API key  │
                           │  Verify signature│
                           └──────┬──────────┘
                                  │  ✓ Authenticated
                           ┌──────▼──────────┐
                           │  objects.go      │
                           │  putObject()     │
                           └──────┬──────────┘
                                  │
                    ┌─────────────▼─────────────┐
                    │     pkg/storage/object.go  │
                    │     PutObject()            │
                    │                            │
                    │  1. Validate bucket & key  │
                    │  2. Build file path        │
                    │  3. Create directories     │
                    │  4. AtomicWriter.Write()   │
                    │  5. Compute MD5 (ETag)     │
                    │  6. AtomicWriter.Commit()  │
                    └─────────────┬─────────────┘
                                  │
                           ┌──────▼───────┐
                           │  Response    │
                           │  200 OK      │
                           │  {etag,size} │
                           └──────────────┘
```

---

<a name="19-example-upload"></a>

## 19. Example Walkthrough: Uploading a File

### Using curl (with auth disabled)

```bash
# 1. Create a bucket
curl -X PUT http://localhost:7777/api/v1/buckets/photos

# 2. Upload a file (raw body)
curl -X PUT \
  --data-binary @beach.jpg \
  http://localhost:7777/api/v1/buckets/photos/vacation/beach.jpg

# 3. Upload a file (multipart form)
curl -X POST \
  -F "file=@beach.jpg" \
  http://localhost:7777/api/v1/buckets/photos/vacation/beach.jpg
```

### Using curl (with auth enabled)

```bash
# 1. Create an API key (via web UI or session-authenticated request)
curl -X POST http://localhost:7777/api/v1/keys \
  -H "Content-Type: application/json" \
  -d '{"name": "my-key"}'

# Response: { "accessKeyId": "BDK_abc123", "secretKey": "sk_xyz789" }

# 2. Sign and upload
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
METHOD="PUT"
PATH="/api/v1/buckets/photos/vacation/beach.jpg"

# Compute signature: HMAC-SHA256 of "METHOD\nPATH\nTIMESTAMP"
SIGNATURE=$(echo -n "$METHOD\n$PATH\n$TIMESTAMP" \
  | openssl dgst -sha256 -hmac "sk_xyz789" -binary \
  | base64)

curl -X PUT \
  -H "Authorization: Bearer BDK_abc123:$SIGNATURE" \
  -H "X-Beamdrop-Date: $TIMESTAMP" \
  --data-binary @beach.jpg \
  http://localhost:7777/api/v1/buckets/photos/vacation/beach.jpg
```

### What Happens on Disk

Before upload:

```
shared_dir/buckets/photos/          ← bucket directory exists
```

After upload:

```
shared_dir/buckets/photos/
└── vacation/
    └── beach.jpg                   ← your file!
```

The `vacation/` directory was auto-created by `os.MkdirAll` in `PutObject()`.

---

<a name="20-example-download"></a>

## 20. Example Walkthrough: Downloading a File

```bash
# Download the file
curl http://localhost:7777/api/v1/buckets/photos/vacation/beach.jpg \
  --output beach.jpg

# Just check if it exists (HEAD request)
curl -I http://localhost:7777/api/v1/buckets/photos/vacation/beach.jpg

# List all files in the bucket
curl http://localhost:7777/api/v1/buckets/photos

# List files with a prefix filter
curl "http://localhost:7777/api/v1/buckets/photos?prefix=vacation/&delimiter=/"
```

---

<a name="21-glossary"></a>

## 21. Glossary

| Term                   | Meaning                                                                                                                                                                                                                                                                                                   |
| ---------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Bucket**             | A top-level container (directory) for objects. Like a folder.                                                                                                                                                                                                                                             |
| **Object**             | A file stored inside a bucket.                                                                                                                                                                                                                                                                            |
| **Key**                | The path of an object within its bucket (e.g., `images/photo.jpg`).                                                                                                                                                                                                                                       |
| **Prefix**             | A key prefix used for filtering (e.g., `images/` to list all images).                                                                                                                                                                                                                                     |
| **Delimiter**          | A character (usually `/`) used to group keys into virtual directories.                                                                                                                                                                                                                                    |
| **ETag**               | MD5 hash of the file content, used for change detection.                                                                                                                                                                                                                                                  |
| **Access Key ID**      | Public identifier for an API key (starts with `BDK_`).                                                                                                                                                                                                                                                    |
| **Secret Key**         | Private key used for signing requests (starts with `sk_`). Shown once.                                                                                                                                                                                                                                    |
| **HMAC-SHA256**        | A cryptographic algorithm that proves you know a secret without revealing it.                                                                                                                                                                                                                             |
| **Presigned URL**      | A temporary URL with auth baked in, so anyone with the link can access the file. Beamdrop supports two types: (1) client-side HMAC URLs computed locally (long, with query params) and (2) server-side pretty URLs created via `POST /api/v1/presign` (short `/dl/{token}` URLs, individually revocable). |
| **Atomic Write**       | A write strategy that prevents corrupted files by using temp file + rename.                                                                                                                                                                                                                               |
| **File-Level Lock**    | A per-object read/write lock that prevents concurrent writes to the same key.                                                                                                                                                                                                                             |
| **Write Lock**         | An exclusive lock only one goroutine can hold it at a time. Used for uploads and deletes.                                                                                                                                                                                                                 |
| **Read Lock**          | A shared lock multiple goroutines can hold it simultaneously. Used for downloads and HEAD requests.                                                                                                                                                                                                       |
| **Lock Timeout**       | Max time to wait for a lock (default 30s). Returns HTTP 423 if exceeded.                                                                                                                                                                                                                                  |
| **Deadlock Detection** | Background scan that warns when a write lock is held too long (>5 min).                                                                                                                                                                                                                                   |
| **Transaction**        | A group of DB operations that either all succeed or all roll back. Uses `WithTransaction()`.                                                                                                                                                                                                              |
| **Saga**               | A pattern for coordinating operations across different systems (DB + filesystem) with compensating rollbacks.                                                                                                                                                                                             |
| **Compensation**       | The undo action for a saga step e.g., deleting a DB record if the subsequent file write fails.                                                                                                                                                                                                            |
| **Orphan Cleaner**     | Background job that removes DB records pointing to files that no longer exist on disk.                                                                                                                                                                                                                    |
| **Sentinel Error**     | A named error variable (e.g., `ErrBucketNotFound`) used for error comparison.                                                                                                                                                                                                                             |
| **Mux**                | HTTP request multiplexer matches URLs to handler functions.                                                                                                                                                                                                                                               |
| **Token Bucket**       | A rate limiting algorithm where tokens refill at a steady rate and each request consumes one token.                                                                                                                                                                                                       |
| **Rate Limiting Tier** | One of three endpoint categories (general/auth/upload) with different request-per-minute limits.                                                                                                                                                                                                          |
| **Retry-After**        | HTTP header telling a rate-limited client how many seconds to wait before retrying.                                                                                                                                                                                                                       |
| **slog**               | Go's standard structured logging package (`log/slog`). Beamdrop uses it for all log output.                                                                                                                                                                                                               |
| **Structured Logging** | Log events as key-value pairs instead of free-form text, enabling machine parsing and querying.                                                                                                                                                                                                           |

---

## Quick Reference: All S3 API Endpoints

All endpoints are subject to per-IP rate limiting when enabled (see [Section 5](#5-rate-limiting)). Upload endpoints (`PUT` object) use the stricter upload tier.

| Method   | Endpoint                                          | Rate Tier | Description                       |
| -------- | ------------------------------------------------- | --------- | --------------------------------- |
| `GET`    | `/api/v1/buckets`                                 | General   | List all buckets                  |
| `PUT`    | `/api/v1/buckets/{bucket}`                        | General   | Create a bucket                   |
| `PUT`    | `/api/v1/buckets/{bucket}?createIfNotExists=true` | General   | Create a bucket (idempotent)      |
| `GET`    | `/api/v1/buckets/{bucket}`                        | General   | Get bucket info / list objects    |
| `HEAD`   | `/api/v1/buckets/{bucket}`                        | General   | Check if bucket exists            |
| `DELETE` | `/api/v1/buckets/{bucket}`                        | General   | Delete an empty bucket            |
| `GET`    | `/api/v1/buckets/{bucket}/{key}`                  | General   | Download an object                |
| `PUT`    | `/api/v1/buckets/{bucket}/{key}`                  | Upload    | Upload an object (raw body)       |
| `POST`   | `/api/v1/buckets/{bucket}/{key}`                  | Upload    | Upload an object (multipart form) |
| `HEAD`   | `/api/v1/buckets/{bucket}/{key}`                  | General   | Get object metadata               |
| `DELETE` | `/api/v1/buckets/{bucket}/{key}`                  | General   | Delete an object                  |
| `GET`    | `/api/v1/keys`                                    | General   | List API keys                     |
| `POST`   | `/api/v1/keys`                                    | General   | Create an API key                 |
| `DELETE` | `/api/v1/keys?accessKeyId=...`                    | General   | Delete an API key                 |
