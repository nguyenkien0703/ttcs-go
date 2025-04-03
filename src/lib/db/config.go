package lib_db

import (
	"application/src/config"
	"fmt"
	"strconv"
)

type connectionSetting struct {
	Host          string
	Port          string
	Database      string
	User          string
	Password      string
	MaxConnection int
	Pool          int
}

func (cs *connectionSetting) Address() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", cs.User, cs.Password, cs.Host, cs.Port, cs.Database)
}

var settings map[string]*connectionSetting = map[string]*connectionSetting{}
var dbNames []string

func init() {
	dbConnectionSettings := config.GetDbConnectionSettings()
	settings = make(map[string]*connectionSetting, len(dbConnectionSettings))
	dbNames = []string{}
	for name, paramMap := range dbConnectionSettings {
		host := paramMap["Host"]
		if len(host) == 0 {
			host = "127.0.0.1"
		}
		//lib_debug.Debug("db name=%s, host=%s", name, host)
		port := paramMap["Port"]
		if len(port) == 0 {
			port = "3306"
		}
		pool, _ := strconv.Atoi(paramMap["Pool"])
		maxConn, _ := strconv.Atoi(paramMap["MaxConnection"])
		if maxConn == 0 {
			maxConn = 100
		}
		if pool < maxConn {
			pool = maxConn + 1
		}
		database := paramMap["Database"]
		user := paramMap["User"]
		password := paramMap["Password"]
		setting := &connectionSetting{
			Host:          host,
			Port:          port,
			Database:      database,
			User:          user,
			Password:      password,
			Pool:          pool,
			MaxConnection: maxConn,
		}
		settings[name] = setting
		dbNames = append(dbNames, name)
	}
}
func GetConnectionSetting(name string) *connectionSetting {
	if _, ok := settings[name]; !ok {
		// 無い.
		return nil
	}
	return settings[name]
}
func GetConnectionSettingNames() []string {
	return dbNames[:]
}
