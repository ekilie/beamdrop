package storage

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"
	"time"

	"github.com/ekilie/beamdrop/pkg/system"
)

// DirUsage represents storage usage for a directory.
type DirUsage struct {
	UsedBytes    uint64    `json:"usedBytes"`
	TotalBytes   uint64    `json:"totalBytes,omitempty"`
	FreeBytes    uint64    `json:"freeBytes,omitempty"`
	UsagePercent float64   `json:"usagePercent"`
	LastUpdated  time.Time `json:"lastUpdated"`
}

// Cached usage to avoid frequent disk walks.
var (
	usageCache     DirUsage
	usageCacheLock sync.RWMutex
	usageCacheTTL  = 5 * time.Second
)

// GetDirStorageUsage returns cached or freshly computed usage for dir.
func GetDirStorageUsage(dir string) (DirUsage, error) {
	usageCacheLock.RLock()
	if time.Since(usageCache.LastUpdated) < usageCacheTTL && usageCache.UsedBytes > 0 {
		cached := usageCache
		usageCacheLock.RUnlock()
		return cached, nil
	}
	usageCacheLock.RUnlock()

	var usage DirUsage
	usage.TotalBytes, usage.FreeBytes = getFilesystemStats(dir)

	used, err := calculateDirSize(dir)
	if err != nil {
		return DirUsage{}, fmt.Errorf("failed to calculate directory size: %w", err)
	}

	usage.UsedBytes = used
	if usage.TotalBytes > 0 {
		usage.UsagePercent = float64(usage.UsedBytes) / float64(usage.TotalBytes) * 100
	}

	usage.LastUpdated = time.Now()

	usageCacheLock.Lock()
	usageCache = usage
	usageCacheLock.Unlock()

	return usage, nil
}

// calculateDirSize walks root, summing file sizes (skips .beamdrop dirs).
func calculateDirSize(root string) (uint64, error) {
	var size uint64

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			name := d.Name()
			if name == ".beamdrop" || name == ".beamdrop_data" || name == ".beamdrop_trash" {
				return filepath.SkipDir
			}
			return nil
		}

		if info, err := d.Info(); err == nil && !info.IsDir() {
			size += uint64(info.Size())
		}
		return nil
	})

	return size, err
}

// getFilesystemStats returns total and free bytes via statfs syscall.
func getFilesystemStats(path string) (total, free uint64) {
	stat := system.GetDiskUsage(path)
	return stat.Total, stat.Free
}

// ValidateMaxStorage checks that maxBytes does not exceed the filesystem's total capacity.
func ValidateMaxStorage(dir string, maxBytes int64) error {
	if maxBytes <= 0 {
		return nil
	}
	total, _ := getFilesystemStats(dir)
	if total == 0 {
		return fmt.Errorf("unable to determine filesystem size for %q", dir)
	}
	if uint64(maxBytes) > total {
		return fmt.Errorf(
			"max-storage (%d bytes) exceeds filesystem capacity (%d bytes) on %q",
			maxBytes, total, dir,
		)
	}
	return nil
}
