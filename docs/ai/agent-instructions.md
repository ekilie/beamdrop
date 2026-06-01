# Beamdrop Agent Instructions

This document provides everything an AI agent needs to interact with a Beamdrop file storage server via its HTTP API.

## What is Beamdrop?

Beamdrop is a self-hosted file storage server with an S3-compatible REST API. You can use it to upload files, download files, manage storage buckets, and generate shareable download links.

## Connection

You need three values:

- **Base URL**: The server address (e.g., `http://localhost:7777`)
- **Access Key ID**: Public key identifier (format: `BDK_*`)
- **Secret Key**: Private signing key (format: `sk_*`)

## Authentication

Every API request must be signed with HMAC-SHA256.

### Signing Algorithm

1. Get the current time in UTC, formatted as ISO 8601 (e.g., `2024-01-15T10:30:00Z`)
2. Build the string to sign:
   ```
   StringToSign = HTTP_METHOD + "\n" + REQUEST_PATH + "\n" + TIMESTAMP
   ```
   Example: `GET\n/api/v1/buckets\n2024-01-15T10:30:00Z`
3. Compute the signature:
   ```
   Signature = Base64(HMAC-SHA256(StringToSign, SecretKey))
   ```
4. Add these headers to every request:
   ```
   Authorization: Bearer ACCESS_KEY_ID:SIGNATURE
   X-Beamdrop-Date: TIMESTAMP
   ```

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

**Bucket name rules**: 3-63 chars, lowercase alphanumeric + hyphens/dots, start/end with letter/number.

### Objects

| Method   | Path                                                      | Description                         |
| -------- | --------------------------------------------------------- | ----------------------------------- |
| `PUT`    | `/api/v1/buckets/{bucket}/{key}`                          | Upload object (body = file content) |
| `GET`    | `/api/v1/buckets/{bucket}/{key}`                          | Download object                     |
| `HEAD`   | `/api/v1/buckets/{bucket}/{key}`                          | Get object metadata (no body)       |
| `DELETE` | `/api/v1/buckets/{bucket}/{key}`                          | Delete object                       |
| `GET`    | `/api/v1/buckets/{bucket}?list=true&prefix=X&delimiter=/` | List objects                        |

**Object key rules**: Max 1024 bytes, no `..`, no leading `/`.

**List parameters**:

- `prefix` — Filter by key prefix (e.g., `folder/`)
- `delimiter` — Group by delimiter (use `/` for directory-like listing)
- `max-keys` — Max results (default 1000)

### Presigned URLs

| Method   | Path                      | Description                                  |
| -------- | ------------------------- | -------------------------------------------- |
| `POST`   | `/api/v1/presign`         | Create presigned download URL                |
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

### API Keys

| Method   | Path                | Description                        |
| -------- | ------------------- | ---------------------------------- |
| `POST`   | `/api/v1/keys`      | Create API key (secret shown once) |
| `GET`    | `/api/v1/keys`      | List API keys (no secrets)         |
| `DELETE` | `/api/v1/keys/{id}` | Delete API key                     |

## Response Formats

### Success responses

**List buckets**:

```json
{
  "buckets": [{ "name": "my-bucket", "created_at": "2024-01-15T10:30:00Z" }],
  "count": 1
}
```

**Create bucket**:

```json
{
  "bucket": "my-bucket",
  "created": true,
  "location": "/api/v1/buckets/my-bucket"
}
```

**Upload object**:

```json
{
  "bucket": "my-bucket",
  "key": "file.txt",
  "etag": "abc123",
  "size": 1234,
  "url": "/api/v1/buckets/my-bucket/file.txt"
}
```

**List objects**:

```json
{
  "bucket": "my-bucket",
  "prefix": "",
  "delimiter": "/",
  "max_keys": 1000,
  "is_truncated": false,
  "contents": [
    {
      "key": "file.txt",
      "size": 1234,
      "last_modified": "2024-01-15T10:30:00Z",
      "etag": "abc123",
      "content_type": "text/plain"
    }
  ],
  "common_prefixes": ["folder/"]
}
```

**Download object**: Returns raw file content with `Content-Type`, `Content-Length`, `ETag`, `Last-Modified` headers.

**HEAD object**: Same headers as download, but no body.

### Error responses

```json
{
  "error": "BUCKET_NOT_FOUND",
  "category": "NOT_FOUND",
  "message": "The specified bucket does not exist",
  "retryable": false,
  "retry_after": 0
}
```

| HTTP Status | Error Code            | Meaning                                       | Retry?                         |
| ----------- | --------------------- | --------------------------------------------- | ------------------------------ |
| 400         | INVALID_BUCKET_NAME   | Bad bucket name                               | No                             |
| 400         | INVALID_OBJECT_KEY    | Bad object key                                | No                             |
| 401         | UNAUTHORIZED          | Bad credentials or timestamp                  | No                             |
| 404         | BUCKET_NOT_FOUND      | Bucket doesn't exist                          | No                             |
| 404         | OBJECT_NOT_FOUND      | Object doesn't exist                          | No                             |
| 409         | BUCKET_ALREADY_EXISTS | Bucket exists (use `?createIfNotExists=true`) | No                             |
| 409         | BUCKET_NOT_EMPTY      | Must delete objects first                     | No                             |
| 429         | RATE_LIMIT_EXCEEDED   | Too many requests                             | Yes (see `Retry-After` header) |

## Common Agent Workflows

### 1. Store a generated file

```
1. PUT /api/v1/buckets/ai-outputs?createIfNotExists=true  (create bucket)
2. PUT /api/v1/buckets/ai-outputs/session-123/result.json  (upload file, body = content)
3. POST /api/v1/presign  (create shareable link)
   Body: {"bucket": "ai-outputs", "key": "session-123/result.json", "method": "GET", "expires_in": 86400}
4. Return the presigned URL to the user
```

### 2. Read a file from storage

```
1. GET /api/v1/buckets/my-bucket/config.json  (download file)
2. Parse the response body
```

### 3. List and browse files

```
1. GET /api/v1/buckets  (list buckets)
2. GET /api/v1/buckets/my-bucket?list=true&delimiter=/  (list top-level)
3. GET /api/v1/buckets/my-bucket?list=true&prefix=folder/&delimiter=/  (list subfolder)
```

### 4. Clean up old files

```
1. GET /api/v1/buckets/temp?list=true&prefix=old/  (find files to delete)
2. DELETE /api/v1/buckets/temp/old/file1.txt  (delete each file)
3. DELETE /api/v1/buckets/temp/old/file2.txt
```

## Rate Limits

- General: 100 requests/minute per IP
- Upload (PUT objects): 10 requests/minute per IP
- If you get 429, wait for the number of seconds in the `Retry-After` response header

## Tips for Agents

- Always use `?createIfNotExists=true` when creating buckets to avoid 409 errors
- Use structured key paths like `{purpose}/{session-id}/{filename}` for organization
- Timestamps must be within 15 minutes of server time (use UTC)
- For large files, consider generating a presigned URL and letting the user download directly
- The `/dl/{token}` presigned URL endpoint requires no authentication — share it freely
- Object ETags are MD5 hashes of content — use for deduplication
