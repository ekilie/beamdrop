# Beamdrop — Complete Documentation

> Turn any VPS or server into a private, self-hosted Google Drive + S3 in seconds. Built with Go and React.

---

## Table of Contents

1. [Overview](#overview)
2. [Installation](#installation)
3. [Configuration](#configuration)
4. [Authentication](#authentication)
   - [Password Auth (Web UI)](#password-auth-web-ui)
   - [API Key Auth (S3 API)](#api-key-auth-s3-api)
   - [HMAC Signature Generation](#hmac-signature-generation)
5. [File Management API](#file-management-api)
   - [List Files](#list-files)
   - [Upload File](#upload-file)
   - [Download File](#download-file)
   - [Create Directory](#create-directory)
   - [Move File](#move-file)
   - [Copy File](#copy-file)
   - [Rename File](#rename-file)
   - [Trash File](#trash-file)
   - [Write File](#write-file)
   - [Search Files](#search-files)
   - [Star / Unstar File](#star--unstar-file)
   - [Get Starred Files](#get-starred-files)
6. [S3-Compatible API](#s3-compatible-api)
   - [API Key Management](#api-key-management)
   - [Bucket Operations](#bucket-operations)
   - [Object Operations](#object-operations)
   - [Presigned URLs](#presigned-urls)
7. [Shareable Links](#shareable-links)
   - [Create Link](#create-link)
   - [List Links](#list-links)
   - [Delete Link](#delete-link)
   - [Access Link](#access-link)
8. [Health & Monitoring](#health--monitoring)
   - [Health Endpoints](#health-endpoints)
   - [Stats](#stats)
   - [WebSocket Stats](#websocket-real-time-stats)
   - [Logs](#logs)
   - [Prometheus Metrics](#prometheus-metrics)
9. [Full Usage Flow (TypeScript)](#full-usage-flow-typescript)
10. [Error Codes Reference](#error-codes-reference)
11. [Storage Structure](#storage-structure)
12. [Docker & Deployment](#docker--deployment)

---

## Overview

Beamdrop is a self-hosted file sharing server that provides:

- **Web UI** — A modern React-based file browser for interactive management
- **File Management API** — REST endpoints for uploading, downloading, moving, copying, renaming, and searching files
- **S3-Compatible API** — Bucket/object storage with HMAC-SHA256 signed authentication
- **Shareable Links** — Generate unique URLs to share files with optional password protection and expiry
- **Real-time Stats** — WebSocket-powered live dashboard
- **Health Probes** — Kubernetes-compatible liveness, readiness, and startup probes
- **Prometheus Metrics** — Full observability with request counters, latency histograms, and storage gauges

---

## Installation

### From Source

```bash
git clone https://github.com/ekilie/beamdrop.git
cd beamdrop
make build
```

### Binary Downloads

```bash
# macOS (Apple Silicon)
curl -L https://github.com/ekilie/beamdrop/releases/latest/download/beamdrop-darwin-arm64.tar.gz | sudo tar -C /usr/local/bin -xz

# macOS (Intel)
curl -L https://github.com/ekilie/beamdrop/releases/latest/download/beamdrop-darwin-amd64.tar.gz | sudo tar -C /usr/local/bin -xz

# Linux (amd64)
curl -L https://github.com/ekilie/beamdrop/releases/latest/download/beamdrop-linux-amd64.tar.gz | sudo tar -C /usr/local/bin -xz

# Linux (arm64)
curl -L https://github.com/ekilie/beamdrop/releases/latest/download/beamdrop-linux-arm64.tar.gz | sudo tar -C /usr/local/bin -xz
```

### Docker

```bash
docker run -d --name beamdrop -p 7777:7777 -v beamdrop-data:/data beamdrop
```

### Docker Compose

```bash
docker compose up -d
```

---

## Configuration

### Command Line Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-dir` | Directory to share | `.` (current) |
| `-port` | Server port | Auto-detect (prefers 7777) |
| `-p` | Password for web authentication | None (disabled) |
| `-api-auth` | Enable API key authentication for S3 API | `false` |
| `-tls-cert` | Path to TLS certificate file | None |
| `-tls-key` | Path to TLS private key file | None |
| `-allowed-origins` | Comma-separated CORS origins | None (CORS disabled) |
| `-db-path` | Path to database file or directory (directory auto-appends `beamdrop.db`) | `~/.beamdrop/beamdrop.db` |
| `-rate-limit` | Requests/min per IP (0 = disabled) | `100` |
| `-log-level` | `debug`, `info`, `warn`, `error` | `info` |
| `-qr` | Enable QR code display | `false` |
| `-shutdown-timeout` | Graceful shutdown timeout | `30s` |
| `-v` | Show version | — |
| `-h` | Show help | — |

### Environment Variables (Docker)

| Variable | Default | Description |
|----------|---------|-------------|
| `BEAMDROP_PORT` | `7777` | Server port |
| `BEAMDROP_PASSWORD` | — | Enable password auth |
| `BEAMDROP_LOG_LEVEL` | `info` | Log level |
| `BEAMDROP_RATE_LIMIT` | `100` | Requests/min per IP |
| `BEAMDROP_API_AUTH` | `false` | Enable S3 API key auth |
| `BEAMDROP_QR` | `false` | Enable QR code display |
| `BEAMDROP_ALLOWED_ORIGINS` | — | CORS origins |
| `BEAMDROP_DB_PATH` | — | Path to database file or directory (directory auto-appends `beamdrop.db`) |
| `BEAMDROP_TLS_CERT` | — | TLS certificate path |
| `BEAMDROP_TLS_KEY` | — | TLS private key path |

### Quick Start Examples

```bash
# Share current directory
beamdrop

# Share a specific directory with password
beamdrop -dir /path/to/share -p mysecretpassword

# Enable S3 API with auth
beamdrop -dir /path/to/share -api-auth

# Full production setup
beamdrop -dir /data -p secret -api-auth -tls-cert cert.pem -tls-key key.pem -rate-limit 200
```

### Upload Limits

- **Max file size**: 100 MB
- **Allowed MIME types**: Images, documents, archives, audio, video, code, and `application/octet-stream`

---

## Authentication

Beamdrop has two independent auth systems:

1. **Password Auth** — Protects the web UI and file management endpoints. Uses JWT tokens stored in cookies.
2. **API Key Auth** — Protects the S3-compatible API (`/api/v1/buckets/*`). Uses HMAC-SHA256 signed requests.

### Password Auth (Web UI)

When started with `-p <password>`, all routes except health probes, login, and static assets require authentication.

**Public routes (always accessible):**
- `/` — Landing page
- `/auth/login`, `/auth/status`
- `/health/*`, `/ready`, `/metrics`
- `/assets/*`, `/static/*`
- `/share/*` — Shareable link pages
- `/api/shares/access/*` — Shareable link API (has own password protection)

### API Key Auth (S3 API)

When started with `-api-auth`, every request to `/api/v1/buckets/*` must include:

1. `Authorization: Bearer <access_key_id>:<signature>` header
2. `X-Beamdrop-Date: <ISO 8601 timestamp>` header

The timestamp must be within **15 minutes** of the server time (clock skew tolerance).

### HMAC Signature Generation

The signature is computed as:

```
string_to_sign = "<METHOD>\n<PATH>\n<TIMESTAMP>"
signature = Base64(HMAC-SHA256(secret_key, string_to_sign))
```

---

## File Management API

All file management endpoints require password authentication if enabled. All responses are JSON.

### List Files

List files and directories at a given path.

```
GET /files?path=<relative_path>
```

**Query Parameters:**
| Parameter | Required | Description |
|-----------|----------|-------------|
| `path` | No | Relative path within the shared directory. Defaults to root. |

**Response:**
```json
[
  {
    "name": "documents",
    "isDir": true,
    "size": "4.0 KB",
    "modTime": "2025-01-15 10:30:00",
    "path": "documents",
    "isStarred": false
  },
  {
    "name": "photo.jpg",
    "isDir": false,
    "size": "2.5 MB",
    "modTime": "2025-01-15 09:00:00",
    "path": "photo.jpg",
    "isStarred": true
  }
]
```

If `path` points to a file, the file content is served directly.

---

### Upload File

Upload a file using multipart form data.

```
POST /upload
Content-Type: multipart/form-data
```

**Form Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `file` | File | The file to upload (max 100 MB) |

**Response:**
```json
{
  "message": "Uploaded",
  "file": "photo.jpg"
}
```

**Errors:**
- `413` — File too large (> 100 MB)
- `415` — MIME type not allowed

---

### Download File

Download a file by name.

```
GET /download?file=<filename>
```

**Query Parameters:**
| Parameter | Required | Description |
|-----------|----------|-------------|
| `file` | Yes | File path relative to the shared directory |

**Response:** Raw file content.

---

### Create Directory

```
POST /mkdir
Content-Type: application/json
```

**Body:**
```json
{
  "dirPath": "path/to/new-directory"
}
```

**Response:**
```json
{
  "message": "Directory created successfully",
  "path": "path/to/new-directory"
}
```

---

### Move File

```
POST /move
Content-Type: application/json
```

**Body:**
```json
{
  "sourcePath": "old/location/file.txt",
  "targetPath": "new/location/file.txt"
}
```

**Response:**
```json
{
  "message": "File moved successfully",
  "from": "old/location/file.txt",
  "to": "new/location/file.txt"
}
```

---

### Copy File

```
POST /copy
Content-Type: application/json
```

**Body:**
```json
{
  "sourcePath": "original/file.txt",
  "targetPath": "copy/file.txt"
}
```

**Response:**
```json
{
  "message": "File copied successfully",
  "from": "original/file.txt",
  "to": "copy/file.txt"
}
```

---

### Rename File

```
POST /rename
Content-Type: application/json
```

**Body:**
```json
{
  "oldPath": "documents/report.txt",
  "newName": "final-report.txt"
}
```

**Response:**
```json
{
  "message": "Renamed successfully",
  "oldPath": "documents/report.txt",
  "newPath": "documents/final-report.txt"
}
```

---

### Trash File

Moves a file to `.beamdrop_trash/` instead of permanently deleting it.

```
POST /trash
Content-Type: application/json
```

**Body:**
```json
{
  "sourcePath": "file-to-delete.txt"
}
```

**Response:**
```json
{
  "message": "File moved to trash successfully",
  "from": "file-to-delete.txt",
  "to": ".beamdrop_trash/file-to-delete.txt"
}
```

---

### Write File

Write content directly to a file (creates parent directories automatically).

```
POST /write
Content-Type: application/json
```

**Body:**
```json
{
  "filePath": "notes/readme.txt",
  "content": "Hello, World!"
}
```

**Response:**
```json
{
  "message": "File written successfully",
  "filePath": "notes/readme.txt"
}
```

---

### Search Files

Recursively search for files by name (case-insensitive substring match).

```
GET /search?q=<query>&path=<optional_path>
```

**Query Parameters:**
| Parameter | Required | Description |
|-----------|----------|-------------|
| `q` | Yes | Search query |
| `path` | No | Restrict search to a subdirectory |

**Response:**
```json
{
  "query": "report",
  "path": "",
  "results": [
    {
      "name": "final-report.txt",
      "isDir": false,
      "size": "1.2 KB",
      "modTime": "2025-01-15 10:30:00",
      "path": "documents/final-report.txt",
      "isStarred": false
    }
  ],
  "count": 1
}
```

---

### Star / Unstar File

Toggle the starred status of a file (starred → unstarred, unstarred → starred).

```
POST /star
Content-Type: application/json
```

**Body:**
```json
{
  "filePath": "documents/important.pdf"
}
```

**Response (starred):**
```json
{
  "message": "File starred",
  "filePath": "documents/important.pdf",
  "starred": "true"
}
```

**Response (unstarred):**
```json
{
  "message": "File unstarred",
  "filePath": "documents/important.pdf",
  "starred": "false"
}
```

---

### Get Starred Files

List all starred files.

```
GET /starred
```

**Response:**
```json
{
  "starred": [
    {
      "filePath": "documents/important.pdf",
      "createdAt": "2025-01-15T10:30:00Z"
    }
  ]
}
```

---

## S3-Compatible API

The S3-compatible API lives under `/api/v1/` and provides bucket and object storage with HMAC-SHA256 signed authentication (when `-api-auth` is enabled).

### API Key Management

API keys are managed via `/api/v1/keys`. These endpoints use **session auth** (cookies from the web UI login), not API key auth.

#### Create API Key

```
POST /api/v1/keys
Content-Type: application/json
```

**Body:**
```json
{
  "name": "My CI Pipeline",
  "permissions": "read,write",
  "bucketScope": "my-bucket",
  "expiresIn": 2592000
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Human-readable name |
| `permissions` | string | No | Comma-separated permissions |
| `bucketScope` | string | No | Restrict key to a specific bucket |
| `expiresIn` | number | No | Expiry in seconds (null = never) |

**Response (201):**
```json
{
  "id": 1,
  "name": "My CI Pipeline",
  "accessKeyId": "BDK_a1b2c3d4e5f67890",
  "secretKey": "sk_1234567890abcdef1234567890abcdef12345678",
  "permissions": "read,write",
  "bucketScope": "my-bucket",
  "expiresAt": "2025-02-14T10:30:00Z",
  "createdAt": "2025-01-15T10:30:00Z",
  "warning": "Save the secret key now. It cannot be retrieved later."
}
```

> **Important:** The `secretKey` is shown **only once**. Store it securely.

#### List API Keys

```
GET /api/v1/keys
```

**Response:**
```json
{
  "keys": [
    {
      "id": 1,
      "name": "My CI Pipeline",
      "accessKeyId": "BDK_a1b2c3d4e5f67890",
      "permissions": "read,write",
      "bucketScope": "my-bucket",
      "expiresAt": "2025-02-14T10:30:00Z",
      "lastUsedAt": "2025-01-20T09:15:00Z",
      "createdAt": "2025-01-15T10:30:00Z",
      "disabled": false
    }
  ],
  "count": 1
}
```

#### Delete API Key

```
DELETE /api/v1/keys?accessKeyId=BDK_a1b2c3d4e5f67890
```

**Response:** `204 No Content`

---

### Bucket Operations

All bucket endpoints require API key auth when `-api-auth` is enabled.

#### List Buckets

```
GET /api/v1/buckets
```

**Response:**
```json
{
  "buckets": [
    {
      "name": "my-bucket",
      "createdAt": "2025-01-15T10:30:00Z"
    }
  ],
  "count": 1
}
```

#### Create Bucket

```
PUT /api/v1/buckets/{bucket-name}
```

**Bucket naming rules (S3-compatible):**
- 3–63 characters
- Lowercase letters, numbers, hyphens, dots
- Must start and end with a letter or number
- Cannot be an IP address format

**Response (201):**
```json
{
  "bucket": "my-bucket",
  "created": "2025-01-15T10:30:00Z",
  "location": "/api/v1/buckets/my-bucket"
}
```

#### Check Bucket Exists

```
HEAD /api/v1/buckets/{bucket-name}
```

**Response:** `200 OK` or `404 Not Found`

#### Get Bucket Info

```
GET /api/v1/buckets/{bucket-name}
```

**Response:**
```json
{
  "bucket": "my-bucket",
  "exists": true
}
```

#### Delete Bucket

```
DELETE /api/v1/buckets/{bucket-name}
```

Bucket must be empty. **Response:** `204 No Content`

**Errors:**
- `409 BUCKET_NOT_EMPTY` — Bucket still has objects

---

### Object Operations

#### Upload Object (Raw Body)

```
PUT /api/v1/buckets/{bucket}/{key}
Content-Type: <mime-type>

<raw file content>
```

**Response:**
```json
{
  "bucket": "my-bucket",
  "key": "images/photo.jpg",
  "etag": "d41d8cd98f00b204e9800998ecf8427e",
  "size": 2048576,
  "url": "/api/v1/buckets/my-bucket/images/photo.jpg"
}
```

#### Upload Object (Multipart)

```
POST /api/v1/buckets/{bucket}/{key}
Content-Type: multipart/form-data
```

**Form Fields:**
| Field | Description |
|-------|-------------|
| `file` | The file to upload |

**Response:** Same as PUT upload.

#### Download Object

```
GET /api/v1/buckets/{bucket}/{key}
```

**Response Headers:**
- `Content-Length` — File size in bytes
- `Content-Type` — Detected MIME type
- `Last-Modified` — Last modification time
- `ETag` — MD5 hash of content

Supports HTTP Range requests for partial content downloads.

#### Get Object Metadata

```
HEAD /api/v1/buckets/{bucket}/{key}
```

Returns the same headers as GET but without the body.

#### Delete Object

```
DELETE /api/v1/buckets/{bucket}/{key}
```

**Response:** `204 No Content`

#### List Objects

```
GET /api/v1/buckets/{bucket}?prefix=<prefix>&delimiter=<delimiter>&max-keys=<n>
```

**Query Parameters:**
| Parameter | Default | Description |
|-----------|---------|-------------|
| `prefix` | — | Filter objects by key prefix |
| `delimiter` | — | Group keys by delimiter (e.g., `/` for virtual directories) |
| `max-keys` | `1000` | Maximum number of objects to return (max 1000) |

**Response:**
```json
{
  "bucket": "my-bucket",
  "prefix": "images/",
  "delimiter": "/",
  "maxKeys": 1000,
  "isTruncated": false,
  "contents": [
    {
      "key": "images/photo.jpg",
      "size": 2048576,
      "lastModified": "2025-01-15T10:30:00Z",
      "etag": "d41d8cd98f00b204e9800998ecf8427e"
    }
  ],
  "commonPrefixes": [
    "images/thumbnails/"
  ]
}
```

---

### Presigned URLs

Access objects without full HMAC auth by generating a presigned URL with a time-limited token. The URL is self-contained — anyone with the link can access the file until it expires. No API key is required on the client side.

**URL Format:**
```
GET /api/v1/buckets/{bucket}/{key}?token=<token>&expires=<timestamp>&access_key=<access_key_id>
```

#### Token Generation (Client-Side)

The token is an HMAC-SHA256 signature of four fields joined by newlines:

```
message = "<METHOD>\n<BUCKET>\n<KEY>\n<UNIX_TIMESTAMP>"
token = Base64URL(HMAC-SHA256(secret_key, message))
```

| Field | Example | Description |
|-------|---------|-------------|
| `METHOD` | `GET` | HTTP method the URL is valid for |
| `BUCKET` | `photos` | Bucket name (not the full path) |
| `KEY` | `vacation/beach.jpg` | Object key within the bucket |
| `UNIX_TIMESTAMP` | `1707741600` | Expiration time as Unix seconds |

#### Query Parameters

| Parameter | Example | Description |
|-----------|---------|-------------|
| `token` | `aB3d...` | Base64URL-encoded HMAC-SHA256 token |
| `expires` | `2026-02-12T12:00:00Z` | When the link expires (RFC 3339 or compact ISO `20260212T120000Z`) |
| `access_key` | `BDK_abc123` | Access Key ID used to generate the token |

#### Example: Generate a presigned URL with cURL

```bash
# Configuration
SECRET_KEY="sk_your_secret_key"
ACCESS_KEY="BDK_your_access_key"
BUCKET="photos"
KEY="vacation/beach.jpg"
EXPIRES_UNIX=$(date -d '+1 hour' +%s)  # 1 hour from now

# Generate token
MESSAGE=$(printf "GET\n%s\n%s\n%s" "$BUCKET" "$KEY" "$EXPIRES_UNIX")
TOKEN=$(printf '%s' "$MESSAGE" | openssl dgst -sha256 -hmac "$SECRET_KEY" -binary | base64 | tr '+/' '-_' | tr -d '=')
EXPIRES_ISO=$(date -u -d@${EXPIRES_UNIX} +%Y-%m-%dT%H:%M:%SZ)

# Use the URL
curl "http://localhost:8090/api/v1/buckets/${BUCKET}/${KEY}?token=${TOKEN}&expires=${EXPIRES_ISO}&access_key=${ACCESS_KEY}"
```

#### Important Behaviors & Limitations

- **Always expires** — there is no "permanent" presigned URL. The server always checks `time.Now().After(expiresAt)`.
- **Tied to API key** — deleting, disabling, or rotating the API key invalidates all presigned URLs generated with it.
- **Method-specific** — a token generated for `GET` will not work for `PUT`. The HTTP method is part of the signature.
- **Key-specific** — a token for `photos/pic.jpg` cannot be reused for `photos/other.jpg`.
- **No maximum expiry** — you can set expiry far in the future (e.g. 10 years), but it breaks on key rotation.
- **No individual revocation** — to invalidate a leaked URL, either delete the object or disable the entire API key.

#### Suggested Expiry Durations

| Use case | Duration |
|----------|----------|
| One-time download link (email, chat) | 1–24 hours |
| Embedded in a web page (avatars) | 1–7 days |
| Client portal / invoice download | 7–30 days |
| Semi-permanent asset | 1–10 years (breaks on key rotation) |

#### Alternatives for Permanent Public Access

If you need files to be permanently accessible without signing:

1. **Run without `-api-auth`** — all S3 API reads are public
2. **Proxy through your app** — your backend authenticates to Beamdrop and streams the file to end users
3. **Use shareable links** — the `/api/shares` feature provides token-based sharing with optional password protection and expiry

---

### Server-Side Pretty Presigned URLs (URL Registry)

In addition to client-side HMAC presigned URLs (above), Beamdrop supports a **server-side presigned URL registry** that produces short, clean `/dl/{token}` URLs. Both methods can be used together.

#### Why Use Pretty URLs?

| Feature | Client-Side (HMAC) | Server-Side (Pretty) |
|---------|-------------------|---------------------|
| URL format | `/api/v1/buckets/…?token=…&expires=…&access_key=…` | `/dl/{token}` |
| Generated by | Client (no server call) | Server (`POST /api/v1/presign`) |
| Max downloads | No | Yes |
| Individually revocable | No | Yes |
| Download tracking | No | Yes |
| Survives API key rotation | No | Yes |

#### Create a Pretty Presigned URL

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
```

Response:
```json
{
  "token": "a1b2c3d4e5f6789a0b1c2d3e4f5a6b7c",
  "url": "https://server/dl/a1b2c3d4e5f6789a0b1c2d3e4f5a6b7c",
  "bucket": "photos",
  "key": "vacation/beach.jpg",
  "method": "GET",
  "expiresAt": "2026-02-24T13:00:00Z",
  "maxDownloads": 100,
  "createdAt": "2026-02-24T12:00:00Z"
}
```

#### Download (Public — No Auth)

```bash
curl https://server/dl/a1b2c3d4e5f6789a0b1c2d3e4f5a6b7c -o beach.jpg
```

#### Revoke a Pretty Presigned URL

```bash
curl -X DELETE https://server/api/v1/presign/a1b2c3d4e5f6789a0b1c2d3e4f5a6b7c \
  -H "Authorization: Bearer BDK_xxx:signature" \
  -H "X-Beamdrop-Date: ..."
```

#### List All Pretty Presigned URLs

```bash
curl https://server/api/v1/presign \
  -H "Authorization: Bearer BDK_xxx:signature" \
  -H "X-Beamdrop-Date: ..."
```

---

## Shareable Links

Shareable links allow sharing files/folders via unique URLs with optional password protection and expiry. They bypass normal authentication.

### Create Link

```
POST /api/shares
Content-Type: application/json
```

**Body:**
```json
{
  "path": "documents/report.pdf",
  "password": "optional-password",
  "expiresIn": 86400
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | Path to the file or folder |
| `password` | string | No | Password-protect the link |
| `expiresIn` | number | No | Expiry in seconds |

**Response (201):**
```json
{
  "token": "abc123def456",
  "url": "http://localhost:7777/share/abc123def456",
  "path": "documents/report.pdf",
  "expiresAt": "2025-01-16T10:30:00Z",
  "createdAt": "2025-01-15T10:30:00Z"
}
```

### List Links

```
GET /api/shares/list
```

**Response:**
```json
[
  {
    "id": 1,
    "path": "documents/report.pdf",
    "token": "abc123def456",
    "hasPassword": true,
    "expiresAt": "2025-01-16T10:30:00Z",
    "accessCount": 5,
    "createdAt": "2025-01-15T10:30:00Z"
  }
]
```

### Delete Link

```
DELETE /api/shares/delete?token=abc123def456
```

**Response:**
```json
{
  "message": "Shareable link deleted successfully"
}
```

### Access Link

Public endpoint — no authentication required (but may require link password).

```
GET /api/shares/access/{token}
```

**If password-protected (no password provided):**
```json
{
  "requiresPassword": true,
  "path": "documents/report.pdf"
}
```

**Provide password via POST or query parameter:**
```
GET /api/shares/access/{token}?password=my-password
```
or
```
POST /api/shares/access/{token}
Content-Type: application/json

{ "password": "my-password" }
```

**File response (metadata):**
```json
{
  "path": "documents/report.pdf",
  "name": "report.pdf",
  "size": "1.2 MB",
  "sizeBytes": 1258291,
  "contentType": "application/pdf",
  "isDir": false,
  "isFile": true
}
```

**Directory response:**
```json
{
  "path": "documents",
  "files": [...],
  "isDir": true
}
```

**Download mode:**
```
GET /api/shares/access/{token}?mode=download
```

**Inline preview mode:**
```
GET /api/shares/access/{token}?mode=inline
```

---

## Health & Monitoring

### Health Endpoints

Kubernetes-compatible health probes with component-level status.

| Endpoint | Purpose | Checks |
|----------|---------|--------|
| `GET /health` | Full health overview | Process, startup, database, storage, runtime |
| `GET /health/live` | Liveness probe | Process is running (no I/O) |
| `GET /health/ready` | Readiness probe | Database + storage accessible |
| `GET /health/startup` | Startup probe | Server initialization complete |
| `GET /ready` | Legacy readiness alias | Same as `/health/ready` |

**Response Example:**
```json
{
  "status": "healthy",
  "service": "beamdrop",
  "version": "0.0.1",
  "timestamp": "2025-01-15T10:30:00Z",
  "components": {
    "process": { "status": "ok", "message": "running" },
    "startup": { "status": "ok", "message": "initialisation complete" },
    "database": { "status": "ok", "message": "connected", "latency": "1.23ms" },
    "storage": { "status": "ok", "message": "writable" },
    "runtime": { "status": "ok", "message": "goroutines: 12" }
  }
}
```

### Stats

Get server statistics.

```
GET /stats
```

**Response:**
```json
{
  "downloads": 42,
  "uploads": 15,
  "requests": 1234,
  "startTime": "2025-01-15T08:00:00Z"
}
```

### WebSocket Real-Time Stats

Connect for live stats updates every minute.

```
ws://localhost:7777/ws/stats
```

**Message format:**
```json
{
  "downloads": 42,
  "requests": 1234,
  "uploads": 15,
  "startTime": "2025-01-15T08:00:00Z",
  "system": {
    "memory": { "total": 16000000000, "used": 8000000000, "percent": 50.0 },
    "disk": { "total": 500000000000, "used": 200000000000, "percent": 40.0 },
    "cpu": { "percent": 25.0 }
  }
}
```

### Logs

Retrieve structured JSON logs from the server.

```
GET /api/logs?limit=200&offset=0&level=error&search=upload
```

**Query Parameters:**
| Parameter | Default | Description |
|-----------|---------|-------------|
| `limit` | `200` | Max entries (max 5000) |
| `offset` | `0` | Entries to skip (for pagination) |
| `level` | — | Filter by log level |
| `search` | — | Case-insensitive message search |

**Response:**
```json
{
  "logs": [
    {
      "time": "2025-01-15T10:30:00Z",
      "level": "INFO",
      "msg": "File uploaded successfully",
      "file": "photo.jpg"
    }
  ],
  "total": 500,
  "returned": 200,
  "hasMore": true,
  "logPath": "/data/.beamdrop/beamdrop.log"
}
```

### Prometheus Metrics

```
GET /metrics
```

Returns metrics in Prometheus text format. Key metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `beamdrop_requests_total` | counter | HTTP requests by method, path, status |
| `beamdrop_request_duration_seconds` | histogram | Request latency (p50/p95/p99) |
| `beamdrop_auth_failures_total` | counter | Auth failures by reason |
| `beamdrop_uploads_total` | counter | Completed uploads |
| `beamdrop_downloads_total` | counter | Completed downloads |
| `beamdrop_upload_size_bytes` | histogram | Upload file sizes |
| `beamdrop_storage_bytes` | gauge | Bytes used by stored files |
| `beamdrop_objects_total` | gauge | Number of stored files |
| `beamdrop_active_connections` | gauge | In-flight HTTP requests |
| `beamdrop_storage_free_bytes` | gauge | Free disk space |
| `beamdrop_storage_total_bytes` | gauge | Total disk capacity |
| `beamdrop_goroutines_count` | gauge | Go goroutine count |

A pre-built Grafana dashboard is at `docs/grafana-dashboard.json`.

---

## Full Usage Flow (TypeScript)

Below is a complete TypeScript example demonstrating **every feature** of the Beamdrop API — from authentication through file management, S3 buckets/objects, shareable links, and monitoring.

### Setup

```ts
import crypto from "crypto";

const BASE_URL = "http://localhost:7777";

// ─── Helpers ──────────────────────────────────────────────────────────────────

async function request(
  method: string,
  path: string,
  options: {
    body?: any;
    headers?: Record<string, string>;
    token?: string;
    formData?: FormData;
  } = {}
): Promise<any> {
  const headers: Record<string, string> = {
    ...options.headers,
  };

  if (options.token) {
    headers["Authorization"] = `Bearer ${options.token}`;
  }

  let body: any;
  if (options.formData) {
    body = options.formData;
    // Don't set Content-Type for FormData — the runtime sets the boundary
  } else if (options.body) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }

  const res = await fetch(`${BASE_URL}${path}`, { method, headers, body });

  const contentType = res.headers.get("content-type") || "";
  if (contentType.includes("application/json")) {
    return res.json();
  }
  return res;
}
```

### Step 1 — Check Auth Status

```ts
// Check if password authentication is enabled and whether we're authenticated
async function checkAuthStatus(): Promise<{
  authEnabled: boolean;
  authenticated: boolean;
}> {
  return request("GET", "/auth/status");
}

const authStatus = await checkAuthStatus();
console.log("Auth enabled:", authStatus.authEnabled);
console.log("Authenticated:", authStatus.authenticated);
```

### Step 2 — Login (If Password Auth Enabled)

```ts
async function login(password: string): Promise<{
  success: boolean;
  token: string;
  message: string;
}> {
  return request("POST", "/auth/login", {
    body: { password },
  });
}

let authToken: string | undefined;

if (authStatus.authEnabled && !authStatus.authenticated) {
  const loginResult = await login("mysecretpassword");
  console.log("Login:", loginResult.message);
  authToken = loginResult.token; // Save for subsequent requests
}
```

### Step 3 — List Files

```ts
// List files at the root directory
async function listFiles(path: string = ""): Promise<any[]> {
  return request("GET", `/files?path=${encodeURIComponent(path)}`, {
    token: authToken,
  });
}

const files = await listFiles();
console.log("Files:", files);

// List files in a subdirectory
const subFiles = await listFiles("documents");
console.log("Documents:", subFiles);
```

### Step 4 — Upload a File

```ts
async function uploadFile(
  filePath: string,
  content: Blob | Buffer
): Promise<any> {
  const formData = new FormData();
  formData.append("file", new Blob([content]), filePath);

  return request("POST", "/upload", {
    formData,
    token: authToken,
  });
}

const uploadResult = await uploadFile(
  "hello.txt",
  Buffer.from("Hello from Beamdrop!")
);
console.log("Upload:", uploadResult);
// => { message: "Uploaded", file: "hello.txt" }
```

### Step 5 — Download a File

```ts
async function downloadFile(filename: string): Promise<string> {
  const res = await request("GET", `/download?file=${encodeURIComponent(filename)}`, {
    token: authToken,
  });
  if (res instanceof Response) {
    return res.text();
  }
  return res;
}

const content = await downloadFile("hello.txt");
console.log("Downloaded content:", content);
// => "Hello from Beamdrop!"
```

### Step 6 — Create a Directory

```ts
async function mkdir(dirPath: string): Promise<any> {
  return request("POST", "/mkdir", {
    body: { dirPath },
    token: authToken,
  });
}

const mkdirResult = await mkdir("projects/my-app");
console.log("Mkdir:", mkdirResult);
// => { message: "Directory created successfully", path: "projects/my-app" }
```

### Step 7 — Write a File (Inline Content)

```ts
async function writeFile(filePath: string, content: string): Promise<any> {
  return request("POST", "/write", {
    body: { filePath, content },
    token: authToken,
  });
}

const writeResult = await writeFile(
  "projects/my-app/README.md",
  "# My App\n\nBuilt with Beamdrop!"
);
console.log("Write:", writeResult);
// => { message: "File written successfully", filePath: "projects/my-app/README.md" }
```

### Step 8 — Copy a File

```ts
async function copyFile(
  sourcePath: string,
  targetPath: string
): Promise<any> {
  return request("POST", "/copy", {
    body: { sourcePath, targetPath },
    token: authToken,
  });
}

const copyResult = await copyFile("hello.txt", "hello-backup.txt");
console.log("Copy:", copyResult);
// => { message: "File copied successfully", from: "hello.txt", to: "hello-backup.txt" }
```

### Step 9 — Rename a File

```ts
async function renameFile(
  oldPath: string,
  newName: string
): Promise<any> {
  return request("POST", "/rename", {
    body: { oldPath, newName },
    token: authToken,
  });
}

const renameResult = await renameFile("hello-backup.txt", "hello-copy.txt");
console.log("Rename:", renameResult);
// => { message: "Renamed successfully", oldPath: "hello-backup.txt", newPath: "hello-copy.txt" }
```

### Step 10 — Move a File

```ts
async function moveFile(
  sourcePath: string,
  targetPath: string
): Promise<any> {
  return request("POST", "/move", {
    body: { sourcePath, targetPath },
    token: authToken,
  });
}

const moveResult = await moveFile(
  "hello-copy.txt",
  "projects/my-app/hello-copy.txt"
);
console.log("Move:", moveResult);
// => { message: "File moved successfully", from: "hello-copy.txt", to: "projects/my-app/hello-copy.txt" }
```

### Step 11 — Search Files

```ts
async function searchFiles(
  query: string,
  path: string = ""
): Promise<any> {
  const params = new URLSearchParams({ q: query });
  if (path) params.set("path", path);
  return request("GET", `/search?${params}`, {
    token: authToken,
  });
}

const searchResult = await searchFiles("hello");
console.log("Search results:", searchResult.count, "files found");
searchResult.results.forEach((f: any) => console.log(" -", f.path));
```

### Step 12 — Star & Unstar Files

```ts
async function toggleStar(filePath: string): Promise<any> {
  return request("POST", "/star", {
    body: { filePath },
    token: authToken,
  });
}

// Star a file
const starResult = await toggleStar("hello.txt");
console.log("Star:", starResult);
// => { message: "File starred", filePath: "hello.txt", starred: "true" }

// Get all starred files
async function getStarred(): Promise<any> {
  return request("GET", "/starred", { token: authToken });
}

const starred = await getStarred();
console.log("Starred files:", starred.starred);

// Unstar the file (toggle again)
const unstarResult = await toggleStar("hello.txt");
console.log("Unstar:", unstarResult);
// => { message: "File unstarred", filePath: "hello.txt", starred: "false" }
```

### Step 13 — Trash a File

```ts
async function trashFile(sourcePath: string): Promise<any> {
  return request("POST", "/trash", {
    body: { sourcePath },
    token: authToken,
  });
}

const trashResult = await trashFile("hello.txt");
console.log("Trash:", trashResult);
// => { message: "File moved to trash successfully", from: "hello.txt", to: ".beamdrop_trash/hello.txt" }
```

### Step 14 — Create Shareable Links

```ts
async function createShareLink(
  path: string,
  options: { password?: string; expiresIn?: number } = {}
): Promise<any> {
  return request("POST", "/api/shares", {
    body: { path, ...options },
    token: authToken,
  });
}

// Share a file with password and 24h expiry
const shareResult = await createShareLink("projects/my-app/README.md", {
  password: "share-secret",
  expiresIn: 86400, // 24 hours in seconds
});
console.log("Share URL:", shareResult.url);
console.log("Token:", shareResult.token);

// List all shareable links
async function listShareLinks(): Promise<any[]> {
  return request("GET", "/api/shares/list", { token: authToken });
}

const shareLinks = await listShareLinks();
console.log("Active share links:", shareLinks.length);
```

### Step 15 — Access a Shareable Link (Public)

```ts
// Access without auth — this is a public endpoint
async function accessShareLink(
  shareToken: string,
  password?: string
): Promise<any> {
  if (password) {
    return request("POST", `/api/shares/access/${shareToken}`, {
      body: { password },
    });
  }
  return request("GET", `/api/shares/access/${shareToken}`);
}

// Access the password-protected share
const shareAccess = await accessShareLink(shareResult.token, "share-secret");
console.log("Shared file:", shareAccess);
// => { path: "projects/my-app/README.md", name: "README.md", size: "36 B", ... }
```

### Step 16 — Delete a Shareable Link

```ts
async function deleteShareLink(shareToken: string): Promise<any> {
  return request("DELETE", `/api/shares/delete?token=${shareToken}`, {
    token: authToken,
  });
}

const deleteShareResult = await deleteShareLink(shareResult.token);
console.log("Deleted share:", deleteShareResult);
```

### Step 17 — Create an API Key (S3 API)

```ts
async function createAPIKey(
  name: string,
  options: {
    permissions?: string;
    bucketScope?: string;
    expiresIn?: number;
  } = {}
): Promise<any> {
  return request("POST", "/api/v1/keys", {
    body: { name, ...options },
    token: authToken,
  });
}

const apiKey = await createAPIKey("my-app-key", {
  permissions: "read,write",
  expiresIn: 30 * 24 * 60 * 60, // 30 days
});
console.log("Access Key ID:", apiKey.accessKeyId);
console.log("Secret Key:", apiKey.secretKey); // SAVE THIS! Shown only once.

const ACCESS_KEY_ID: string = apiKey.accessKeyId;
const SECRET_KEY: string = apiKey.secretKey;
```

### Step 18 — HMAC Signature Helper

```ts
function generateSignature(
  secretKey: string,
  method: string,
  path: string,
  timestamp: string
): string {
  const message = `${method}\n${path}\n${timestamp}`;
  const hmac = crypto.createHmac("sha256", secretKey);
  hmac.update(message);
  return hmac.digest("base64");
}

function generatePresignedToken(
  secretKey: string,
  method: string,
  bucket: string,
  key: string,
  expiresAt: Date
): string {
  const message = `${method}\n${bucket}\n${key}\n${Math.floor(
    expiresAt.getTime() / 1000
  )}`;
  const hmac = crypto.createHmac("sha256", secretKey);
  hmac.update(message);
  return hmac.digest("base64url");
}

async function s3Request(
  method: string,
  path: string,
  options: { body?: any; stream?: ReadableStream | Buffer } = {}
): Promise<any> {
  const timestamp = new Date().toISOString();
  const signature = generateSignature(SECRET_KEY, method, path, timestamp);

  const headers: Record<string, string> = {
    Authorization: `Bearer ${ACCESS_KEY_ID}:${signature}`,
    "X-Beamdrop-Date": timestamp,
  };

  let body: any;
  if (options.stream) {
    body = options.stream;
  } else if (options.body) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }

  const res = await fetch(`${BASE_URL}${path}`, { method, headers, body });

  if (res.status === 204) return null;
  const contentType = res.headers.get("content-type") || "";
  if (contentType.includes("application/json")) {
    return res.json();
  }
  return res;
}
```

### Step 19 — Create & Manage Buckets

```ts
// Create a bucket
const createBucket = await s3Request("PUT", "/api/v1/buckets/my-bucket");
console.log("Created bucket:", createBucket);
// => { bucket: "my-bucket", created: "...", location: "/api/v1/buckets/my-bucket" }

// Create a bucket if it doesn't exist (idempotent)
const ensureBucket = await s3Request("PUT", "/api/v1/buckets/my-bucket?createIfNotExists=true");
console.log("Ensure bucket:", ensureBucket);
// => 201: { bucket: "my-bucket", created: "...", location: "..." } if new
// => 200: { bucket: "my-bucket", exists: true, location: "..." } if already existed

// List all buckets
const buckets = await s3Request("GET", "/api/v1/buckets");
console.log("Buckets:", buckets);
// => { buckets: [{ name: "my-bucket", createdAt: "..." }], count: 1 }

// Check if bucket exists (HEAD)
const timestamp = new Date().toISOString();
const sig = generateSignature(SECRET_KEY, "HEAD", "/api/v1/buckets/my-bucket", timestamp);
const headRes = await fetch(`${BASE_URL}/api/v1/buckets/my-bucket`, {
  method: "HEAD",
  headers: {
    Authorization: `Bearer ${ACCESS_KEY_ID}:${sig}`,
    "X-Beamdrop-Date": timestamp,
  },
});
console.log("Bucket exists:", headRes.status === 200);
```

### Step 20 — Upload Objects to Buckets

```ts
// Upload with raw body (PUT)
const putObject = await s3Request(
  "PUT",
  "/api/v1/buckets/my-bucket/configs/app.json",
  {
    stream: Buffer.from(JSON.stringify({ version: "1.0", debug: false })),
  }
);
console.log("PUT object:", putObject);
// => { bucket: "my-bucket", key: "configs/app.json", etag: "...", size: 38 }

// Upload with raw body — text file
const putTextFile = await s3Request(
  "PUT",
  "/api/v1/buckets/my-bucket/docs/readme.txt",
  {
    stream: Buffer.from("Welcome to my bucket!"),
  }
);
console.log("PUT text file:", putTextFile);

// Upload with multipart form (POST)
async function uploadToS3Multipart(
  bucket: string,
  key: string,
  content: Buffer,
  filename: string
): Promise<any> {
  const timestamp = new Date().toISOString();
  const path = `/api/v1/buckets/${bucket}/${key}`;
  const signature = generateSignature(SECRET_KEY, "POST", path, timestamp);

  const formData = new FormData();
  formData.append("file", new Blob([content]), filename);

  const res = await fetch(`${BASE_URL}${path}`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${ACCESS_KEY_ID}:${signature}`,
      "X-Beamdrop-Date": timestamp,
    },
    body: formData,
  });

  return res.json();
}

const multipartResult = await uploadToS3Multipart(
  "my-bucket",
  "uploads/data.csv",
  Buffer.from("name,age\nAlice,30\nBob,25"),
  "data.csv"
);
console.log("Multipart upload:", multipartResult);
```

### Step 21 — Download Objects

```ts
async function downloadObject(
  bucket: string,
  key: string
): Promise<string> {
  const res = await s3Request(
    "GET",
    `/api/v1/buckets/${bucket}/${key}`
  );
  if (res instanceof Response) {
    return res.text();
  }
  return JSON.stringify(res);
}

const objectContent = await downloadObject("my-bucket", "docs/readme.txt");
console.log("Object content:", objectContent);
// => "Welcome to my bucket!"
```

### Step 22 — List Objects with Prefix & Delimiter

```ts
// List all objects
const allObjects = await s3Request("GET", "/api/v1/buckets/my-bucket");
console.log("All objects:", allObjects);

// List with prefix filter
const configObjects = await s3Request(
  "GET",
  "/api/v1/buckets/my-bucket?prefix=configs/"
);
console.log("Config objects:", configObjects);

// List virtual directories using delimiter
const virtualDirs = await s3Request(
  "GET",
  "/api/v1/buckets/my-bucket?delimiter=/"
);
console.log("Top-level keys:", virtualDirs.contents);
console.log("Virtual directories:", virtualDirs.commonPrefixes);
// => ["configs/", "docs/", "uploads/"]

// Paginate results
const pagedObjects = await s3Request(
  "GET",
  "/api/v1/buckets/my-bucket?max-keys=2"
);
console.log("Truncated:", pagedObjects.isTruncated);
```

### Step 23 — Object Metadata (HEAD)

```ts
async function headObject(
  bucket: string,
  key: string
): Promise<Record<string, string>> {
  const timestamp = new Date().toISOString();
  const path = `/api/v1/buckets/${bucket}/${key}`;
  const signature = generateSignature(SECRET_KEY, "HEAD", path, timestamp);

  const res = await fetch(`${BASE_URL}${path}`, {
    method: "HEAD",
    headers: {
      Authorization: `Bearer ${ACCESS_KEY_ID}:${signature}`,
      "X-Beamdrop-Date": timestamp,
    },
  });

  return {
    contentLength: res.headers.get("content-length") || "",
    contentType: res.headers.get("content-type") || "",
    lastModified: res.headers.get("last-modified") || "",
    etag: res.headers.get("etag") || "",
    status: String(res.status),
  };
}

const metadata = await headObject("my-bucket", "configs/app.json");
console.log("Object metadata:", metadata);
// => { contentLength: "38", contentType: "application/json", ... }
```

### Step 24 — Generate Presigned URL

Beamdrop supports two methods for presigned URLs. You can use either or both.

#### Method 1: Client-Side HMAC Presigned URL

```ts
function generatePresignedUrl(
  bucket: string,
  key: string,
  expiresInSeconds: number
): string {
  const expiresAt = new Date(Date.now() + expiresInSeconds * 1000);
  const token = generatePresignedToken(
    SECRET_KEY,
    "GET",
    bucket,
    key,
    expiresAt
  );
  const expiresFormatted = expiresAt
    .toISOString()
    .replace(/[-:]/g, "")
    .replace(/\.\d{3}/, "");
  // Format: 20250115T103000Z

  return (
    `${BASE_URL}/api/v1/buckets/${bucket}/${key}` +
    `?token=${encodeURIComponent(token)}` +
    `&expires=${expiresFormatted}` +
    `&access_key=${ACCESS_KEY_ID}`
  );
}

const presignedUrl = generatePresignedUrl("my-bucket", "docs/readme.txt", 3600);
console.log("Presigned URL (valid 1 hour):", presignedUrl);
// → https://server/api/v1/buckets/my-bucket/docs/readme.txt?token=...&expires=...&access_key=BDK_xxx
```

#### Method 2: Server-Side Pretty Presigned URL

```ts
// Create a pretty presigned URL via the server-side registry
const prettyUrl = await s3Request("POST", "/api/v1/presign", {
  body: {
    bucket: "my-bucket",
    key: "docs/readme.txt",
    expiresIn: 3600,
    maxDownloads: 50,
  },
});
console.log("Pretty URL:", prettyUrl.url);
// → https://server/dl/a1b2c3d4e5f6789a...

// Anyone can download using this URL without API key auth
// const res = await fetch(prettyUrl.url);

// Revoke a pretty presigned URL
await s3Request("DELETE", `/api/v1/presign/${prettyUrl.token}`);

// List all pretty presigned URLs
const allUrls = await s3Request("GET", "/api/v1/presign");
console.log("Active URLs:", allUrls.count);
```

Both methods produce URLs that anyone can download from without authentication. Use client-side for quick ephemeral links; use server-side for clean URLs with download tracking and revocation.

### Step 25 — Delete Objects

```ts
// Delete a single object
const deleteResult = await s3Request(
  "DELETE",
  "/api/v1/buckets/my-bucket/uploads/data.csv"
);
console.log("Deleted object:", deleteResult === null ? "success" : deleteResult);
```

### Step 26 — Delete Bucket

```ts
// Must delete all objects first, then delete the empty bucket
await s3Request("DELETE", "/api/v1/buckets/my-bucket/configs/app.json");
await s3Request("DELETE", "/api/v1/buckets/my-bucket/docs/readme.txt");

const deleteBucket = await s3Request("DELETE", "/api/v1/buckets/my-bucket");
console.log("Deleted bucket:", deleteBucket === null ? "success" : deleteBucket);
```

### Step 27 — List & Delete API Keys

```ts
// List all API keys
async function listAPIKeys(): Promise<any> {
  return request("GET", "/api/v1/keys", { token: authToken });
}

const keys = await listAPIKeys();
console.log("API keys:", keys);

// Delete the API key we created
async function deleteAPIKey(accessKeyId: string): Promise<void> {
  await request("DELETE", `/api/v1/keys?accessKeyId=${accessKeyId}`, {
    token: authToken,
  });
}

await deleteAPIKey(ACCESS_KEY_ID);
console.log("API key deleted");
```

### Step 28 — Health Checks

```ts
// Full health overview
const health = await request("GET", "/health");
console.log("Health:", health.status);
console.log("Components:", health.components);

// Liveness probe (cheapest — no I/O)
const live = await request("GET", "/health/live");
console.log("Live:", live.status);

// Readiness probe (checks DB + storage)
const ready = await request("GET", "/health/ready");
console.log("Ready:", ready.status);

// Startup probe
const startup = await request("GET", "/health/startup");
console.log("Startup:", startup.status);
```

### Step 29 — Server Stats

```ts
const stats = await request("GET", "/stats", { token: authToken });
console.log("Server Stats:");
console.log("  Downloads:", stats.downloads);
console.log("  Uploads:", stats.uploads);
console.log("  Requests:", stats.requests);
console.log("  Uptime since:", stats.startTime);
```

### Step 30 — Retrieve Logs

```ts
async function getLogs(options: {
  limit?: number;
  offset?: number;
  level?: string;
  search?: string;
} = {}): Promise<any> {
  const params = new URLSearchParams();
  if (options.limit) params.set("limit", String(options.limit));
  if (options.offset) params.set("offset", String(options.offset));
  if (options.level) params.set("level", options.level);
  if (options.search) params.set("search", options.search);

  return request("GET", `/api/logs?${params}`, { token: authToken });
}

// Get recent error logs
const errorLogs = await getLogs({ level: "error", limit: 10 });
console.log("Error logs:", errorLogs.returned, "of", errorLogs.total);

// Search logs for upload events
const uploadLogs = await getLogs({ search: "upload", limit: 50 });
console.log("Upload logs:", uploadLogs.returned);
```

### Step 31 — WebSocket Real-Time Stats

```ts
function connectStatsWebSocket(): void {
  const ws = new WebSocket(`ws://localhost:7777/ws/stats`);

  ws.onopen = () => {
    console.log("WebSocket connected for real-time stats");
  };

  ws.onmessage = (event) => {
    const stats = JSON.parse(event.data);
    console.log("Live stats update:");
    console.log("  Downloads:", stats.downloads);
    console.log("  Uploads:", stats.uploads);
    console.log("  Requests:", stats.requests);
    if (stats.system) {
      console.log("  Memory:", stats.system.memory.percent + "%");
      console.log("  Disk:", stats.system.disk.percent + "%");
    }
  };

  ws.onerror = (error) => {
    console.error("WebSocket error:", error);
  };

  ws.onclose = () => {
    console.log("WebSocket disconnected");
  };

  // The server pings every 30s to keep alive
  // Stats update every 60s
}

connectStatsWebSocket();
```

### Step 32 — Logout

```ts
async function logout(): Promise<any> {
  return request("POST", "/auth/logout", { token: authToken });
}

const logoutResult = await logout();
console.log("Logout:", logoutResult.message);
// => "Logged out successfully"
```

---

## Error Codes Reference

All API errors follow a structured JSON format:

```json
{
  "code": "BUCKET_NOT_FOUND",
  "category": "NOT_FOUND",
  "message": "Bucket 'my-bucket' not found"
}
```

### Categories

| Category | Description |
|----------|-------------|
| `VALIDATION` | Input validation errors |
| `STORAGE` | Storage/filesystem errors |
| `AUTH` | Authentication/authorization |
| `NOT_FOUND` | Resource not found |
| `CONFLICT` | Resource conflict |
| `RATE_LIMIT` | Rate limiting |
| `INTERNAL` | Internal server errors |
| `UNAVAILABLE` | Service unavailable |

### Validation Codes

| Code | HTTP | Description |
|------|------|-------------|
| `INVALID_REQUEST` | 400 | Malformed request body or parameters |
| `INVALID_BUCKET_NAME` | 400 | Bucket name doesn't meet naming rules |
| `INVALID_OBJECT_KEY` | 400 | Invalid object key (empty, traversal, too long) |
| `INVALID_PATH` | 400 | Path traversal attempt or invalid path |
| `INVALID_MIME_TYPE` | 415 | File MIME type not in allowed list |
| `FILE_TOO_LARGE` | 413 | File exceeds 100 MB limit |
| `MISSING_FIELD` | 400 | Required field missing |

### Storage Codes

| Code | HTTP | Description |
|------|------|-------------|
| `OBJECT_LOCKED` | 423 | Object is locked by another operation |
| `WRITE_FAILED` | 500 | Failed to write file |
| `READ_FAILED` | 500 | Failed to read file |
| `DELETE_FAILED` | 500 | Failed to delete file |
| `IO_ERROR` | 500 | General filesystem I/O error |

### Auth Codes

| Code | HTTP | Description |
|------|------|-------------|
| `UNAUTHORIZED` | 401 | Missing or invalid auth credentials |
| `FORBIDDEN` | 403 | Invalid signature or insufficient permissions |
| `INVALID_TOKEN` | 401 | JWT token is invalid |
| `TOKEN_EXPIRED` | 401 | JWT or timestamp expired |
| `INVALID_API_KEY` | 401 | API key not found or disabled |
| `INVALID_PASSWORD` | 401 | Wrong password |

### Not Found Codes

| Code | HTTP | Description |
|------|------|-------------|
| `BUCKET_NOT_FOUND` | 404 | Bucket doesn't exist |
| `OBJECT_NOT_FOUND` | 404 | Object doesn't exist |
| `FILE_NOT_FOUND` | 404 | File doesn't exist |

### Conflict Codes

| Code | HTTP | Description |
|------|------|-------------|
| `BUCKET_EXISTS` | 409 | Bucket already exists |
| `FILE_EXISTS` | 409 | File already exists |
| `BUCKET_NOT_EMPTY` | 409 | Cannot delete non-empty bucket |

### Rate Limit Codes

| Code | HTTP | Description |
|------|------|-------------|
| `RATE_LIMIT_EXCEEDED` | 429 | Per-IP rate limit exceeded |
| `TOO_MANY_REQUESTS` | 429 | Too many requests |

---

## Storage Structure

```
shared-directory/
├── buckets/                    # S3-compatible bucket storage
│   ├── my-bucket/
│   │   ├── images/
│   │   │   └── photo.jpg
│   │   └── data.json
│   └── backups/
│       └── db.sql
├── .beamdrop/                  # Server logs
│   └── beamdrop.log            # Structured JSON log file
├── .beamdrop_data/             # Internal SQLite database
│   └── beamdrop.db
├── .beamdrop_trash/            # Soft-deleted files (recoverable)
│   └── deleted-file.txt
├── documents/                  # User files (flat storage)
│   └── report.pdf
└── photo.jpg
```

---

## Docker & Deployment

### Docker Run

```bash
docker run -d \
  --name beamdrop \
  -p 7777:7777 \
  -v beamdrop-data:/data \
  -e BEAMDROP_PASSWORD="secret" \
  -e BEAMDROP_API_AUTH=true \
  -e BEAMDROP_RATE_LIMIT=100 \
  beamdrop
```

### Docker Compose

```yaml
# docker-compose.yml
services:
  beamdrop:
    build: .
    ports:
      - "7777:7777"
    volumes:
      - ./data:/data
    environment:
      BEAMDROP_PORT: ${BEAMDROP_PORT:-7777}
      BEAMDROP_PASSWORD: ${BEAMDROP_PASSWORD:-}
      BEAMDROP_API_AUTH: ${BEAMDROP_API_AUTH:-false}
      BEAMDROP_RATE_LIMIT: ${BEAMDROP_RATE_LIMIT:-100}
      BEAMDROP_LOG_LEVEL: ${BEAMDROP_LOG_LEVEL:-info}
      BEAMDROP_QR: ${BEAMDROP_QR:-false}
      BEAMDROP_ALLOWED_ORIGINS: ${BEAMDROP_ALLOWED_ORIGINS:-}
      BEAMDROP_DB_PATH: ${BEAMDROP_DB_PATH:-}
      BEAMDROP_TLS_CERT: ${BEAMDROP_TLS_CERT:-}
      BEAMDROP_TLS_KEY: ${BEAMDROP_TLS_KEY:-}
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:7777/health/live"]
      interval: 30s
      timeout: 5s
      retries: 3
```

### Kubernetes

Use the health probes for K8s deployment:

```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 7777
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /health/ready
    port: 7777
  initialDelaySeconds: 10
  periodSeconds: 15

startupProbe:
  httpGet:
    path: /health/startup
    port: 7777
  failureThreshold: 30
  periodSeconds: 2
```

### Prometheus Scrape Config

```yaml
scrape_configs:
  - job_name: beamdrop
    static_configs:
      - targets: ["localhost:7777"]
```

Import the Grafana dashboard from `docs/grafana-dashboard.json`.

### Caddy Reverse Proxy (Auto HTTPS)

```
{$BEAMDROP_DOMAIN} {
  reverse_proxy beamdrop:7777
}
```

### Rate Limiting Tiers

When rate limiting is enabled (`-rate-limit N`):

| Tier | Rate | Routes |
|------|------|--------|
| General | N req/min | All endpoints |
| Auth | N/20 req/min (min 1) | `/auth/login` |
| Upload | N/10 req/min (min 1) | `/upload` |

---

## API Endpoint Summary

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/auth/status` | No | Check auth status |
| `POST` | `/auth/login` | No | Login with password |
| `POST` | `/auth/logout` | Yes | Logout (clear session) |
| `GET` | `/files` | Yes | List files |
| `POST` | `/upload` | Yes | Upload file |
| `GET` | `/download` | Yes | Download file |
| `POST` | `/mkdir` | Yes | Create directory |
| `POST` | `/move` | Yes | Move file |
| `POST` | `/copy` | Yes | Copy file |
| `POST` | `/rename` | Yes | Rename file |
| `POST` | `/trash` | Yes | Trash file |
| `POST` | `/write` | Yes | Write file content |
| `GET` | `/search` | Yes | Search files |
| `POST` | `/star` | Yes | Toggle star |
| `GET` | `/starred` | Yes | List starred files |
| `POST` | `/api/shares` | Yes | Create shareable link |
| `GET` | `/api/shares/list` | Yes | List shareable links |
| `DELETE` | `/api/shares/delete` | Yes | Delete shareable link |
| `GET/POST` | `/api/shares/access/{token}` | No | Access shared content |
| `GET` | `/api/v1/keys` | Session | List API keys |
| `POST` | `/api/v1/keys` | Session | Create API key |
| `DELETE` | `/api/v1/keys` | Session | Delete API key |
| `GET` | `/api/v1/buckets` | API Key | List buckets |
| `PUT` | `/api/v1/buckets/{name}` | API Key | Create bucket |
| `PUT` | `/api/v1/buckets/{name}?createIfNotExists=true` | API Key | Create bucket (idempotent) |
| `HEAD` | `/api/v1/buckets/{name}` | API Key | Check bucket |
| `GET` | `/api/v1/buckets/{name}` | API Key | Bucket info / list objects |
| `DELETE` | `/api/v1/buckets/{name}` | API Key | Delete bucket |
| `PUT` | `/api/v1/buckets/{b}/{key}` | API Key | Upload object (raw) |
| `POST` | `/api/v1/buckets/{b}/{key}` | API Key | Upload object (multipart) |
| `GET` | `/api/v1/buckets/{b}/{key}` | API Key | Download object |
| `HEAD` | `/api/v1/buckets/{b}/{key}` | API Key | Object metadata |
| `DELETE` | `/api/v1/buckets/{b}/{key}` | API Key | Delete object |
| `GET` | `/stats` | Yes | Server statistics |
| `GET` | `/api/logs` | Yes | Server logs |
| `GET` | `/health` | No | Health overview |
| `GET` | `/health/live` | No | Liveness probe |
| `GET` | `/health/ready` | No | Readiness probe |
| `GET` | `/health/startup` | No | Startup probe |
| `GET` | `/metrics` | No | Prometheus metrics |
| `WS` | `/ws/stats` | No | Real-time stats |

---

*Beamdrop is licensed under the [GNU Affero General Public License v3.0](../LICENSE).*
