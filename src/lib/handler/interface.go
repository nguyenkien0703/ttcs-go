package lib_handler

import (
	lib_db "application/src/lib/db"
	lib_db_manager "application/src/lib/db/manager"
	lib_redis "application/src/lib/redis"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"time"
)

type HandlerInterface interface {
	SetGlobalDbConnections(map[string]*gorm.DB)
	Setup(*gin.Context) error
	Terminate()
	GetContext() *gin.Context
	GetDbClient(string) (*lib_db.Client, error)
	GetModelManager(...string) (*lib_db_manager.ModelManager, error)
	GetRedisClient(string) (*lib_redis.Client, error)
	GetNow() time.Time
	SetHttpStatus(int)
	WriteString(string) error

	WriteHtml(string) error

	WriteJson() error
	PreProcess() error
	Process() error
	AfterProcess() error
	CheckUser() error
	IsEnd() bool
	ProcessError(error) error
}
