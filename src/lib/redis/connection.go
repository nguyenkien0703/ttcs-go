package lib_redis

import (
	"application/src/config"
	lib_debug "application/src/lib/debug"
	lib_error "application/src/lib/error"
	"crypto/tls" // Hỗ trợ TLS (Transport Layer Security)
	"fmt"

	"github.com/redis/go-redis/v9"
)


//Map lưu trữ các kết nối đến Redis Cluster, với khóa là tên kết nối
var clusterClients = map[string]*redis.ClusterClient{}
// Map lưu trữ các kết nối đến Redis đơn lẻ, với khóa là tên kết nối
var clients = map[string]*redis.Client{}


//Hàm init() được Go tự động gọi khi package được import
func init() {
	for name := range settings {
		_, err := setupClient(name)
		if err != nil {
			lib_debug.Error("lib_redis.init error:name=%s:%v", name, err)
		}
	}
}

func setupClient(name string) (redis.Cmdable, error) {
	setting := GetConnectionSetting(name)

	if setting == nil {
		return nil, lib_error.NewAppErrorWithStackTrace(lib_error.DefaultErrorCode, "Illegal redis db name:%s", name)
	} else if len(setting.Hosts) == 0 {//(không có danh sách máy chủ
		/*
		Tạo kết nối đến Redis đơn lẻ với các thông số từ cài đặt
		Bật TLS nếu được cấu hình
		Lưu client vào map clients
		*/
		opt := &redis.Options{
			Addr:         fmt.Sprintf("%s:%s", setting.Host, setting.Port),
			Username:     setting.Username,
			Password:     setting.Password,
			DB:           setting.Db,
			PoolSize:     setting.Pool,
			MinIdleConns: setting.MinIdleConns,
		}
		if setting.TLS {
			opt.TLSConfig = &tls.Config{}
		}
		clients[name] = redis.NewClient(opt)
		return clients[name], nil
	} else {//Nếu setting.Hosts không rỗng:, tức là  có danh sách máy chủ

		addrs := make([]string, len(setting.Hosts))
		for i, host := range setting.Hosts {
			addrs[i] = fmt.Sprintf("%s:%s", host, setting.Port)
		}
		opt := &redis.ClusterOptions{
			Addrs:        addrs,
			PoolSize:     setting.Pool,
			MinIdleConns: setting.MinIdleConns,
		}
		if !config.GetIsDebugMode() {
			opt.TLSConfig = &tls.Config{}
		}
		clusterClients[name] = redis.NewClusterClient(opt)
		return clusterClients[name], nil
	}
}


func GetConnection(name string) (redis.Cmdable, error) {
	if clients[name] != nil {
		return clients[name], nil
	} else if clusterClients[name] != nil {
		return clusterClients[name], nil
	}
	return setupClient(name)
}


func CloseConnections() {
	for _, client := range clients {
		client.Close()
	}
	for _, client := range clusterClients {
		client.Close()
	}
}
