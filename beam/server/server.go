package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ekilie/beamdrop/beam/server/handlers"
	"github.com/ekilie/beamdrop/config"
	"github.com/ekilie/beamdrop/pkg/auth"
	"github.com/ekilie/beamdrop/pkg/db"
	"github.com/ekilie/beamdrop/pkg/logger"
	"github.com/ekilie/beamdrop/pkg/metrics"
	"github.com/ekilie/beamdrop/pkg/middleware"
	"github.com/ekilie/beamdrop/pkg/qr"
	"github.com/ekilie/beamdrop/pkg/reqctx"
	"github.com/ekilie/beamdrop/pkg/storage"
	"github.com/ekilie/beamdrop/pkg/styles"
)

type Server struct {
	sharedDir             string
	flags                 config.Flags
	mux                   *http.ServeMux
	passwordService       *auth.PasswordService
	authMiddleware        *auth.AuthMiddleware
	rateLimiter           *middleware.RateLimiter
	httpServer            *http.Server
	orphanCleaner         *db.OrphanCleaner
	metricsCollector      *metrics.Collector
	stopRevocationCleanup func()
}

func New(sharedDir string, flags config.Flags) *Server {
	// Initialize password service
	passwordService := auth.NewPasswordService(flags.Password)

	// Initialize rate limiter
	rlConfig := middleware.DefaultRateLimiterConfig()
	if flags.RateLimit <= 0 {
		rlConfig.Enabled = false
	} else {
		rlConfig.GeneralRate = flags.RateLimit
		// Auth and upload tiers are stricter: 5% and 10% of general rate, with minimums
		rlConfig.AuthRate = max(1, flags.RateLimit/20)
		rlConfig.UploadRate = max(1, flags.RateLimit/10)
	}
	rlConfig.TrustedProxies = middleware.ParseTrustedProxies(flags.TrustedProxies)

	s := &Server{
		sharedDir:       sharedDir,
		flags:           flags,
		mux:             http.NewServeMux(),
		passwordService: passwordService,
		authMiddleware:  auth.NewAuthMiddleware(passwordService),
		rateLimiter:     middleware.NewRateLimiter(rlConfig),
	}
	s.setupRoutes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	db.IncrementRequests()

	// Build middleware chain
	var handler http.Handler = s.mux

	// Apply max storage check before writes
	handler = middleware.MaxStorageCheck(s.sharedDir, s.flags.MaxStorage)(handler)

	// Apply auth middleware
	handler = s.authMiddleware.Middleware(handler)

	// Apply CSRF protection (after auth, before rate limiting)
	handler = middleware.CSRFProtection(s.flags.DisableCSRF)(handler)

	// Apply rate limiting middleware
	handler = s.rateLimiter.Middleware(handler)

	// Apply CORS middleware
	corsConfig := s.getCORSConfig()
	handler = middleware.CORS(corsConfig)(handler)

	// Apply Prometheus metrics middleware (outermost to capture all requests)
	handler = metrics.Middleware(handler)

	// Apply security headers middleware
	enableHSTS := s.flags.TLSCert != "" && s.flags.TLSKey != ""
	handler = middleware.SecurityHeaders(enableHSTS, s.flags.DisableCSP)(handler)

	// Apply request context middleware (outermost – sets X-Request-ID and enriches context)
	handler = reqctx.Middleware()(handler)

	handler.ServeHTTP(w, r)
}

