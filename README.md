# Beamdrop

A lightweight, self-hosted file sharing server with S3-compatible API. Built with Go and React.

## Overview

Beamdrop provides both a web interface for interactive file management and a programmatic API for application integration. Use it as a personal file server, artifact storage for CI/CD pipelines, or a development S3 alternative.

## Features

- Web-based file browser with modern UI
- File upload and download
- File operations: move, copy, rename, create directories
- File search functionality
- Real-time statistics via WebSocket
- Password authentication support
- QR code generation for easy access
- **Shareable Links**: Generate unique links to share files/folders with optional password protection and expiry
- Cross-platform support
- **Security features**:
  - HTTPS/TLS support for encrypted connections
  - Configurable CORS with strict defaults (disabled by default)
  - Security headers (HSTS, CSP, X-Frame-Options, etc.)
  - HTTP method restrictions on all endpoints

## Installation

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

## Quick Start

### Basic Usage

```bash
# Share current directory
beamdrop

# Share specific directory
beamdrop -dir /path/to/share

# With password protection
beamdrop -dir /path/to/share -p mysecretpassword

# With custom port
beamdrop -dir /path/to/share -port 9000
```

### With S3-Compatible API

```bash
# Enable API authentication
beamdrop -dir /path/to/share -api-auth

# With HTTPS
beamdrop -dir /path/to/share -api-auth -tls-cert cert.pem -tls-key key.pem
```

## Configuration

### Command Line Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-dir` | Directory to share | Current directory |
| `-port` | Server port | Auto-detect |
| `-p` | Password for web authentication | None |
| `-api-auth` | Enable API key authentication | false |
| `-tls-cert` | Path to TLS certificate | None |
| `-tls-key` | Path to TLS private key | None |
| `-allowed-origins` | CORS allowed origins (comma-separated) | None |
| `-no-qr` | Disable QR code display | false |
| `-v` | Show version | - |
| `-h` | Show help | - |

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

## API Documentation

- OpenAPI Specification: [docs/openapi.yaml](docs/openapi.yaml)
- Postman Collection: [docs/beamdrop-api.postman_collection.json](docs/beamdrop-api.postman_collection.json)
- Postman Environment: [docs/beamdrop-api.postman_environment.json](docs/beamdrop-api.postman_environment.json)
- Postman Guide: [docs/POSTMAN-GUIDE.md](docs/POSTMAN-GUIDE.md)
- API Design: [docs/s3-api-design.md](docs/s3-api-design.md)
- Security: [docs/SECURITY.md](docs/SECURITY.md)

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
│   ├── storage/        # Bucket/object storage
│   ├── crypto/         # Signature utilities
│   └── ...
├── static/frontend/    # React frontend
└── docs/               # Documentation
```

## License

[GNU Affero General Public License v3.0](LICENSE)

