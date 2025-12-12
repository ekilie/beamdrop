package server

import (
	"github.com/tachRoutine/beamdrop-go/beam/server/handlers"
)

func (s *Server) setupRoutes() {
	// Health and readiness endpoints (for deployment contexts)
	s.mux.HandleFunc("/health", handlers.HealthHandler)
	s.mux.HandleFunc("/ready", handlers.ReadinessHandler(s.sharedDir))

	// Static files
	s.mux.HandleFunc("/", handlers.StaticHandler)

	// Stats
	s.mux.HandleFunc("/stats", handlers.StatsHandler)
	s.mux.HandleFunc("/ws/stats", StatsSocketHandler) //TODO: will come up with  better structure for the websockts

	// File handlers
	fileHandler := handlers.NewFileHandler(s.sharedDir)
	fileOpsHandler := handlers.NewFileOperationsHandler(s.sharedDir)

	// File operations
	s.mux.HandleFunc("/files", fileHandler.ListFiles)
	s.mux.HandleFunc("/download", fileHandler.Download)
	s.mux.HandleFunc("/upload", fileHandler.Upload)
	s.mux.HandleFunc("/move", fileOpsHandler.Move)
	s.mux.HandleFunc("/copy", fileOpsHandler.Copy)
	s.mux.HandleFunc("/mkdir", fileOpsHandler.Mkdir)
	s.mux.HandleFunc("/rename", fileOpsHandler.Rename)
	s.mux.HandleFunc("/write", fileOpsHandler.Write)
	s.mux.HandleFunc("/search", fileOpsHandler.Search)
	s.mux.HandleFunc("/star", fileOpsHandler.Star)
	s.mux.HandleFunc("/starred", fileOpsHandler.Starred)
}
