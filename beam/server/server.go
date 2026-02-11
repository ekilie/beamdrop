package server

import (
	"fmt"
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
}

func New(sharedDir string, flags config.Flags) *Server {
	// Initialize password service
	passwordService := auth.NewPasswordService(flags.Password)

	s := &Server{
		sharedDir:       sharedDir,
		flags:           flags,
		mux:             http.NewServeMux(),
		passwordService: passwordService,
		authMiddleware:  auth.NewAuthMiddleware(passwordService),
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
		logger.Warn("Failed to clean up orphaned temp files: %v", err)
	}

	if s.flags.Password != "" {
		logger.Info("Password is enabled")
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
		logger.Info("CORS enabled for origins: %s", s.flags.AllowedOrigins)
	} else {
		logger.Info("CORS is disabled (most secure for local file sharing)")
	}

	logger.Info("Server started at %s sharing directory: %s", url, s.sharedDir)

	// Start with TLS if configured
	if s.flags.TLSCert != "" && s.flags.TLSKey != "" {
		// Validate that TLS files exist
		if _, err := os.Stat(s.flags.TLSCert); os.IsNotExist(err) {
			logger.Fatal("TLS certificate file not found: %s", s.flags.TLSCert)
		}
		if _, err := os.Stat(s.flags.TLSKey); os.IsNotExist(err) {
			logger.Fatal("TLS key file not found: %s", s.flags.TLSKey)
		}

		logger.Info("Starting server with TLS enabled")
		return http.ListenAndServeTLS(fmt.Sprintf(":%d", port), s.flags.TLSCert, s.flags.TLSKey, s)
	}

	return http.ListenAndServe(fmt.Sprintf(":%d", port), s)
}

// getPort returns the port to use for the server, either from the flags or by finding an available port
func (s *Server) getPort() int {
	// Find an available port from the default ports list
	port, err := config.FindAvailablePort()
	if err != nil {
		logger.Fatal("Failed to find available port: %v", err)
	}

	// If its greater than zero then the flag was passed in the cli args
	if s.flags.Port > 0 {
		if !config.IsPortAvailable(s.flags.Port) {
			logger.Error("Port %d is not available, falling back to port %d ", s.flags.Port, port)
			return port
		}
		return s.flags.Port
	}
	return port
}

// GetLocalIP returns the local IP address
func GetLocalIP() string {
	logger.Debug("Detecting local IP address")
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logger.Warn("Failed to get network interfaces: %v", err)
		return "localhost"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				logger.Debug("Found local IP: %s", ipnet.IP.String())
				return ipnet.IP.String()
			}
		}
	}

	logger.Warn("No local IP found, using localhost")
	logger.Info("This might be due to no active network connection.")
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
