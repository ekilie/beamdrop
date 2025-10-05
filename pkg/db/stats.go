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

func ResetStats(){
	db := GetDB()
	var stats ServerStats
	err :=db.First(&stats).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound){
			logger.Error("failed to fetch server stats: %v", err)
			return
		}

		// If no record exists, we create one
		stats = ServerStats{
			Downloads: 0,
			Requests:  0,
			Uploads:   0,
			StartTime: time.Now(),
		}
		db.Create(&stats)
		return
	}
	stats.Downloads = 0
	stats.Requests = 0
	stats.Uploads = 0
	stats.StartTime = time.Now()
	db.Save(&stats)
}