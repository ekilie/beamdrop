# BeamDrop

**Your private Google Drive + S3, ready in 10 seconds.**

Turn any server or VPS into secure, self-hosted file storage with:

- Web UI for humans
- S3-compatible API for apps
- Shareable links & real-time stats

No cloud vendor. No complex setup. One command to start.

![BeamDrop Preview](docs/beamdrop.jpeg)

## Quick Start

```bash
# Share current directory instantly
beamdrop

# Access via browser: http://localhost:7777
# Scan QR code from your phone to upload files
```

```bash
# Share a specific directory with password protection
beamdrop -dir /path/to/share -p mysecretpassword

# Enable S3-compatible API
beamdrop -dir /path/to/share -api-auth

# With HTTPS
beamdrop -dir /path/to/share -api-auth -tls-cert cert.pem -tls-key key.pem
```

## Why BeamDrop?

- **Tired of paying for cloud storage?** Run your own S3-compatible server.
- **Need easy file sharing?** Drag, drop, share links — optionally password-protected.
- **Want something developers love?** Works with existing S3 libraries & APIs.
- **Need a single tool for teams?** Web UI + API + real-time stats in one binary.
- **Worried about security & vendor lock-in?** Your server, your data, full control.

## Key Features

- **Web UI + File Browser:** Drag, drop, rename, move, and organize files.
- **S3-Compatible API:** Works with existing AWS SDKs and scripts.
- **Official Go client:** In-repo SDK for signed bucket, object, and presigned URL operations.
- **Shareable Links:** Optional password and expiry.
- **Single Binary:** Runs anywhere, zero dependencies.
- **Secure & Production-Ready:** TLS, rate limiting, structured logging.

## Architecture

![Beamdrop System Architecture](docs/beamdrop-arch.png)

## All Features

- **Web-based file manager** — modern React UI with drag-and-drop upload, search, and file operations (move, copy, rename, mkdir)
- **S3-compatible API** — buckets, objects, presigned URLs, HMAC-SHA256 auth
- **Shareable links** — generate unique URLs with optional password protection and expiry
- **Real-time stats** — live storage metrics via WebSocket
- **Single binary** — zero dependencies, runs on Linux, macOS, and Windows
- **Docker-ready** — ~39 MB image, non-root, health checks included
- QR code generation for easy mobile access
- Cross-platform support
- **Security features**:
  - HTTPS/TLS support for encrypted connections
  - Configurable CORS with strict defaults (disabled by default)
  - Security headers (HSTS, CSP, X-Frame-Options, Permissions-Policy, etc.)
  - HTTP method restrictions on all endpoints
  - **Per-IP rate limiting** with tiered enforcement (general, auth, upload)
  - **CSRF protection** via double-submit cookie pattern
  - **JWT token revocation** on logout with automatic cleanup
  - **AES-256-GCM encryption** for API key secrets at rest
  - **bcrypt password hashing** for shareable link passwords
  - **Cookie-only JWT storage** (no localStorage) with `HttpOnly` + `SameSite=Strict`
  - **Trusted proxy support** for accurate IP detection behind reverse proxies
- **Structured logging**:
  - Colored, human-readable terminal output
  - Structured JSON log file at `<dir>/.beamdrop/beamdrop.log`
  - Configurable log level
- **Docker support**: Multi-stage Dockerfile with ~39 MB image, non-root user, health checks
- **Health probes**: Kubernetes-compatible `/health/live`, `/health/ready`, `/health/startup` endpoints with component-level status
- **Prometheus metrics**: `/metrics` endpoint with request counters, latency histograms, storage gauges, and a ready-to-import Grafana dashboard

## Installation

## Go Client

Beamdrop now includes a first-party Go client in this repository:

```go
import "github.com/ekilie/beamdrop/pkg/client"
```

Minimal example:

```go
ctx := context.Background()

sdk, err := client.New(client.Config{
  BaseURL:     "http://localhost:7777",
  AccessKeyID: "BDK_your_access_key",
  SecretKey:   "sk_your_secret_key",
})
if err != nil {
  log.Fatal(err)
}

if _, err := sdk.CreateBucketIfNotExists(ctx, "uploads"); err != nil {
  log.Fatal(err)
}

if _, err := sdk.PutObject(ctx, "uploads", "hello.txt", []byte("hello beamdrop")); err != nil {
  log.Fatal(err)
}

object, err := sdk.GetObject(ctx, "uploads", "hello.txt")
if err != nil {
  log.Fatal(err)
}

fmt.Println(string(object.Body))
```

Current client scope:

