# Beamdrop Agent Instructions

This document provides everything an AI agent needs to interact with a Beamdrop file storage server via its HTTP API. It covers authentication, all API endpoints with exact request/response formats, error handling, presigned URL strategies, and recommended workflows.

## What is Beamdrop?

Beamdrop is a self-hosted, single-binary file storage server with an S3-compatible REST API built in Go. It provides bucket-based object storage with web UI, presigned URLs, shareable links, real-time stats, and Prometheus metrics. Runs on Linux, macOS, and Windows (Intel + ARM).

## Connection

You need three values:

- **Base URL**: The server address (e.g., `http://localhost:7777`)
- **Access Key ID**: Public key identifier (format: `BDK_` + 16 hex chars, e.g., `BDK_a1b2c3d4e5f67890`)
- **Secret Key**: Private signing key (format: `sk_` + 40 hex chars, e.g., `sk_0123456789abcdef...`)

Keys are created with `POST /api/v1/keys`. The secret key is shown **only once** at creation — it cannot be retrieved later. It is encrypted at rest with AES-256-GCM.

## Authentication

Every `/api/v1/` request must be signed with HMAC-SHA256 when `-api-auth` is enabled.

### Signing Algorithm

1. Get the current time in UTC, formatted as RFC3339 (e.g., `2024-01-15T10:30:00Z`)
2. Build the string to sign:

   ```
   StringToSign = HTTP_METHOD + "\n" + REQUEST_PATH + "\n" + TIMESTAMP
   ```

   - **REQUEST_PATH** is the URL path only — no query string. Example: `/api/v1/buckets/my-bucket`
   - Example: `GET\n/api/v1/buckets\n2024-01-15T10:30:00Z`

3. Compute the signature:
   ```
   Signature = Base64(HMAC-SHA256(StringToSign, SecretKey))
   ```
   Use **standard Base64** encoding (not URL-safe).
4. Add these headers to every request:
   ```
   Authorization: Bearer ACCESS_KEY_ID:SIGNATURE
   X-Beamdrop-Date: TIMESTAMP
   ```

**Clock skew tolerance:** ±15 minutes. Requests outside this window are rejected with 401 UNAUTHORIZED.

### API Key Properties

| Property      | Type   | Description                                                           |
| ------------- | ------ | --------------------------------------------------------------------- |
| `accessKeyId` | string | `BDK_` + 16 hex chars. Public identifier used in Authorization header |
| `secretKey`   | string | `sk_` + 40 hex chars. Shown only at creation. Used for HMAC signing   |
| `permissions` | string | `"read"`, `"write"`, or `"read,write"`                                |
| `bucketScope` | string | Optional. Restricts key to operations on a single bucket              |
| `expiresAt`   | string | Optional. RFC3339 timestamp. Key auto-expires                         |
| `disabled`    | bool   | Can be set to temporarily disable without deleting                    |
| `lastUsedAt`  | string | Auto-tracked. Last time this key was used                             |
| `name`        | string | Optional human-readable label                                         |

### Signing Examples

**curl**:

```bash
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
METHOD="GET"
PATH="/api/v1/buckets"
STRING_TO_SIGN="${METHOD}\n${PATH}\n${TIMESTAMP}"
SIGNATURE=$(printf "${STRING_TO_SIGN}" | openssl dgst -sha256 -hmac "${SECRET_KEY}" -binary | base64)

curl -H "Authorization: Bearer ${ACCESS_KEY_ID}:${SIGNATURE}" \
     -H "X-Beamdrop-Date: ${TIMESTAMP}" \
     "${BASE_URL}${PATH}"
```

**Python**:

```python
import hmac, hashlib, base64, requests
from datetime import datetime, timezone

timestamp = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
method = "GET"
path = "/api/v1/buckets"
string_to_sign = f"{method}\n{path}\n{timestamp}"
signature = base64.b64encode(
    hmac.new(secret_key.encode(), string_to_sign.encode(), hashlib.sha256).digest()
).decode()

response = requests.get(
    f"{base_url}{path}",
    headers={
        "Authorization": f"Bearer {access_key_id}:{signature}",
        "X-Beamdrop-Date": timestamp,
    },
)
```

**JavaScript/TypeScript**:

