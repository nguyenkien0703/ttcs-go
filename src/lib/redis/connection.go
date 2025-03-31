package lib_redis

import "github.com/redis/go-redis/v9"

var clusterClients = map[string]*redis.ClusterClient{}


var clients = map[string]*redis.Client{}


func init() {

}
func setupClient(name string)(redis.Cmdable, error) {
	setting := GetConnectionSetting(name)
	if setting == nil {
		return nil, lib_error.NewApp
	}else {

	}
}

func GetConnection(name string) (redis.Cmdable, error) {
	if clients[name] != nil {
		return clients[name], nil
	}else if clusterClients[name] != nil {
		return clusterClients[name], nil
	}
	return setupClient(name)
}




func CloseConnection() {
	for _, client := range clients {
		client.Close()
	}
	for _, client := range clusterClients {
		client.Close()
	}
}


