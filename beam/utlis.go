package beam

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tachRoutine/beamdrop-go/pkg/logger"
)

func GetLocalIP() string {
	logger.Debug("Detecting local IP address")
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logger.Warn("Failed to get network interfaces: %v", err)
		return "localhost"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				logger.Debug("Found local IP: %s", ipnet.IP.String())
				return ipnet.IP.String()
			}
		}
	}

	logger.Warn("No local IP found, using localhost")
	logger.Info("This might be due to no active network connection.")
	return "localhost"
}

func FormatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
		PB = TB * 1024
	)

	switch {
	case size < KB:
		return fmt.Sprintf("%d B", size)
	case size < MB:
		val := float64(size) / KB
		if val == float64(int64(val)) {
			return fmt.Sprintf("%.0f KB", val)
		}
		return fmt.Sprintf("%.2f KB", val)
	case size < GB:
		val := float64(size) / MB
		if val == float64(int64(val)) {
			return fmt.Sprintf("%.0f MB", val)
		}
		return fmt.Sprintf("%.2f MB", val)
	case size < TB:
		val := float64(size) / GB
		if val == float64(int64(val)) {
			return fmt.Sprintf("%.0f GB", val)
		}
		return fmt.Sprintf("%.2f GB", val)
	case size < PB:
		val := float64(size) / TB
		if val == float64(int64(val)) {
			return fmt.Sprintf("%.0f TB", val)
		}
		return fmt.Sprintf("%.2f TB", val)
	default:
		val := float64(size) / PB
		if val == float64(int64(val)) {
			return fmt.Sprintf("%.0f PB", val)
		}
		return fmt.Sprintf("%.2f PB", val)
	}
}

func FormatModTime(modTime string) string {
	t, err := time.Parse(time.RFC3339, modTime)
	if err != nil {
		return modTime
	}
	return t.Format("2006-01-02 15:04:05")
}

// ResolvePath returns the absolute safe path inside sharedDir
func ResolvePath(sharedDir, raw string) (string, error) {
	clean := filepath.Clean(raw)
	target := filepath.Join(sharedDir, clean)

	absShared, err := filepath.Abs(sharedDir)
	if err != nil {
		return "", err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absTarget, absShared) {
		return "", fmt.Errorf("invalid path")
	}
	return absTarget, nil
}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
