package system

import (
	"fmt"
	"runtime"

	"github.com/tachRoutine/beamdrop-go/pkg/logger"
)

// SystemStats contains system resource statistics
type SystemStats struct {
	Memory MemoryStats `json:"memory"`
	Disk   DiskStats   `json:"disk"`
	CPU    CPUStats    `json:"cpu"`
}

// MemoryStats contains memory usage statistics
type MemoryStats struct {
	Total     uint64  `json:"total"`     // Total memory in bytes
	Available uint64  `json:"available"` // Available memory in bytes
	Used      uint64  `json:"used"`      // Used memory in bytes
	Percent   float64 `json:"percent"`   // Usage percentage (0-100)
}

// DiskStats contains disk usage statistics
type DiskStats struct {
	Total   uint64  `json:"total"`   // Total disk space in bytes
	Free    uint64  `json:"free"`    // Free disk space in bytes
	Used    uint64  `json:"used"`    // Used disk space in bytes
	Percent float64 `json:"percent"` // Usage percentage (0-100)
}

// CPUStats contains CPU statistics
type CPUStats struct {
	Cores      int `json:"cores"`      // Number of CPU cores
	Goroutines int `json:"goroutines"` // Number of active goroutines
}

// GetSystemStats collects current system statistics
func GetSystemStats(sharedDir string) SystemStats {
	var stats SystemStats

	// Get memory stats
	stats.Memory = getMemoryStats()

	// Get disk stats
	stats.Disk = getDiskStats(sharedDir)

	// Get CPU stats
	stats.CPU = getCPUStats()

	return stats
}

// getMemoryStats collects memory usage statistics
func getMemoryStats() MemoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Total memory allocated to the Go runtime
	total := m.Sys
	// Memory currently in use
	used := m.Alloc

	// Calculate available (this is approximate for Go runtime)
	available := total - used

	// Calculate percentage
	var percent float64
	if total > 0 {
		percent = (float64(used) / float64(total)) * 100
	}

	return MemoryStats{
		Total:     total,
		Available: available,
		Used:      used,
		Percent:   percent,
	}
}

// getDiskStats collects disk usage statistics for the shared directory
func getDiskStats(sharedDir string) DiskStats {
	if sharedDir == "" {
		logger.Warn("Shared directory not set, cannot get disk stats")
		return DiskStats{}
	}

	stat, err := getDiskUsage(sharedDir)
	if err != nil {
		logger.Error("Failed to get disk stats: %v", err)
		return DiskStats{}
	}

	total := stat.Total
	free := stat.Free
	used := total - free

	var percent float64
	if total > 0 {
		percent = (float64(used) / float64(total)) * 100
	}

	return DiskStats{
		Total:   total,
		Free:    free,
		Used:    used,
		Percent: percent,
	}
}

// getCPUStats collects CPU statistics
func getCPUStats() CPUStats {
	return CPUStats{
		Cores:      runtime.NumCPU(),
		Goroutines: runtime.NumGoroutine(),
	}
}

// FormatBytes converts bytes to human-readable format
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return formatFloat(float64(bytes)) + " B"
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return formatFloat(float64(bytes)/float64(div)) + " " + string("KMGTPE"[exp]) + "B"
}

// formatFloat formats a float to 1 decimal place
func formatFloat(f float64) string {
	return fmt.Sprintf("%.1f", f)
}
