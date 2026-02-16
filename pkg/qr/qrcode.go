package qr

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/skip2/go-qrcode"
)

// Generate a QR code and save it to a file
func Generate(data string, filename string) error {
	slog.Debug("Generating QR code", "data", data)
	qrCode, err := qrcode.New(data, qrcode.Medium)
	if err != nil {
		slog.Error("Failed to create QR code", "error", err)
		return err
	}

	// Print the QR code
	pngData, err := qrCode.PNG(256)
	if err != nil {
		slog.Error("Failed to generate PNG data", "error", err)
		return err
	}

	// Write PNG data to a file
	filePath := "./" + filename
	slog.Debug("Saving QR code to file", "path", filePath)
	file, err := os.Create(filePath)
	if err != nil {
		slog.Error("Failed to create file", "path", filePath, "error", err)
		return err
	}
	defer file.Close()

	_, err = file.Write(pngData)
	if err != nil {
		slog.Error("Failed to write PNG data to file", "path", filePath, "error", err)
		return err
	}

	slog.Info("QR code successfully saved", "path", filePath)
	return nil
}

// ShowQrCode generates a QR code for the given URL and prints it to the terminal
func ShowQrCode(url string) {
	slog.Debug("Generating terminal QR code", "url", url)
	qrCode, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		slog.Error("Error creating QR code for terminal", "error", err)
		return
	}
	slog.Info("QR code generated", "url", url)
	fmt.Println(qrCode.ToSmallString(false))
}
