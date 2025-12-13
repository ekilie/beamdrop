package system

import (
	"syscall"

	"github.com/tachRoutine/beamdrop-go/pkg/logger"
)

// DiskUsage represents disk usage statistics
type DiskUsage struct {
	Total uint64
	Free  uint64
}

// getDiskUsage gets disk usage for the given path
func getDiskUsage(path string) (*DiskUsage, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		logger.Error("Failed to get disk stats for path %s: %v", path, err)
		return nil, err
	}

	// Calculating total and free bytes
	// Blocks are typically 512 bytes or 4096 bytes depending on the filesystem
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)

	return &DiskUsage{
		Total: total,
		Free:  free,
	}, nil
}
