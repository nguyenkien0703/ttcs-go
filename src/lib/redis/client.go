package lib_redis

import (
	lib_error "application/src/lib/error"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const NX string = "NX"
const XX string = "XX"

type Client struct {
	name string
	conn redis.Cmdable
}
func (self *Client) Terminate() {
	self.conn = nil
}

func NewClient(name string) (*Client, error) {
	conn, err := GetConnection(name)
	if err != nil {
		return nil, lib_error.WrapError(err)
	}
	return &Client{
		name: name,
		conn: conn,
	}, nil
}

// redisのデータベース名.
//Các phương thức tiện ích
func (self *Client) DBName() string {
	return self.name
}

// データベース名の確認.
//Kiểm tra xem tên cơ sở dữ liệu có khớp với tên đã cho không
func (self *Client) AssertDB(name string) error {
	if self.name == name {
		return nil
	}
	return lib_error.WrapError(fmt.Errorf("RedisのDBが一致しません:%s vs %s", self.name, name))
}

// ===========================================================================
// common.

//Tạo khóa Redis với namespace (hữu ích cho Redis Cluster)
func (self *Client) MakeKey(key, namespace string) string {
	// redisのクラスターを使う場合のキー作成.
	// multiとか使うなら必要.
	if len(namespace) == 0 {
		return key
	}
	return fmt.Sprintf("{%s}%s", namespace, key)
}


////. Các phương thức thao tác chung

// Kiểm tra xem khóa có tồn tại không
func (self *Client) Exists(key string) (bool, error) {
	v, err := self.conn.Exists(context.Background(), key).Result()
	return v == int64(1), lib_error.WrapError(err)
}

// /Đặt thời gian hết hạn cho khóa
func (self *Client) Expire(key string, ttl int64) error {
	_, err := self.conn.Expire(context.Background(), key, time.Second*time.Duration(ttl)).Result()
	return lib_error.WrapError(err)
}

// Lấy thời gian còn lại của khóa
func (self *Client) Ttl(key string) (int64, error) {
	i, err := self.conn.TTL(context.Background(), key).Result()
	if err != nil {
		return 0, lib_error.WrapError(err)
	}
	return int64(i.Seconds()), nil
}
//Xóa một hoặc nhiều khóa
func (self *Client) Del(keys ...string) error {
	_, err := self.conn.Del(context.Background(), keys...).Result()
	return lib_error.WrapError(err)
}

//Xóa tất cả các khóa trong cơ sở dữ liệu hiện tại
func (self *Client) FlushDb() error {
	_, err := self.conn.FlushDB(context.Background()).Result()
	return lib_error.WrapError(err)
}

//: Xóa tất cả các khóa trong tất cả các cơ sở dữ liệu
func (self *Client) FlushAll() error {
	_, err := self.conn.FlushAll(context.Background()).Result()
	return lib_error.WrapError(err)
}


// Thực hiện nhiều lệnh trong một giao dịch
func (self *Client) Multi(f func(*Client, ...interface{}) error, args ...interface{}) error {
	_, err := self.conn.TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
		pClient := &Client{
			name: self.name,
			conn: pipe,
		}
		err := f(pClient, args...)
		return lib_error.WrapError(err)
	})
	return lib_error.WrapError(err)
}

//): Quét và xóa các khóa khớp với mẫu
func (self *Client) ScanDel(pattern string, count int64) error {
	ctx := context.Background()
	iter := self.conn.Scan(ctx, 0, pattern, count).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		_, err := self.conn.Del(ctx, key).Result()
		if err != nil {
			return lib_error.WrapError(err)
		}
	}
	return lib_error.WrapError(iter.Err())
}

//Đổi tên khóa
func (self *Client) Rename(key, newkey string) error {
	_, err := self.conn.Rename(context.Background(), key, newkey).Result()
	return lib_error.WrapError(err)
}

// ===========================================================================
// String.
// Lấy giá trị của khóa
func (self *Client) Get(dest interface{}, key string) (bool, error) {
	value, err := self.conn.Get(context.Background(), key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, lib_error.WrapError(err)
	}
	err = unmarshal(value, dest)
	return true, lib_error.WrapError(err)
}

