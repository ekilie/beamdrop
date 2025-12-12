package server

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tachRoutine/beamdrop-go/pkg/db"
	"github.com/tachRoutine/beamdrop-go/pkg/logger"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// StatsSocketHandler handles WebSocket connections for real-time stats updates
// It fetches fresh stats from the database on each interval and sends them to the client
func StatsSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade to WebSocket: %v", err)
		http.Error(w, "Failed to upgrade to WebSocket", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	logger.Debug("WebSocket connection established for stats")

	// Set up ping/pong handlers to keep connection alive
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Sending initial stats immediately
	stats, err := db.GetStats()
	if err != nil {
		logger.Error("Failed to retrieve initial stats: %v", err)
		conn.WriteJSON(map[string]string{"error": "Failed to retrieve stats"})
		return
	}
	if err := conn.WriteJSON(stats); err != nil {
		logger.Debug("WebSocket connection closed during initial stats send: %v", err)
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
					logger.Debug("WebSocket error: %v", err)
				}
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			logger.Debug("WebSocket connection closed by client")
			return

		case <-pingTicker.C:
			// Send ping to keep connection alive
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Debug("Failed to send ping, connection may be closed: %v", err)
				return
			}

		case <-ticker.C:
			// Fetch fresh stats from database on each interval
			stats, err := db.GetStats()
			if err != nil {
				logger.Error("Failed to retrieve stats from database: %v", err)
				// Send error message to client
				if err := conn.WriteJSON(map[string]any{
					"error": "Failed to retrieve stats",
				}); err != nil {
					logger.Debug("WebSocket connection closed during error send: %v", err)
					return
				}
				continue
			}

			// Send fresh stats to client
			if err := conn.WriteJSON(stats); err != nil {
				logger.Debug("WebSocket connection closed during stats send: %v", err)
				return
			}
			logger.Debug("Sent updated stats via WebSocket: Downloads=%d, Uploads=%d, Requests=%d",
				stats.Downloads, stats.Uploads, stats.Requests)
		}
	}
}
