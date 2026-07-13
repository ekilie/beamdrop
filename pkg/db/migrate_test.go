package db

import (
	"testing"
)

func TestAutoMigrate(t *testing.T) {
	setupTestDB(t)

	AutoMigrate()

	// Verify tables were created by checking we can query them
	var stats ServerStats
	if err := db.First(&stats).Error; err != nil {
		t.Fatalf("expected stats table to exist: %v", err)
	}
}

func TestCreateStatsTable(t *testing.T) {
	setupTestDB(t)

	CreateStatsTable()
	InitializeStats()

	var count int64
	db.Model(&ServerStats{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 stats record after CreateStatsTable+InitializeStats, got %d", count)
	}
}
