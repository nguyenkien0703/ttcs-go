package lib_handler

import (

	lib_db "application/src/lib/db"
	lib_redis "application/src/lib/redis"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BaseHandler struct {
	context        *gin.Context
	dbConnections  map[string]*gorm.DB
	dbClients      map[string]*lib_db.Client
	redisClients   map[string]*lib_redis.Client
	now            time.Time
	IsDebugRequest bool
	httpStatus     int
	HtmlParams     gin.H
	JsonParams     map[string]interface{}
	RedisCache     string
}
