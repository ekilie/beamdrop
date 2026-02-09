# Beamdrop

Beamdrop is a simple, self-hosted file sharing server built with Go and React. It provides a web interface for uploading, downloading, and managing files from a shared directory.

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

```bash
git clone https://github.com/ekilie/beamdrop.git
cd beamdrop
go build -o beamdrop ./cmd/beam
```

## Usage

Start the server with a directory to share:

```bash
./beamdrop -dir="/path/to/share"
```

Available flags:
- `-dir` - Directory to share files from (default: current directory)
- `-port` - Port to run on (default: auto-detect available port)
- `-p` - Password for authentication
- `-tls-cert` - Path to TLS certificate file for HTTPS
- `-tls-key` - Path to TLS private key file for HTTPS
- `-allowed-origins` - Comma-separated list of allowed CORS origins (empty = disabled for security)
- `-no-qr` - Disable QR code generation
- `-v` - Show version information
- `-h` - Show help message

For more details on security features, see [docs/SECURITY.md](docs/SECURITY.md).

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

The project consists of:
- Go backend server with RESTful API
- React frontend with TypeScript
- SQLite database for statistics

## Future Plans

Planned features include S3-compatible storage backend support, allowing integration with cloud storage providers like AWS S3, MinIO, and other S3-compatible services for file storage and manipulation.

## License

MIT License
