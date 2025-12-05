package beam

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tachRoutine/beamdrop-go/pkg/db"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func StatsSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade to WebSocket", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	stats, err := db.GetStats()
	if err != nil {
		http.Error(w, "Failed to retrieve stats", http.StatusInternalServerError)
		return
	}

	for {
		if err := conn.WriteJSON(stats); err != nil {
			break
		}

		time.Sleep(1 * time.Minute) // Sends updates every minute
	}
}