```javascript
const crypto = require("crypto");

const timestamp = new Date().toISOString().replace(/\.\d{3}Z$/, "Z");
const method = "GET";
const path = "/api/v1/buckets";
const stringToSign = `${method}\n${path}\n${timestamp}`;
const signature = crypto
  .createHmac("sha256", secretKey)
  .update(stringToSign)
  .digest("base64");

const response = await fetch(`${baseUrl}${path}`, {
  headers: {
    Authorization: `Bearer ${accessKeyId}:${signature}`,
    "X-Beamdrop-Date": timestamp,
  },
});
```

## API Endpoints

### Buckets

| Method   | Path                                            | Description                |
| -------- | ----------------------------------------------- | -------------------------- |
| `GET`    | `/api/v1/buckets`                               | List all buckets           |
| `PUT`    | `/api/v1/buckets/{name}`                        | Create a bucket            |
| `PUT`    | `/api/v1/buckets/{name}?createIfNotExists=true` | Create bucket (idempotent) |
| `HEAD`   | `/api/v1/buckets/{name}`                        | Check if bucket exists     |
| `DELETE` | `/api/v1/buckets/{name}`                        | Delete empty bucket        |

**Bucket name validation (exact rules):**

- Length: 3-63 characters
- Regex: `^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`
- Characters: lowercase a-z, digits 0-9, hyphens `-`, dots `.`
- Must start and end with a letter or digit
- IP-like names rejected (e.g., `192.168.1.1` → INVALID_BUCKET_NAME)
- Valid: `my-bucket`, `data.2024`, `ai-artifacts`
- Invalid: `-bucket`, `MY-BUCKET`, `ab` (too short), `a-bucket-that-exceeds-sixty-three-characters-limit-in-length-here`

### Objects

| Method   | Path                                                      | Description                      |
| -------- | --------------------------------------------------------- | -------------------------------- |
| `PUT`    | `/api/v1/buckets/{bucket}/{key}`                          | Upload object (body = raw bytes) |
| `GET`    | `/api/v1/buckets/{bucket}/{key}`                          | Download object                  |
| `HEAD`   | `/api/v1/buckets/{bucket}/{key}`                          | Get object metadata (no body)    |
| `DELETE` | `/api/v1/buckets/{bucket}/{key}`                          | Delete object                    |
| `GET`    | `/api/v1/buckets/{bucket}?list=true&prefix=X&delimiter=/` | List objects                     |

**Object key validation:**

- Cannot be empty
- Cannot contain `..` (path traversal prevention)
- Cannot start with `/`
- Maximum 1024 bytes (S3 compatible)
- Forward slashes `/` create virtual directory hierarchies
- Valid: `file.txt`, `folder/sub/file.pdf`, `2024/01/report.csv`
- Invalid: `../etc/passwd`, `/absolute`, empty string

**Upload limits:**

- Maximum file size: 5GB (5,242,880,000 bytes)
- Writes are atomic: temp file (`.beamdrop_tmp_*`) → fsync → rename → fsync parent dir. Crash-safe — old file preserved on failure
- Per-object write locks prevent concurrent writes (30-second timeout)
- ETag = MD5 hex hash of uploaded content

**List parameters:**

- `list=true` — Required to trigger list mode (otherwise treated as object download)
- `prefix` — Filter by key prefix (e.g., `folder/`)
- `delimiter` — Group by delimiter (use `/` for directory-like listing). Objects between prefix and next delimiter become `commonPrefixes`
- `max-keys` — Max results (default 1000)

### Presigned URLs

| Method   | Path                      | Description                                  |
| -------- | ------------------------- | -------------------------------------------- |
| `POST`   | `/api/v1/presign`         | Create server-side presigned URL             |
| `GET`    | `/api/v1/presign`         | List all presigned URLs                      |
| `GET`    | `/api/v1/presign/{token}` | Get presigned URL details                    |
| `DELETE` | `/api/v1/presign/{token}` | Revoke presigned URL                         |
| `GET`    | `/dl/{token}`             | Download via presigned URL (public, no auth) |

**Create presigned URL** request body:

```json
{
  "bucket": "my-bucket",
  "key": "file.txt",
  "method": "GET",
  "expires_in": 3600,
  "max_downloads": 10
}
```

Fields:

- `bucket` (required): Target bucket name
- `key` (required): Target object key
- `method` (optional, default "GET"): HTTP method allowed
- `expires_in` (optional): Seconds until expiry. Omit for no expiry
- `max_downloads` (optional): Maximum download count. Omit for unlimited

### API Keys

| Method   | Path                                | Description                        |
| -------- | ----------------------------------- | ---------------------------------- |
| `POST`   | `/api/v1/keys`                      | Create API key (secret shown once) |
| `GET`    | `/api/v1/keys`                      | List API keys (no secrets)         |
| `DELETE` | `/api/v1/keys?accessKeyId=BDK_xxxx` | Delete API key by access key ID    |

