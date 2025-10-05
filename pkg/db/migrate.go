package db

import "github.com/tachRoutine/beamdrop-go/pkg/logger"

func AutoMigrate() {
	logger.Info("Running database migrations")
	err := db.AutoMigrate(&ServerStats{},&Config{})
	if err != nil {
		logger.Error("failed to migrate database: %v", err)
	}

	// We initialize stats record if it doesn't exist //FIXME: Figure out if this is the best place for this
	InitializeStats()
}
