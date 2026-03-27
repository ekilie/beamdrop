# Beamdrop S3-Compatible API Design

## Overview

Transform beamdrop into a lightweight, self-hosted S3-compatible file storage server. This enables programmatic file uploads/downloads via API, making it useful for:

- Application file storage backend
- CI/CD artifact storage
- Backup destination
- Development/testing S3 alternative

## Core Concepts

### Storage Hierarchy

```
shared_directory/
├── .beamdrop_data/          # Internal data (existing trash, db, etc.)
├── .beamdrop_trash/         # Trash bin (existing)
└── buckets/                 # API-managed storage
    ├── my-app-uploads/      # Bucket (directory)
    │   ├── images/          # Prefix (subdirectory)
    │   │   └── photo.jpg    # Object (file)
    │   └── documents/
    │       └── report.pdf
    └── backups/
        └── db-2026-02-09.sql
```

### Terminology Mapping

| S3 Term | Beamdrop Term | Implementation                   |
| ------- | ------------- | -------------------------------- |
| Bucket  | Bucket        | Directory under `buckets/`       |
| Object  | Object/File   | File within bucket directory     |
| Key     | Object Key    | Relative file path within bucket |
| Prefix  | Prefix        | Directory path within bucket     |

---

## Authentication & Authorization

### API Keys

```go
// pkg/db/api_keys.go
type APIKey struct {
    ID          uint      `gorm:"primaryKey"`
    Name        string    `gorm:"not null"`                    // Human-readable name
    AccessKeyID string    `gorm:"uniqueIndex;size:20;not null"` // BDK_XXXXXXXXXXXX
    SecretKey   string    `gorm:"size:64;not null"`            // Hashed
    Permissions []byte    `gorm:"type:json"`                   // JSON permissions
    BucketScope string    `gorm:"size:255"`                    // Optional: limit to specific bucket
    ExpiresAt   *time.Time                                     // Optional expiration
    LastUsedAt  *time.Time
    CreatedAt   time.Time
    Disabled    bool      `gorm:"default:false"`
}

type Permission struct {
    Actions  []string `json:"actions"`  // ["GetObject", "PutObject", "DeleteObject", "ListBucket", "*"]
    Resource string   `json:"resource"` // "bucket-name/*" or "*"
}
```

### Authentication Methods

#### 1. Header-Based (Recommended)

```http
GET /api/v1/buckets/my-bucket/file.txt HTTP/1.1
Authorization: Bearer <access_key_id>:<signature>
X-Beamdrop-Date: 2026-02-09T12:00:00Z
```

#### 2. Query String (Presigned URLs)

```
/api/v1/buckets/my-bucket/file.txt?token=BASE64URL_TOKEN&expires=2026-02-09T13:00:00Z&access_key=BDK_xxx
```

