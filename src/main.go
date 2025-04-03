package main

import (
	"application/src/config"
	lib_db "application/src/lib/db"
	lib_debug "application/src/lib/debug"
	lib_redis "application/src/lib/redis"
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	limit "github.com/aviddiviner/gin-limit"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var runMigrations = flag.Bool("migrate", false, "run database migrations")

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		lib_debug.Error("Error loading .env file: %v", err)
	}

	flag.Parse()

	if *runMigrations {
		gooseMain()
		return
	}

	if config.GetIsLocal() {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/go/src/golang-ttcs-server/tmp/credential.json")
	}
	numCPU := runtime.NumCPU()
	lib_debug.Info("numCPU:%d", numCPU)
	if 2 < numCPU {
		numCPU--
	}
	runtime.GOMAXPROCS(numCPU)
	engine := gin.Default()
	engine.Use(limit.MaxAllowed(40))
	// Set up global db connections
	globalDBConnections := map[string]*gorm.DB{}
	defer func() {
		// Close db connections
		for _, db := range globalDBConnections {
			if sqlDB, _ := db.DB(); sqlDB != nil {
				sqlDB.Close()
			}
		}
		// Close redis connections
		lib_redis.CloseConnections()
	}()
	
	for _, connectionSettingName := range lib_db.GetConnectionSettingNames() {
		dbClient, err := lib_db.Connect(connectionSettingName, nil)
		if err != nil {
			lib_debug.Error("DB connection err: %v", err)
			return
		}
		globalDBConnections[connectionSettingName] = dbClient.GetDB()
	}
	// Set up endpoint routing/handlers
	//app_routers.RouteApi(engine, globalDBConnections)
	//app_routers.RouteMgr(engine)
	// Create server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: engine,
	}

	// Listen
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			lib_debug.Error("listen err: %v", err)
		}
	}()

	// Blocking channel, Monitor os signals for interruptions
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGTERM)
	<-quit
	// Shut down server gracefully
	lib_debug.Info("Server shutting down ...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		lib_debug.Error("Server shutdown err: %v", err)
	} else {
		lib_debug.Info("Server shutdown gracefully")
	}

}
