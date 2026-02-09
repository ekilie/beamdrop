package server

import (
	"net/http"
	"strings"

	"github.com/tachRoutine/beamdrop-go/beam/server/handlers"
	"github.com/tachRoutine/beamdrop-go/beam/server/handlers/api"
)

func (s *Server) setupRoutes() {
	// Health and readiness endpoints (for deployment contexts)
	s.mux.HandleFunc("/health", handlers.HealthHandler)
	s.mux.HandleFunc("/ready", handlers.ReadinessHandler(s.sharedDir))

	// Static files
	s.mux.HandleFunc("/", handlers.StaticHandler)

	// Auth handlers
	authHandler := handlers.NewAuthHandler(s.passwordService)
	s.mux.HandleFunc("/auth/login", authHandler.Login)
	s.mux.HandleFunc("/auth/logout", authHandler.Logout)
	s.mux.HandleFunc("/auth/status", authHandler.Status)

	// Stats
	s.mux.HandleFunc("/stats", handlers.StatsHandler)
	s.mux.HandleFunc("/ws/stats", StatsSocketHandler(s.sharedDir)) //TODO: will come up with  better structure for the websockts

	// File handlers
	fileHandler := handlers.NewFileHandler(s.sharedDir)
	fileOpsHandler := handlers.NewFileOperationsHandler(s.sharedDir)

	// File operations
	s.mux.HandleFunc("/files", fileHandler.ListFiles)
	s.mux.HandleFunc("/download", fileHandler.Download)
	s.mux.HandleFunc("/upload", fileHandler.Upload)
	s.mux.HandleFunc("/move", fileOpsHandler.Move)
	s.mux.HandleFunc("/trash", fileOpsHandler.Trash)
	s.mux.HandleFunc("/copy", fileOpsHandler.Copy)
	s.mux.HandleFunc("/mkdir", fileOpsHandler.Mkdir)
	s.mux.HandleFunc("/rename", fileOpsHandler.Rename)
	s.mux.HandleFunc("/write", fileOpsHandler.Write)
	s.mux.HandleFunc("/search", fileOpsHandler.Search)
	s.mux.HandleFunc("/star", fileOpsHandler.Star)
	s.mux.HandleFunc("/starred", fileOpsHandler.Starred)

	// Shareable links
	shareLinkHandler := handlers.NewShareableLinkHandler(s.sharedDir)
	s.mux.HandleFunc("/api/shares", shareLinkHandler.Create)
	s.mux.HandleFunc("/api/shares/list", shareLinkHandler.List)
	s.mux.HandleFunc("/api/shares/delete", shareLinkHandler.Delete)
	s.mux.HandleFunc("/share/", shareLinkHandler.Access) // Public access endpoint

	// S3-like API endpoints
	s.setupAPIRoutes()
}

// setupAPIRoutes configures the S3-like API endpoints
func (s *Server) setupAPIRoutes() {
	bucketHandler := api.NewBucketHandler(s.sharedDir)
	objectHandler := api.NewObjectHandler(s.sharedDir)
	keysHandler := api.NewKeysHandler()

	// API auth middleware (disabled by default for now - enable with -api-auth flag)
	apiAuth := api.NewAPIAuthMiddleware(s.flags.APIAuth)

	// API v1 routes - handle bucket and object operations
	s.mux.HandleFunc("/api/v1/buckets/", func(w http.ResponseWriter, r *http.Request) {
		// Apply API auth middleware
		apiAuth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Route to appropriate handler based on path
			path := strings.TrimPrefix(r.URL.Path, "/api/v1/buckets/")
			parts := strings.SplitN(path, "/", 2)

			// If there's an object key, use object handler
			if len(parts) > 1 && parts[1] != "" {
				objectHandler.Handle(w, r)
				return
			}

			// Otherwise use bucket handler
			bucketHandler.Handle(w, r)
		})).ServeHTTP(w, r)
	})

	// List buckets endpoint (no trailing path)
	s.mux.HandleFunc("/api/v1/buckets", func(w http.ResponseWriter, r *http.Request) {
		apiAuth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bucketHandler.Handle(w, r)
		})).ServeHTTP(w, r)
	})

	// API key management endpoint (no auth required - managed via web UI with session auth)
	s.mux.HandleFunc("/api/v1/keys", keysHandler.Handle)
}
