package handlers

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/tachRoutine/beamdrop-go/pkg/db"
	"github.com/tachRoutine/beamdrop-go/pkg/logger"
)

// HealthHandler handles the /health endpoint for liveness checks
// This is a simple check that the server is running and responding
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "beamdrop",
	})
}

// ReadinessHandler handles the /ready endpoint for readiness checks
// This checks if the service is ready to accept traffic by verifying:
// - Database connection is working
// - Shared directory is accessible
func ReadinessHandler(sharedDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.Header().Set("Allow", "GET")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		checks := make(map[string]string)
		allHealthy := true

		// Check database connection
		dbInstance := db.GetDB()
		if dbInstance == nil {
			checks["database"] = "unavailable"
			allHealthy = false
		} else {
			sqlDB, err := dbInstance.DB()
			if err != nil {
				checks["database"] = "error: " + err.Error()
				allHealthy = false
			} else {
				if err := sqlDB.Ping(); err != nil {
					checks["database"] = "error: " + err.Error()
					allHealthy = false
				} else {
					checks["database"] = "ok"
				}
			}
		}

		// Check shared directory accessibility
		if sharedDir == "" {
			checks["shared_directory"] = "not configured"
			allHealthy = false
		} else {
			info, err := os.Stat(sharedDir)
			if err != nil {
				checks["shared_directory"] = "error: " + err.Error()
				allHealthy = false
			} else if !info.IsDir() {
				checks["shared_directory"] = "error: not a directory"
				allHealthy = false
			} else {
				// Check if directory is readable
				_, err := os.ReadDir(sharedDir)
				if err != nil {
					checks["shared_directory"] = "error: not readable - " + err.Error()
					allHealthy = false
				} else {
					checks["shared_directory"] = "ok"
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		
		if allHealthy {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "ready",
				"service": "beamdrop",
				"checks": checks,
			})
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			logger.Warn("Readiness check failed: %v", checks)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "not ready",
				"service": "beamdrop",
				"checks": checks,
			})
		}
	}
}