//: Lấy giá trị và xóa khóa
func (self *Client) GetDel(dest interface{}, key string) (bool, error) {
	value, err := self.conn.GetDel(context.Background(), key).Result()
	if err != nil {
		if 0 <= strings.Index(err.Error(), "nil returned") || 0 <= strings.Index(err.Error(), "redis: nil") {
			return false, nil
		}
		return false, lib_error.WrapError(err)
	}
	err = unmarshal(value, dest)
	return true, lib_error.WrapError(err)
}

// Lấy giá trị của nhiều khóa
func (self *Client) MGet(dest interface{}, keys ...string) error {
	v, err := self.conn.MGet(context.Background(), keys...).Result()
	if err != nil {
		return lib_error.WrapError(err)
	}
	values := make([]string, len(v))
	for i, val := range v {
		if val == nil {
			continue
		}
		values[i] = val.(string)
	}
	return unmarshalSlice(values, dest)
}
// Đặt giá trị cho khóa
func (self *Client) Set(key string, value interface{}, ttl ...int64) error {
	redisValue, err := marshal(value)
	if err != nil {
		return lib_error.WrapError(err)
	}
	ttlSec := int64(0)
	if len(ttl) > 0 {
		ttlSec = ttl[0]
	}
	_, err = self.conn.Set(context.Background(), key, redisValue, time.Second*time.Duration(ttlSec)).Result()
	return lib_error.WrapError(err)
}


// Đặt giá trị cho nhiều khóa

func (self *Client) MSet(data map[string]interface{}) error {
	args := make([]interface{}, len(data)*2)
	i := 0
	for k, v := range data {
		redisValue, err := marshal(v)
		if err != nil {
			return lib_error.WrapError(err)
		}
		args[i] = k
		i++
		args[i] = redisValue
		i++
	}
	_, err := self.conn.MSet(context.Background(), args...).Result()
	return lib_error.WrapError(err)
}

// Đặt giá trị cho khóa nếu khóa chưa tồn tại
func (self *Client) SetNx(key string, value interface{}, ttl ...int64) error {
	redisValue, err := marshal(value)
	if err != nil {
		return lib_error.WrapError(err)
	}
	ttlSec := int64(0)
	if len(ttl) > 0 {
		ttlSec = ttl[0]
	}
	_, err = self.conn.SetNX(context.Background(), key, redisValue, time.Second*time.Duration(ttlSec)).Result()
	return lib_error.WrapError(err)
}
// Đặt giá trị cho nhiều khóa nếu tất cả các khóa chưa tồn tại
func (self *Client) MSetNX(data map[string]interface{}) error {
	args := make([]interface{}, len(data)*2)
	i := 0
	for k, v := range data {
		redisValue, err := marshal(v)
		if err != nil {
			return lib_error.WrapError(err)
		}
		args[i] = k
		i++
		args[i] = redisValue
		i++
	}
	_, err := self.conn.MSetNX(context.Background(), args...).Result()
	return lib_error.WrapError(err)
}
// Tăng giá trị của khóa lên 1
func (self *Client) Incr(key string) (int64, error) {
	v, err := self.IncrBy(key, 1)
	return v, lib_error.WrapError(err)
}
// Tăng giá trị của khóa lên một số lượng cụ thể
func (self *Client) IncrBy(key string, incr int64) (int64, error) {
	value, err := self.conn.IncrBy(context.Background(), key, incr).Result()
	if err != nil {
		return 0, lib_error.WrapError(err)
	}
	return value, nil
}

// ===========================================================================
// List.
//Các phương thức thao tác với kiểu dữ liệu List trong Redis:


