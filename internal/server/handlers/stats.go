package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ekilie/beamdrop/pkg/db"
	"github.com/ekilie/beamdrop/pkg/storage"
	"github.com/ekilie/beamdrop/pkg/system"
)

// StorageUsage provides Beamdrop-specific storage metrics for the HTTP stats endpoint.
type StorageUsage struct {
	UsedBytes      uint64  `json:"usedBytes"`
	TotalBytes     uint64  `json:"totalBytes"`
	FreeBytes      uint64  `json:"freeBytes"`
	AllocatedBytes int64   `json:"allocatedBytes"`
	AvailableBytes uint64  `json:"availableBytes"`
	UsagePercent   float64 `json:"usagePercent"`
}

// StatsHandlerParams holds the parameters needed for the stats handler.
type StatsHandlerParams struct {
	SharedDir          string
	DisableSystemStats bool
	MaxStorage         int64
}

// StatsHandlerWithStorage returns an HTTP handler that includes storage metrics.
func StatsHandlerWithStorage(sharedDir string, disableSystemStats bool, maxStorage int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
			return
		}

		stats, err := db.GetStats()
		if err != nil {
			slog.Error("Failed to get server stats", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get server stats"})
			return
		}

		// Compute storage usage
		var storageUsage StorageUsage
		dirUsage, dirErr := storage.GetDirStorageUsage(sharedDir)
		if dirErr == nil {
			storageUsage = computeStorageUsage(dirUsage.UsedBytes, dirUsage.TotalBytes, dirUsage.FreeBytes, maxStorage)
		} else {
			sysStats := system.GetSystemStats(sharedDir)
			storageUsage = computeStorageUsage(0, sysStats.Disk.Total, sysStats.Disk.Free, maxStorage)
		}

		response := map[string]any{
			"downloads":       stats.Downloads,
			"requests":        stats.Requests,
			"uploads":         stats.Uploads,
			"bytesUploaded":   stats.BytesUploaded,
			"bytesDownloaded": stats.BytesDownloaded,
			"startTime":       stats.StartTime,
			"storage":         storageUsage,
		}

		if !disableSystemStats {
			sysStats := system.GetSystemStats(sharedDir)
			response["system"] = sysStats
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// computeStorageUsage builds the StorageUsage struct from raw filesystem
// metrics and the optional maxStorage allocation.
func computeStorageUsage(usedBytes, totalBytes, freeBytes uint64, maxStorage int64) StorageUsage {
	s := StorageUsage{
		UsedBytes:      usedBytes,
		TotalBytes:     totalBytes,
		FreeBytes:      freeBytes,
		AllocatedBytes: maxStorage,
	}

	if maxStorage > 0 {
		allocated := uint64(maxStorage)
		if usedBytes >= allocated {
			s.AvailableBytes = 0
			s.UsagePercent = 100.0
		} else {
			s.AvailableBytes = allocated - usedBytes
			if allocated > 0 {
				s.UsagePercent = float64(usedBytes) / float64(allocated) * 100
			}
		}
	} else {
		s.AvailableBytes = freeBytes
		if totalBytes > 0 {
			s.UsagePercent = float64(usedBytes) / float64(totalBytes) * 100
		}
	}

	return s
}

// StatsHandler is the legacy handler that returns DB stats only (no storage metrics).
// It is kept for backward compatibility; StatsHandlerWithStorage is preferred.
func StatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	stats, err := db.GetStats()
	if err != nil {
		slog.Error("Failed to get server stats", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get server stats"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
