package db

import "log/slog"

func AutoMigrate() {
	slog.Info("Running database migrations")
	err := db.AutoMigrate(
		&ServerStats{}, 
		&Config{}, 
		&StarredFile{}, 
		&APIKey{}, 
		&ShareableLink{},
		&PresignedURL{},
		&Webhook{},
		&WebhookEvent{},
		&WebhookDelivery{})
	if err != nil {
		slog.Error("Failed to migrate database", "error", err)
	}

	// We initialize stats record if it doesn't exist //FIXME: Figure out if this is the best place for this
	InitializeStats()
}
