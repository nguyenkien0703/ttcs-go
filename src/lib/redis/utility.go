package lib_redis

import (
	lib_error "application/src/lib/error"
	lib_util_object "application/src/lib/util/object"
	"fmt"
	"math"
	"reflect"

	"github.com/mmcloughlin/geohash"
	"github.com/redis/go-redis/v9"
)

const MinLatitude = -85.05112878
const MaxLatitude = 85.05112878

func marshal(value interface{}) (string, error) {
	s, err := lib_util_object.Marshal(value)
	return s, err
}
func unmarshal(redisValue string, dest interface{}) error {
	return lib_util_object.Unmarshal(redisValue, dest)
}

func unmarshalSlice(redisValues []string, dest interface{}) error {

	length := len(redisValues)
	if length == 0 {
		return nil
	}

	destV := reflect.ValueOf(dest)
	elem := destV.Elem()
	sl := reflect.MakeSlice(elem.Type(), length, length)

	for i := 0; i < length; i++ {
		item := sl.Index(i)
		if len(redisValues[i]) == 0 {
			if item.Kind() != reflect.Ptr {
				item.Set(reflect.New(item.Type()).Elem())
			}
			continue
		}
		var itemV reflect.Value
		if item.Kind() == reflect.Ptr {
			itemV = reflect.New(item.Type().Elem())
			item.Set(itemV)
		} else {
			item.Set(reflect.New(item.Type()).Elem())
			itemV = item.Addr()
		}
		err := unmarshal(redisValues[i], itemV.Interface())
		if err != nil {
			return lib_error.WrapError(err)
		}
	}
	elem.Set(sl)

	return nil
}
func unmarshalMap(redisValues map[string]string, dest interface{}) error {

length := len(redisValues)
if length == 0 {
	return nil
}

destV := reflect.ValueOf(dest)
elem := destV.Elem()

mp := reflect.MakeMap(elem.Type())
valType := mp.Type().Elem()

for k, redisValue := range redisValues {
	key := reflect.ValueOf(k)
	var itemV reflect.Value
	if valType.Kind() == reflect.Ptr {
		itemV = reflect.New(valType.Elem())
		err := unmarshal(redisValue, itemV.Interface())
		if err != nil {
			return lib_error.WrapError(err)
		}
	} else {
		itemV = reflect.New(valType).Elem()
		err := unmarshal(redisValue, itemV.Addr().Interface())
		if err != nil {
			return lib_error.WrapError(err)
		}
	}
	mp.SetMapIndex(key, itemV)
}
elem.Set(mp)

return nil
}
func loadSortedSetMembers(redisPairs []redis.Z) []*SortedSetMember {
	dest := make([]*SortedSetMember, len(redisPairs))
	for i, v := range redisPairs {
		dest[i] = &SortedSetMember{
			Name:  v.Member.(string),
			Score: uint64(v.Score),
		}
	}
	return dest
}
func IsValidLatitute(lat float64) bool {
	return lat <= MaxLatitude && MinLatitude <= lat
}
func RoundLatitute(lat float64) float64 {
	return math.Max(math.Min(lat, MaxLatitude), MinLatitude)
}

func rangeToString(v interface{}) string {
	switch v.(type) {
	case string:
		return v.(string)
	default:
		return fmt.Sprintf("%d", v)
	}
}
type GeoLocationData struct {
	Lat  float64
	Lon  float64
	Name interface{}
}

func MakeRedisGeoLocations(geoLocData []*GeoLocationData) ([]*redis.GeoLocation, error) {
	var err error = nil
	locations := make([]*redis.GeoLocation, len(geoLocData))
	for i, data := range geoLocData {
		loc := &redis.GeoLocation{
			Latitude:  data.Lat,
			Longitude: data.Lon,
		}
		loc.Name, err = marshal(data.Name)
		if err != nil {
			return nil, lib_error.WrapError(err)
		}
		locations[i] = loc
	}
	return locations, nil
}

func MakeGeoHash(lat, lon float64, precision uint) string {
	if precision == 0 {
		return geohash.Encode(lat, lon)
	}
	if precision > 12 {
		precision = 12
	}
	return geohash.EncodeWithPrecision(lat, lon, precision)
}

func GetGeoHashNeighbors(hash string) []string {
	return geohash.Neighbors(hash)
}

func GetGeoHashCenter(hash string) (float64, float64) {
	return geohash.BoundingBox(hash).Center()
}

func GeoHashIntToString(intHash int64, precision uint) string {
	return geohash.ConvertIntToString(uint64(intHash), precision)
}
