package db

import (
	"errors"
	"time"

	"github.com/tachRoutine/beamdrop-go/pkg/logger"
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
		logger.Error("failed to check if file is starred: %v", err)
		return err
	}

	// Create new starred file record
	starredFile := StarredFile{
		FilePath:  filePath,
		CreatedAt: time.Now(),
	}

	if err := db.Create(&starredFile).Error; err != nil {
		logger.Error("failed to star file: %v", err)
		return err
	}

	logger.Info("File starred: %s", filePath)
	return nil
}

// UnstarFile removes a file from the starred files list
func UnstarFile(filePath string) error {
	db := GetDB()
	result := db.Where("file_path = ?", filePath).Delete(&StarredFile{})

	if result.Error != nil {
		logger.Error("failed to unstar file: %v", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		// File was not starred, but that's okay for toggle behavior
		logger.Debug("File was not starred: %s", filePath)
		return nil
	}

	logger.Info("File unstarred: %s", filePath)
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
		logger.Error("failed to get starred files: %v", err)
		return nil, err
	}
	return starredFiles, nil
}
