package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/tachRoutine/beamdrop-go/config"
	"github.com/tachRoutine/beamdrop-go/pkg/db"
)

// ---------------------------------------------------------------------------
// Startup tracking
// ---------------------------------------------------------------------------

// startupReady is set to 1 once server initialization is complete.
var startupReady atomic.Int32

// MarkStartupReady should be called after the server has finished all
// initialisation steps (DB migration, orphan cleanup, etc.) and is
// ready to serve traffic for the first time.
func MarkStartupReady() {
	startupReady.Store(1)
	slog.Info("Startup probe: server initialisation complete")
}

// ---------------------------------------------------------------------------
// Response types (K8s-compatible JSON)
// ---------------------------------------------------------------------------

// ComponentStatus describes the status of a single subsystem.
type ComponentStatus struct {
	Status  string `json:"status"`            // "ok" | "degraded" | "unavailable"
	Message string `json:"message,omitempty"` // human-readable detail
	Latency string `json:"latency,omitempty"` // e.g. "1.23ms"
}

// HealthResponse is the envelope returned by all health endpoints.
type HealthResponse struct {
	Status     string                     `json:"status"`     // "healthy" | "unhealthy"
	Service    string                     `json:"service"`    // always "beamdrop"
	Version    string                     `json:"version"`    // build version
	Timestamp  string                     `json:"timestamp"`  // RFC 3339
	Components map[string]ComponentStatus `json:"components"` // per-component checks
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// LivenessHandler returns 200 as long as the process is alive.
// This is the cheapest possible probe – no I/O.
//
//	GET /health/live
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeHealth(w, http.StatusOK, HealthResponse{
		Status:    "healthy",
		Service:   "beamdrop",
		Version:   config.VERSION,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Components: map[string]ComponentStatus{
			"process": {Status: "ok", Message: "running"},
		},
	})
}

// EnhancedReadinessHandler checks that the service can serve traffic:
// database is reachable and the shared directory is available.
//
//	GET /health/ready
func EnhancedReadinessHandler(sharedDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		components := make(map[string]ComponentStatus)
		allOK := true

		// --- Database ---
		components["database"] = checkDatabase()
		if components["database"].Status != "ok" {
			allOK = false
		}

		// --- Storage ---
		components["storage"] = checkStorage(sharedDir)
		if components["storage"].Status != "ok" {
			allOK = false
		}

		// --- Runtime ---
		components["runtime"] = checkRuntime()

		status := http.StatusOK
		statusText := "healthy"
		if !allOK {
			status = http.StatusServiceUnavailable
			statusText = "unhealthy"
			slog.Warn("Readiness check failed", "components", components)
		}

		writeHealth(w, status, HealthResponse{
			Status:     statusText,
			Service:    "beamdrop",
			Version:    config.VERSION,
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			Components: components,
		})
	}
}

// StartupHandler returns 200 once MarkStartupReady() has been called,
// 503 before that.  K8s uses this to know when the container has
// finished initialising so it can switch to liveness/readiness probes.
//
//	GET /health/startup
func StartupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ready := startupReady.Load() == 1
	status := http.StatusOK
	statusText := "healthy"
	compStatus := ComponentStatus{Status: "ok", Message: "initialisation complete"}

	if !ready {
		status = http.StatusServiceUnavailable
		statusText = "unhealthy"
		compStatus = ComponentStatus{Status: "unavailable", Message: "server is still starting"}
	}

	writeHealth(w, status, HealthResponse{
		Status:    statusText,
		Service:   "beamdrop",
		Version:   config.VERSION,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Components: map[string]ComponentStatus{
			"startup": compStatus,
		},
	})
}

// HealthIndexHandler responds on the base /health path with an overview
// of all probes, so operators can hit a single URL.
//
//	GET /health
func HealthIndexHandler(sharedDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		components := make(map[string]ComponentStatus)
		allOK := true

		// Process
		components["process"] = ComponentStatus{Status: "ok", Message: "running"}

		// Startup
		if startupReady.Load() == 1 {
			components["startup"] = ComponentStatus{Status: "ok", Message: "initialisation complete"}
		} else {
			components["startup"] = ComponentStatus{Status: "unavailable", Message: "server is still starting"}
			allOK = false
		}

		// Database
		components["database"] = checkDatabase()
		if components["database"].Status != "ok" {
			allOK = false
		}

		// Storage
		components["storage"] = checkStorage(sharedDir)
		if components["storage"].Status != "ok" {
			allOK = false
		}

		// Runtime
		components["runtime"] = checkRuntime()

		status := http.StatusOK
		statusText := "healthy"
		if !allOK {
			status = http.StatusServiceUnavailable
			statusText = "unhealthy"
		}

		writeHealth(w, status, HealthResponse{
			Status:     statusText,
			Service:    "beamdrop",
			Version:    config.VERSION,
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			Components: components,
		})
	}
}

// ---------------------------------------------------------------------------
// Component checks
// ---------------------------------------------------------------------------

func checkDatabase() ComponentStatus {
	dbInstance := db.GetDB()
	if dbInstance == nil {
		return ComponentStatus{Status: "unavailable", Message: "database instance is nil"}
	}

	sqlDB, err := dbInstance.DB()
	if err != nil {
		return ComponentStatus{Status: "unavailable", Message: err.Error()}
	}

	start := time.Now()
	if err := sqlDB.Ping(); err != nil {
		return ComponentStatus{Status: "unavailable", Message: err.Error()}
	}
	latency := time.Since(start)

	return ComponentStatus{
		Status:  "ok",
		Message: "connected",
		Latency: latency.String(),
	}
}

func checkStorage(sharedDir string) ComponentStatus {
	if sharedDir == "" {
		return ComponentStatus{Status: "unavailable", Message: "shared directory not configured"}
	}

	start := time.Now()
	info, err := os.Stat(sharedDir)
	if err != nil {
		return ComponentStatus{Status: "unavailable", Message: err.Error()}
	}
	if !info.IsDir() {
		return ComponentStatus{Status: "unavailable", Message: "path is not a directory"}
	}
	if _, err := os.ReadDir(sharedDir); err != nil {
		return ComponentStatus{Status: "unavailable", Message: "directory not readable: " + err.Error()}
	}
	latency := time.Since(start)

	return ComponentStatus{
		Status:  "ok",
		Message: "accessible",
		Latency: latency.String(),
	}
}

func checkRuntime() ComponentStatus {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return ComponentStatus{
		Status: "ok",
		Message: formatHealthBytes(m.Alloc) + " heap, " +
			formatHealthUint(uint64(runtime.NumGoroutine())) + " goroutines",
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func writeHealth(w http.ResponseWriter, status int, resp HealthResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func formatHealthBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return formatHealthUint(b) + "B"
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	suffix := []string{"KB", "MB", "GB", "TB"}
	val := float64(b) / float64(div)
	whole := int(val)
	frac := int((val - float64(whole)) * 10)
	return formatHealthUint(uint64(whole)) + "." + formatHealthUint(uint64(frac)) + suffix[exp]
}

func formatHealthUint(n uint64) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
