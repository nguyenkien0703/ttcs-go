package lib_db

import "gorm.io/gorm"

type Client struct {
	db *gorm.DB
	transaction *gorm.DB
	redis *lib_redis.Client

	modelManager *lib_db_manager.ModelManager


}