Presigned URLs embed authentication into the URL itself. The `token` is generated client-side using HMAC-SHA256. See the [Presigned URLs](#presigned-urls) section below for full details.

Alternatively, you can use the server-side presigned URL registry to create short `/dl/{token}` URLs. See [Server-Side Pretty Presigned URLs](#server-side-pretty-presigned-urls-url-registry).

### Signature Generation (Simplified HMAC)

```go
// Client-side signature generation
func GenerateSignature(secretKey, method, path, timestamp string) string {
    message := fmt.Sprintf("%s\n%s\n%s", method, path, timestamp)
    h := hmac.New(sha256.New, []byte(secretKey))
    h.Write([]byte(message))
    return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
```

---

## API Endpoints

### Base URL

```
/api/v1/
```

### Bucket Operations

#### Create Bucket

```http
PUT /api/v1/buckets/{bucket-name}
Authorization: Bearer <credentials>
Content-Type: application/json

{
    "versioning": false,        // Optional: enable versioning
    "maxSizeBytes": 10737418240 // Optional: 10GB limit
}
```

**Response:**

```json
{
  "bucket": "my-bucket",
  "created": "2026-02-09T12:00:00Z",
  "location": "/api/v1/buckets/my-bucket"
}
```

#### List Buckets

```http
GET /api/v1/buckets
Authorization: Bearer <credentials>
```

**Response:**

```json
{
  "buckets": [
    {
      "name": "my-bucket",
      "createdAt": "2026-02-09T12:00:00Z",
      "objectCount": 42,
      "totalSize": "1.2 GB"
    }
  ],
  "count": 1
}
```

#### Delete Bucket

```http
DELETE /api/v1/buckets/{bucket-name}
Authorization: Bearer <credentials>
```

#### Get Bucket Info

```http
HEAD /api/v1/buckets/{bucket-name}
Authorization: Bearer <credentials>
```

---

### Object Operations

#### Upload Object (PUT)

```http
PUT /api/v1/buckets/{bucket}/{key}
Authorization: Bearer <credentials>
Content-Type: application/octet-stream
Content-Length: 1048576
X-Beamdrop-Meta-Custom: value

<binary data>
```

**Response:**

```json
{
  "bucket": "my-bucket",
  "key": "images/photo.jpg",
  "etag": "d41d8cd98f00b204e9800998ecf8427e",
  "size": 1048576,
  "url": "/api/v1/buckets/my-bucket/images/photo.jpg"
}
```

#### Upload Object (Multipart Form)

```http
POST /api/v1/buckets/{bucket}/{key}
Authorization: Bearer <credentials>
Content-Type: multipart/form-data

file=@photo.jpg
```

#### Download Object

```http
GET /api/v1/buckets/{bucket}/{key}
Authorization: Bearer <credentials>
Range: bytes=0-1023  # Optional: partial download
```

#### Delete Object

```http
DELETE /api/v1/buckets/{bucket}/{key}
Authorization: Bearer <credentials>
```

#### Get Object Metadata

```http
HEAD /api/v1/buckets/{bucket}/{key}
Authorization: Bearer <credentials>
```

**Response Headers:**

```
Content-Length: 1048576
Content-Type: image/jpeg
ETag: "d41d8cd98f00b204e9800998ecf8427e"
Last-Modified: Sun, 09 Feb 2026 12:00:00 GMT
X-Beamdrop-Meta-Custom: value
```

#### List Objects

```http
GET /api/v1/buckets/{bucket}?prefix=images/&delimiter=/&max-keys=100&continuation-token=xxx
Authorization: Bearer <credentials>
```

**Response:**

```json
{
  "bucket": "my-bucket",
  "prefix": "images/",
  "delimiter": "/",
  "maxKeys": 100,
  "isTruncated": false,
  "contents": [
    {
      "key": "images/photo1.jpg",
      "size": 1048576,
      "lastModified": "2026-02-09T12:00:00Z",
      "etag": "d41d8cd98f00b204e9800998ecf8427e"
    }
  ],
  "commonPrefixes": [{ "prefix": "images/thumbnails/" }]
}
```

---

### Presigned URLs

Presigned URLs let you share a time-limited, self-contained download (or upload) link. The URL embeds all authentication info the recipient does not need an API key.

**Presigned URLs are generated client-side**, not via a server endpoint. The server only verifies them on access.

#### How to Generate a Presigned URL

```
Step 1: Choose an expiry time → expiresAt = now + duration (as Unix timestamp)
Step 2: Build the HMAC message → "METHOD\nBUCKET\nKEY\nUNIX_TIMESTAMP"
Step 3: Compute the token     → Base64URL(HMAC-SHA256(secret_key, message))
Step 4: Build the URL          → /api/v1/buckets/{bucket}/{key}?token=TOKEN&expires=ISO8601&access_key=BDK_xxx
```

#### Token Generation (All Languages)

**Go:**

```go
message := fmt.Sprintf("%s\n%s\n%s\n%d", method, bucket, key, expiresAt.Unix())
h := hmac.New(sha256.New, []byte(secretKey))
h.Write([]byte(message))
token := base64.URLEncoding.EncodeToString(h.Sum(nil))
```

**Python:**

```python
import hmac, hashlib, base64
message = f"{method}\n{bucket}\n{key}\n{int(expires_at.timestamp())}"
token = base64.urlsafe_b64encode(
    hmac.new(secret_key.encode(), message.encode(), hashlib.sha256).digest()
).rstrip(b"=").decode()
```

**PHP:**

```php
$message = implode("\n", [$method, $bucket, $key, (string) $expiresAtUnix]);
$token = rtrim(strtr(base64_encode(
    hash_hmac('sha256', $message, $secretKey, binary: true)
), '+/', '-_'), '=');
```

**JavaScript / Node.js:**

```javascript
const crypto = require("crypto");
const message = [method, bucket, key, Math.floor(expiresAt / 1000)].join("\n");
const token = crypto
  .createHmac("sha256", secretKey)
  .update(message)
  .digest("base64url");
```

**cURL (bash):**

```bash
EXPIRES_UNIX=$(date -d '+1 hour' +%s)
MESSAGE="GET\n${BUCKET}\n${KEY}\n${EXPIRES_UNIX}"
TOKEN=$(printf '%s' "$MESSAGE" | openssl dgst -sha256 -hmac "$SECRET_KEY" -binary | base64 | tr '+/' '-_' | tr -d '=')
curl "http://server:8090/api/v1/buckets/${BUCKET}/${KEY}?token=${TOKEN}&expires=$(date -u -d@${EXPIRES_UNIX} +%Y-%m-%dT%H:%M:%SZ)&access_key=${ACCESS_KEY}"
```

#### Query Parameters

| Parameter    | Required | Description                                                                                |
| ------------ | -------- | ------------------------------------------------------------------------------------------ |
| `token`      | Yes      | Base64URL-encoded HMAC-SHA256 token                                                        |
| `expires`    | Yes      | Expiration timestamp RFC 3339 (`2026-02-09T13:00:00Z`) or compact ISO (`20260209T130000Z`) |
| `access_key` | Yes      | Access Key ID (`BDK_xxx`) used to generate the token                                       |

#### Server Verification Flow

1. Check `expires` if `time.Now()` is past the timestamp, reject with 403
2. Extract bucket and key from the URL path
3. Look up the API key by `access_key`
4. Recompute the token using the stored secret key
5. Constant-time comparison (`hmac.Equal`) of computed vs. provided token

#### Important Behaviors & Limitations

| Behavior                     | Detail                                                                                         |
| ---------------------------- | ---------------------------------------------------------------------------------------------- |
| **Always expires**           | There is no "permanent" presigned URL. The server always checks `time.Now().After(expiresAt)`. |
| **Tied to API key**          | Deleting, disabling, or rotating the API key invalidates all presigned URLs generated with it. |
| **Method-specific**          | A token for `GET` will not work for `PUT` the method is part of the signed message.            |
| **Key-specific**             | A token for `photos/pic.jpg` will not work for `photos/other.jpg`.                             |
| **No maximum expiry**        | You can set expiry to year 2100, but URLs break on key rotation.                               |
| **No individual revocation** | To revoke a URL, delete the object or disable the API key.                                     |

#### Suggested Expiry Durations

| Use case                             | Duration                                    |
| ------------------------------------ | ------------------------------------------- |
| One-time download link (email, chat) | 1–24 hours                                  |
| Embedded in web page (avatars)       | 1–7 days                                    |
| Client portal / invoice download     | 7–30 days                                   |
| Semi-permanent asset                 | 1–10 years (fragile breaks on key rotation) |

#### Alternatives to Presigned URLs for Permanent Access

- **Disable `-api-auth`** all reads are public, no signing needed
- **Proxy through your app** your backend fetches from Beamdrop and streams to the client
- **Use shareable links** Beamdrop's `/api/shares` feature provides token-based sharing with optional password protection

---

### Server-Side Pretty Presigned URLs (URL Registry)

In addition to client-side HMAC presigned URLs, Beamdrop supports a server-side presigned URL registry that produces short, clean `/dl/{token}` download links.

#### Why Use Pretty URLs?

Client-side HMAC presigned URLs work but produce long URLs with token, expires, and access_key query parameters. The server-side registry creates short `/dl/{token}` URLs that are:

- Clean enough for emails, chat, and client portals
- Individually revocable (`DELETE /api/v1/presign/{token}`)
- Trackable (server counts downloads)
- Support max download limits
- **Not tied to any API key** they survive key rotation

Both methods can be used together in the same application.

#### API Endpoints

| Method   | Endpoint                  | Auth | Description                    |
| -------- | ------------------------- | ---- | ------------------------------ |
| `POST`   | `/api/v1/presign`         | HMAC | Create a pretty presigned URL  |
| `GET`    | `/api/v1/presign`         | HMAC | List all pretty presigned URLs |
| `GET`    | `/api/v1/presign/{token}` | HMAC | Get details of a specific URL  |
| `DELETE` | `/api/v1/presign/{token}` | HMAC | Revoke a pretty presigned URL  |
| `GET`    | `/dl/{token}`             | None | Download a file (public)       |

#### Create Request

```http
POST /api/v1/presign
Content-Type: application/json

{
  "bucket": "photos",
  "key": "vacation/beach.jpg",
  "method": "GET",
  "expiresIn": 3600,
  "maxDownloads": 100
}
```

| Field          | Required | Description                             |
| -------------- | -------- | --------------------------------------- |
| `bucket`       | Yes      | Bucket name                             |
| `key`          | Yes      | Object key                              |
| `method`       | No       | `GET` (default) or `PUT`                |
| `expiresIn`    | No       | Seconds until expiry (null = no expiry) |
| `maxDownloads` | No       | Max download count (null = unlimited)   |

#### Create Response

```json
{
  "token": "a1b2c3d4e5f6789a0b1c2d3e4f5a6b7c",
  "url": "https://files.example.com/dl/a1b2c3d4e5f6789a0b1c2d3e4f5a6b7c",
  "bucket": "photos",
  "key": "vacation/beach.jpg",
  "method": "GET",
  "expiresAt": "2026-02-24T13:00:00Z",
  "maxDownloads": 100,
  "createdAt": "2026-02-24T12:00:00Z"
}
```

#### Comparison: Client-Side vs Server-Side Presigned URLs

| Feature                        | Client-Side (HMAC)                                 | Server-Side (Pretty)            |
| ------------------------------ | -------------------------------------------------- | ------------------------------- |
| URL format                     | `/api/v1/buckets/…?token=…&expires=…&access_key=…` | `/dl/{token}`                   |
| Generated by                   | Client (no server call)                            | Server (`POST /api/v1/presign`) |
| Max downloads                  | No                                                 | Yes                             |
| Individually revocable         | No                                                 | Yes                             |
| Download tracking              | No                                                 | Yes                             |
| Survives API key rotation      | No                                                 | Yes                             |
| Works without registry feature | Yes                                                | No                              |

---

### API Key Management

#### Create API Key

```http
POST /api/v1/keys
Authorization: Bearer <admin-credentials>
Content-Type: application/json

{
    "name": "CI Pipeline",
    "permissions": [
        {"actions": ["PutObject", "GetObject"], "resource": "artifacts/*"}
    ],
    "bucketScope": "artifacts",  // Optional
    "expiresIn": 2592000         // 30 days, optional
}
```

**Response:**

```json
{
  "name": "CI Pipeline",
  "accessKeyId": "BDK_a1b2c3d4e5f6g7h8",
  "secretKey": "sk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "warning": "Save the secret key now. It cannot be retrieved later."
}
```

#### List API Keys

```http
GET /api/v1/keys
Authorization: Bearer <admin-credentials>
```

#### Revoke API Key

```http
DELETE /api/v1/keys/{access_key_id}
Authorization: Bearer <admin-credentials>
```

---

## Implementation Structure

### Directory Layout

```
beam/server/
├── handlers/
│   ├── api/
│   │   ├── auth.go          # API key auth, signature verification
│   │   ├── buckets.go       # Bucket CRUD operations
│   │   ├── objects.go       # Object CRUD operations
│   │   ├── presign.go       # Presigned URL generation
│   │   ├── keys.go          # API key management
│   │   └── middleware.go    # API-specific middleware
│   └── ... (existing handlers)
├── routes.go                 # Add API routes
└── server.go

pkg/
├── db/
│   ├── api_keys.go          # API key model & operations
│   ├── buckets.go           # Bucket metadata
│   └── objects.go           # Object metadata (optional)
├── storage/
│   ├── bucket.go            # Bucket filesystem operations
│   ├── object.go            # Object filesystem operations
│   └── multipart.go         # Multipart upload handling
└── crypto/
    └── signature.go         # HMAC signature utilities
```

### Database Models

```go
// pkg/db/buckets.go
type Bucket struct {
    ID          uint      `gorm:"primaryKey"`
    Name        string    `gorm:"uniqueIndex;size:63;not null"`
    OwnerKeyID  uint      `gorm:"index"`              // API key that created it
    Versioning  bool      `gorm:"default:false"`
    MaxSizeBytes int64    `gorm:"default:0"`          // 0 = unlimited
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// pkg/db/objects.go (optional - for metadata/versioning)
type ObjectMeta struct {
    ID          uint      `gorm:"primaryKey"`
    BucketID    uint      `gorm:"index;not null"`
    Key         string    `gorm:"size:1024;not null"`
    Size        int64
    ETag        string    `gorm:"size:32"`
    ContentType string    `gorm:"size:128"`
    Metadata    []byte    `gorm:"type:json"`          // Custom metadata
    VersionID   string    `gorm:"size:64"`            // For versioning
    IsDeleted   bool      `gorm:"default:false"`      // Soft delete for versioning
    CreatedAt   time.Time

    // Composite unique index
    // UNIQUE(bucket_id, key, version_id)
}
```

---

## Implementation Phases

### Phase 1: Core API (MVP)

1. **API Key Management**
   - Create/list/revoke keys via CLI flag or config
   - Simple auth middleware

2. **Basic Bucket Operations**
   - Create bucket (creates directory)
   - List buckets
   - Delete bucket (if empty)

3. **Basic Object Operations**
   - PUT object (simple upload)
   - GET object (download)
   - DELETE object
   - List objects

### Phase 2: Enhanced Features

1. **Presigned URLs**
   - Client-side HMAC presigned URLs (time-limited signed query params)
   - Server-side pretty presigned URLs (`POST /api/v1/presign` → `/dl/{token}`)
   - Support GET and PUT

2. **Permissions System**
   - Per-key bucket scoping
   - Action-based permissions

3. **Object Metadata**
   - Custom headers (X-Beamdrop-Meta-\*)
   - Content-Type handling

### Phase 3: Advanced Features

1. **Multipart Uploads**
   - Initiate/upload parts/complete
   - For files > 100MB

2. **Versioning**
   - Enable per-bucket
   - Version listing

3. **Lifecycle Policies**
   - Auto-delete after N days
   - Move to trash after N days

---

## CLI Flags

```bash
# Start with API enabled
beam -dir /data -api-enabled

# Generate initial admin key
beam -dir /data -api-enabled -generate-key "Admin Key"
# Output: Access Key: BDK_xxx, Secret: sk_xxx

# Specify API port (if different from main)
beam -dir /data -api-enabled -api-port 9000
```

### Config File (Optional)

```yaml
# ~/.beamdrop/config.yaml
api:
  enabled: true
  port: 9000 # Optional separate port
  maxUploadSize: 5GB

  # Initial admin key (created on first run)
  adminKey:
    name: "Admin"
    # Secret is generated and shown once
```

---

## Client Usage Examples

### cURL

```bash
# Upload file
curl -X PUT \
  -H "Authorization: Bearer BDK_xxx:$(generate_sig PUT /api/v1/buckets/uploads/file.txt)" \
  -H "X-Beamdrop-Date: $(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -H "Content-Type: text/plain" \
  --data-binary @file.txt \
  http://192.168.1.100:8080/api/v1/buckets/uploads/file.txt

# Download file
curl -H "Authorization: Bearer BDK_xxx:signature" \
  http://192.168.1.100:8080/api/v1/buckets/uploads/file.txt -o file.txt

# Using client-side HMAC presigned URL (no auth header needed)
curl -X PUT \
  -H "Content-Type: text/plain" \
  --data-binary @file.txt \
  "http://192.168.1.100:8080/api/v1/buckets/uploads/file.txt?token=xxx&expires=1707483600"

# Using server-side pretty presigned URL (no auth header needed)
curl "http://192.168.1.100:8080/dl/a1b2c3d4e5f6789a..."
```

### JavaScript/TypeScript

```typescript
import { BeamdropClient } from "beamdrop-sdk"; // Future SDK

const client = new BeamdropClient({
  endpoint: "http://192.168.1.100:8080",
  accessKeyId: "BDK_xxx",
  secretKey: "sk_xxx",
});

// Upload
await client.putObject({
  bucket: "uploads",
  key: "images/photo.jpg",
  body: fileBuffer,
  contentType: "image/jpeg",
});

// Download
const data = await client.getObject({
  bucket: "uploads",
  key: "images/photo.jpg",
});

// Presigned URL for frontend upload (client-side HMAC)
const { url } = await client.getSignedUrl({
  bucket: "uploads",
  key: "user-uploads/avatar.png",
  method: "PUT",
  expiresIn: 3600,
});

// Pretty presigned URL (server-side registry)
const { url: prettyUrl } = await client.createPresignedUrl({
  bucket: "uploads",
  key: "user-uploads/avatar.png",
  expiresIn: 3600,
  maxDownloads: 50,
});
// prettyUrl → "https://server/dl/a1b2c3d4..."
```

### Python

```python
import requests
import hmac
import hashlib
import base64
from datetime import datetime

def upload_file(endpoint, access_key, secret_key, bucket, key, file_path):
    url = f"{endpoint}/api/v1/buckets/{bucket}/{key}"
    timestamp = datetime.utcnow().strftime('%Y-%m-%dT%H:%M:%SZ')

    # Generate signature
    message = f"PUT\n/api/v1/buckets/{bucket}/{key}\n{timestamp}"
    signature = base64.b64encode(
        hmac.new(secret_key.encode(), message.encode(), hashlib.sha256).digest()
    ).decode()

    with open(file_path, 'rb') as f:
        response = requests.put(
            url,
            data=f,
            headers={
                'Authorization': f'Bearer {access_key}:{signature}',
                'X-Beamdrop-Date': timestamp,
                'Content-Type': 'application/octet-stream',
            }
        )
    return response.json()
```

---

## Error Responses

```json
{
  "error": {
    "code": "BucketNotFound",
    "message": "The specified bucket does not exist",
    "resource": "my-bucket",
    "requestId": "req_abc123"
  }
}
```

### Error Codes

| Code                  | HTTP Status | Description                                     |
| --------------------- | ----------- | ----------------------------------------------- |
| `AccessDenied`        | 403         | Invalid credentials or insufficient permissions |
| `BucketAlreadyExists` | 409         | Bucket name is taken                            |
| `BucketNotEmpty`      | 409         | Cannot delete non-empty bucket                  |
| `BucketNotFound`      | 404         | Bucket doesn't exist                            |
| `InvalidBucketName`   | 400         | Invalid bucket name format                      |
| `InvalidRequest`      | 400         | Malformed request                               |
| `KeyNotFound`         | 404         | Object doesn't exist                            |
| `SignatureExpired`    | 401         | Request timestamp too old                       |
| `SignatureMismatch`   | 401         | Invalid signature                               |

---

## Security Considerations

1. **Bucket Names**: Validate against regex `^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`
2. **Key Paths**: Prevent path traversal (`..`, absolute paths)
3. **Rate Limiting**: Consider per-key rate limits
4. **Signature Expiry**: Max 15 minutes for signed requests
5. **HTTPS**: Strongly recommend TLS in production
6. **Key Rotation**: Support key expiration and rotation

---

## Compatibility Notes

This is a **simplified** S3-like API, not full AWS S3 compatibility. Key differences:

- Simplified signature scheme (not AWS Signature V4)
- No IAM policies (simple permission model)
- No bucket policies or ACLs
- No cross-region replication
- Single-node only (no clustering)

For apps needing full S3 compatibility, consider MinIO. Beamdrop's API is designed for simplicity and ease of self-hosting.