- bucket operations
- object upload, download, metadata, and delete
- client-side presigned URL generation
- server-side presigned URL management via `/api/v1/presign`

### Quick Install (macOS & Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/ekilie/beamdrop/main/docs/install.sh | sh
```

Or inspect the script first:

```bash
curl -fsSL https://raw.githubusercontent.com/ekilie/beamdrop/main/docs/install.sh -o install.sh
less install.sh
sh install.sh
```

Options via environment variables:

```bash
# Install a specific version
BEAMDROP_VERSION=v1.0.0 sh install.sh

# Install to a custom directory
BEAMDROP_INSTALL=~/.local/bin sh install.sh
```

### From Source

```bash
git clone https://github.com/ekilie/beamdrop.git
cd beamdrop
make build
```

### macOS (Apple Silicon)

```bash
curl -L https://github.com/ekilie/beamdrop/releases/latest/download/beamdrop-darwin-arm64.tar.gz -o beamdrop-darwin-arm64.tar.gz
sudo tar -C /usr/local/bin -xzf beamdrop-darwin-arm64.tar.gz
rm beamdrop-darwin-arm64.tar.gz
```

### macOS (Intel)

```bash
curl -L https://github.com/ekilie/beamdrop/releases/latest/download/beamdrop-darwin-amd64.tar.gz -o beamdrop-darwin-amd64.tar.gz
sudo tar -C /usr/local/bin -xzf beamdrop-darwin-amd64.tar.gz
rm beamdrop-darwin-amd64.tar.gz
```

### Linux (amd64)

```bash
curl -L https://github.com/ekilie/beamdrop/releases/latest/download/beamdrop-linux-amd64.tar.gz -o beamdrop-linux-amd64.tar.gz
sudo tar -C /usr/local/bin -xzf beamdrop-linux-amd64.tar.gz
rm beamdrop-linux-amd64.tar.gz
```

### Linux (arm64)

```bash
curl -L https://github.com/ekilie/beamdrop/releases/latest/download/beamdrop-linux-arm64.tar.gz -o beamdrop-linux-arm64.tar.gz
sudo tar -C /usr/local/bin -xzf beamdrop-linux-arm64.tar.gz
rm beamdrop-linux-arm64.tar.gz
```

### Windows

Download the latest `.zip` from the [releases page](https://github.com/ekilie/beamdrop/releases), extract it, and add `beamdrop.exe` to your PATH.

### Docker

```bash
# Build the image
docker build -t beamdrop .

# Run with a persistent volume
docker run -d \
  --name beamdrop \
  -p 7777:7777 \
  -v beamdrop-data:/data \
  beamdrop

# Run with all options
docker run -d \
  --name beamdrop \
  -p 7777:7777 \
  -v beamdrop-data:/data \
  -e BEAMDROP_PASSWORD="secret" \
  -e BEAMDROP_API_AUTH=true \
  -e BEAMDROP_RATE_LIMIT=100 \
  -e BEAMDROP_MAX_STORAGE=10GB \
  beamdrop
```

The image is ~39 MB, runs as non-root, and includes a `HEALTHCHECK` against `/health/live`.

### Docker Compose (recommended)

The easiest way to run BeamDrop:

```bash
# Start in background
docker compose up -d

# View logs
docker compose logs -f beamdrop

# Stop
docker compose down
```

Configure via environment variables create a `.env` file or export them:

```bash
# .env (optional)
BEAMDROP_PORT=7777
BEAMDROP_PASSWORD=your-secret-password
BEAMDROP_LOG_LEVEL=info
BEAMDROP_RATE_LIMIT=100
BEAMDROP_MAX_STORAGE=0
BEAMDROP_API_AUTH=true
BEAMDROP_QR=false
BEAMDROP_ALLOWED_ORIGINS=https://example.com
```

| Variable                   | Default  | Description                                                         |
| -------------------------- | -------- | ------------------------------------------------------------------- |
| `BEAMDROP_PORT`            | `7777`   | Port to listen on                                                   |
| `BEAMDROP_PASSWORD`        | _(none)_ | Enable password authentication                                      |
| `BEAMDROP_LOG_LEVEL`       | `info`   | Log level: `debug`, `info`, `warn`, `error`                         |
| `BEAMDROP_RATE_LIMIT`      | `100`    | Requests/min per IP (`0` = disabled)                                |
| `BEAMDROP_API_AUTH`        | _(off)_  | Set to `true` to enable S3 API key auth                             |
| `BEAMDROP_QR`              | `false`  | Set to `true` to print startup QR code                              |
| `BEAMDROP_ALLOWED_ORIGINS` | _(none)_ | Comma-separated CORS origins                                        |
| `BEAMDROP_DB_PATH`         | _(none)_ | Path to DB file or directory (directory auto-appends `beamdrop.db`) |
| `BEAMDROP_TLS_CERT`        | _(none)_ | Path to TLS certificate (inside container)                          |
| `BEAMDROP_TLS_KEY`         | _(none)_ | Path to TLS private key (inside container)                          |

**Development mode** (debug logging, rate limiting off):

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

**With Caddy reverse proxy** (automatic HTTPS):

1. Uncomment the `caddy` service in `docker-compose.yml`
2. Set your domain: `export BEAMDROP_DOMAIN=files.example.com`
3. Run: `docker compose up -d`

Data is persisted in `./data/` on the host.

### Prometheus & Grafana

Beamdrop exposes a `/metrics` endpoint in Prometheus text format. Add it as a scrape target:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: beamdrop
    static_configs:
      - targets: ["localhost:7777"]
```

