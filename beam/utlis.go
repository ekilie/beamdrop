package beam

import (
	"fmt"
	"net"
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
	return "localhost"
}

func FormatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
	} else {
		return fmt.Sprintf("%.2f GB", float64(size)/(1024*1024*1024))
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
func ResolvePath(sharedDir,raw string) (string, error) {
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