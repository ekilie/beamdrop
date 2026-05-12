package config

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	PORT = 7777

	ConfigDirName = ".beamdrop"

	// MaxUploadSize defines the maximum upload file size (500MB by default)
	MaxUploadSize int64 = 500 * 1024 * 1024 // 500MB in bytes //TODO: Will fix this,
	//  probably also should user should set this via a flag

	MultipartFormMaxMemory int64 = 10 << 30 // 10GB in bytes, for parsing multipart forms
)

// Set via -ldflags at build time
var (
	VERSION   = "0.0.1" // All these are set during the build
	Commit    = "unknown"
	BuildDate = "unknown"
)

var (
	ConfigDir  string
	ConfigPath string
	DBName     = "beamdrop.db"
	DBPath     string
	// AllowedMIMETypes defines the allowed MIME types for uploads
	// Empty list means all types are allowed
	AllowedMIMETypes = []string{
		// Images
		"image/jpeg", "image/png", "image/gif", "image/webp", "image/svg+xml", "image/bmp",
		// Documents
		"application/pdf", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint", "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"text/plain", "text/csv", "text/html", "text/css", "text/javascript",
		// Archives
		"application/zip", "application/x-rar-compressed", "application/x-7z-compressed",
		"application/x-tar", "application/gzip",
		// Audio
		"audio/mpeg", "audio/wav", "audio/ogg", "audio/mp4", "audio/webm", "audio/flac",
		// Video
		"video/mp4", "video/mpeg", "video/webm", "video/ogg", "video/x-msvideo", "video/quicktime",
		// Code
		"application/json", "application/xml", "application/javascript",
		// Other
		"application/octet-stream",
	}
)

type Config struct {
	PORT int
}

type Flags struct {
	SharedDir          string
	QR                 bool
	Port               int
	Help               bool
	Password           string
	TLSCert            string
	TLSKey             string
	AllowedOrigins     string        // Comma-separated list of allowed CORS origins
	APIAuth            bool          // Enable API key authentication for S3-like API
	LogLevel           string        // "debug", "info", "warn", "error" (default "info")
	RateLimit          int           // General rate limit in requests/min (0 = disabled)
	ShutdownTimeout    time.Duration // Graceful shutdown timeout (default 30s)
	DBPath             string        // Path to database file (default: <sharedDir>/.beamdrop/beamdrop.db)
	MaxStorage         int64         // Maximum total storage in bytes (0 = unlimited)
	TrustedProxies     string        // Comma-separated list of trusted proxy IPs/CIDRs
	JWTSecret          string        // Explicit JWT signing secret (min 32 bytes; empty = auto-generate and persist)
	DisableCSP         bool          // Skip setting Content-Security-Policy header
	DisableCSRF        bool          // Skip CSRF token validation
	DisableSystemStats bool          // Hide server disk/memory/CPU stats from the usage dashboard
}

func GetDBPath() string {
	return filepath.Join(ConfigDir, DBName)
}

func GetConfig() Config {
	return Config{
		PORT: PORT,
	}
}

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("failed to get home directory: %v", err)
	}
	ConfigDir = filepath.Join(homeDir, ConfigDirName)
	ConfigPath = filepath.Join(ConfigDir, "beamdrop.db") //FIXME: will fix this
	DBPath = GetDBPath()

	createConfigDir()

	if _, err := os.Stat(ConfigPath); os.IsNotExist(err) {
		createConfigDb()
	}
}

func createConfigDir() {
	if err := os.MkdirAll(ConfigDir, 0755); err != nil {
		log.Fatalf("failed to create config directory: %v", err)
	}
}

// CreateTrashBin creates the trash bin directory if it doesn't exist
// inside the sharedDir
func CreateTrashBin(sharedDir string) error {
	trashBinDir := filepath.Join(sharedDir, ".beamdrop_trash")
	if err := os.MkdirAll(trashBinDir, 0755); err != nil {
		return err
	}
	return nil
}

func createConfigDb() {
	file, err := os.Create(ConfigPath)
	if err != nil {
		log.Fatalf("failed to create config file: %v", err)
	}
	defer file.Close()
	// TODO: Load initial settings
	log.Printf("Created default config file at: %s", ConfigPath)
}
