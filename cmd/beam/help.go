package main

import (
	"log/slog"
)

func Help() string {
	return `beamdrop - Self-hosted file storage with web UI and S3-compatible API

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
  -db-path string
    Path to database file or directory (default: ~/.beamdrop/beamdrop.db). If a directory is provided, beamdrop.db is appended automatically
  -max-storage string
		Maximum total storage, e.g. 500MB, 10GB, 1TB (0 = unlimited)
  -shutdown-timeout duration
		Graceful shutdown timeout for draining connections (default 30s)
  -qr
		Enable QR code generation
  -h
		Show this help message
  -v
		Show version information

Environment Variables:
  All flags can be set via environment variables. CLI flags take precedence.

    BEAMDROP_DIR              -dir
    BEAMDROP_PASSWORD         -p
    BEAMDROP_PORT             -port
    BEAMDROP_DB_PATH          -db-path
    BEAMDROP_TLS_CERT         -tls-cert
    BEAMDROP_TLS_KEY          -tls-key
    BEAMDROP_ALLOWED_ORIGINS  -allowed-origins
    BEAMDROP_API_AUTH         -api-auth        ("true"/"1" to enable)
    BEAMDROP_QR               -qr              ("true"/"1" to enable)
    BEAMDROP_LOG_LEVEL        -log-level
    BEAMDROP_RATE_LIMIT       -rate-limit
    BEAMDROP_MAX_STORAGE      -max-storage
    BEAMDROP_SHUTDOWN_TIMEOUT -shutdown-timeout (Go duration, e.g. "30s")

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
