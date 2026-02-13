# Beamdrop S3 API — Deep Code Walkthrough

A plain-English, file-by-file explanation of how the S3-compatible API works inside beamdrop, so you can read the code with confidence.

---

## Table of Contents

1. [Big Picture — How Everything Fits Together](#1-big-picture)
2. [How a Request Travels Through the Code](#2-request-lifecycle)
3. [Route Registration (`beam/server/routes.go`)](#3-route-registration)
4. [Authentication Middleware (`beam/server/handlers/api/middleware.go`)](#4-authentication-middleware)
5. [Bucket Handler (`beam/server/handlers/api/buckets.go`)](#5-bucket-handler)
6. [Object Handler (`beam/server/handlers/api/objects.go`)](#6-object-handler)
7. [Storage Layer — BucketManager (`pkg/storage/bucket.go`)](#7-bucket-manager)
8. [Storage Layer — ObjectManager (`pkg/storage/object.go`)](#8-object-manager)
9. [Atomic Writes (`pkg/storage/atomic.go`)](#9-atomic-writes)
10. [File-Level Locking (`pkg/storage/locks.go`)](#10-file-locking)
11. [API Key Management — Database (`pkg/db/api_keys.go`)](#11-api-key-database)
12. [API Key Management — Handler (`beam/server/handlers/api/keys.go`)](#12-keys-handler)
13. [Cryptography & Signatures (`pkg/crypto/signature.go`)](#13-cryptography)
14. [Error Handling (`pkg/errors/errors.go`)](#14-error-handling)
15. [How It All Connects — Diagrams](#15-diagrams)
16. [Example Walkthrough: Uploading a File](#16-example-upload)
17. [Example Walkthrough: Downloading a File](#17-example-download)
18. [Glossary](#18-glossary)

---

<a name="1-big-picture"></a>
## 1. Big Picture — How Everything Fits Together

Beamdrop's S3 API lets you manage files programmatically using **buckets** (like folders) and **objects** (files inside those folders). It mimics the concepts from Amazon S3 but is backed by your **local filesystem** instead of cloud storage.

### The Four Layers

```
┌─────────────────────────────────────────────────┐
│  HTTP Layer  (routes.go)                        │
│  Receives HTTP requests, routes to handlers     │
├─────────────────────────────────────────────────┤
│  Auth Layer  (api/middleware.go)                │
│  Validates API keys and request signatures      │
├─────────────────────────────────────────────────┤
│  Handler Layer  (api/buckets.go, objects.go)    │
│  Business logic — what to do with the request   │
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
              (beam/server/server.go — ServeHTTP method)

Step 2  →  Route matching
              (beam/server/routes.go — matches "/api/v1/buckets/")

Step 3  →  API Auth Middleware runs
              (api/middleware.go — checks API key + signature)

Step 4  →  Router decides: bucket or object handler?
              Path has "photos/vacation/beach.jpg"
              → parts[0] = "photos" (bucket)
              → parts[1] = "vacation/beach.jpg" (key, non-empty)
              → Routes to ObjectHandler

Step 5  →  ObjectHandler.Handle() dispatches by HTTP method
              Method is PUT → calls putObject()

Step 6  →  putObject() uses ObjectManager.PutObject()
              (pkg/storage/object.go)

Step 7  →  ObjectManager acquires a write lock
              (pkg/storage/locks.go — per-object locking)

Step 8  →  ObjectManager validates names, creates dirs,
              uses AtomicWriter to write the file safely

Step 9  →  Write lock released, response sent back
              with ETag, size, URL
```

---

<a name="3-route-registration"></a>
## 3. Route Registration — `beam/server/routes.go`

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
1. Creates three handlers — one for buckets, one for objects, one for API keys
2. Creates the auth middleware (enabled/disabled by the `-api-auth` command-line flag)
3. Registers URL patterns on the HTTP mux

### URL Patterns

| Pattern | What it matches |
|---------|----------------|
| `/api/v1/buckets` | List all buckets (no trailing slash) |
| `/api/v1/buckets/` | Anything under buckets — the router inspects the rest of the path |
| `/api/v1/keys` | Manage API keys |

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
## 4. Authentication Middleware — `beam/server/handlers/api/middleware.go`

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

Presigned URLs let you share a temporary download/upload link without revealing your API key. The middleware:

1. Checks if the URL has expired
2. Extracts the bucket/key from the path
3. Looks up the API key
4. Verifies the token using `crypto.VerifyPresignedToken()`

#### What If Auth Is Disabled?

```go
if !m.enabled {
    next.ServeHTTP(w, r)  // Just pass the request through
    return
}
```

When you start beamdrop **without** the `-api-auth` flag, all API requests go straight through. This is handy for development.

---

<a name="5-bucket-handler"></a>
## 5. Bucket Handler — `beam/server/handlers/api/buckets.go`

Manages buckets (creating, listing, deleting directories).

### Structure

```go
type BucketHandler struct {
    bucketManager *storage.BucketManager  // Does the actual filesystem work
}
```

### The `Handle()` Method — Request Router

```go
func (h *BucketHandler) Handle(w http.ResponseWriter, r *http.Request) {
    // Extract bucket name from URL
    bucketName := ... // e.g., "my-bucket" from "/api/v1/buckets/my-bucket"

    switch r.Method {
    case GET:    → listBuckets() or getBucketInfo()
    case PUT:    → createBucket()
    case DELETE: → deleteBucket()
    case HEAD:   → headBucket()
    }
}
```

### Operations Explained

| Operation | HTTP | URL | What it does |
|-----------|------|-----|--------------|
| List all buckets | `GET` | `/api/v1/buckets` | Returns all bucket names + count |
| Create bucket | `PUT` | `/api/v1/buckets/my-bucket` | Creates directory on disk |
| Delete bucket | `DELETE` | `/api/v1/buckets/my-bucket` | Removes directory (must be empty) |
| Check bucket exists | `HEAD` | `/api/v1/buckets/my-bucket` | Returns 200 or 404 (no body) |
| Get bucket info | `GET` | `/api/v1/buckets/my-bucket` | Returns `{bucket, exists}` |

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

<a name="6-object-handler"></a>
## 6. Object Handler — `beam/server/handlers/api/objects.go`

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

| Operation | HTTP | URL | What it does |
|-----------|------|-----|--------------|
| Download file | `GET` | `/api/v1/buckets/photos/pic.jpg` | Streams file content back |
| Upload (raw body) | `PUT` | `/api/v1/buckets/photos/pic.jpg` | Reads request body, saves to disk |
| Upload (form) | `POST` | `/api/v1/buckets/photos/pic.jpg` | Reads multipart form data |
| Delete file | `DELETE` | `/api/v1/buckets/photos/pic.jpg` | Removes the file |
| Get file info | `HEAD` | `/api/v1/buckets/photos/pic.jpg` | Returns headers only (size, type) |
| List files | `GET` | `/api/v1/buckets/photos` | Lists objects, supports prefix/delimiter |

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

<a name="7-bucket-manager"></a>
## 7. Storage Layer — BucketManager (`pkg/storage/bucket.go`)

This is the **lowest-level** code that touches the filesystem for bucket operations. The handlers never write files directly — they always go through these managers.

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
- `my-bucket` — valid
- `My-Bucket` — invalid (uppercase)
- `ab` — invalid (too short)
- `192.168.1.1` — invalid (looks like IP)

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
- `../../etc/passwd` — path traversal attack
- `/root/.ssh/id_rsa` — absolute path escape

### Key Methods

| Method | What it does |
|--------|-------------|
| `CreateBucket(name)` | Validates name, checks if exists, creates directory |
| `DeleteBucket(name)` | Validates name, checks if empty, removes directory |
| `BucketExists(name)` | Validates name, checks if directory exists |
| `ListBuckets()` | Reads all directories, returns `[]BucketInfo` |
| `GetBucketPath(name)` | Returns filesystem path (e.g., `/shared/buckets/my-bucket`) |
| `EnsureBucketsDir()` | Creates the `buckets/` directory if it doesn't exist |

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

<a name="8-object-manager"></a>
## 8. Storage Layer — ObjectManager (`pkg/storage/object.go`)

Handles file read/write operations. Uses `BucketManager` internally to resolve paths and check bucket existence.

### Structure

```go
type ObjectManager struct {
    bucketManager *BucketManager
    LockManager   *LockManager   // Per-object read/write locking
}
```

The `LockManager` is created automatically in `NewObjectManager()` and provides file-level locking for all object operations (see [Section 10](#10-file-locking) for details).

### `PutObject()` — Writing a File

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

**Concurrency note:** Step 3 ensures that if two clients upload to the same key simultaneously, uploads are **serialized** — one waits for the other to finish. Uploads to *different* keys proceed in parallel without blocking.

### `GetObject()` — Reading a File

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

The caller (the handler) is responsible for **closing the file** and **calling the unlock function** after streaming it to the client. Multiple readers can read the same file concurrently — the read lock is shared.

### `ListObjects()` — Listing Files

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

### ObjectInfo — The Metadata Struct

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

<a name="9-atomic-writes"></a>
## 9. Atomic Writes — `pkg/storage/atomic.go`

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

**Why rename is atomic:** On POSIX systems (Linux, macOS), `os.Rename` within the same filesystem is a single atomic operation. At no point is the file in a half-written state. Either you see the old file or the new file — never something in between.

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

<a name="10-file-locking"></a>
## 10. File-Level Locking — `pkg/storage/locks.go`

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

Uploads to **different** keys are fully independent — `Lock("photos", "pic.jpg")` and `Lock("photos", "other.jpg")` never block each other.

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

| Operation | Lock Type | Behavior |
|-----------|-----------|----------|
| `PutObject` | **Write** (exclusive) | Only one writer at a time per key |
| `DeleteObject` | **Write** (exclusive) | Only one writer at a time per key |
| `GetObject` | **Read** (shared) | Multiple concurrent readers allowed |
| `HeadObject` | **Read** (shared) | Multiple concurrent readers allowed |
| `ListObjects` | **None** | Directory scan, no per-object lock needed |

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
WARN: Potential deadlock: write lock on "photos/pic.jpg" held for 5m30s (threshold 5m0s)
```

This helps operators identify stuck uploads or crashes that left a lock in a bad state. The thresholds:

| Constant | Default | Meaning |
|----------|---------|----------|
| `DefaultLockTimeout` | 30s | Max wait time to acquire a lock |
| `DeadlockDetectionInterval` | 10s | How often the detector scans |
| `StaleLockerThreshold` | 5min | How long before a held lock is "suspicious" |

### Lock Stats

```go
stats := objectManager.LockManager.Stats()
// stats.ActiveLocks — number of keys with active locks
// stats.WriteLocks  — number of write locks currently held
// stats.ReadLocks   — number of read locks currently held
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

<a name="11-api-key-database"></a>
## 11. API Key Management — Database (`pkg/db/api_keys.go`)

API keys are stored in a SQLite database using GORM (a Go ORM).

### The APIKey Model

```go
type APIKey struct {
    ID          uint       // Auto-incrementing primary key
    Name        string     // Human-readable name, e.g., "CI/CD Pipeline"
    AccessKeyID string     // Public identifier: "BDK_a1b2c3d4e5f6g7h8"
    SecretKey   string     // Secret for HMAC signing (stored in DB for verification)
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

The access key ID is **public** — it identifies which key is being used.
The secret key is **private** — it's used to sign requests and is **only shown once** at creation time.

### Key Functions

| Function | What it does |
|----------|-------------|
| `CreateAPIKey(name, permissions, bucketScope, expiresIn)` | Generates keys, saves to DB, returns key + secret |
| `GetAPIKeyByAccessID(id)` | Looks up a key, checks if disabled/expired |
| `UpdateLastUsed(id)` | Updates the `last_used_at` timestamp |
| `ListAPIKeys()` | Returns all keys (ordered by creation date) |
| `DeleteAPIKey(id)` | Hard-deletes a key from the database |
| `DisableAPIKey(id)` | Soft-disables a key (sets `disabled = true`) |

---

<a name="12-keys-handler"></a>
## 12. API Key Management — Handler (`beam/server/handlers/api/keys.go`)

This handler provides the HTTP API for managing API keys from the web UI or curl.

**Important:** This endpoint does NOT use API key auth — it's protected by the web UI's session authentication instead. This makes sense because you need to be able to create your first API key without already having one.

### Endpoints

| Method | URL | What it does |
|--------|-----|-------------|
| `GET` | `/api/v1/keys` | List all keys (secrets are hidden) |
| `POST` | `/api/v1/keys` | Create a new key |
| `DELETE` | `/api/v1/keys?accessKeyId=BDK_...` | Delete a key |

### Creating a Key — Request

```json
POST /api/v1/keys
{
  "name": "My CI Pipeline",
  "permissions": "read,write",
  "bucketScope": "artifacts",
  "expiresIn": 2592000
}
```

- `name` — required, human-readable label
- `permissions` — optional, comma-separated permission list
- `bucketScope` — optional, restricts key to a specific bucket
- `expiresIn` — optional, seconds until the key expires (e.g., 2592000 = 30 days)

### Creating a Key — Response

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

### Listing Keys — Response

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

<a name="13-cryptography"></a>
## 13. Cryptography & Signatures — `pkg/crypto/signature.go`

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

Similar concept, but the token includes the expiration time:

```go
message = "GET\nphotos\npic.jpg\n1707741600"  // Unix timestamp
token   = HMAC-SHA256(secret_key, message)
token   = base64url_encode(token)
```

### Timestamp Validation

```go
func IsTimestampValid(timestamp string) bool {
    diff := now - parsedTimestamp
    return diff >= -15min && diff <= 15min
}
```

Requests must have a timestamp within **15 minutes** of the server's clock. This prevents **replay attacks** — someone can't capture a valid request and send it again hours later.

---

<a name="14-error-handling"></a>
## 14. Error Handling — `pkg/errors/errors.go`

Beamdrop has a structured error system with **codes**, **categories**, and **HTTP status mapping**.

### Error Categories

| Category | Description |
|----------|-------------|
| `VALIDATION` | Bad input from the client |
| `STORAGE` | File system problems |
| `AUTH` | Authentication/authorization failures |
| `NOT_FOUND` | Resource doesn't exist |
| `CONFLICT` | Resource already exists |
| `INTERNAL` | Server-side bugs |

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

---

<a name="15-diagrams"></a>
## 15. How It All Connects — Diagrams

### File Dependency Graph

```
beam/server/routes.go
    ├── beam/server/handlers/api/middleware.go  (auth)
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

beam/server/handlers/api/middleware.go
    ├── pkg/crypto/signature.go                  (HMAC signing/verification)
    └── pkg/db/api_keys.go                       (key lookup)

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

<a name="16-example-upload"></a>
## 16. Example Walkthrough: Uploading a File

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

<a name="17-example-download"></a>
## 17. Example Walkthrough: Downloading a File

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

<a name="18-glossary"></a>
## 18. Glossary

| Term | Meaning |
|------|---------|
| **Bucket** | A top-level container (directory) for objects. Like a folder. |
| **Object** | A file stored inside a bucket. |
| **Key** | The path of an object within its bucket (e.g., `images/photo.jpg`). |
| **Prefix** | A key prefix used for filtering (e.g., `images/` to list all images). |
| **Delimiter** | A character (usually `/`) used to group keys into virtual directories. |
| **ETag** | MD5 hash of the file content, used for change detection. |
| **Access Key ID** | Public identifier for an API key (starts with `BDK_`). |
| **Secret Key** | Private key used for signing requests (starts with `sk_`). Shown once. |
| **HMAC-SHA256** | A cryptographic algorithm that proves you know a secret without revealing it. |
| **Presigned URL** | A temporary URL with auth baked in, so anyone with the link can access the file. |
| **Atomic Write** | A write strategy that prevents corrupted files by using temp file + rename. |
| **File-Level Lock** | A per-object read/write lock that prevents concurrent writes to the same key. |
| **Write Lock** | An exclusive lock — only one goroutine can hold it at a time. Used for uploads and deletes. |
| **Read Lock** | A shared lock — multiple goroutines can hold it simultaneously. Used for downloads and HEAD requests. |
| **Lock Timeout** | Max time to wait for a lock (default 30s). Returns HTTP 423 if exceeded. |
| **Deadlock Detection** | Background scan that warns when a write lock is held too long (>5 min). |
| **Sentinel Error** | A named error variable (e.g., `ErrBucketNotFound`) used for error comparison. |
| **Mux** | HTTP request multiplexer — matches URLs to handler functions. |

---

## Quick Reference: All S3 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/buckets` | List all buckets |
| `PUT` | `/api/v1/buckets/{bucket}` | Create a bucket |
| `GET` | `/api/v1/buckets/{bucket}` | Get bucket info / list objects |
| `HEAD` | `/api/v1/buckets/{bucket}` | Check if bucket exists |
| `DELETE` | `/api/v1/buckets/{bucket}` | Delete an empty bucket |
| `GET` | `/api/v1/buckets/{bucket}/{key}` | Download an object |
| `PUT` | `/api/v1/buckets/{bucket}/{key}` | Upload an object (raw body) |
| `POST` | `/api/v1/buckets/{bucket}/{key}` | Upload an object (multipart form) |
| `HEAD` | `/api/v1/buckets/{bucket}/{key}` | Get object metadata |
| `DELETE` | `/api/v1/buckets/{bucket}/{key}` | Delete an object |
| `GET` | `/api/v1/keys` | List API keys |
| `POST` | `/api/v1/keys` | Create an API key |
| `DELETE` | `/api/v1/keys?accessKeyId=...` | Delete an API key |
