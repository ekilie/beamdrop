package main

import (
	"log/slog"
)

func Help() string {
	return `beamdrop - A simple file sharing tool with S3-like API

NOTE: YOU NEED TO BE IN THE SAME NETWORK AS THE RECEIVER

Usage:
  beam [options]

Options:
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
  -log-level string
		Log level: debug, info, warn, error (default "info")
		Colored output in terminal; JSON logs saved to <dir>/.beamdrop/beamdrop.log
  -rate-limit int
		General rate limit in requests/min per IP (default 100, 0 = disabled)
		Auth endpoints: 5% of general rate; Upload endpoints: 10% of general rate
  -no-qr
		Disable QR code generation
  -h
		Show this help message
  -v
		Show version information

S3-like API:
  When running, beamdrop exposes an S3-compatible API at /api/v1/
  Buckets are stored under the shared directory in a 'buckets' folder.
  
  Endpoints:
    GET    /api/v1/buckets           - List all buckets
    PUT    /api/v1/buckets/{name}    - Create bucket
    DELETE /api/v1/buckets/{name}    - Delete bucket
    GET    /api/v1/buckets/{b}/{key} - Download object
    PUT    /api/v1/buckets/{b}/{key} - Upload object
    DELETE /api/v1/buckets/{b}/{key} - Delete object`
}

func PrintHelp() {
	slog.Info(Help())
}