func (s *Server) Start() error {
	// Validate max storage against filesystem capacity
	if err := storage.ValidateMaxStorage(s.sharedDir, s.flags.MaxStorage); err != nil {
		return fmt.Errorf("storage validation failed: %w", err)
	}

	// Cleaningup any orphaned temp files from interrupted writes
	if err := storage.CleanupOrphanedTempFiles(s.sharedDir); err != nil {
		slog.Warn("Failed to clean up orphaned temp files", "error", err)
	}

	if s.flags.Password != "" {
		slog.Info("Password is enabled")
	}

	// Log rate limiting status
	if s.flags.RateLimit > 0 {
		slog.Info("Rate limiting enabled", "general", s.flags.RateLimit, "unit", "req/min")
	} else {
		slog.Warn("Rate limiting is disabled")
	}

	// Start orphan cleaner for background DB maintenance
	s.orphanCleaner = db.NewOrphanCleaner(s.sharedDir)
	s.orphanCleaner.Start()

	// Start Prometheus metrics background collector
	s.metricsCollector = metrics.NewCollector(s.sharedDir, 15*time.Second)
	s.metricsCollector.Start()

	// Start token revocation cleanup
	s.stopRevocationCleanup = auth.StartRevocationCleanup()

	port := s.getPort()
	ip := GetLocalIP()

	// Determine protocol based on TLS configuration
	protocol := "http"
	if s.flags.TLSCert != "" && s.flags.TLSKey != "" {
		protocol = "https"
	}

	url := fmt.Sprintf("%s://%s:%d", protocol, ip, port)

	if s.flags.QR {
		qr.ShowQrCode(url)
	}

	// Log CORS configuration
	if s.flags.AllowedOrigins != "" {
		slog.Info("CORS enabled", "origins", s.flags.AllowedOrigins)
	} else {
		slog.Info("CORS is disabled (most secure for local file sharing)")
	}

	slog.Info("Server started ", "shared_dir", s.sharedDir)
	styles.InfoStyle.Println("Server started at " + url)

	// Mark startup complete so /health/startup returns 200
	handlers.MarkStartupReady()

	addr := fmt.Sprintf(":%d", port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s,
	}

	// Start with TLS if configured
	if s.flags.TLSCert != "" && s.flags.TLSKey != "" {
		// Validate that TLS files exist
		if _, err := os.Stat(s.flags.TLSCert); os.IsNotExist(err) {
			logger.Fatal("TLS certificate file not found", "path", s.flags.TLSCert)
		}
		if _, err := os.Stat(s.flags.TLSKey); os.IsNotExist(err) {
			logger.Fatal("TLS key file not found", "path", s.flags.TLSKey)
		}

		slog.Info("Starting server with TLS enabled")
		return s.httpServer.ListenAndServeTLS(s.flags.TLSCert, s.flags.TLSKey)
	}

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server and all managed resources.
// It drains in-flight HTTP requests within the given context deadline,
// then closes the rate limiter, orphan cleaner, database, and logger.
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Initiating graceful shutdown...")

	// 1. Stop accepting new connections and drain in-flight requests
	var shutdownErr error
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			slog.Error("HTTP server shutdown error", "error", err)
			shutdownErr = err
		} else {
			slog.Info("HTTP server drained successfully")
		}
	}

	// 2. Stop rate limiter background cleanup
	if s.rateLimiter != nil {
		s.rateLimiter.Close()
		slog.Info("Rate limiter stopped")
	}

	// 3. Stop metrics collector
	if s.metricsCollector != nil {
		s.metricsCollector.Stop()
		slog.Info("Metrics collector stopped")
	}

	// 4. Stop orphan cleaner
	if s.orphanCleaner != nil {
		s.orphanCleaner.Stop()
		slog.Info("Orphan cleaner stopped")
	}

	// 5. Stop token revocation cleanup
	if s.stopRevocationCleanup != nil {
		s.stopRevocationCleanup()
		slog.Info("Token revocation cleanup stopped")
	}

	// 6. Close database connection
	if err := db.Close(); err != nil {
		slog.Error("Database close error", "error", err)
		if shutdownErr == nil {
			shutdownErr = err
		}
	} else {
		slog.Info("Database connection closed")
	}

	// 6. Close logger (flush remaining logs)
	logger.Close()
	slog.Info("Logger closed")

	return shutdownErr
}

// getPort returns the port to use for the server, either from the flags or by finding an available port
func (s *Server) getPort() int {
	// Find an available port from the default ports list
	port, err := config.FindAvailablePort()
	if err != nil {
		logger.Fatal("Failed to find available port", "error", err)
	}

	// If its greater than zero then the flag was passed in the cli args
	if s.flags.Port > 0 {
		if !config.IsPortAvailable(s.flags.Port) {
			slog.Error("Port not available, falling back", "requested", s.flags.Port, "fallback", port)
			return port
		}
		return s.flags.Port
	}
	return port
}

// GetLocalIP returns the local IP address
func GetLocalIP() string {
	slog.Debug("Detecting local IP address")
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		slog.Warn("Failed to get network interfaces", "error", err)
		return "localhost"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				slog.Debug("Found local IP", "ip", ipnet.IP.String())
				return ipnet.IP.String()
			}
		}
	}

	slog.Warn("No local IP found, using localhost")
	slog.Info("This might be due to no active network connection")
	return "localhost"
}

// getCORSConfig returns CORS configuration based on flags
func (s *Server) getCORSConfig() middleware.CORSConfig {
	config := middleware.DefaultCORSConfig()

	// Parse allowed origins from comma-separated string
	if s.flags.AllowedOrigins != "" {
		origins := strings.Split(s.flags.AllowedOrigins, ",")
		for i, origin := range origins {
			origins[i] = strings.TrimSpace(origin)
		}
		config.AllowedOrigins = origins
	}

	return config
}

// getAllowedOrigins returns the parsed list of allowed CORS origins
func (s *Server) getAllowedOrigins() []string {
	if s.flags.AllowedOrigins == "" {
		return nil
	}
	origins := strings.Split(s.flags.AllowedOrigins, ",")
	for i, origin := range origins {
		origins[i] = strings.TrimSpace(origin)
	}
	return origins
}