// Thêm giá trị vào đầu danh sách
func (self *Client) LPush(key string, value interface{}) error {
	redisValue, err := marshal(value)
	if err != nil {
		return lib_error.WrapError(err)
	}
	_, err = self.conn.LPush(context.Background(), key, redisValue).Result()
	return lib_error.WrapError(err)
}
// Lấy và xóa giá trị từ đầu danh sách
func (self *Client) LPop(dest interface{}, key string) error {
	v, err := self.conn.LPop(context.Background(), key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}
		return lib_error.WrapError(err)
	}
	return unmarshal(v, dest)
}
// Thêm giá trị vào cuối danh sách
func (self *Client) RPush(key string, value interface{}) error {
	redisValue, err := marshal(value)
	if err != nil {
		return lib_error.WrapError(err)
	}
	_, err = self.conn.RPush(context.Background(), key, redisValue).Result()
	return lib_error.WrapError(err)
}
// : Lấy và xóa giá trị từ cuối danh sách
func (self *Client) RPop(dest interface{}, key string) error {
	v, err := self.conn.RPop(context.Background(), key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}
		return lib_error.WrapError(err)
	}
	return unmarshal(v, dest)
}
// Lấy độ dài của danh sách
func (self *Client) LLen(key string) (int64, error) {
	value, err := self.conn.LLen(context.Background(), key).Result()
	if err != nil {
		return 0, lib_error.WrapError(err)
	}
	return value, nil
}
// Lấy một phạm vi giá trị từ danh sách
func (self *Client) LRange(dest interface{}, key string, start, end int64) error {
	values, err := self.conn.LRange(context.Background(), key, start, end).Result()
	if err != nil {
		return lib_error.WrapError(err)
	}
	return unmarshalSlice(values, dest)
}
// Cắt danh sách để chỉ giữ lại một phạm vi cụ thể
func (self *Client) LTrim(key string, start, end int64) error {
	_, err := self.conn.LTrim(context.Background(), key, start, end).Result()
	return lib_error.WrapError(err)
}

// ===========================================================================
// Set.
//Các phương thức thao tác với kiểu dữ liệu Set trong Redis:


// Hàm nội bộ để chuẩn bị đối số cho các lệnh Set
func (self *Client) makeSetCommandArgs(key string, members []interface{}) ([]interface{}, error) {
	args := make([]interface{}, len(members))
	for i, member := range members {
		s, err := marshal(member)
		if err != nil {
			return args, lib_error.WrapError(err)
		}
		args[i] = interface{}(s)
	}
	return args, nil
}
// Thêm một hoặc nhiều thành viên vào tập hợp
func (self *Client) SAdd(key string, members ...interface{}) error {
	args, err := self.makeSetCommandArgs(key, members)
	if err != nil {
		return lib_error.WrapError(err)
	}
	_, err = self.conn.SAdd(context.Background(), key, args...).Result()
	return lib_error.WrapError(err)
}
// Xóa một hoặc nhiều thành viên khỏi tập hợp
func (self *Client) SRem(key string, members ...interface{}) error {
	args, err := self.makeSetCommandArgs(key, members)
	if err != nil {
		return lib_error.WrapError(err)
	}
	_, err = self.conn.SRem(context.Background(), key, args...).Result()
	return lib_error.WrapError(err)
}
// Lấy tất cả các thành viên của tập hợp
func (self *Client) SMembers(dest interface{}, key string) error {
	values, err := self.conn.SMembers(context.Background(), key).Result()
	if err != nil {
		return lib_error.WrapError(err)
	}
	return unmarshalSlice(values, dest)
}
// Lấy số lượng thành viên của tập hợp
func (self *Client) SCard(key string) (int64, error) {
	value, err := self.conn.SCard(context.Background(), key).Result()
	if err != nil {
		return 0, lib_error.WrapError(err)
	}
	return value, nil
}
// Kiểm tra xem một giá trị có phải là thành viên của tập hợp không
func (self *Client) SIsMember(key string, member interface{}) (bool, error) {
	memberValue, err := marshal(member)
	if err != nil {
		return false, lib_error.WrapError(err)
	}
	v, err := self.conn.SIsMember(context.Background(), key, memberValue).Result()
	return v, lib_error.WrapError(err)
}
// Lấy ngẫu nhiên một thành viên từ tập hợp
func (self *Client) SRandMember(dest interface{}, key string) error {
	value, err := self.conn.SRandMember(context.Background(), key).Result()
	if err != nil {
		return lib_error.WrapError(err)
	}
	return unmarshal(value, dest)
}

