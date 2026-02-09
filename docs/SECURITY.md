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

### X-XSS-Protection: 1; mode=block
Enables browser's XSS filter to block pages if XSS attack is detected.

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
6. **Keep the software updated** to get the latest security patches

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
