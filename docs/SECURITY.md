# Security Features

This document describes the security hardening features added to Beamdrop.

> **See also:** [Security Threat Model](THREAT-MODEL.md) assets, trust boundaries, threat actors, attack vectors, and incident response.

## CORS (Cross-Origin Resource Sharing)

By default, CORS is **disabled** for maximum security. This is the recommended configuration for local file sharing as it prevents unauthorized cross-origin access.

### Enabling CORS

To enable CORS for specific origins, use the `-allowed-origins` flag:

```bash
./beamdrop -dir=/path/to/share -allowed-origins="http://localhost:3000,http://example.com"
```

When CORS is enabled:

- Only specified origins can make cross-origin requests
- Preflight (OPTIONS) requests are properly handled
- Credentials (cookies, auth headers) are allowed
- The following headers are set:
  - `Access-Control-Allow-Origin`: Set to the requesting origin if allowed
  - `Access-Control-Allow-Credentials`: true
  - `Access-Control-Allow-Methods`: GET, POST, OPTIONS
  - `Access-Control-Allow-Headers`: Content-Type, Authorization
  - `Access-Control-Max-Age`: 86400 (24 hours)

When CORS is disabled:

- No CORS headers are sent
- Preflight requests are rejected with 403 Forbidden
- Only same-origin requests work

## TLS/HTTPS Support

Beamdrop now supports TLS/HTTPS for encrypted connections.

### Using TLS

Generate a certificate and key (or use existing ones), then start the server:

```bash
./beamdrop -dir=/path/to/share -tls-cert=/path/to/cert.pem -tls-key=/path/to/key.pem
```

For development/testing, you can generate self-signed certificates:

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"
```

When TLS is enabled:

- Server runs on HTTPS instead of HTTP
- HSTS (HTTP Strict Transport Security) header is added
- QR code shows HTTPS URL

## Security Headers

The following security headers are automatically added to all responses:

### X-Frame-Options: DENY

Prevents the page from being embedded in iframes, protecting against clickjacking attacks.

### X-Content-Type-Options: nosniff

Prevents browsers from MIME-sniffing the content type, reducing XSS risks.

### Referrer-Policy: strict-origin-when-cross-origin

Controls how much referrer information is sent with requests:

- Same origin: full URL
- Cross-origin: only the origin (no path)

### Content-Security-Policy

Restricts resource loading to prevent XSS and data injection attacks:

- `default-src 'self'`: Only load resources from same origin
- `script-src 'self'`: Allow scripts from same origin only (no inline scripts or eval)
- `style-src 'self' 'unsafe-inline'`: Allow styles from same origin and inline styles (required for Tailwind CSS)
- `img-src 'self' data: blob:`: Allow images from same origin, data URIs, and blob URIs
- `font-src 'self' data:`: Allow fonts from same origin and data URIs
- `connect-src 'self' ws: wss:`: Allow connections to same origin and WebSocket
- `object-src 'none'`: Block all plugin content (Flash, Java applets, etc.)
- `base-uri 'self'`: Restrict `<base>` tag to same origin
- `form-action 'self'`: Restrict form submissions to same origin
- `frame-ancestors 'none'`: Prevent embedding in frames (equivalent to X-Frame-Options: DENY)

### Permissions-Policy

Restricts access to powerful browser features:

- `geolocation=()`: Disabled
- `microphone=()`: Disabled
- `camera=()`: Disabled

### Strict-Transport-Security (HTTPS only)

When TLS is enabled, HSTS header is added:

- `max-age=31536000`: Force HTTPS for 1 year
- `includeSubDomains`: Apply to all subdomains

## HTTP Method Restrictions

All endpoints now enforce strict HTTP method requirements:

- **GET only**: `/health`, `/ready`, `/files`, `/download`, `/search`, `/starred`, `/auth/status`
- **POST only**: `/upload`, `/move`, `/copy`, `/mkdir`, `/rename`, `/write`, `/star`, `/auth/login`, `/auth/logout`

Requests with incorrect methods receive a `405 Method Not Allowed` response.

## Storage Limits

Beamdrop supports a configurable maximum storage limit. When enabled, write requests (POST, PUT, PATCH) are rejected with `STORAGE_FULL` once usage exceeds the threshold.

### Configuration

```bash
# Limit storage to 10GB
beamdrop -dir /path/to/share -max-storage 10GB

# Unlimited (default)
beamdrop -dir /path/to/share -max-storage 0
```

Storage usage is cached for 5 seconds to avoid disk overhead on every request. Only user files count toward the limit — internal `.beamdrop` directories are excluded.

## Rate Limiting

Beamdrop includes built-in per-IP rate limiting to protect against brute-force attacks, upload flooding, and general abuse.

### How It Works

Rate limiting uses a **token-bucket algorithm** with three endpoint tiers, each enforced independently per client IP:

| Tier        | Endpoints                | Default Rate | Purpose                               |
| ----------- | ------------------------ | ------------ | ------------------------------------- |
| **General** | All other endpoints      | 100 req/min  | Prevents general abuse                |
| **Auth**    | `/auth/login`            | 5 req/min    | Prevents brute-force password attacks |
| **Upload**  | `/upload`, S3 PUT object | 10 req/min   | Prevents upload flooding              |

Auth and upload tier rates are derived from the general rate (5% and 10% respectively, minimum 1).

### Configuration

```bash
# Default: 100 requests/min per IP
beamdrop -dir /path/to/share

