package db

import (
	"testing"
)

func TestInitializeStats(t *testing.T) {
	setupTestDB(t)

	InitializeStats()

	stats, err := GetStats()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Requests != 0 || stats.Downloads != 0 || stats.Uploads != 0 {
		t.Fatalf("expected zero stats, got %+v", stats)
	}
	if stats.StartTime.IsZero() {
		t.Fatal("expected non-zero start time")
	}
}

func TestInitializeStats_Idempotent(t *testing.T) {
	setupTestDB(t)

	InitializeStats()
	InitializeStats()

	stats, err := GetStats()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = stats
}

func TestIncrementDownloads(t *testing.T) {
	setupTestDB(t)
	InitializeStats()

	IncrementDownloads()
	IncrementDownloads()

	stats, _ := GetStats()
	if stats.Downloads != 2 {
		t.Fatalf("expected 2 downloads, got %d", stats.Downloads)
	}
}

func TestIncrementRequests(t *testing.T) {
	setupTestDB(t)
	InitializeStats()

	IncrementRequests()

	stats, _ := GetStats()
	if stats.Requests != 1 {
		t.Fatalf("expected 1 request, got %d", stats.Requests)
	}
}

func TestIncrementUploads(t *testing.T) {
	setupTestDB(t)
	InitializeStats()

	IncrementUploads()

	stats, _ := GetStats()
	if stats.Uploads != 1 {
		t.Fatalf("expected 1 upload, got %d", stats.Uploads)
	}
}

func TestAddBytesUploaded(t *testing.T) {
	setupTestDB(t)
	InitializeStats()

	AddBytesUploaded(1000)
	AddBytesUploaded(500)

	stats, _ := GetStats()
	if stats.BytesUploaded != 1500 {
		t.Fatalf("expected 1500 bytes uploaded, got %d", stats.BytesUploaded)
	}
}

func TestAddBytesDownloaded(t *testing.T) {
	setupTestDB(t)
	InitializeStats()

	AddBytesDownloaded(2000)

	stats, _ := GetStats()
	if stats.BytesDownloaded != 2000 {
		t.Fatalf("expected 2000 bytes downloaded, got %d", stats.BytesDownloaded)
	}
}

func TestIncrement(t *testing.T) {
	setupTestDB(t)
	InitializeStats()

	Increment("downloads")
	Increment("requests")
	Increment("uploads")
	Increment("unknown")

	stats, _ := GetStats()
	if stats.Downloads != 1 {
		t.Fatalf("expected 1 download, got %d", stats.Downloads)
	}
	if stats.Requests != 1 {
		t.Fatalf("expected 1 request, got %d", stats.Requests)
	}
	if stats.Uploads != 1 {
		t.Fatalf("expected 1 upload, got %d", stats.Uploads)
	}
}

func TestResetStats(t *testing.T) {
	setupTestDB(t)
	InitializeStats()

	IncrementDownloads()
	IncrementDownloads()
	IncrementRequests()

	ResetStats()

	stats, _ := GetStats()
	if stats.Downloads != 0 || stats.Requests != 0 || stats.Uploads != 0 {
		t.Fatalf("expected all stats zero after reset, got %+v", stats)
	}
}
