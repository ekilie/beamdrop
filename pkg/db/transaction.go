package db

import (
	"fmt"

	"github.com/tachRoutine/beamdrop-go/pkg/logger"
	"gorm.io/gorm"
)

// WithTransaction executes fn inside a database transaction.
// If fn returns an error the transaction is rolled back; otherwise it is committed.
// This is the canonical helper for wrapping multi-step DB operations.
func WithTransaction(fn func(tx *gorm.DB) error) error {
	database := GetDB()
	tx := database.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Error("Transaction panicked and was rolled back: %v", r)
			panic(r) // re-panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback().Error; rbErr != nil {
			logger.Error("Transaction rollback failed: %v (original error: %v)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
