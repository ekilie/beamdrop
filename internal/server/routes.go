package server

import (
	"net/http"
	"strings"

	"github.com/ekilie/beamdrop/internal/server/handlers"
	"github.com/ekilie/beamdrop/internal/server/handlers/api"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *Server) setupRoutes() {
	// Health probe endpoints (K8s-compatible)
	s.mux.HandleFunc("/health", handlers.HealthIndexHandler(s.sharedDir))
	s.mux.HandleFunc("/health/live", handlers.LivenessHandler)
	s.mux.HandleFunc("/health/ready", handlers.EnhancedReadinessHandler(s.sharedDir))
	s.mux.HandleFunc("/health/startup", handlers.StartupHandler)
	// Legacy alias
	s.mux.HandleFunc("/ready", handlers.EnhancedReadinessHandler(s.sharedDir))

	// Prometheus metrics
	s.mux.Handle("/metrics", promhttp.Handler())

	// LLM discoverability
	s.mux.HandleFunc("/llms.txt", handlers.LLMsTxtHandler)

	// MCP endpoint (Model Context Protocol) — requires API key auth
	mcpHandler := handlers.NewMCPHandler(s.sharedDir)
	apiAuthForMCP := api.NewAPIAuthMiddleware(s.flags.APIAuth)
	s.mux.HandleFunc("/api/mcp", func(w http.ResponseWriter, r *http.Request) {
		// GET /api/mcp is public (discovery info)
		if r.Method == http.MethodGet || r.Method == http.MethodOptions {
			mcpHandler.ServeHTTP(w, r)
			return
		}
		// POST /api/mcp requires API key authentication
		apiAuthForMCP.Middleware(mcpHandler).ServeHTTP(w, r)
	})

	// Static files
	s.mux.HandleFunc("/", handlers.StaticHandler)

	// Auth handlers
	authHandler := handlers.NewAuthHandler(s.passwordService)
	s.mux.HandleFunc("/auth/login", authHandler.Login)
	s.mux.HandleFunc("/auth/logout", authHandler.Logout)
	s.mux.HandleFunc("/auth/status", authHandler.Status)

	// Stats
	s.mux.HandleFunc("/stats", handlers.StatsHandlerWithStorage(s.sharedDir, s.flags.DisableSystemStats, s.flags.MaxStorage))
	s.mux.HandleFunc("/ws/stats", StatsSocketHandler(s.sharedDir, s.getAllowedOrigins(), s.flags.DisableSystemStats, s.flags.MaxStorage))

	// Logs
	s.mux.HandleFunc("/api/logs", handlers.LogsHandler())

	// File handlers
	fileHandler := handlers.NewFileHandler(s.sharedDir)
	fileOpsHandler := handlers.NewFileOperationsHandler(s.sharedDir)

	// Presigned URL downloads (public   no auth)
	downloadHandler := handlers.NewDownloadHandler(s.sharedDir)
	s.mux.Handle("/dl/", downloadHandler)

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
	s.mux.HandleFunc("/api/shares/access/", shareLinkHandler.Access) // Public access API endpoint

	// S3-like API endpoints
	s.setupS3APIRoutes()
}

// setupS3APIRoutes configures the S3-like API endpoints
func (s *Server) setupS3APIRoutes() {
	bucketHandler := api.NewBucketHandler(s.sharedDir)
	objectHandler := api.NewObjectHandler(s.sharedDir)
	keysHandler := api.NewKeysHandler()

	// API auth middleware (enabled by default - disable with -api-auth=false flag)
	apiAuth := api.NewAPIAuthMiddleware(s.flags.APIAuth)

	// API v1 routes - handle bucket and object operations
	s.mux.HandleFunc("/api/v1/buckets/", func(w http.ResponseWriter, r *http.Request) {
		// Apply API auth middleware
		apiAuth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// We route to appropriate handler based on path
			path := strings.TrimPrefix(r.URL.Path, "/api/v1/buckets/")
			parts := strings.SplitN(path, "/", 2)

			// If there's an object key, we use object handler
			if len(parts) > 1 && parts[1] != "" {
				objectHandler.Handle(w, r)
				return
			}

			// Otherwise we use bucket handler
			bucketHandler.Handle(w, r)
		})).ServeHTTP(w, r)
	})

	// List buckets endpoint (no trailing path)
	s.mux.HandleFunc("/api/v1/buckets", func(w http.ResponseWriter, r *http.Request) {
		apiAuth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bucketHandler.Handle(w, r)
		})).ServeHTTP(w, r)
	})

	// API key management endpoint (requires session auth via AuthMiddleware)
	s.mux.HandleFunc("/api/v1/keys", keysHandler.Handle)

	// Presigned URL management
	presignHandler := api.NewPresignHandler(s.sharedDir)

	// Webhook management
	webhookHandler := api.NewWebhookHandler()

	// We support both /api/v1/presign and /api/v1/presign/ for convenience
	// (some clients may add trailing slash)
	s.mux.HandleFunc("/api/v1/presign/", func(w http.ResponseWriter, r *http.Request) {
		apiAuth.Middleware(http.HandlerFunc(presignHandler.Handle)).ServeHTTP(w, r)
	})
	s.mux.HandleFunc("/api/v1/presign", func(w http.ResponseWriter, r *http.Request) {
		apiAuth.Middleware(http.HandlerFunc(presignHandler.Handle)).ServeHTTP(w, r)
	})

	// Webhook management — S3 API auth (programmatic access)
	s.mux.HandleFunc("/api/v1/webhooks/", func(w http.ResponseWriter, r *http.Request) {
		apiAuth.Middleware(http.HandlerFunc(webhookHandler.Handle)).ServeHTTP(w, r)
	})
	s.mux.HandleFunc("/api/v1/webhooks", func(w http.ResponseWriter, r *http.Request) {
		apiAuth.Middleware(http.HandlerFunc(webhookHandler.Handle)).ServeHTTP(w, r)
	})

	// Webhook management — session auth (dashboard UI)
	s.mux.HandleFunc("/api/webhooks/", webhookHandler.Handle)
	s.mux.HandleFunc("/api/webhooks", webhookHandler.Handle)
}
