package database

import (
	"fmt"
	"log"
	"sync"
	"telegram-tickets-bot/src/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	db   *gorm.DB
	once sync.Once
)

func ConnectDatabase(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Set connection pool parameters
	sqlDB.SetMaxOpenConns(0)
	sqlDB.SetMaxIdleConns(0)

	return db, nil
}

func InitializeAndPrintDBInfo(cfg *config.Config) error {
	var err error
	once.Do(func() {
		db, err = ConnectDatabase(cfg)
		if err != nil {
			return
		}

		sqlDB, err := db.DB()
		if err != nil {
			return
		}

		stats := sqlDB.Stats()
		log.Printf("[INFO] Database connection successful. Connection info: Max open connections: %d, Idle connections: %d, In-use connections: %d",
			stats.MaxOpenConnections, stats.Idle, stats.InUse)
	})

	return err
}

func InitializeDB() (*gorm.DB, error) {
	if db == nil {
		return nil, fmt.Errorf("[ERROR] Database not initialized")
	}
	return db, nil
}

func Create(value interface{}) error {
	return db.Create(value).Error
}

func Find(dest interface{}, conds ...interface{}) error {
	return db.Find(dest, conds...).Error
}

func Update(model interface{}, updates interface{}) error {
	return db.Model(model).Updates(updates).Error
}

func Delete(value interface{}, conds ...interface{}) error {
	return db.Delete(value, conds...).Error
}
