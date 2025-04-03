package lib_handler

import (
	config "application/src/config"
	lib_cache "application/src/lib/cache"
	lib_db "application/src/lib/db"
	lib_db_manager "application/src/lib/db/manager"
	lib_debug "application/src/lib/debug"
	lib_error "application/src/lib/error"
	lib_redis "application/src/lib/redis"
	"encoding/json"
	"fmt"
	"net/http"
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

func (baseHandler *BaseHandler) SetGlobalDbConnections(dbConnections map[string]*gorm.DB) {
	baseHandler.dbConnections = dbConnections
}

// ベースハンドラのセットアップ.
func (baseHandler *BaseHandler) Setup(context *gin.Context) error {
	baseHandler.context = context
	baseHandler.dbClients = map[string]*lib_db.Client{}
	baseHandler.redisClients = map[string]*lib_redis.Client{}
	baseHandler.now = time.Now().UTC()
	baseHandler.httpStatus = http.StatusOK
	baseHandler.HtmlParams = gin.H{}
	baseHandler.IsDebugRequest = config.GetIsDebugMode() && !config.GetIsClosedAlpha()
	baseHandler.JsonParams = map[string]interface{}{}
	return nil
}

// 終了処理.
func (baseHandler *BaseHandler) Terminate() {
	for name, client := range baseHandler.dbClients {
		if client == nil {
			continue
		}
		client.Close()
		if baseHandler.dbConnections != nil && baseHandler.dbConnections[name] != nil {
			continue
		}
		lib_db.Disconnect(client)
	}
	for _, client := range baseHandler.redisClients {
		if client == nil {
			continue
		}
		client.Terminate()
	}
}

// ginのContext.
func (baseHandler *BaseHandler) GetContext() *gin.Context {
	return baseHandler.context
}

// レスポンス返却済みフラグ.
func (baseHandler *BaseHandler) IsEnd() bool {
	return baseHandler.GetContext().GetBool("isEnd")
}

// レスポンス返却済みフラグを設定.
func (baseHandler *BaseHandler) setEnd() {
	baseHandler.GetContext().Set("isEnd", true)
}

// dbの接続クライアントを取得.
func (baseHandler *BaseHandler) GetDbClient(name string) (*lib_db.Client, error) {
	client := baseHandler.dbClients[name]
	if client != nil {
		return client, nil
	}
	var err error = nil
	// キャッシュ用redisクライアント.
	dbName := baseHandler.RedisCache
	if len(dbName) == 0 {
		dbName = config.RedisCache
	}
	redis, err := baseHandler.GetRedisClient(dbName)
	if err != nil {
		return nil, lib_error.WrapError(err)
	}
	// DBクライアント.
	if baseHandler.dbConnections != nil && baseHandler.dbConnections[name] != nil {
		client = lib_db.NewClient(baseHandler.dbConnections[name], redis)
	} else {
		client, err = lib_db.Connect(name, redis)
		if err != nil {
			return nil, lib_error.WrapError(err)
		}
	}
	baseHandler.dbClients[name] = client
	return client, nil
}

// dbのモデル管理インスタンスを取得.
func (baseHandler *BaseHandler) GetModelManager(args ...string) (*lib_db_manager.ModelManager, error) {
	name := config.DbDefault
	if 0 < len(args) {
		name = args[0]
	}
	client, err := baseHandler.GetDbClient(name)
	if err != nil {
		return nil, lib_error.WrapError(err)
	}
	return client.GetModelManager(), nil
}

// Redisの接続クライアントを取得.
func (baseHandler *BaseHandler) GetRedisClient(name string) (*lib_redis.Client, error) {
	redisClient := baseHandler.redisClients[name]
	if redisClient != nil {
		return redisClient, nil
	}
	var err error = nil
	redisClient, err = lib_redis.NewClient(name)
	if err != nil {
		return nil, lib_error.WrapError(err)
	}
	baseHandler.redisClients[name] = redisClient
	return redisClient, nil
}

// 現在時刻.
func (baseHandler *BaseHandler) GetNow() time.Time {
	return baseHandler.now
}

// processの直前に呼ばれます.
func (baseHandler *BaseHandler) PreProcess() error {
	return nil
}

// メイン処理.
func (baseHandler *BaseHandler) Process() error {
	return nil
}

// メイン処理後.
func (baseHandler *BaseHandler) AfterProcess() error {
	return nil
}

// ユーザーの確認.
func (baseHandler *BaseHandler) CheckUser() error {
	return nil
}

// エラー処理.
func (baseHandler *BaseHandler) ProcessError(err error) error {
	baseHandler.httpStatus = http.StatusInternalServerError
	baseHandler.WriteString(err.Error())
	lib_debug.Error("ProcessError:StatusInternalServerError: %+v", err)
	return nil
}

func (baseHandler *BaseHandler) SetHttpStatus(status int) {
	baseHandler.httpStatus = status
}

func (baseHandler *BaseHandler) GetHttpStatus() int {
	return baseHandler.httpStatus
}

// レスポンスのbodyにtextを書き込む.
func (baseHandler *BaseHandler) WriteString(text string) error {
	context := baseHandler.GetContext()
	context.String(baseHandler.httpStatus, text)
	err := context.Err()
	if err != nil {
		return err
	}
	baseHandler.setEnd()
	return nil
}

// レスポンスのbodyにバイトデータを書き込む.
func (self *BaseHandler) WriteBytes(data []byte, contentType string) error {
	context := self.GetContext()
	context.Data(self.httpStatus, contentType, data)
	err := context.Err()
	if err != nil {
		return err
	}
	self.setEnd()
	return nil
}

// レスポンスのbodyにファイルを書き込む.
func (baseHandler *BaseHandler) WriteFile(data []byte, name, contentType string) error {
	context := baseHandler.GetContext()
	context.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", name))
	context.Data(baseHandler.httpStatus, contentType, data)
	err := context.Err()
	if err != nil {
		return err
	}
	baseHandler.setEnd()
	return nil
}

// レスポンスのbodyにhtmlを書き込む.
func (baseHandler *BaseHandler) WriteHtml(htmlName string) error {
	context := baseHandler.GetContext()
	context.Writer.Header().Set("Cache-Control", "private")
	context.HTML(baseHandler.httpStatus, htmlName, baseHandler.HtmlParams)
	err := context.Err()
	if err != nil {
		return err
	}
	baseHandler.setEnd()
	return nil
}

// レスポンスのbodyにjsonを書き込む.
func (baseHandler *BaseHandler) WriteJsonData(data interface{}) error {
	context := baseHandler.GetContext()
	context.Writer.Header().Set("Cache-Control", "no-cache")
	if baseHandler.IsDebugRequest {
		// デバッグ時はインデントを付けた形で返す.
		context.IndentedJSON(baseHandler.httpStatus, data)
	} else {
		b, err := json.Marshal(data)
		if err != nil {
			return err
		}
		context.Data(baseHandler.httpStatus, "application/json; charset=utf-8", b)
	}
	err := context.Err()
	if err != nil {
		return err
	}
	baseHandler.setEnd()
	return nil
}

// レスポンスのbodyにjsonを書き込む.
func (baseHandler *BaseHandler) WriteJson() error {
	return baseHandler.WriteJsonData(baseHandler.JsonParams)
}

// リダイレクト.
func (baseHandler *BaseHandler) Redirect(url string) {
	context := baseHandler.GetContext()
	context.Writer.Header().Set("Cache-Control", "no-cache")
	context.Redirect(http.StatusMovedPermanently, url)
	baseHandler.setEnd()
}

// ログ.
func (baseHandler *BaseHandler) DebugLog(msg string, args ...interface{}) {
	lib_debug.Debug(msg, args...)
}
func (baseHandler *BaseHandler) InfoLog(msg string, args ...interface{}) {
	lib_debug.Info(msg, args...)
}
func (baseHandler *BaseHandler) ErrorLog(msg string, args ...interface{}) {
	lib_debug.Error(msg, args...)
}

// 実行処理.
func Run(handler HandlerInterface) error {
	userAgent := handler.GetContext().Request.Header.Get("User-Agent")
	if userAgent == "Mozilla/4.0 (compatible; MSIE 6.0; Windows NT 5.1)" {
		// ウィルスバスターのアクセス.IE6も死ぬけど仕方ない.
		handler.SetHttpStatus(http.StatusBadRequest)
		return handler.WriteString("No support")
	}
	flag := make(chan bool, 2)
	defer func() {
		flag <- true
	}()
	f := func(requestURI string) func(chan bool) {
		return func(ch chan bool) {
			time.Sleep(time.Second * 5)
			if len(ch) == 0 {
				lib_debug.Log("timeout", "%s", requestURI)
			}
		}
	}(handler.GetContext().Request.URL.RequestURI())
	go f(flag)
	// ローカルキャッシュ初期化.
	redis, err := handler.GetRedisClient(config.RedisCache)
	if err != nil {
		return handler.ProcessError(err)
	}
	err = lib_cache.InitLocalCache(redis)
	if err != nil {
		return handler.ProcessError(err)
	}
	// // ユーザー確認.
	err = handler.CheckUser()
	if err != nil {
		return handler.ProcessError(err)
	} else if handler.IsEnd() {
		return nil
	}
	// プロセス前の処理.
	err = handler.PreProcess()
	if err != nil {
		return handler.ProcessError(err)
	} else if handler.IsEnd() {
		return nil
	}
	// メイン処理.
	err = handler.Process()
	if err != nil {
		return handler.ProcessError(err)
	}

	// メイン処理後の処理.
	err = handler.AfterProcess()
	if err != nil {
		return handler.ProcessError(err)
	}

	if !handler.IsEnd() {
		// レスポンスを返していない.
		return handler.ProcessError(lib_error.NewAppError(lib_error.DefaultErrorCode, "no response"))
	}
	return nil
}

// ルータのラッパ.
func Wrap(f func() HandlerInterface, dbConnections map[string]*gorm.DB) gin.HandlerFunc {
	return func(context *gin.Context) {
		var handler HandlerInterface = nil
		defer func() {
			if err := recover(); err != nil {
				lib_debug.Error("%s\npanic: %+v\n%s", context.Request.URL.String(), err, lib_error.StackTrace())
				if handler != nil && !handler.IsEnd() {
					handler.ProcessError(fmt.Errorf("panic: %+v", err))
				}
			}
			if handler != nil {
				handler.Terminate()
			}
		}()
		// ハンドラ作成.
		handler = f()
		handler.SetGlobalDbConnections(dbConnections)
		// セットアップ.
		err := handler.Setup(context)
		if err == nil {
			// 実行.
			err = Run(handler)
		}
		if err != nil {
			message := ""
			if config.GetIsDebugMode() {
				message = err.Error()
			}
			lib_debug.Error("%s\nStatusInternalServerError: %+v", context.Request.URL.String(), err)
			context.String(http.StatusInternalServerError, message)
		}
	}
}
