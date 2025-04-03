package lib_db

import (
	"application/src/config"
	lib_error "application/src/lib/error"
	lib_redis "application/src/lib/redis"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Connect(name string, redis *lib_redis.Client) (*Client, error) {
	dbConfig := config.GetDatabaseConfig()
	dsn := dbConfig.GetDSN()

	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:               dsn,
		DefaultStringSize: 256,
	}), &gorm.Config{})
	if err != nil {
		return nil, lib_error.WrapError(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, lib_error.WrapError(err)
	}

	// Set connection pool settings
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return &Client{db: db}, nil
}

func Disconnect(client *Client) error {
	client.Close()
	sqlDB, _ := client.GetDB().DB()
	if sqlDB != nil {
		return sqlDB.Close()
	}
	return nil
}