**Important:** Delete uses query parameter `?accessKeyId=BDK_xxxx`, NOT a path parameter.

**Create API key** request body:

```json
{
  "name": "my-key",
  "permissions": "read,write",
  "bucket_scope": "optional-bucket",
  "expires_in": "720h"
}
```

## Response Formats

### Success responses

**List buckets** — `GET /api/v1/buckets`:

```json
{
  "buckets": [{ "name": "my-bucket", "createdAt": "2024-01-15T10:30:00Z" }],
  "count": 1
}
```

**Create bucket** — `PUT /api/v1/buckets/{name}`:

```json
{
  "bucket": "my-bucket",
  "created": "2024-01-15T10:30:00Z",
  "location": "/api/v1/buckets/my-bucket"
}
```

**Create bucket (idempotent, already existed)** — `PUT /api/v1/buckets/{name}?createIfNotExists=true`:

```json
{
  "bucket": "my-bucket",
  "exists": true,
  "location": "/api/v1/buckets/my-bucket"
}
```

**Upload object** — `PUT /api/v1/buckets/{bucket}/{key}`:

```json
{
  "bucket": "my-bucket",
  "key": "file.txt",
  "etag": "d41d8cd98f00b204e9800998ecf8427e",
  "size": 1234,
  "url": "/api/v1/buckets/my-bucket/file.txt"
}
```

**List objects** — `GET /api/v1/buckets/{bucket}?list=true`:

```json
{
  "bucket": "my-bucket",
  "prefix": "",
  "delimiter": "/",
  "maxKeys": 1000,
  "isTruncated": false,
  "contents": [
    {
      "key": "file.txt",
      "size": 1234,
      "lastModified": "2024-01-15T10:30:00Z",
      "etag": "d41d8cd98f00b204e9800998ecf8427e",
      "contentType": "text/plain"
    }
  ],
  "commonPrefixes": [{ "prefix": "folder/" }]
}
```

**Download object** — `GET /api/v1/buckets/{bucket}/{key}`: Returns raw file content with headers:

- `Content-Type`: MIME type
- `Content-Length`: size in bytes
- `ETag`: quoted MD5 hash (e.g., `"abc123"`)
- `Last-Modified`: RFC 2822 format
- Supports `Range` header for partial downloads

**HEAD object**: Same headers as download, no body. Use for existence checks and metadata.

**Create API key** — `POST /api/v1/keys`:

```json
{
  "id": 1,
  "name": "my-key",
  "accessKeyId": "BDK_a1b2c3d4e5f67890",
  "secretKey": "sk_0123456789abcdef0123456789abcdef01234567",
  "permissions": "read,write",
  "bucketScope": "optional-bucket",
  "expiresAt": "2024-02-15T10:30:00Z",
  "createdAt": "2024-01-15T10:30:00Z",
  "warning": "Save the secret key now — it cannot be retrieved later"
}
```

⚠️ `secretKey` is shown ONLY in the creation response. Store it immediately.

**Create presigned URL** — `POST /api/v1/presign`:

```json
{
  "id": 1,
  "token": "a1b2c3d4e5f67890a1b2c3d4e5f67890",
  "url": "https://server/dl/a1b2c3d4e5f67890a1b2c3d4e5f67890",
  "bucket": "my-bucket",
  "key": "file.txt",
  "method": "GET",
  "expiresAt": "2024-01-15T11:30:00Z",
  "maxDownloads": 10,
  "downloadCount": 0,
  "createdBy": "BDK_a1b2c3d4e5f67890",
  "createdAt": "2024-01-15T10:30:00Z"
}
```

### Error responses

All errors return consistent JSON:

```json
{
  "error": {
    "code": "BUCKET_NOT_FOUND",
    "category": "NOT_FOUND",
    "message": "The specified bucket does not exist",
    "details": {}
  }
}
```

**Complete error code reference:**

