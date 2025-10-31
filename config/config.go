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
)

var (
	ConfigDir  string
	ConfigPath string
	DBName     = "beamdrop.db"
	DBPath     string
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
		if isPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports found from the default list: %v", DefaultPorts)
}

// isPortAvailable checks if a port is available for use
func isPortAvailable(port int) bool {
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

func createConfigDb() {
	file, err := os.Create(ConfigPath)
	if err != nil {
		log.Fatalf("failed to create config file: %v", err)
	}
	defer file.Close()
	// TODO: Load initial settings
	log.Printf("Created default config file at: %s", ConfigPath)
}
