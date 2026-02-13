package qr

import (
	"fmt"
	"os"

	"github.com/skip2/go-qrcode"
	"github.com/tachRoutine/beamdrop-go/pkg/logger"
)

// Generate a QR code and save it to a file
func Generate(data string, filename string) error {
	logger.Debug("Generating QR code for data: %s", data)
	qrCode, err := qrcode.New(data, qrcode.Medium)
	if err != nil {
		logger.Error("Failed to create QR code: %v", err)
		return err
	}

	// Print the QR code
	pngData, err := qrCode.PNG(256)
	if err != nil {
		logger.Error("Failed to generate PNG data: %v", err)
		return err
	}

	// Write PNG data to a file
	filePath := "./" + filename
	logger.Debug("Saving QR code to file: %s", filePath)
	file, err := os.Create(filePath)
	if err != nil {
		logger.Error("Failed to create file %s: %v", filePath, err)
		return err
	}
	defer file.Close()

	_, err = file.Write(pngData)
	if err != nil {
		logger.Error("Failed to write PNG data to file %s: %v", filePath, err)
		return err
	}

	logger.Info("QR code successfully saved to: %s", filePath)
	return nil
}

// ShowQrCode generates a QR code for the given URL and prints it to the terminal
func ShowQrCode(url string) {
	logger.Debug("Generating terminal QR code for URL: %s", url)
	qrCode, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		logger.Error("Error creating QR code for terminal: %v", err)
		return
	}
	logger.Info("QR code for %s:", url)
	fmt.Println(qrCode.ToSmallString(false))
}