# Custom rate limit
beamdrop -dir /path/to/share -rate-limit 200

# Disable rate limiting
beamdrop -dir /path/to/share -rate-limit 0
```

### Rate Limit Response

When a client exceeds the rate limit, the server responds with:

- **HTTP 429 Too Many Requests**
- `Retry-After` header (seconds until the client can retry)
- `X-Retryable: true` header
- JSON body with error code `RATE_LIMIT_EXCEEDED`

Example response:

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded",
    "category": "RATE_LIMIT"
  }
}
```

### Trusted Proxy Support

By default, `X-Forwarded-For` and `X-Real-IP` headers are **ignored** to prevent IP spoofing. When running behind a reverse proxy, configure trusted proxies so headers from known proxies are honoured:

```bash
# Trust a single proxy
beamdrop -dir /path/to/share -trusted-proxies "10.0.0.1/32"

# Trust a CIDR range (e.g., Docker network)
beamdrop -dir /path/to/share -trusted-proxies "172.16.0.0/12"

# Trust multiple ranges
beamdrop -dir /path/to/share -trusted-proxies "10.0.0.0/8,172.16.0.0/12"
```

### IP Detection

The rate limiter identifies clients by IP address, checking in order:

1. `X-Forwarded-For` header (first IP in the chain) — **only if the request comes from a trusted proxy**
2. `X-Real-IP` header — **only if the request comes from a trusted proxy**
3. Connection remote address

### Internals

- Stale client entries (unseen for 10+ minutes) are automatically evicted every 5 minutes
- Each IP gets independent buckets for each tier hitting the auth limit does not affect general requests
- Tokens refill continuously (not in fixed windows), providing smooth rate enforcement

## Structured Logging

Beamdrop uses Go's `log/slog` for structured logging with dual output:

### Terminal Output

Human-readable, colored output showing timestamp, level, and message with key-value pairs:

```
11:03:13.973 INFO  Starting beamdrop application
11:03:13.973 INFO  Starting server shared_dir=/tmp/share
11:03:13.974 INFO  Rate limiting enabled general=100 unit=req/min
11:03:13.985 INFO  Server started url=http://192.168.1.13:7777
```

### File Output

Structured JSON logs are written to `<dir>/.beamdrop/beamdrop.log` with full source locations:

```json
{
  "time": "2026-02-16T11:03:13.973+03:00",
  "level": "INFO",
  "source": { "function": "main.main", "file": "cmd/beam/main.go", "line": 65 },
  "msg": "Starting beamdrop application"
}
```

### Configuration

```bash
# Set log level (debug, info, warn, error)
beamdrop -dir /path/to/share -log-level debug
```

## Command-Line Options

```
-dir string
      Directory to share files from (default ".")
-port int
      Set the port that beamdrop will run on (default: auto-detect)
-p string
      Password for authentication
-tls-cert string
      Path to TLS certificate file for HTTPS
-tls-key string
      Path to TLS private key file for HTTPS
-allowed-origins string
      Comma-separated list of allowed CORS origins (empty = CORS disabled for security)
-api-auth
      Enable API key authentication for S3-like API endpoints
-rate-limit int
      General rate limit in requests/min per IP (default 100, 0 = disabled)
-max-storage string
      Maximum total storage, e.g. 500MB, 10GB, 1TB (0 = unlimited)
-trusted-proxies string
      Comma-separated CIDR ranges of trusted reverse proxies (e.g. "10.0.0.0/8,172.16.0.0/12")
-log-level string
      Log level: debug, info, warn, error (default "info")
-qr
      Enable QR code generation
-h
      Show help message
-v
      Show version information
```

## CSRF Protection

Beamdrop uses **double-submit cookie** CSRF protection to prevent cross-site request forgery attacks.

### How It Works

1. On any `GET` request, the server sets a `beamdrop_csrf` cookie with a random token.
2. The frontend reads this cookie and attaches the value as an `X-CSRF-Token` header on all unsafe requests (`POST`, `PUT`, `DELETE`, `PATCH`).
3. The server validates that the header matches the cookie on every state-changing request.

### Exemptions

The following requests are exempt from CSRF validation:

- **Safe methods**: `GET`, `HEAD`, `OPTIONS`
- **API key authenticated requests**: Requests with an `Authorization` header (API keys use HMAC, which is inherently CSRF-safe)
- **Requests without a session cookie**: Non-authenticated requests or first-time visitors
- **Shareable link access**: `/api/shares/access/` paths (public endpoints with their own password protection)

