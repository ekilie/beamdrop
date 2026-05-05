package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ekilie/beamdrop/beam/server"
	"github.com/ekilie/beamdrop/config"
	"github.com/ekilie/beamdrop/pkg/db"
	"github.com/ekilie/beamdrop/pkg/logger"
	"github.com/ekilie/beamdrop/pkg/styles"
	"github.com/ekilie/beamdrop/pkg/system"
	"github.com/ekilie/beamdrop/static"
)

// envStr returns the environment variable value if the flag was not explicitly set.
func envStr(flagName, envName string, flagVal *string) {
	if isFlagSet(flagName) {
		return
	}
	if v := os.Getenv(envName); v != "" {
		*flagVal = v
	}
}

// envBool sets the bool flag from an env var ("true"/"1") when the flag was not explicitly set.
func envBool(flagName, envName string, flagVal *bool) {
	if isFlagSet(flagName) {
		return
	}
	if v := os.Getenv(envName); v != "" {
		*flagVal = strings.EqualFold(v, "true") || v == "1"
	}
}

// envInt sets the int flag from an env var when the flag was not explicitly set.
func envInt(flagName, envName string, flagVal *int) {
	if isFlagSet(flagName) {
		return
	}
	if v := os.Getenv(envName); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			*flagVal = n
		}
	}
}

// envDuration sets the duration flag from an env var when the flag was not explicitly set.
func envDuration(flagName, envName string, flagVal *time.Duration) {
	if isFlagSet(flagName) {
		return
	}
	if v := os.Getenv(envName); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			*flagVal = d
		}
	}
}

// isFlagSet reports whether a flag was explicitly provided on the command line.
func isFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// parseByteSize parses a human-readable size string (e.g. "10GB", "500MB", "1TB") into bytes.
// Also accepts raw byte values (e.g. "1073741824"). Returns 0 for empty or "0".
func parseByteSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" {
		return 0, nil
	}

	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n, nil
	}

	upper := strings.ToUpper(s)
	multipliers := []struct {
		suffix string
		mult   int64
	}{
		{"TB", 1 << 40},
		{"GB", 1 << 30},
		{"MB", 1 << 20},
		{"KB", 1 << 10},
		{"B", 1},
	}

	for _, m := range multipliers {
		if strings.HasSuffix(upper, m.suffix) {
			numStr := strings.TrimSpace(strings.TrimSuffix(upper, m.suffix))
			n, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid size value: %q", s)
			}
			return int64(n * float64(m.mult)), nil
		}
	}

	return 0, fmt.Errorf("invalid size format: %q (use e.g. 500MB, 10GB, 1TB)", s)
}

