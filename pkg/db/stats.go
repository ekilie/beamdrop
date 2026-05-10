package db

import (
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

type ServerStats struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Downloads      int       `gorm:"column:downloads;default:0" json:"downloads"`
	Requests       int       `gorm:"column:requests;default:0" json:"requests"`
	Uploads        int       `gorm:"column:uploads;default:0" json:"uploads"`
	BytesUploaded  int64     `gorm:"column:bytes_uploaded;default:0" json:"bytesUploaded"`
	BytesDownloaded int64    `gorm:"column:bytes_downloaded;default:0" json:"bytesDownloaded"`
	StartTime      time.Time `gorm:"column:start_time;default:CURRENT_TIMESTAMP" json:"startTime"`
}

func (ServerStats) TableName() string {
	return "server_stats"
}
func CreateStatsTable() {
	err := db.AutoMigrate(&ServerStats{})
	if err != nil {
		slog.Error("Failed to migrate server stats table", "error", err)
	}
}

// InitializeStats creates an initial stats record if one doesn't exist
func InitializeStats() {
	db := GetDB()
	var stats ServerStats
	err := db.First(&stats).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		stats = ServerStats{
			Downloads: 0,
			Requests:  0,
			Uploads:   0,
			StartTime: time.Now(),
		}
		err = db.Create(&stats).Error
		if err != nil {
			slog.Error("Failed to create initial server stats", "error", err)
		} else {
			slog.Info("Initialized server stats record")
		}
	} else if err != nil {
		slog.Error("Failed to check for existing server stats", "error", err)
	}
}

// ResetStats resets the server stats to zero and updates the start time to now
func ResetStats() {
	db := GetDB()
	var stats ServerStats
	err := db.First(&stats).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Error("Failed to fetch server stats", "error", err)
			return
		}

		CreateStatsTable()
		return
	}
	stats.Downloads = 0
	stats.Requests = 0
	stats.Uploads = 0
	stats.BytesUploaded = 0
	stats.BytesDownloaded = 0
	stats.StartTime = time.Now()
	db.Save(&stats)
}

// IncrementDownloads increments the download count by 1
func IncrementDownloads() {
	db := GetDB()
	var stats ServerStats
	err := db.First(&stats).Error
	if err != nil {
		slog.Error("Failed to fetch server stats", "error", err)
		return
	}
	stats.Downloads++
	db.Save(&stats)
}

// IncrementRequests increments the request count by 1
func IncrementRequests() {
	db := GetDB()
	var stats ServerStats
	err := db.First(&stats).Error
	if err != nil {
		slog.Error("Failed to fetch server stats", "error", err)
		return
	}
	stats.Requests++
	db.Save(&stats)
}

// IncrementUploads increments the upload count by 1
func IncrementUploads() {
	db := GetDB()
	var stats ServerStats
	err := db.First(&stats).Error
	if err != nil {
		slog.Error("Failed to fetch server stats", "error", err)
		return
	}
	stats.Uploads++
	db.Save(&stats)
}

// AddBytesUploaded adds n bytes to the total bytes uploaded counter
func AddBytesUploaded(n int64) {
	db := GetDB()
	db.Model(&ServerStats{}).Where("id = 1").
		UpdateColumn("bytes_uploaded", gorm.Expr("bytes_uploaded + ?", n))
}

// AddBytesDownloaded adds n bytes to the total bytes downloaded counter
func AddBytesDownloaded(n int64) {
	db := GetDB()
	db.Model(&ServerStats{}).Where("id = 1").
		UpdateColumn("bytes_downloaded", gorm.Expr("bytes_downloaded + ?", n))
}

// Increment increments the specified field by 1
func Increment(field string) {
	switch field {
	case "downloads":
		IncrementDownloads()
	case "requests":
		IncrementRequests()
	case "uploads":
		IncrementUploads()
	default:
		slog.Warn("Unknown field to increment", "field", field)
	}
}

// GetStats retrieves the current server stats
func GetStats() (ServerStats, error) {
	db := GetDB()
	var stats ServerStats
	err := db.First(&stats).Error
	if err != nil {
		return stats, err
	}
	return stats, nil
}