// ===========================================================================
// Sorted Set.
//Các phương thức thao tác với kiểu dữ liệu Sorted Set trong Redis:


//  Hàm nội bộ để thêm thành viên vào sorted set với các tùy chọn khác nhau
func (self *Client) zadd(key string, members map[string]uint64, nxxx string) error {
	args := make([]redis.Z, len(members))
	i := 0
	for k, v := range members {
		args[i] = redis.Z{
			Member: k,
			Score:  float64(v),
		}
		i++
	}
	var v *redis.IntCmd
	// NX|XX.
	switch nxxx {
	case NX:
		v = self.conn.ZAddNX(context.Background(), key, args...)
	case XX:
		v = self.conn.ZAddXX(context.Background(), key, args...)
	default:
		v = self.conn.ZAdd(context.Background(), key, args...)
	}

	_, err := v.Result()
	return lib_error.WrapError(err)
}

func (self *Client) ZAdd(key string, members map[string]uint64) error {
	return self.zadd(key, members, "")
}

func (self *Client) ZAddNX(key string, members map[string]uint64) error {
	return self.zadd(key, members, NX)
}

func (self *Client) ZAddXX(key string, members map[string]uint64) error {
	return self.zadd(key, members, XX)
}

func (self *Client) ZRem(key string, members ...interface{}) error {
	_, err := self.conn.ZRem(context.Background(), key, members...).Result()
	return lib_error.WrapError(err)
}

func (self *Client) ZRemRangeByScore(key string, min, max interface{}) error {
	minS := rangeToString(min)
	maxS := rangeToString(max)
	_, err := self.conn.ZRemRangeByScore(context.Background(), key, minS, maxS).Result()
	return lib_error.WrapError(err)
}

func (self *Client) ZRank(key string, member string) (int64, error) {
	v, err := self.conn.ZRank(context.Background(), key, member).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return -1, nil
		}
		return -1, lib_error.WrapError(err)
	}
	return v, nil
}

func (self *Client) ZRevRank(key string, member string) (int64, error) {
	v, err := self.conn.ZRevRank(context.Background(), key, member).Result()
	if err != nil {
		return -1, lib_error.WrapError(err)
	}
	return v, nil
}

func (self *Client) ZIncrBy(key string, member string, score uint64) error {
	_, err := self.conn.ZIncrBy(context.Background(), key, float64(score), member).Result()
	return lib_error.WrapError(err)
}

func (self *Client) ZRange(key string, start, end int64, reverse bool) ([]string, error) {
	var v *redis.StringSliceCmd
	if reverse {
		v = self.conn.ZRevRange(context.Background(), key, start, end)
	} else {
		v = self.conn.ZRange(context.Background(), key, start, end)
	}
	val, err := v.Result()
	return val, lib_error.WrapError(err)
}

func (self *Client) ZRangeWithScores(key string, start, end int64, reverse bool) ([]*SortedSetMember, error) {
	var v *redis.ZSliceCmd
	if reverse {
		v = self.conn.ZRevRangeWithScores(context.Background(), key, start, end)
	} else {
		v = self.conn.ZRangeWithScores(context.Background(), key, start, end)
	}
	arr, err := v.Result()
	if err != nil {
		return nil, lib_error.WrapError(err)
	}
	return loadSortedSetMembers(arr), nil
}

func (self *Client) ZRangeByScore(key string, min, max interface{}, reverse bool) ([]string, error) {
	var v *redis.StringSliceCmd
	if reverse {
		v = self.conn.ZRevRangeByScore(context.Background(), key, &redis.ZRangeBy{
			Min: rangeToString(max),
			Max: rangeToString(min),
		})
	} else {
		v = self.conn.ZRangeByScore(context.Background(), key, &redis.ZRangeBy{
			Min: rangeToString(min),
			Max: rangeToString(max),
		})
	}
	values, err := v.Result()
	return values, lib_error.WrapError(err)
}

