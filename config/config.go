package config

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
)

const (
	PORT          = 7777
	VERSION       = "0.0.1"
	ConfigDirName = ".beamdrop"
	// MaxUploadSize defines the maximum upload file size (100MB by default)
	MaxUploadSize int64 = 100 * 1024 * 1024 // 100MB in bytes
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
	SharedDir string
	NoQR      bool
	Port      int
	Help      bool
	Password  string
}

func GetDBPath() string {
	return filepath.Join(ConfigDir, DBName)
}

func GetConfig() Config {
	return Config{
		PORT: PORT,
	}
}

// FindAvailablePort tries to find an available port from the default ports list
func FindAvailablePort() (int, error) {
	for _, port := range DefaultPorts {
		if IsPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports found from the default list: %v", DefaultPorts)
}

// IsPortAvailable checks if a port is available for use
func IsPortAvailable(port int) bool {
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	defer listener.Close()
	return true
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
	} else {
		// For now, we just log that we're loading the existing config
		log.Printf("Loading existing config from: %s", ConfigPath)
	}
}

func createConfigDir() {
	if err := os.MkdirAll(ConfigDir, 0755); err != nil {
		log.Fatalf("failed to create config directory: %v", err)
	}
}

// CreateTrashBin creates the trash bin directory if it doesn't exist
// inside the sharedDir
func CreateTrashBin(sharedDir string) error{
	trashBinDir := filepath.Join(sharedDir, ".beamdrop_trash")
	if err := os.MkdirAll(trashBinDir,0755); err != nil {	
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
