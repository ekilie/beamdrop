package db

import (
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// StarredFile represents a starred file in the database
type StarredFile struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	FilePath  string    `gorm:"column:file_path;uniqueIndex;not null" json:"filePath"`
	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"createdAt"`
}

func (StarredFile) TableName() string {
	return "starred_files"
}

// StarFile adds a file to the starred files list
func StarFile(filePath string) error {
	db := GetDB()
	var existing StarredFile
	err := db.Where("file_path = ?", filePath).First(&existing).Error

	// If already exists, return nil (already starred)
	if err == nil {
		return nil
	}

	// If error is not "record not found", return the error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		slog.Error("Failed to check if file is starred", "error", err)
		return err
	}

	// Create new starred file record
	starredFile := StarredFile{
		FilePath:  filePath,
		CreatedAt: time.Now(),
	}

	if err := db.Create(&starredFile).Error; err != nil {
		slog.Error("Failed to star file", "error", err)
		return err
	}

	slog.Info("File starred", "path", filePath)
	return nil
}

// UnstarFile removes a file from the starred files list
func UnstarFile(filePath string) error {
	db := GetDB()
	result := db.Where("file_path = ?", filePath).Delete(&StarredFile{})

	if result.Error != nil {
		slog.Error("Failed to unstar file", "error", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		// File was not starred, but that's okay for toggle behavior
		slog.Debug("File was not starred", "path", filePath)
		return nil
	}

	slog.Info("File unstarred", "path", filePath)
	return nil
}

// IsStarred checks if a file is starred
func IsStarred(filePath string) bool {
	db := GetDB()
	var starredFile StarredFile
	err := db.Where("file_path = ?", filePath).First(&starredFile).Error
	return err == nil
}

// GetStarredFiles retrieves all starred files
func GetStarredFiles() ([]StarredFile, error) {
	db := GetDB()
	var starredFiles []StarredFile
	err := db.Order("created_at DESC").Find(&starredFiles).Error
	if err != nil {
		slog.Error("Failed to get starred files", "error", err)
		return nil, err
	}
	return starredFiles, nil
}