func (self *Client) ZRangeByScoreWithScores(key string, min, max interface{}, reverse bool) ([]*SortedSetMember, error) {
	var v *redis.ZSliceCmd
	if reverse {
		v = self.conn.ZRevRangeByScoreWithScores(context.Background(), key, &redis.ZRangeBy{
			Min: rangeToString(max),
			Max: rangeToString(min),
		})
	} else {
		v = self.conn.ZRangeByScoreWithScores(context.Background(), key, &redis.ZRangeBy{
			Min: rangeToString(min),
			Max: rangeToString(max),
		})
	}
	values, err := v.Result()
	return loadSortedSetMembers(values), lib_error.WrapError(err)
}

func (self *Client) ZCount(key string, min, max interface{}) (int64, error) {
	value, err := self.conn.ZCount(context.Background(), key, rangeToString(min), rangeToString(max)).Result()
	if err != nil {
		return 0, lib_error.WrapError(err)
	}
	return value, nil
}

func (self *Client) ZCard(key string) (int64, error) {
	value, err := self.conn.ZCard(context.Background(), key).Result()
	if err != nil {
		return 0, lib_error.WrapError(err)
	}
	return value, nil
}

func (self *Client) ZScore(key, member string) (uint64, error) {
	v, err := self.conn.ZScore(context.Background(), key, member).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, lib_error.WrapError(err)
	}
	return uint64(v), nil
}

//===========================================================================
// Hash.
// Các phương thức thao tác với kiểu dữ liệu Hash trong Redis:



// Lấy giá trị của một trường trong hash
func (self *Client) HGet(dest interface{}, key, member string) error {
	value, err := self.conn.HGet(context.Background(), key, member).Result()
	if err != nil {
		return lib_error.WrapError(err)
	}
	err = unmarshal(value, dest)
	return lib_error.WrapError(err)
}

// Lấy giá trị của nhiều trường trong hash
func (self *Client) HMGet(dest interface{}, key string, members ...string) error {
	v, err := self.conn.HMGet(context.Background(), key, members...).Result()
	if err != nil {
		return lib_error.WrapError(err)
	}
	values := make([]string, len(v))
	for i, val := range v {
		values[i], _ = val.(string)
	}
	return unmarshalSlice(values, dest)
}
// Đặt giá trị cho một hoặc nhiều trường trong hash
func (self *Client) HSet(key string, nameAndValues ...interface{}) error {
	fields := make(map[string]interface{}, len(nameAndValues)/2)
	for i := 0; i < len(nameAndValues); i += 2 {
		name := nameAndValues[i].(string)
		redisValue, err := marshal(nameAndValues[i+1])
		if err != nil {
			return lib_error.WrapError(err)
		}
		fields[name] = redisValue
	}
	_, err := self.conn.HSet(context.Background(), key, fields).Result()
	return lib_error.WrapError(err)
}

// Xóa một hoặc nhiều trường trong hash
func (self *Client) HDel(key string, members ...string) error {
	_, err := self.conn.HDel(context.Background(), key, members...).Result()
	return lib_error.WrapError(err)
}
// : Tăng giá trị của một trường trong hash
func (self *Client) HIncrBy(key string, member string, incr int64) (int64, error) {
	value, err := self.conn.HIncrBy(context.Background(), key, member, incr).Result()
	if err != nil {
		return 0, lib_error.WrapError(err)
	}
	return value, nil
}
// Lấy số lượng trường trong hash
func (self *Client) HLen(key string) (int64, error) {
	value, err := self.conn.HLen(context.Background(), key).Result()
	if err != nil {
		return 0, lib_error.WrapError(err)
	}
	return value, nil
}
// Lấy tất cả các trường và giá trị trong hash
func (self *Client) HGetAll(dest interface{}, key string) error {
	values, err := self.conn.HGetAll(context.Background(), key).Result()
	if err != nil {
		return lib_error.WrapError(err)
	}
	return unmarshalMap(values, dest)
}

