package db

import "time"

type ServerStats struct{
	Downloads int       `gorm:"column:downloads, default:0" json:"downloads"`
	Requests  int       `gorm:"column:requests, default:0" json:"requests"`
    Uploads   int       `gorm:"column:uploads, default:0" json:"uploads"`
    StartTime time.Time `gorm:"column:start_time, default:CURRENT_TIMESTAMP" json:"startTime"`
}

func (ServerStats) TableName() string {
	return "server_stats"
}