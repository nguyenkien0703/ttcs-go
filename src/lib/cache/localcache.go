package lib_cache

import (
	"application/src/config"
	lib_redis "application/src/lib/redis"
	"time"
)

/*
+----------------+      +----------------+
|  Redis Chính   |      |  Redis Local   |
|                |      |                |
| localcacheTime |----->| localcacheTime |
|     (t2)      |      |     (t1)       |
+----------------+      +----------------+

	|                      ^
	|                      |
	+----------------------+
	  Nếu t2 > t1: Flush local
*/
func InitLocalCache(client *lib_redis.Client) error {
	var err error = nil

	if client == nil {
		client, err = lib_redis.NewClient(config.RedisCache)
		if err != nil {
			return err
		}
		defer client.Terminate()
	}

	redisLocal, err := lib_redis.NewClient(config.RedisLocal)
	if err != nil {
		return err
	}
	defer redisLocal.Terminate()
	var localcacheTime int64 = 0
	var t int64 = 0
	_, err = redisLocal.Get(&localcacheTime, "localcacheTime")
	if err != nil {
		return err
	}
	_, err = client.Get(&t, "localcacheTime")
	if err != nil {
		return err
	} else if t == 0 {
		// 設定がない.
		t = time.Now().UnixNano()
		err = client.SetNx("localcacheTime", t)
		if err != nil {
			return err
		}
	}
	if localcacheTime < t {
		// ローカルのキャッシュを破棄.
		err = redisLocal.FlushDb()
		if err != nil {
			return err
		}
		err = redisLocal.Set("localcacheTime", t)
		if err != nil {
			return err
		}
	}
	return nil

}
func DeleteLocalCache(client *lib_redis.Client) error {
	var err error = nil
	if client == nil {
		client, err = lib_redis.NewClient(config.RedisCache)
		if err != nil {
			return err
		}
		defer client.Terminate()
	}
	err = client.Del("localcacheTime")
	return err
}
func GetLocalCache(key string, dest interface{}) bool {
	redisLocal, err := lib_redis.NewClient(config.RedisLocal)
	if err != nil {
		return false
	}
	defer redisLocal.Terminate()
	flag, err := redisLocal.Get(dest, key)
	return flag && err == nil
}
func GetLocalCacheMany(keys []string, dest interface{}) {
	if len(keys) == 0 {
		return
	}
	redisLocal, err := lib_redis.NewClient(config.RedisLocal)
	if err != nil {
		return
	}
	defer redisLocal.Terminate()
	redisLocal.MGet(dest, keys...)
}
func SetLocalCache(key string, value interface{}) error {
	redisLocal, err := lib_redis.NewClient(config.RedisLocal)
	if err != nil {
		return err
	}
	defer redisLocal.Terminate()
	return redisLocal.Set(key, value)
}
func SetLocalCacheMany(data map[string]interface{}) error {
	redisLocal, err := lib_redis.NewClient(config.RedisLocal)
	if err != nil {
		return err
	}
	defer redisLocal.Terminate()
	return redisLocal.MSet(data)
}

func SetLocalCacheWithTTL(key string, value interface{}, ttl int64) error {
	redisLocal, err := lib_redis.NewClient(config.RedisLocal)
	if err != nil {
		return err
	}
	defer redisLocal.Terminate()
	err = redisLocal.Set(key, value)
	if err != nil {
		return err
	}
	err = redisLocal.Expire(key, ttl)
	if err != nil {
		redisLocal.Del(key)
	}
	return nil
}
