package main

import (
	"flag"
	"log/slog"

	"github.com/tachRoutine/beamdrop-go/beam/server"
	"github.com/tachRoutine/beamdrop-go/config"
	"github.com/tachRoutine/beamdrop-go/pkg/logger"
	"github.com/tachRoutine/beamdrop-go/pkg/styles"
)

func main() {
	sharedDir := flag.String("dir", ".", "Directory to share files from")
	noQR := flag.Bool("no-qr", false, "Disable QR code generation")
	help := flag.Bool("h", false, "Show help message")
	password := flag.String("p", "", "Password authentication")
	versionFlag := flag.Bool("v", false, "Show version information")
	tlsCert := flag.String("tls-cert", "", "Path to TLS certificate file for HTTPS")
	tlsKey := flag.String("tls-key", "", "Path to TLS private key file for HTTPS")
	allowedOrigins := flag.String("allowed-origins", "", "Comma-separated list of allowed CORS origins (empty = CORS disabled)")
	apiAuth := flag.Bool("api-auth", false, "Enable API key authentication for S3-like API endpoints")
	logLevel := flag.String("log-level", "info", "Log level: debug, info, warn, error")
	rateLimit := flag.Int("rate-limit", 100, "General rate limit in requests/min per IP (0 = disabled)")

	// NOTE:Here i default it to 0 so when it zero we know that the flag wasnt passed
	// Since the flag is a non-boolean value
	port := flag.Int("port", 0, "Set the port that beamdrop will run on")
	if *versionFlag {
		styles.InfoStyle.Println("Beamdrop Version:", config.VERSION)
		return
	}
	flag.Parse()

	// Initialize structured logger before anything else.
	// Terminal gets colored human-readable output; JSON logs go to <sharedDir>/.beamdrop/beamdrop.log
	logger.Init(*logLevel, *sharedDir)

	flags := config.Flags{
		SharedDir:      *sharedDir,
		NoQR:           *noQR,
		Help:           *help,
		Password:       *password,
		Port:           *port,
		TLSCert:        *tlsCert,
		TLSKey:         *tlsKey,
		AllowedOrigins: *allowedOrigins,
		APIAuth:        *apiAuth,
		LogLevel:       *logLevel,
		RateLimit:      *rateLimit,
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
	if *help {
		PrintHelp()
		return
	}

	slog.Info("Starting beamdrop application")
	slog.Info("Starting server", "shared_dir", *sharedDir)

	srv := server.New(*sharedDir, flags)

	err := config.CreateTrashBin(*sharedDir)

	if err != nil {
		logger.Fatal("Failed to create trash bin", "error", err)
	}

	if err := srv.Start(); err != nil {
		logger.Fatal("Server error", "error", err)
	}
}
