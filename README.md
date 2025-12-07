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
- Cross-platform support

## Installation

```bash
git clone https://github.com/ekilie/beamdrop.git
cd beamdrop
go build -o beamdrop ./cmd/beam
```

## Usage

Start the server with a directory to share:

```bash
./beamdrop -dir /path/to/share
```

Available flags:
- `-dir` - Directory to share files from (default: current directory)
- `-port` - Port to run on (default: auto-detect available port)
- `-p` - Password for authentication
- `-no-qr` - Disable QR code generation
- `-v` - Show version information
- `-h` - Show help message

## Development

The project consists of:
- Go backend server with RESTful API
- React frontend with TypeScript
- SQLite database for statistics

## Future Plans

Planned features include S3-compatible storage backend support, allowing integration with cloud storage providers like AWS S3, MinIO, and other S3-compatible services for file storage and manipulation.

## License

MIT License
