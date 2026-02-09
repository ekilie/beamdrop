# Beamdrop

A lightweight, self-hosted file sharing server with S3-compatible API. Built with Go and React.

## Overview

Beamdrop provides both a web interface for interactive file management and a programmatic API for application integration. Use it as a personal file server, artifact storage for CI/CD pipelines, or a development S3 alternative.

## Features

### Web Interface
- Modern file browser with grid and list views
- Drag-and-drop file upload
- File operations: move, copy, rename, delete, restore from trash
- Directory creation and navigation
- File search and advanced filtering
- File preview for images, video, audio, code, and documents
- Real-time server statistics via WebSocket
- Responsive design for desktop and mobile

### S3-Compatible API
- RESTful API for programmatic access
- Bucket-based storage organization
- Object upload/download with streaming support
- Prefix-based listing (directory-like structure)
- HMAC-SHA256 signature authentication
- API key management with optional bucket scoping and expiration

### Security
- Password authentication for web interface
- API key authentication with HMAC signatures
- HTTPS/TLS support for encrypted connections
- Security headers (HSTS, CSP, X-Frame-Options)
- Configurable CORS policies
- HTTP method restrictions

## Installation

### From Source

```bash
git clone https://github.com/ekilie/beamdrop.git
cd beamdrop
make build
```

### Binary

Download the latest release from the releases page.

## Quick Start

### Basic Usage

```bash
# Share current directory
./beamdrop

# Share specific directory
./beamdrop -dir /path/to/share

# With password protection
./beamdrop -dir /path/to/share -p mysecretpassword

# With custom port
./beamdrop -dir /path/to/share -port 9000
```

### With S3-Compatible API

```bash
# Enable API authentication
./beamdrop -dir /path/to/share -api-auth

# With HTTPS
./beamdrop -dir /path/to/share -api-auth -tls-cert cert.pem -tls-key key.pem
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

MIT License
