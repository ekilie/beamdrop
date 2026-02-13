package db

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// setupTestDB creates an in-memory SQLite database for testing.
// It replaces the package-level `db` variable and runs migrations.
func setupTestDB(t *testing.T) {
	t.Helper()
	var err error
	db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(&ServerStats{}, &Config{}, &StarredFile{}, &APIKey{}, &ShareableLink{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
}

// --- Transaction tests ---

func TestWithTransaction_CommitsOnSuccess(t *testing.T) {
	setupTestDB(t)

	err := WithTransaction(func(tx *gorm.DB) error {
		return tx.Create(&StarredFile{FilePath: "/test/file.txt", CreatedAt: time.Now()}).Error
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var count int64
	db.Model(&StarredFile{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 starred file, got %d", count)
	}
}

func TestWithTransaction_RollsBackOnError(t *testing.T) {
	setupTestDB(t)

	err := WithTransaction(func(tx *gorm.DB) error {
		if err := tx.Create(&StarredFile{FilePath: "/test/file.txt", CreatedAt: time.Now()}).Error; err != nil {
			return err
		}
		return errors.New("simulated failure")
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var count int64
	db.Model(&StarredFile{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 starred files after rollback, got %d", count)
	}
}

func TestWithTransaction_MultipleOpsAtomicity(t *testing.T) {
	setupTestDB(t)

	// All-or-nothing: create two records, fail on third
	err := WithTransaction(func(tx *gorm.DB) error {
		if err := tx.Create(&StarredFile{FilePath: "/a.txt", CreatedAt: time.Now()}).Error; err != nil {
			return err
		}
		if err := tx.Create(&StarredFile{FilePath: "/b.txt", CreatedAt: time.Now()}).Error; err != nil {
			return err
		}
		return errors.New("fail after two inserts")
	})
	if err == nil {
		t.Fatal("expected error")
	}

	var count int64
	db.Model(&StarredFile{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 records after rollback, got %d", count)
	}
}

// --- Saga tests ---

func TestSaga_AllStepsSucceed(t *testing.T) {
	var order []string

	saga := NewSaga("test-succeed")
	saga.AddStep(SagaStep{
		Name:       "step-1",
		Action:     func() error { order = append(order, "a1"); return nil },
		Compensate: func() error { order = append(order, "c1"); return nil },
	})
	saga.AddStep(SagaStep{
		Name:       "step-2",
		Action:     func() error { order = append(order, "a2"); return nil },
		Compensate: func() error { order = append(order, "c2"); return nil },
	})

	if err := saga.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(order) != 2 || order[0] != "a1" || order[1] != "a2" {
		t.Fatalf("expected [a1, a2], got %v", order)
	}
}

func TestSaga_CompensatesOnFailure(t *testing.T) {
	var order []string

	saga := NewSaga("test-compensate")
	saga.AddStep(SagaStep{
		Name:       "step-1",
		Action:     func() error { order = append(order, "a1"); return nil },
		Compensate: func() error { order = append(order, "c1"); return nil },
	})
	saga.AddStep(SagaStep{
		Name:       "step-2",
		Action:     func() error { order = append(order, "a2"); return nil },
		Compensate: func() error { order = append(order, "c2"); return nil },
	})
	saga.AddStep(SagaStep{
		Name:       "step-3",
		Action:     func() error { return errors.New("boom") },
		Compensate: func() error { order = append(order, "c3"); return nil },
	})

	err := saga.Execute()
	if err == nil {
		t.Fatal("expected error")
	}

	// step-3 failed, so compensate step-2 then step-1 (reverse order)
	// step-3's compensate should NOT be called (it never completed)
	expected := []string{"a1", "a2", "c2", "c1"}
	if len(order) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Fatalf("expected %v, got %v", expected, order)
		}
	}
}

func TestSaga_DBPlusFS_Coordination(t *testing.T) {
	setupTestDB(t)

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	saga := NewSaga("db-fs-write")
	saga.AddStep(SagaStep{
		Name: "create-db-record",
		Action: func() error {
			return db.Create(&StarredFile{FilePath: "/test.txt", CreatedAt: time.Now()}).Error
		},
		Compensate: func() error {
			return db.Where("file_path = ?", "/test.txt").Delete(&StarredFile{}).Error
		},
	})
	saga.AddStep(SagaStep{
		Name: "write-file",
		Action: func() error {
			return os.WriteFile(filePath, []byte("hello"), 0644)
		},
		Compensate: func() error {
			return os.Remove(filePath)
		},
	})

	if err := saga.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Both DB record and file should exist
	var count int64
	db.Model(&StarredFile{}).Where("file_path = ?", "/test.txt").Count(&count)
	if count != 1 {
		t.Fatal("expected DB record to exist")
	}
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
}

func TestSaga_DBPlusFS_RollsBackDBWhenFSFails(t *testing.T) {
	setupTestDB(t)

	saga := NewSaga("db-fs-fail")
	saga.AddStep(SagaStep{
		Name: "create-db-record",
		Action: func() error {
			return db.Create(&StarredFile{FilePath: "/doomed.txt", CreatedAt: time.Now()}).Error
		},
		Compensate: func() error {
			return db.Where("file_path = ?", "/doomed.txt").Delete(&StarredFile{}).Error
		},
	})
	saga.AddStep(SagaStep{
		Name: "write-file-fails",
		Action: func() error {
			// Simulate FS failure: write to a path that can't exist
			return os.WriteFile("/nonexistent-dir-12345/file.txt", []byte("x"), 0644)
		},
		Compensate: func() error { return nil },
	})

	err := saga.Execute()
	if err == nil {
		t.Fatal("expected error from FS failure")
	}

	// DB record should have been rolled back (compensated)
	var count int64
	db.Model(&StarredFile{}).Where("file_path = ?", "/doomed.txt").Count(&count)
	if count != 0 {
		t.Fatalf("expected DB record to be cleaned up after FS failure, got %d", count)
	}
}

// --- Orphaned records cleanup tests ---

func TestOrphanCleaner_RemovesOrphanedStarredFiles(t *testing.T) {
	setupTestDB(t)
	tmpDir := t.TempDir()

	// Create a file that exists
	existingFile := filepath.Join(tmpDir, "exists.txt")
	if err := os.WriteFile(existingFile, []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}

	// Star both existing and non-existing files
	db.Create(&StarredFile{FilePath: "exists.txt", CreatedAt: time.Now()})
	db.Create(&StarredFile{FilePath: "gone.txt", CreatedAt: time.Now()})

	cleaner := NewOrphanCleaner(tmpDir)
	cleaner.RunOnce()

	var files []StarredFile
	db.Find(&files)

	if len(files) != 1 {
		t.Fatalf("expected 1 starred file after cleanup, got %d", len(files))
	}
	if files[0].FilePath != "exists.txt" {
		t.Fatalf("expected 'exists.txt' to remain, got %q", files[0].FilePath)
	}
}

func TestOrphanCleaner_RemovesOrphanedShareableLinks(t *testing.T) {
	setupTestDB(t)
	tmpDir := t.TempDir()

	// Create a file that exists
	existingFile := filepath.Join(tmpDir, "real.pdf")
	if err := os.WriteFile(existingFile, []byte("pdf"), 0644); err != nil {
		t.Fatal(err)
	}

	db.Create(&ShareableLink{Path: "real.pdf", Token: "tok1", CreatedAt: time.Now()})
	db.Create(&ShareableLink{Path: "deleted.pdf", Token: "tok2", CreatedAt: time.Now()})

	cleaner := NewOrphanCleaner(tmpDir)
	cleaner.RunOnce()

	var links []ShareableLink
	db.Find(&links)

	if len(links) != 1 {
		t.Fatalf("expected 1 link after cleanup, got %d", len(links))
	}
	if links[0].Token != "tok1" {
		t.Fatalf("expected 'tok1' to remain, got %q", links[0].Token)
	}
}

func TestOrphanCleaner_RemovesExpiredLinks(t *testing.T) {
	setupTestDB(t)
	tmpDir := t.TempDir()

	// Create files so they aren't cleaned up as orphans
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(1 * time.Hour)

	db.Create(&ShareableLink{Path: "a.txt", Token: "expired", ExpiresAt: &past, CreatedAt: time.Now()})
	db.Create(&ShareableLink{Path: "b.txt", Token: "valid", ExpiresAt: &future, CreatedAt: time.Now()})

	cleaner := NewOrphanCleaner(tmpDir)
	cleaner.RunOnce()

	var links []ShareableLink
	db.Find(&links)

	if len(links) != 1 {
		t.Fatalf("expected 1 link after cleanup, got %d", len(links))
	}
	if links[0].Token != "valid" {
		t.Fatalf("expected 'valid' to remain, got %q", links[0].Token)
	}
}

// --- Test DB failure during FS operations ---

func TestDBFailureDuringFSOperation_StarFile(t *testing.T) {
	setupTestDB(t)
	tmpDir := t.TempDir()

	filePath := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(filePath, []byte("image"), 0644); err != nil {
		t.Fatal(err)
	}

	// Simulate: DB insert succeeds, then FS rename fails → DB should roll back
	dbErr := errors.New("simulated DB constraint violation")

	saga := NewSaga("star-with-db-failure")
	saga.AddStep(SagaStep{
		Name: "verify-file-exists",
		Action: func() error {
			if _, err := os.Stat(filePath); err != nil {
				return err
			}
			return nil
		},
		Compensate: nil,
	})
	saga.AddStep(SagaStep{
		Name: "star-in-db",
		Action: func() error {
			return dbErr // Simulate DB failure
		},
		Compensate: nil,
	})

	err := saga.Execute()
	if err == nil {
		t.Fatal("expected saga to fail")
	}

	// File should still exist (FS untouched since it was a DB failure)
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("file should still exist after DB failure: %v", err)
	}
}

func TestTransaction_ConcurrentAPIKeyOperations(t *testing.T) {
	setupTestDB(t)

	// Test that creating and immediately disabling in a transaction is atomic
	err := WithTransaction(func(tx *gorm.DB) error {
		key := &APIKey{
			Name:        "test-key",
			AccessKeyID: "BDK_test123",
			SecretKey:   "sk_secret",
			Permissions: "read",
			CreatedAt:   time.Now(),
		}
		if err := tx.Create(key).Error; err != nil {
			return err
		}
		return tx.Model(&APIKey{}).Where("access_key_id = ?", "BDK_test123").Update("disabled", true).Error
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var key APIKey
	db.Where("access_key_id = ?", "BDK_test123").First(&key)
	if !key.Disabled {
		t.Fatal("expected key to be disabled after atomic transaction")
	}
}