A pre-built Grafana dashboard is available at [`docs/grafana-dashboard.json`](docs/grafana-dashboard.json). Import it via **Dashboards > Import** in Grafana.

**Exported metrics:**

| Metric                              | Type      | Description                           |
| ----------------------------------- | --------- | ------------------------------------- |
| `beamdrop_requests_total`           | counter   | HTTP requests by method, path, status |
| `beamdrop_request_duration_seconds` | histogram | Request latency (p50/p95/p99)         |
| `beamdrop_auth_failures_total`      | counter   | Auth failures by reason               |
| `beamdrop_uploads_total`            | counter   | Completed uploads                     |
| `beamdrop_downloads_total`          | counter   | Completed downloads                   |
| `beamdrop_upload_size_bytes`        | histogram | Upload file sizes                     |
| `beamdrop_storage_bytes`            | gauge     | Bytes used by stored files            |
| `beamdrop_objects_total`            | gauge     | Number of stored files                |
| `beamdrop_active_connections`       | gauge     | In-flight HTTP requests               |
| `beamdrop_storage_free_bytes`       | gauge     | Free disk space                       |
| `beamdrop_storage_total_bytes`      | gauge     | Total disk capacity                   |
| `beamdrop_goroutines_count`         | gauge     | Go goroutine count                    |

## Configuration

### Command Line Flags

| Flag               | Description                                                               | Default                   |
| ------------------ | ------------------------------------------------------------------------- | ------------------------- |
| `-dir`             | Directory to share                                                        | Current directory         |
| `-port`            | Server port                                                               | Auto-detect               |
| `-p`               | Password for web authentication                                           | None                      |
| `-api-auth`        | Enable API key authentication                                             | false                     |
| `-tls-cert`        | Path to TLS certificate                                                   | None                      |
| `-tls-key`         | Path to TLS private key                                                   | None                      |
| `-allowed-origins` | CORS allowed origins (comma-separated)                                    | None                      |
| `-db-path`         | Path to database file or directory (directory auto-appends `beamdrop.db`) | `~/.beamdrop/beamdrop.db` |
| `-rate-limit`      | Rate limit in requests/min per IP (0 = disabled)                          | 100                       |
| `-max-storage`     | Maximum total storage, e.g. 500MB, 10GB, 1TB (0 = unlimited)              | 0                         |
| `-log-level`       | Log level: debug, info, warn, error                                       | info                      |
| `-qr`              | Enable QR code display                                                    | false                     |
| `-v`               | Show version                                                              | -                         |
| `-h`               | Show help                                                                 | -                         |

## API Usage

### Creating an API Key

Via the web interface:

1. Navigate to API Keys in the sidebar
2. Click "Create New Key"
3. Save the secret key (shown only once)

Via API:

```bash
curl -X POST http://localhost:8080/api/v1/keys \
  -H "Content-Type: application/json" \
  -d '{"name": "My App", "expiresIn": 2592000}'
```

### Authentication

All API requests require HMAC-SHA256 signed authentication:

```bash
# Generate signature
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
STRING_TO_SIGN="GET\n/api/v1/buckets\n${TIMESTAMP}"
SIGNATURE=$(echo -n "$STRING_TO_SIGN" | openssl dgst -sha256 -hmac "$SECRET_KEY" -binary | base64)

# Make request
curl http://localhost:8080/api/v1/buckets \
  -H "Authorization: Bearer ${ACCESS_KEY}:${SIGNATURE}" \
  -H "X-Beamdrop-Date: ${TIMESTAMP}"
```

### Example Operations

