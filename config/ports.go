package config

import (
	"fmt"
	"net"
)

// DefaultPorts is a list of ports to try if the primary port is unavailable
var DefaultPorts = []int{
	7777,
	8080,
	8888,
	9000,
	9999,
	3000,
	4000,
	5000,
	6000,
	8000,
	2020,
	3030,
	4040,
	5050,
	6060,
	7070,
	8181,
	8443,
	9090,
	9191,
}

// FindAvailablePort tries to find an available port from the default ports list
func FindAvailablePort() (int, error) {
	for _, port := range DefaultPorts {
		if IsPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports found from the default list: %v", DefaultPorts)
}

// IsPortAvailable checks if a port is available for use
func IsPortAvailable(port int) bool {
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	defer listener.Close()
	return true
}