| HTTP | Code                | Category   | Retryable | Description                                     | Agent Action                                               |
| ---- | ------------------- | ---------- | --------- | ----------------------------------------------- | ---------------------------------------------------------- |
| 400  | INVALID_REQUEST     | VALIDATION | No        | Malformed request body or params                | Fix request format                                         |
| 400  | INVALID_BUCKET_NAME | VALIDATION | No        | Name doesn't match regex                        | Fix name: 3-63 chars, lowercase, start/end with alnum      |
| 400  | INVALID_OBJECT_KEY  | VALIDATION | No        | Key empty, has `..`, starts `/`, or >1024 bytes | Fix key path                                               |
| 400  | INVALID_PATH        | VALIDATION | No        | Request path invalid                            | Check URL construction                                     |
| 400  | MISSING_FIELD       | VALIDATION | No        | Required field missing                          | Add missing field to request body                          |
| 401  | UNAUTHORIZED        | AUTH       | No        | Missing/invalid Authorization header            | Check key ID, signing algorithm, timestamp ±15 min         |
| 401  | INVALID_TOKEN       | AUTH       | No        | Invalid JWT token (web auth)                    | Re-authenticate                                            |
| 401  | TOKEN_EXPIRED       | AUTH       | No        | JWT has expired                                 | Re-authenticate                                            |
| 401  | INVALID_API_KEY     | AUTH       | No        | API key not found or disabled                   | Verify key exists and is not disabled                      |
| 401  | API_KEY_EXPIRED     | AUTH       | No        | API key past expiresAt                          | Create a new key                                           |
| 403  | FORBIDDEN           | AUTH       | No        | Valid auth but insufficient permissions         | Use key with correct permissions                           |
| 403  | PERMISSION_DENIED   | AUTH       | No        | Key lacks required permission                   | Key needs "read" for GET, "write" for PUT/DELETE           |
| 404  | BUCKET_NOT_FOUND    | NOT_FOUND  | No        | Bucket doesn't exist                            | Create it first with `?createIfNotExists=true`             |
| 404  | OBJECT_NOT_FOUND    | NOT_FOUND  | No        | Object doesn't exist                            | Verify bucket and key spelling                             |
| 404  | FILE_NOT_FOUND      | NOT_FOUND  | No        | File not found (web UI)                         | Check path                                                 |
| 404  | LINK_NOT_FOUND      | NOT_FOUND  | No        | Shareable link token invalid                    | Token may have been revoked                                |
| 409  | BUCKET_EXISTS       | CONFLICT   | No        | Bucket already exists                           | Use `?createIfNotExists=true`                              |
| 409  | BUCKET_NOT_EMPTY    | CONFLICT   | No        | Can't delete bucket with objects                | Delete all objects first                                   |
| 409  | OBJECT_EXISTS       | CONFLICT   | No        | Object exists (create-only ops)                 | Object already uploaded                                    |
| 413  | FILE_TOO_LARGE      | VALIDATION | No        | Exceeds 5GB limit                               | Split or compress the file                                 |
| 415  | INVALID_MIME_TYPE   | VALIDATION | No        | Content type not in allowed list                | Check allowed MIME types                                   |
| 423  | OBJECT_LOCKED       | STORAGE    | **Yes**   | Another write in progress                       | Retry after 1-2 seconds. Lock timeout is 30s               |
| 429  | RATE_LIMIT_EXCEEDED | RATE_LIMIT | **Yes**   | Too many requests                               | Wait `Retry-After` header seconds. Has `X-Retryable: true` |
| 507  | STORAGE_FULL        | STORAGE    | No        | Server at max-storage limit                     | Cannot upload — server full                                |

## Presigned URL Types — Decision Guide

Beamdrop supports two types of presigned URLs. Choose deliberately:

### Type 1: Client-side HMAC (self-contained)

Generated locally using your secret key — no server API call needed.

```
Token = Base64URL(HMAC-SHA256("METHOD\nBUCKET\nKEY\nUNIX_TIMESTAMP", SecretKey))
URL = https://server/api/v1/buckets/{bucket}/{key}?access_key=BDK_xxx&expires=2024-01-16T10:30:00Z&token=TOKEN
```

**Pros:** Zero server overhead, works offline, batch-generate thousands instantly
**Cons:** Cannot be revoked, no download tracking, invalidated by key rotation, long URLs

### Type 2: Server-side (database-tracked, RECOMMENDED)

Created via `POST /api/v1/presign`. Stored in database with 32-char hex token.

```
URL = https://server/dl/a1b2c3d4e5f67890a1b2c3d4e5f67890
```

**Pros:** Revocable, download counting, download limits, audit trail (createdBy), short clean URLs, survives key rotation
**Cons:** Requires API call to create, stored in database

### When to use which

