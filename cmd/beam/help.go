package main

import "github.com/tachRoutine/beamdrop-go/pkg/logger"

func Help() string {
	return `beamdrop - A simple file sharing tool

NOTE: YOU NEED TO BE IN THE SAME NETWORK AS THE RECEIVER

Usage:
  beam [options]

Options:
  -dir string
		Directory to share files from (default ".")
  -port int
		Set the port that beamdrop will run on (default: auto-detect)
  -p string
		Password for authentication
  -tls-cert string
		Path to TLS certificate file for HTTPS
  -tls-key string
		Path to TLS private key file for HTTPS
  -allowed-origins string
		Comma-separated list of allowed CORS origins (empty = CORS disabled for security)
  -no-qr
		Disable QR code generation
  -h
		Show this help message
  -v
		Show version information`
}

func PrintHelp() {
	logger.Info(Help())
}
