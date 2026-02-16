# Security Features

This document describes the security hardening features added to Beamdrop.

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
- `script-src 'self' 'unsafe-inline' 'unsafe-eval'`: Allow scripts from same origin and inline scripts
- `style-src 'self' 'unsafe-inline'`: Allow styles from same origin and inline styles
- `img-src 'self' data:`: Allow images from same origin and data URIs
- `font-src 'self' data:`: Allow fonts from same origin and data URIs
- `connect-src 'self' ws: wss:`: Allow connections to same origin and WebSocket

### Strict-Transport-Security (HTTPS only)
When TLS is enabled, HSTS header is added:
- `max-age=31536000`: Force HTTPS for 1 year
- `includeSubDomains`: Apply to all subdomains

## HTTP Method Restrictions

All endpoints now enforce strict HTTP method requirements:

- **GET only**: `/health`, `/ready`, `/files`, `/download`, `/search`, `/starred`, `/auth/status`
- **POST only**: `/upload`, `/move`, `/copy`, `/mkdir`, `/rename`, `/write`, `/star`, `/auth/login`, `/auth/logout`

Requests with incorrect methods receive a `405 Method Not Allowed` response.

## Rate Limiting

Beamdrop includes built-in per-IP rate limiting to protect against brute-force attacks, upload flooding, and general abuse.

### How It Works

Rate limiting uses a **token-bucket algorithm** with three endpoint tiers, each enforced independently per client IP:

| Tier | Endpoints | Default Rate | Purpose |
|------|-----------|-------------|---------|
| **General** | All other endpoints | 100 req/min | Prevents general abuse |
| **Auth** | `/auth/login` | 5 req/min | Prevents brute-force password attacks |
| **Upload** | `/upload`, S3 PUT object | 10 req/min | Prevents upload flooding |

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

### IP Detection

The rate limiter identifies clients by IP address, checking in order:
1. `X-Forwarded-For` header (first IP in the chain)
2. `X-Real-IP` header
3. Connection remote address

### Internals

- Stale client entries (unseen for 10+ minutes) are automatically evicted every 5 minutes
- Each IP gets independent buckets for each tier — hitting the auth limit does not affect general requests
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
{"time":"2026-02-16T11:03:13.973+03:00","level":"INFO","source":{"function":"main.main","file":"cmd/beam/main.go","line":65},"msg":"Starting beamdrop application"}
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
-log-level string
      Log level: debug, info, warn, error (default "info")
-no-qr
      Disable QR code generation
-h
      Show help message
-v
      Show version information
```

## Best Practices

1. **Keep CORS disabled** unless you specifically need cross-origin access
2. **Use TLS in production** to encrypt data in transit
3. **Use strong passwords** with the `-p` flag for authentication
4. **Restrict allowed origins** to only trusted domains when enabling CORS
5. **Use valid TLS certificates** in production (e.g., from Let's Encrypt)
6. **Keep rate limiting enabled** — the default of 100 req/min is suitable for most use cases
7. **Monitor logs** — check `<dir>/.beamdrop/beamdrop.log` for rate limit warnings and suspicious activity
8. **Keep the software updated** to get the latest security patches

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
