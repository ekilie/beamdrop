package db

import (
	"errors"
	"time"

	"github.com/tachRoutine/beamdrop-go/pkg/logger"
	"gorm.io/gorm"
)

type ServerStats struct{
	Downloads int       `gorm:"column:downloads, default:0" json:"downloads"`
	Requests  int       `gorm:"column:requests, default:0" json:"requests"`
    Uploads   int       `gorm:"column:uploads, default:0" json:"uploads"`
    StartTime time.Time `gorm:"column:start_time, default:CURRENT_TIMESTAMP" json:"startTime"`
}

func (ServerStats) TableName() string {
	return "server_stats"
}
func CreateStatsTable() {
	db := GetDB()
	err := db.AutoMigrate(&ServerStats{})
	if err != nil {
		logger.Error("failed to migrate server stats table: %v", err)
	}
}

// ResetStats resets the server stats to zero and updates the start time to now
func ResetStats(){
	db := GetDB()
	var stats ServerStats
	err :=db.First(&stats).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound){
			logger.Error("failed to fetch server stats: %v", err)
			return
		}

		CreateStatsTable()
		return
	}
	stats.Downloads = 0
	stats.Requests = 0
	stats.Uploads = 0
	stats.StartTime = time.Now()
	db.Save(&stats)
}

// IncrementDownloads increments the download count by 1
func IncrementDownloads(){
	db := GetDB()
	var stats ServerStats
	err := db.First(&stats).Error
	if err != nil {
		// If no record exists, I dont expect this to happen, so there is no need to create one
		logger.Error("failed to fetch server stats: %v", err)
		return
	}
	stats.Downloads++
	db.Save(&stats)
}

// IncrementDownloads increments the download count by 1
func IncrementRequests(){
	db := GetDB()
	var stats ServerStats
	err := db.First(&stats).Error
	if err != nil {
		// If no record exists, I also dont expect this to happen, so there is no need to create one
		logger.Error("failed to fetch server stats: %v", err)
		return
	}
	stats.Requests++
	db.Save(&stats)
}

// IncrementDownloads increments the download count by 1
func IncrementUploads(){
	db := GetDB()
	var stats ServerStats
	err := db.First(&stats).Error
	if err != nil {
		// If no record exists, I also dont expect this to happen, so there is no need to create one
		logger.Error("failed to fetch server stats: %v", err)
		return
	}
	stats.Uploads++
	db.Save(&stats)
}

// Increment increments the specified field by 1
func Increment(field string){
	switch field {
	case "downloads":
		IncrementDownloads()
	case "requests":
		IncrementRequests()
	case "uploads":
		IncrementUploads()
	default:
		logger.Warn("Unknown field to increment: %s", field)
	}
}

// GetStats retrieves the current server stats
func GetStats() (ServerStats, error){
	db := GetDB()
	var stats ServerStats
	err := db.First(&stats).Error
	if err != nil {
		return stats, err
	}
	return stats, nil
}