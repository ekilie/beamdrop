package main

import (
	"flag"

	"github.com/tachRoutine/beamdrop-go/beam"
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


	// NOTE:Here i default it to 0 so when it zero we know that the flag wasnt passed
	// Since the flag is a non-boolean value
	port := flag.Int("port", 0, "Set the port that beamdrop will run on") 
	if *versionFlag {
		styles.InfoStyle.Println("Beamdrop Version:", config.VERSION)
		return
	}
	flag.Parse()

	flags := config.Flags{
		SharedDir: *sharedDir,
		NoQR:      *noQR,
		Help:      *help,
		Password:  *password,
		Port:      *port,
	}

	if flag.NArg() > 0 {
		logger.Debug("Extra arguments provided, showing help")
		PrintHelp()
		return
	}
	if *sharedDir == "" {
		logger.Error("Shared directory is required")
		return
	}
	if *help {
		// logger.Debug("Help flag provided, showing help")
		PrintHelp()
		return
	}

	logger.Info("Starting beamdrop application")
	logger.Info("Starting server with shared directory: %s", *sharedDir)
	beam.StartServer(*sharedDir, flags)
}