func main() {
	sharedDir := flag.String("dir", ".", "Directory to share files from")
	QR := flag.Bool("qr", false, "Enable QR code generation")
	help := flag.Bool("h", false, "Show help message")
	password := flag.String("p", "", "Password authentication")
	dbPath := flag.String("db-path", "", "Path to database file (default: <sharedDir>/.beamdrop/beamdrop.db)")
	versionFlag := flag.Bool("v", false, "Show version information")
	tlsCert := flag.String("tls-cert", "", "Path to TLS certificate file for HTTPS")
	tlsKey := flag.String("tls-key", "", "Path to TLS private key file for HTTPS")
	allowedOrigins := flag.String("allowed-origins", "", "Comma-separated list of allowed CORS origins (empty = CORS disabled)")
	apiAuth := flag.Bool("api-auth", true, "Enable API key authentication for S3-like API endpoints (disable with -api-auth=false)")
	logLevel := flag.String("log-level", "info", "Log level: debug, info, warn, error")
	rateLimit := flag.Int("rate-limit", 100, "General rate limit in requests/min per IP (0 = disabled)")
	shutdownTimeout := flag.Duration("shutdown-timeout", 30*time.Second, "Graceful shutdown timeout")
	maxStorageStr := flag.String("max-storage", "0", "Maximum total storage, e.g. 500MB, 10GB, 1TB (0 = unlimited)")
	trustedProxies := flag.String("trusted-proxies", "", "Comma-separated list of trusted proxy IPs/CIDRs for X-Forwarded-For (empty = trust direct connection only)")

	// NOTE:Here i default it to 0 so when it zero we know that the flag wasnt passed
	// Since the flag is a non-boolean value
	port := flag.Int("port", 0, "Set the port that beamdrop will run on")
	flag.Parse()

	if *versionFlag {
		styles.InfoStyle.Println("Beamdrop Version:", config.VERSION)
		return
	}

	if *help {
		PrintHelp()
		return
	}

	fmt.Println(static.BeamdropAscii)

	// Environment variable fallbacks (CLI flags always take precedence)
	envStr("dir", "BEAMDROP_DIR", sharedDir)
	envStr("p", "BEAMDROP_PASSWORD", password)
	envStr("db-path", "BEAMDROP_DB_PATH", dbPath)
	envStr("tls-cert", "BEAMDROP_TLS_CERT", tlsCert)
	envStr("tls-key", "BEAMDROP_TLS_KEY", tlsKey)
	envStr("allowed-origins", "BEAMDROP_ALLOWED_ORIGINS", allowedOrigins)
	envStr("log-level", "BEAMDROP_LOG_LEVEL", logLevel)
	envBool("qr", "BEAMDROP_QR", QR)
	envBool("api-auth", "BEAMDROP_API_AUTH", apiAuth)
	envInt("port", "BEAMDROP_PORT", port)
	envInt("rate-limit", "BEAMDROP_RATE_LIMIT", rateLimit)
	envDuration("shutdown-timeout", "BEAMDROP_SHUTDOWN_TIMEOUT", shutdownTimeout)
	envStr("max-storage", "BEAMDROP_MAX_STORAGE", maxStorageStr)
	envStr("trusted-proxies", "BEAMDROP_TRUSTED_PROXIES", trustedProxies)

	// Parse max-storage from human-readable format
	maxStorage, err := parseByteSize(*maxStorageStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid -max-storage value %q: %v\n", *maxStorageStr, err)
		os.Exit(1)
	}

	// Validate max-storage does not exceed the device's disk capacity
	if maxStorage > 0 {
		disk := system.GetDiskUsage(*sharedDir)
		if disk.Total > 0 && maxStorage > int64(disk.Total) {
			fmt.Fprintf(os.Stderr, "Error: -max-storage %s exceeds device capacity of %s\n",
				*maxStorageStr, system.FormatBytes(disk.Total))
			os.Exit(1)
		}
	}

	// Initialize structured logger before anything else.
	// Terminal gets colored human-readable output; JSON logs go to <sharedDir>/.beamdrop/beamdrop.log
	logger.Init(*logLevel, *sharedDir)

	// Apply custom DB path if provided in the flag
	if *dbPath != "" {
		config.SetDBPath(*dbPath)
	}

	// Initialize database now that the DB path is finalized
	db.Init()
	db.AutoMigrate()

	flags := config.Flags{
		SharedDir:       *sharedDir,
		QR:              *QR,
		Help:            *help,
		Password:        *password,
		Port:            *port,
		TLSCert:         *tlsCert,
		TLSKey:          *tlsKey,
		AllowedOrigins:  *allowedOrigins,
		APIAuth:         *apiAuth,
		LogLevel:        *logLevel,
		RateLimit:       *rateLimit,
		ShutdownTimeout: *shutdownTimeout,
		DBPath:          *dbPath,
		MaxStorage:      maxStorage,
		TrustedProxies:  *trustedProxies,
	}

	if flag.NArg() > 0 {
		slog.Debug("Extra arguments provided, showing help")
		PrintHelp()
		return
	}
	if *sharedDir == "" {
		slog.Error("Shared directory is required")
		return
	}

	slog.Info("Starting beamdrop application")
	slog.Info("Starting server", "shared_dir", *sharedDir)

	srv := server.New(*sharedDir, flags)

	err = config.CreateTrashBin(*sharedDir)

	if err != nil {
		logger.Fatal("Failed to create trash bin", "error", err)
	}

	// Channel to receive server errors
	serverErr := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	// Wait for interrupt signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("Received shutdown signal", "signal", sig.String())
	case err := <-serverErr:
		if err != nil {
			slog.Error("Server failed", "error", err)
		}
	}

	// Graceful shutdown with timeout
	slog.Info("Shutting down gracefully", "timeout", flags.ShutdownTimeout)
	ctx, cancel := context.WithTimeout(context.Background(), flags.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Shutdown completed with errors", "error", err)
		os.Exit(1)
	}

	slog.Info("Shutdown complete")
}
