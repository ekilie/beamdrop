package server

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ekilie/beamdrop/pkg/db"
	"github.com/ekilie/beamdrop/pkg/storage"
	"github.com/ekilie/beamdrop/pkg/system"
	"github.com/gorilla/websocket"
)

// newUpgrader creates a WebSocket upgrader with origin validation.
// If allowedOrigins is empty, only same-origin requests are allowed.
func newUpgrader(allowedOrigins []string) websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				// No origin header — same-origin request, allow it
				return true
			}

			// If explicit origins are configured, check against them
			for _, allowed := range allowedOrigins {
				if allowed == "*" || strings.EqualFold(allowed, origin) {
					return true
				}
			}

			// If no CORS origins configured, allow same-origin only
			if len(allowedOrigins) == 0 {
				host := r.Host
				return strings.HasSuffix(origin, "://"+host)
			}

			slog.Warn("WebSocket origin rejected", "origin", origin)
			return false
		},
	}
}

// StorageUsage provides Beamdrop-specific storage metrics.
// It includes the actual disk usage of the shared directory, filesystem
// capacity, and any max-storage allocation set via the -max-storage flag.
type StorageUsage struct {
	// UsedBytes is the total bytes consumed by files in the shared directory
	// (excluding internal .beamdrop metadata directories).
	UsedBytes uint64 `json:"usedBytes"`
	// TotalBytes is the total capacity of the filesystem where the shared directory lives.
	TotalBytes uint64 `json:"totalBytes"`
	// FreeBytes is the free space on that filesystem.
	FreeBytes uint64 `json:"freeBytes"`
	// AllocatedBytes is the storage cap set via -max-storage (0 = unlimited).
	AllocatedBytes int64 `json:"allocatedBytes"`
	// AvailableBytes is the effective space available for new uploads:
	//   - If maxStorage is set: max(0, allocatedBytes - usedBytes)
	//   - If unlimited: free space on disk
	AvailableBytes uint64 `json:"availableBytes"`
	// UsagePercent is the percentage of the effective limit used:
	//   - If maxStorage is set: usedBytes / allocatedBytes * 100
	//   - If unlimited: usedBytes / totalBytes * 100
	UsagePercent float64 `json:"usagePercent"`
}

// ExtendedStats contains both database stats, system stats, and storage usage.
type ExtendedStats struct {
	Downloads       int                 `json:"downloads"`
	Requests        int                 `json:"requests"`
	Uploads         int                 `json:"uploads"`
	BytesUploaded   int64               `json:"bytesUploaded"`
	BytesDownloaded int64               `json:"bytesDownloaded"`
	StartTime       time.Time           `json:"startTime"`
	Storage         StorageUsage        `json:"storage"`
	System          *system.SystemStats `json:"system,omitempty"`
}

// StatsSocketHandler handles WebSocket connections for real-time stats updates
// It fetches fresh stats from the database and system on each interval and sends them to the client
func StatsSocketHandler(sharedDir string, allowedOrigins []string, disableSystemStats bool, maxStorage int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleStatsSocket(w, r, sharedDir, allowedOrigins, disableSystemStats, maxStorage)
	}
}

func handleStatsSocket(w http.ResponseWriter, r *http.Request, sharedDir string, allowedOrigins []string, disableSystemStats bool, maxStorage int64) {
	wsUpgrader := newUpgrader(allowedOrigins)
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade to WebSocket", "error", err)
		http.Error(w, "Failed to upgrade to WebSocket", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	slog.Debug("WebSocket connection established for stats")

	// Set up ping/pong handlers to keep connection alive
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Helper function to get extended stats
	getExtendedStats := func() (ExtendedStats, error) {
		dbStats, err := db.GetStats()
		if err != nil {
			return ExtendedStats{}, err
		}

		// Compute Beamdrop-specific storage usage
		var storageUsage StorageUsage
		dirUsage, dirErr := storage.GetDirStorageUsage(sharedDir)
		if dirErr == nil {
			storageUsage = computeStorageUsage(dirUsage.UsedBytes, dirUsage.TotalBytes, dirUsage.FreeBytes, maxStorage)
		} else {
			// Fallback: try to get filesystem stats directly
			sysStats := system.GetSystemStats(sharedDir)
			storageUsage = computeStorageUsage(0, sysStats.Disk.Total, sysStats.Disk.Free, maxStorage)
		}

		ext := ExtendedStats{
			Downloads:       dbStats.Downloads,
			Requests:        dbStats.Requests,
			Uploads:         dbStats.Uploads,
			BytesUploaded:   dbStats.BytesUploaded,
			BytesDownloaded: dbStats.BytesDownloaded,
			StartTime:       dbStats.StartTime,
			Storage:         storageUsage,
		}
		if !disableSystemStats {
			sysStats := system.GetSystemStats(sharedDir)
			ext.System = &sysStats
		}
		return ext, nil
	}

	// Sending initial stats immediately
	stats, err := getExtendedStats()
	if err != nil {
		slog.Error("Failed to retrieve initial stats", "error", err)
		conn.WriteJSON(map[string]string{"error": "Failed to retrieve stats"})
		return
	}
	if err := conn.WriteJSON(stats); err != nil {
		slog.Debug("WebSocket connection closed during initial stats send", "error", err)
		return
	}

	// Create a ticker for periodic updates (every minute)
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Create a ping ticker to keep connection alive (every 30 seconds)
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Channel to handle connection close
	done := make(chan struct{})

	// Handle incoming messages (for graceful close)
	go func() {
		defer close(done)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					slog.Debug("WebSocket error", "error", err)
				}
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			slog.Debug("WebSocket connection closed by client")
			return

		case <-pingTicker.C:
			// Send ping to keep connection alive
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Debug("Failed to send ping, connection may be closed", "error", err)
				return
			}

		case <-ticker.C:
			// Fetch fresh stats from database and system on each interval
			stats, err := getExtendedStats()
			if err != nil {
				slog.Error("Failed to retrieve stats", "error", err)
				// Send error message to client
				if err := conn.WriteJSON(map[string]any{
					"error": "Failed to retrieve stats",
				}); err != nil {
					slog.Debug("WebSocket connection closed during error send", "error", err)
					return
				}
				continue
			}

			// Send fresh stats to client
			if err := conn.WriteJSON(stats); err != nil {
				slog.Debug("WebSocket connection closed during stats send", "error", err)
				return
			}
			slog.Debug("Sent updated stats via WebSocket",
				"downloads", stats.Downloads,
				"uploads", stats.Uploads,
				"requests", stats.Requests)
		}
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
		// A storage cap is configured: available = max(0, cap - used)
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
		// No cap: available = free space on disk
		s.AvailableBytes = freeBytes
		if totalBytes > 0 {
			s.UsagePercent = float64(usedBytes) / float64(totalBytes) * 100
		}
	}

	return s
}
