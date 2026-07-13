package db

import (
	"testing"
)

func TestConfigTableName(t *testing.T) {
	var cfg Config
	if cfg.TableName() != "server_config" {
		t.Fatalf("expected 'server_config', got %q", cfg.TableName())
	}
}

func TestConfigModel(t *testing.T) {
	setupTestDB(t)

	cfg := Config{Password: "test-hash"}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found Config
	db.First(&found)
	if found.Password != "test-hash" {
		t.Fatalf("expected 'test-hash', got %q", found.Password)
	}
}
