package storage

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/unix"
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
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, 0
	}

	total = uint64(stat.Blocks) * uint64(stat.Bsize)
	free = uint64(stat.Bavail) * uint64(stat.Bsize)
	return total, free
}
