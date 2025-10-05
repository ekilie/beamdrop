package db

import (
	"sync"

	"github.com/tachRoutine/beamdrop-go/config"
	"github.com/tachRoutine/beamdrop-go/pkg/logger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	db   *gorm.DB
	once sync.Once
)

func init() {
	once.Do(openDB)
}

func openDB() {
	var dbPath string = config.DBPath
	logger.Info("Opening database at: %s", dbPath)
	var err error
	db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		logger.Error("failed to connect database: %v", err)
	}
}

func GetDB() *gorm.DB {
	return db
}
