package lib_redis

import (
	config "application/src/config"
	"strconv"
	"strings"
)

type connectionSetting struct {
	Host         string
	Hosts        []string
	Port         string
	Username     string
	Password     string
	Db           int
	Pool         int
	MinIdleConns int
	TLS          bool
}

var settings map[string]*connectionSetting = map[string]*connectionSetting{}

var dbNames []string

// Hàm init() được Go tự động gọi khi package được import
func init() {

	redisConnectionSettings := config.GetRedisConnectionSettings()
	settings = make(map[string]*connectionSetting, len(redisConnectionSettings))

	for name, paramMap := range redisConnectionSettings {
		host := ""
		hosts:= []string{}

		if len(paramMap["Hosts"]) ==0 {
			host = paramMap["Host"]
			if len(host) ==0 {
				host = "127.0.0.1"
			}
		}else {
			hosts = strings.Split(paramMap["Hosts"], ",")
		}

		port := paramMap["Port"]
		if len(port) == 0 {
			port = "6379"
		}
		username := paramMap["Username"]
		password := paramMap["Password"]
		db := 0
		if _, ok := paramMap["DB"]; ok {
			db, _ = strconv.Atoi(paramMap["Db"]) // Nếu tham số "Db" được cung cấp, chuyển đổi từ chuỗi sang số nguyên

		}
		pool := 0
		if _, ok := paramMap["Pool"]; ok {
			pool, _ = strconv.Atoi(paramMap["Pool"])
		}
		if pool < 1 {
			pool = 1
		}
		minIdleConns := 0
		if _, ok := paramMap["MinIdleConns"]; ok {
			minIdleConns, _ = strconv.Atoi(paramMap["MinIdleConns"])
		}
		settings[name] = &connectionSetting{
			Host:         host,
			Hosts:        hosts,
			Port:         port,
			Username:     username,
			Password:     password,
			Db:           db,
			Pool:         pool,
			MinIdleConns: minIdleConns,
			TLS:          paramMap["TLS"] == "1",
		}
		dbNames = append(dbNames, name)

	}


}

func GetConnectionSetting(name string) *connectionSetting {
	if _, ok := settings[name]; !ok {
		return nil
	}
	return settings[name]
}

func GetConnectionNames() []string {
	return dbNames[:]
}
