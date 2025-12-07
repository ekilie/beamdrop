package server

import (
	"fmt"
	"net"
	"net/http"

	"github.com/tachRoutine/beamdrop-go/config"
	"github.com/tachRoutine/beamdrop-go/pkg/db"
	"github.com/tachRoutine/beamdrop-go/pkg/logger"
	"github.com/tachRoutine/beamdrop-go/pkg/qr"
)

type Server struct {
	sharedDir string
	flags     config.Flags
	mux       *http.ServeMux
}

func New(sharedDir string, flags config.Flags) *Server {
	s := &Server{
		sharedDir: sharedDir,
		flags:     flags,
		mux:       http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: Will add other common middleware here
	db.IncrementRequests()
	s.mux.ServeHTTP(w, r)
}

func (s *Server) Start() error {
	db.AutoMigrate()

	if s.flags.Password != "" {
		logger.Info("Password is enabled")
	}

	port := s.getPort()
	ip := GetLocalIP()
	url := fmt.Sprintf("http://%s:%d", ip, port)

	if !s.flags.NoQR {
		qr.ShowQrCode(url)
	}

	logger.Info("Server started at %s sharing directory: %s", url, s.sharedDir)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), s)
}

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
