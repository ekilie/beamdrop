package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/ekilie/beamdrop/pkg/db"
	"github.com/ekilie/beamdrop/pkg/system"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ExtendedStats contains both database stats and system stats
type ExtendedStats struct {
	Downloads int                `json:"downloads"`
	Requests  int                `json:"requests"`
	Uploads   int                `json:"uploads"`
	StartTime time.Time          `json:"startTime"`
	System    system.SystemStats `json:"system"`
}

// StatsSocketHandler handles WebSocket connections for real-time stats updates
// It fetches fresh stats from the database and system on each interval and sends them to the client
func StatsSocketHandler(sharedDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleStatsSocket(w, r, sharedDir)
	}
}

func handleStatsSocket(w http.ResponseWriter, r *http.Request, sharedDir string) {
	conn, err := upgrader.Upgrade(w, r, nil)
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
		sysStats := system.GetSystemStats(sharedDir)
		return ExtendedStats{
			Downloads: dbStats.Downloads,
			Requests:  dbStats.Requests,
			Uploads:   dbStats.Uploads,
			StartTime: dbStats.StartTime,
			System:    sysStats,
		}, nil
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
				"requests", stats.Requests,
				"memory_pct", stats.System.Memory.Percent,
				"disk_pct", stats.System.Disk.Percent)
		}
	}
}
