package db

import (
	"log/slog"
	"sync"

	"github.com/glebarez/sqlite"
	"github.com/tachRoutine/beamdrop-go/config"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var (
	db   *gorm.DB
	once sync.Once
)

func init() {
	once.Do(openDB)
	CreateStatsTable()
}

func openDB() {
	var dbPath string = config.DBPath
	// logger.Info("Opening database at: %s", dbPath)
	var err error
	db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		slog.Error("Failed to connect database", "error", err)
	}
}

func GetDB() *gorm.DB {
	return db
}
