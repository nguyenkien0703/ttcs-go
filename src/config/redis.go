package config

const RedisDb string = "db"
const RedisCache string = "cache"
const RedisSession string = "session"
const RedisLocal string = "local"

var RedisDefault string = RedisDb

func GetRedisConnectionSettings() map[string]map[string]string {
	return map[string]map[string]string{
		RedisDb: {
			"Host":         "redis",
			"Port":         "6379",
			"Username":     "ttcs",
			"Password":     "password",
			"Db":           "0",
			"Pool":         "100",
			"MinIdleConns": "10",
		},
		RedisCache: {
			"Host":         "redis",
			"Port":         "6379",
			"Username":     "ttcs",
			"Password":     "password",
			"Db":           "1",
			"Pool":         "100",
			"MinIdleConns": "10",
		},
		RedisSession: {
			"Host":         "redis",
			"Port":         "6379",
			"Username":     "ttcs",
			"Password":     "password",
			"Db":           "2",
			"Pool":         "100",
			"MinIdleConns": "10",
		},
		RedisLocal: {
			"Host":         "redis",
			"Port":         "6379",
			"Username":     "ttcs",
			"Password":     "password",
			"Db":           "3",
			"Pool":         "100",
			"MinIdleConns": "10",
		},
	}
}