//===========================================================================
// Geospatial.
// Các phương thức thao tác với kiểu dữ liệu Geospatial trong Redis:



func (self *Client) GeoAddMany(key string, locations []*redis.GeoLocation) error {
	_, err := self.conn.GeoAdd(context.Background(), key, locations...).Result()
	return lib_error.WrapError(err)
}



// Thêm nhiều vị trí địa lý vào một khóa
func (self *Client) GeoAdd(key string, lat, lon float64, data interface{}) error {
	var err error
	location := &redis.GeoLocation{
		Latitude:  lat,
		Longitude: lon,
	}
	location.Name, err = marshal(data)
	if err != nil {
		return lib_error.WrapError(err)
	}
	_, err = self.conn.GeoAdd(context.Background(), key, location).Result()
	return lib_error.WrapError(err)
}

// Thêm một vị trí địa lý vào một khóa
func (self *Client) GeoDel(key string, args ...interface{}) error {
	var err error = nil
	members := make([]interface{}, len(args))
	for i, v := range args {
		members[i], err = marshal(v)
		if err != nil {
			return lib_error.WrapError(err)
		}
	}
	return self.ZRem(key, members...)
}
func (self *Client) GeoSearch(dest interface{}, key string, lat, lon, radius float64, unit string, count int) error {
	values, err := self.conn.GeoSearch(context.Background(), key, &redis.GeoSearchQuery{
		Longitude:  lon,
		Latitude:   lat,
		Radius:     radius,
		RadiusUnit: unit,
		Count:      count,
	}).Result()
	if err != nil {
		return lib_error.WrapError(err)
	}
	return unmarshalSlice(values, dest)
}
func (self *Client) GeoSearchManyWithDist(dest interface{}, keys []string, lat, lon, radius float64, unit string, count int) error {
	geoLocations := []redis.GeoLocation{}
	// Search all keys for geo locations
	for _, key := range keys {
		query := &redis.GeoSearchLocationQuery{}
		query.Latitude = lat
		query.Longitude = lon
		query.Radius = radius
		query.RadiusUnit = unit
		query.Count = count
		query.WithDist = true

		res, err := self.conn.GeoSearchLocation(context.Background(), key, query).Result()
		if err != nil {
			return lib_error.WrapError(err)
		}
		geoLocations = append(geoLocations, res...)
	}

	if count > len(geoLocations) {
		count = len(geoLocations)
	}

	// Too many results so we need to sort and return the closest results
	sort.SliceStable(geoLocations, func(i, j int) bool {
		return geoLocations[i].Dist < geoLocations[j].Dist
	})
	values := make([]string, count)
	for i := 0; i < count; i++ {
		values[i] = geoLocations[i].Name
	}
	return unmarshalSlice(values, dest)
}

func (self *Client) GeoPos(key string, members ...interface{}) ([][]float64, error) {
	var err error = nil
	tmp := make([]string, len(members))
	for i, member := range members {
		tmp[i], err = marshal(member)
		if err != nil {
			return nil, lib_error.WrapError(err)
		}
	}
	locations, err := self.conn.GeoPos(context.Background(), key, tmp...).Result()
	if err != nil {
		return nil, lib_error.WrapError(err)
	}
	datas := make([][]float64, len(locations))
	for i, loc := range locations {
		if loc == nil {
			continue
		}
		datas[i] = []float64{
			loc.Longitude, loc.Latitude,
		}
	}
	return datas, nil
}
func (self *Client) GeoRadius(dest interface{}, key string, lon, lat, radius float64, unit string) error {
	locations, err := self.conn.GeoRadius(context.Background(), key, lon, lat, &redis.GeoRadiusQuery{
		Radius: radius,
		Unit:   unit,
	}).Result()
	if err != nil {
		return lib_error.WrapError(err)
	}
	values := make([]string, len(locations))
	for i, loc := range locations {
		values[i] = loc.Name
	}
	return unmarshalSlice(values, dest)
}