| Scenario                                  | Client-side | Server-side |
| ----------------------------------------- | :---------: | :---------: |
| Quick temporary link, no tracking         |     ✅      |             |
| Need to revoke access after sharing       |             |     ✅      |
| Limit number of downloads                 |             |     ✅      |
| Track download count                      |             |     ✅      |
| Batch-generate many links (zero overhead) |     ✅      |             |
| Share in download portal                  |             |     ✅      |
| Must survive API key rotation             |             |     ✅      |
| Sharing sensitive files with audit trail  |             |     ✅      |
| Embed in automated emails/notifications   |     ✅      |             |

**Default recommendation:** Use server-side URLs unless you have a specific reason for client-side.

## Rate Limits

Three independent token-bucket tiers per IP address:

| Tier    | Endpoints              | Default Rate | Burst |
| ------- | ---------------------- | ------------ | ----- |
| General | All endpoints          | 100 req/min  | 100   |
| Auth    | `/auth/login`          | 5 req/min    | 5     |
| Upload  | PUT objects, `/upload` | 10 req/min   | 10    |

- When rate-limited, response includes `Retry-After: <seconds>` header and `X-Retryable: true`
- Buckets start full and refill continuously
- Configure with `-rate-limit N` (auth = 5% of N, upload = 10% of N). Disable with `-rate-limit 0`
- IP detection: RemoteAddr by default. Only trusts X-Forwarded-For/X-Real-IP from `-trusted-proxies`

## Common Agent Workflows

### 1. Store a generated file with shareable link

```
1. PUT /api/v1/buckets/ai-outputs?createIfNotExists=true
   → 201 or 200 (idempotent — always safe)

2. PUT /api/v1/buckets/ai-outputs/session-123/result.json
   Content-Type: application/octet-stream
   Body: {"generated": "content"}
   → 200 {"bucket":"ai-outputs", "key":"session-123/result.json", "etag":"...", "size":...}

3. POST /api/v1/presign
   Body: {"bucket":"ai-outputs", "key":"session-123/result.json", "method":"GET", "expires_in":86400}
   → 201 {"url":"https://server/dl/abc123...", "token":"abc123..."}

4. Return the presigned URL to the user — they can download without authentication
```

### 2. Read a configuration file

```
1. GET /api/v1/buckets/config/app-settings.json
   → 200 with raw JSON body
   Headers: Content-Type: application/json, Content-Length: 1234

2. Parse the response body as JSON
```

### 3. Browse files (directory-style)

```
1. GET /api/v1/buckets
   → {"buckets": [...], "count": N}

2. GET /api/v1/buckets/my-bucket?list=true&delimiter=/
   → top-level files in "contents", "directories" in "commonPrefixes"

3. GET /api/v1/buckets/my-bucket?list=true&prefix=folder/&delimiter=/
   → files in folder/ and subdirectories as commonPrefixes
```

### 4. Upload with deduplication (ETag check)

```
1. HEAD /api/v1/buckets/my-bucket/file.txt
   → 200 with ETag header (or 404 if new)

2. Compare ETag to MD5 hex hash of new content
   If same → skip upload (content unchanged)
   If different → proceed to upload

3. PUT /api/v1/buckets/my-bucket/file.txt
   Body: <new content>
```

### 5. Clean up old files

```
1. GET /api/v1/buckets/temp?list=true&prefix=old/
   → list of objects to delete

2. For each object:
   DELETE /api/v1/buckets/temp/{key}
   → 204

3. If bucket should be removed:
   DELETE /api/v1/buckets/temp
   → 204 (only works if empty)
```

### 6. Create scoped API key for limited access

```
POST /api/v1/keys
Body: {"name":"readonly-reports", "permissions":"read", "bucket_scope":"reports", "expires_in":"720h"}
→ 201 {"accessKeyId":"BDK_...", "secretKey":"sk_...", ...}

This key can only read from the "reports" bucket and expires in 30 days.
```

### 7. Set up a webhook for object events

```
POST /api/v1/webhooks
Body: {"name":"object-watcher","url":"https://my-app.com/webhook","event_types":["beamdrop.object.*"]}
→ 201 {"webhook":{...}, "secret":"whsec_...", "warning":"Save the secret now..."}

Save the secret — use it to verify webhook signatures in your handler.
```

### 8. Verify webhook signature in your handler

```
1. Extract headers: X-Beamdrop-Signature, X-Beamdrop-Timestamp, X-Beamdrop-Delivery-Id
2. Build signature base: timestamp + "\n" + delivery_id + "\n" + request_body
3. Compute: expected = "v1=" + hex(HMAC-SHA256(signature_base, webhook_secret))
4. Compare expected == X-Beamdrop-Signature (constant-time comparison)
5. Reject if timestamp is too old (replay protection)
```

