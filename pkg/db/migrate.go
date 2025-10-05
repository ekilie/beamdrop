package db

import "github.com/tachRoutine/beamdrop-go/pkg/logger"

func AutoMigrate() {
	logger.Info("Running database migrations")
	err := db.AutoMigrate(&ServerStats{})
	if err != nil {
		logger.Error("failed to migrate database: %v", err)
	}
}