### Frontend Integration

The frontend automatically installs a global `fetch` interceptor that reads the `beamdrop_csrf` cookie and attaches the `X-CSRF-Token` header on all unsafe requests. No manual intervention is needed when using the built-in web UI.

For custom integrations, include the `X-CSRF-Token` header with the value from the `beamdrop_csrf` cookie on all POST/PUT/DELETE/PATCH requests.

## Token Revocation

JWT tokens are revoked on logout to prevent reuse of stolen tokens.

### How It Works

1. When a user logs out, the token's unique identifier (JTI) is added to an in-memory revocation list.
2. Every token validation check now verifies the JTI is not in the revocation list.
3. Revoked entries are automatically cleaned up when the token would have expired (or after 24 hours, whichever is sooner).
4. A background goroutine runs every 10 minutes to purge expired revocation entries.

### Implications

- Tokens are immediately invalidated on logout — they cannot be replayed.
- The revocation list is in-memory and cleared on server restart (which also regenerates the JWT secret, invalidating all tokens).
- Both cookie-based and `Authorization: Bearer` header tokens are revoked during logout.

## API Key Secret Encryption

API key secrets are encrypted at rest using **AES-256-GCM** before being stored in the database.

### How It Works

1. When an API key is created, the secret key is encrypted with AES-256-GCM using a 32-byte key derived from the JWT secret.
2. The encrypted ciphertext (base64-encoded) is stored in the database instead of a plain SHA-256 hash.
3. During HMAC signature verification, the secret is decrypted in memory for the comparison.
4. The encryption key is regenerated on every server restart (along with the JWT secret), so API keys created on one process cannot be decrypted by another. API key secrets are shown once at creation time — users must save them.

### Why Encryption Instead of Hashing?

API key secrets are used for HMAC signature computation, which requires the raw secret. Unlike passwords (where we only need to verify), API key secrets must be recoverable in the server process. AES-256-GCM provides authenticated encryption with integrity verification.

## Shareable Link Password Hashing

Shareable link passwords are hashed using **bcrypt** (cost 10) instead of SHA-256.

### Backward Compatibility

Existing links with SHA-256 password hashes continue to work. The password verification function detects the hash format:

- If the stored hash starts with `$2a$` (bcrypt prefix), bcrypt verification is used.
- Otherwise, SHA-256 verification is used as a fallback.

New shareable links always use bcrypt.

## Session Security

### Cookie-Only Token Storage

JWT tokens are stored exclusively in `HttpOnly`, `SameSite=Strict` cookies. The frontend does not use `localStorage` or `sessionStorage` for token storage, eliminating the risk of token theft via XSS.

### Cookie Attributes

| Attribute  | Value    | Purpose                           |
| ---------- | -------- | --------------------------------- |
| `HttpOnly` | `true`   | Prevents JavaScript access        |
| `SameSite` | `Strict` | Prevents cross-site request usage |
| `Path`     | `/`      | Available to all routes           |
| `Secure`   | Auto     | Set when TLS is enabled           |

## Best Practices

1. **Keep CORS disabled** unless you specifically need cross-origin access
2. **Use TLS in production** to encrypt data in transit
3. **Use strong passwords** with the `-p` flag for authentication
4. **Restrict allowed origins** to only trusted domains when enabling CORS
5. **Use valid TLS certificates** in production (e.g., from Let's Encrypt)
6. **Keep rate limiting enabled** the default of 100 req/min is suitable for most use cases
7. **Monitor logs** check `<dir>/.beamdrop/beamdrop.log` for rate limit warnings and suspicious activity
8. **Keep the software updated** to get the latest security patches
9. **Use short-lived presigned URLs** prefer 1–24 hour expiry for download links; never rely on very long expiry times as they break on API key rotation. For individually revocable links, use the server-side pretty presigned URL registry (`POST /api/v1/presign`)
10. **Rotate API keys periodically** create a new key, update your application, then delete the old key; be aware this invalidates all client-side HMAC presigned URLs generated with the old key (server-side pretty URLs are not affected by key rotation)
11. **Prefer server-side pretty presigned URLs for sensitive content** they support download limits, individual revocation, and download tracking, giving you more control than client-side HMAC URLs

## Examples

### Secure local sharing (recommended)

```bash
./beamdrop -dir=/path/to/share -p="strong-password"
```

### With HTTPS and specific CORS origins

```bash
./beamdrop -dir=/path/to/share \
  -tls-cert=/etc/beamdrop/cert.pem \
  -tls-key=/etc/beamdrop/key.pem \
  -allowed-origins="https://app.example.com" \
  -p="strong-password"
```

### Development with HTTPS

```bash
# Generate self-signed cert first
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes

# Start server
./beamdrop -dir=. -tls-cert=cert.pem -tls-key=key.pem
```