## Webhooks API Reference

Webhooks send HMAC-SHA256 signed HTTP POST requests to your endpoints when events occur.

### Create webhook

```
POST /api/v1/webhooks
Body: {"name":"my-hook", "url":"https://example.com/webhook", "event_types":["beamdrop.object.*"], "bucket_scope":"my-bucket"}
→ 201 {"webhook":{id, name, targetUrl, enabled, eventTypes, ...}, "secret":"whsec_...", "warning":"Save the secret now — it will not be shown again."}
```

### List webhooks

```
GET /api/v1/webhooks
→ 200 {"webhooks":[...], "count":N}
```

### Update webhook

```
PATCH /api/v1/webhooks/{id}
Body: {"name":"new-name", "enabled":false, "event_types":["beamdrop.object.created"], "rotate_secret":true}
→ 200 {"updated":true, "secret":"whsec_new_..."} (secret only if rotate_secret=true)
```

### Delete webhook

```
DELETE /api/v1/webhooks/{id}
→ 204
```

### Send test event

```
POST /api/v1/webhooks/{id}/test
→ 200 {"sent":true, "message":"Test event queued for delivery"}
```

### List deliveries

```
GET /api/v1/webhooks/{id}/deliveries
→ 200 {"deliveries":[{id, status, attemptCount, lastHttpStatus, lastError, ...}], "count":N}
```

### Event types

- `beamdrop.object.created`, `beamdrop.object.updated`, `beamdrop.object.deleted`
- `beamdrop.bucket.created`, `beamdrop.bucket.deleted`
- `beamdrop.share.created`, `beamdrop.share.deleted`
- `beamdrop.presign.created`, `beamdrop.presign.deleted`
- Wildcards: `beamdrop.object.*`, `beamdrop.bucket.*`, `beamdrop.share.*`, `beamdrop.presign.*`

### Webhook delivery headers

- `X-Beamdrop-Webhook-Id` — Webhook ID
- `X-Beamdrop-Event` — Event type
- `X-Beamdrop-Delivery-Id` — Unique delivery UUID
- `X-Beamdrop-Timestamp` — Unix timestamp
- `X-Beamdrop-Signature` — `v1=` + hex HMAC-SHA256 signature

### Delivery behavior

- Max 8 retry attempts with exponential backoff (2s base, 15min max, jitter)
- Retryable status codes: 408, 425, 429, 500, 502, 503, 504
- 10-second timeout per attempt
- Dead-lettered after max attempts exhausted
- 7-day retention for delivery history

## MCP Server

Beamdrop includes a built-in MCP (Model Context Protocol) server at `/mcp`. It uses Streamable HTTP transport with JSON-RPC 2.0.

- `GET /mcp` — Public discovery (no auth). Returns server info, protocol version, transport, available methods.
- `POST /mcp` — JSON-RPC 2.0 requests. Requires API key auth (same HMAC-SHA256 as S3 API).

**Methods:** `initialize`, `ping`, `tools/list`, `tools/call`

**16 tools available:** list_buckets, create_bucket, delete_bucket, bucket_exists, list_objects, put_object, get_object, head_object, delete_object, create_presigned_url, list_presigned_urls, get_presigned_url, delete_presigned_url, list_api_keys, create_api_key, delete_api_key

**Configure in Claude Desktop / VS Code:**

```json
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

## Tips for Agents

- **Always** use `?createIfNotExists=true` when creating buckets to avoid 409 errors
- Use structured key paths: `{purpose}/{session-id}/{filename}` for organization
- Timestamps must be within ±15 minutes of server time (use UTC)
- For large files, generate a presigned URL and let the user download directly via `/dl/{token}`
- The `/dl/{token}` endpoint requires **no authentication** — it's the public download URL
- Object ETags are MD5 hex hashes — use for deduplication (compare before uploading)
- When getting 401 errors, check: (1) key format is `BDK_*`, (2) timestamp is RFC3339 UTC, (3) signing path matches request path exactly (no query string), (4) Base64 is standard (not URL-safe)
- Permissions are `"read"`, `"write"`, or `"read,write"` — NOT `["GetObject", "PutObject"]`
- API key deletion uses query param `?accessKeyId=BDK_xxx`, not a path segment
- Lock errors (423) are transient — retry after 1-2 seconds
- If isTruncated=true in list response, there are more results to fetch