```bash
# Create bucket
curl -X PUT http://localhost:8080/api/v1/buckets/my-bucket \
  -H "Authorization: Bearer ${ACCESS_KEY}:${SIGNATURE}" \
  -H "X-Beamdrop-Date: ${TIMESTAMP}"

# Upload file
curl -X PUT http://localhost:8080/api/v1/buckets/my-bucket/path/to/file.txt \
  -H "Authorization: Bearer ${ACCESS_KEY}:${SIGNATURE}" \
  -H "X-Beamdrop-Date: ${TIMESTAMP}" \
  -H "Content-Type: text/plain" \
  -d "Hello, World!"

# Download file
curl http://localhost:8080/api/v1/buckets/my-bucket/path/to/file.txt \
  -H "Authorization: Bearer ${ACCESS_KEY}:${SIGNATURE}" \
  -H "X-Beamdrop-Date: ${TIMESTAMP}"

# List objects
curl "http://localhost:8080/api/v1/buckets/my-bucket?list&prefix=path/" \
  -H "Authorization: Bearer ${ACCESS_KEY}:${SIGNATURE}" \
  -H "X-Beamdrop-Date: ${TIMESTAMP}"
```

## Documentation

### For Operators

- **[Operations Runbook](docs/OPERATIONS-RUNBOOK.md)** - Production deployment, backup, monitoring, scaling, and troubleshooting

### For Developers

- **[Developer Guide](docs/DEVELOPER-GUIDE.md)** - Comprehensive guide to the codebase structure and S3 API implementation
- API Design: [docs/s3-api-design.md](docs/s3-api-design.md)
- Security: [docs/SECURITY.md](docs/SECURITY.md)

### API Documentation

- OpenAPI Specification: [docs/openapi.yaml](docs/openapi.yaml)
- Postman Collection: [docs/beamdrop-api.postman_collection.json](docs/beamdrop-api.postman_collection.json)
- Postman Environment: [docs/beamdrop-api.postman_environment.json](docs/beamdrop-api.postman_environment.json)
- Postman Guide: [docs/POSTMAN-GUIDE.md](docs/POSTMAN-GUIDE.md)

## Storage Structure

```
shared-directory/
├── buckets/              # API-managed storage
│   ├── my-bucket/
│   │   ├── images/
│   │   │   └── photo.jpg
│   │   └── data.json
│   └── backups/
│       └── db.sql
├── .beamdrop/            # Logs
│   └── beamdrop.log      # Structured JSON log file
├── .beamdrop_data/       # Internal database
└── .beamdrop_trash/      # Deleted files (recoverable)
```

## Shareable Links

Beamdrop supports creating shareable links for files and folders, similar to Google Drive:

### Creating a Shareable Link

1. Navigate to the file browser
2. Right-click on a file or folder and select "Share Link" from the context menu
3. Configure optional settings:
   - **Password**: Protect the link with a password
   - **Expiry**: Set when the link should expire (in hours)
4. Click "Generate Link" to create the shareable URL
5. Copy the link and share it with others

### Managing Shareable Links

- View all active shareable links in the "Shares" section of the sidebar
- See access statistics including view count
- Delete links when they're no longer needed
- Links are automatically removed after expiration

### Security Considerations

- Password-protected links require the correct password to access
- Expired links are automatically rejected
- Access to shareable links is tracked for monitoring
- Links can be revoked at any time by deleting them
- Public share links bypass authentication but can still be password-protected

### API Endpoints

- `POST /api/shares` - Create a new shareable link
- `GET /api/shares/list` - List all shareable links
- `DELETE /api/shares/delete?token=<token>` - Delete a shareable link
- `GET /share/<token>` - Public access endpoint (no auth required)

## Development

### Prerequisites

- Go 1.21+
- Node.js 18+ (for frontend development)
- Make

### Building

```bash
# Build everything
make build

# Build backend only
go build -o beamdrop ./cmd/beam

# Build frontend
cd static/frontend && bun install && bun run build
```

### Running in Development

```bash
# Backend with hot reload
make dev

# Frontend dev server
cd static/frontend && bun run dev
```

## Project Structure

```
beamdrop/
├── cmd/beam/           # CLI entry point
├── beam/server/        # HTTP server and handlers
├── config/             # Configuration
├── pkg/
│   ├── auth/           # Authentication
│   ├── db/             # Database and models
│   ├── errors/         # Structured error types
│   ├── middleware/      # CORS, security headers, rate limiting
│   ├── storage/        # Bucket/object storage
│   ├── crypto/         # Signature utilities
│   ├── logger/         # Dual-output structured logging
│   └── ...
├── static/frontend/    # React frontend
└── docs/               # Documentation
```

## License

[GNU Affero General Public License v3.0](LICENSE)
