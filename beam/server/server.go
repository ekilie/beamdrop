package server

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/tachRoutine/beamdrop-go/config"
	"github.com/tachRoutine/beamdrop-go/pkg/auth"
	"github.com/tachRoutine/beamdrop-go/pkg/db"
	"github.com/tachRoutine/beamdrop-go/pkg/logger"
	"github.com/tachRoutine/beamdrop-go/pkg/middleware"
	"github.com/tachRoutine/beamdrop-go/pkg/qr"
	"github.com/tachRoutine/beamdrop-go/pkg/storage"
)

type Server struct {
	sharedDir       string
	flags           config.Flags
	mux             *http.ServeMux
	passwordService *auth.PasswordService
	authMiddleware  *auth.AuthMiddleware
	rateLimiter     *middleware.RateLimiter
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
	// Increment request counter
	db.IncrementRequests()

	// Build middleware chain
	var handler http.Handler = s.mux

	// Apply auth middleware
	handler = s.authMiddleware.Middleware(handler)

	// Apply rate limiting middleware
	handler = s.rateLimiter.Middleware(handler)

	// Apply CORS middleware
	corsConfig := s.getCORSConfig()
	handler = middleware.CORS(corsConfig)(handler)

	// Apply security headers middleware
	enableHSTS := s.flags.TLSCert != "" && s.flags.TLSKey != ""
	handler = middleware.SecurityHeaders(enableHSTS)(handler)

	handler.ServeHTTP(w, r)
}

func (s *Server) Start() error {
	db.AutoMigrate()

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

	port := s.getPort()
	ip := GetLocalIP()

	// Determine protocol based on TLS configuration
	protocol := "http"
	if s.flags.TLSCert != "" && s.flags.TLSKey != "" {
		protocol = "https"
	}

	url := fmt.Sprintf("%s://%s:%d", protocol, ip, port)

	if !s.flags.NoQR {
		qr.ShowQrCode(url)
	}

	// Log CORS configuration
	if s.flags.AllowedOrigins != "" {
		slog.Info("CORS enabled", "origins", s.flags.AllowedOrigins)
	} else {
		slog.Info("CORS is disabled (most secure for local file sharing)")
	}

	slog.Info("Server started", "url", url, "shared_dir", s.sharedDir)

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
		return http.ListenAndServeTLS(fmt.Sprintf(":%d", port), s.flags.TLSCert, s.flags.TLSKey, s)
	}

	return http.ListenAndServe(fmt.Sprintf(":%d", port), s)
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
