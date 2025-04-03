package lib_db_manager

import (
	"application/src/config"
	lib_cache "application/src/lib/cache"
	lib_debug "application/src/lib/debug"
	lib_error "application/src/lib/error"
	lib_redis "application/src/lib/redis"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"reflect"
	"sort"
	"strings"
)

/*
5. Tối ưu hóa hiệu suất
Mã này triển khai nhiều cơ chế tối ưu hóa:
Batch processing: Xử lý nhiều thay đổi cùng một lúc
Partial updates: Chỉ cập nhật các trường đã thay đổi
Prioritized execution: Thực hiện các thay đổi theo thứ tự ưu tiên
Deadlock detection: Phát hiện và báo cáo deadlock

6. Xử lý đặc biệt
Mã này xử lý nhiều trường hợp đặc biệt:
Force insert/update: Bắt buộc thêm mới hoặc cập nhật
Custom save functions: Cho phép tùy chỉnh logic lưu
Pre/post tasks: Thực hiện các tác vụ trước/sau khi lưu
Empty key handling: Xử lý trường hợp khóa chính rỗng
*/
const DefaultCacheTTL = 86400

/*
Mã này triển khai một hệ thống cache hai tầng:
Cache trong bộ nhớ: Lưu trữ các model đã truy vấn trong bộ nhớ của ứng dụng
Cache Redis: Lưu trữ các model trong Redis để chia sẻ giữa các instance của ứng dụng
LocalCache: Một lớp cache bổ sung cho dữ liệu master (ít thay đổi)
Hệ thống phân biệt giữa:
Dữ liệu master: Dữ liệu tham chiếu ít thay đổi, được cache trong LocalCache
Dữ liệu transaction: Dữ liệu thường xuyên thay đổi, được cache trong Redis

*/

type ModelManager struct {
	db                    *gorm.DB
	redis                 *lib_redis.Client
	modelInfos            map[string]*ModelInfo
	reservedSaveModelKeys []string
	reservedSaveInfos     map[string]*ReservedInfo
	reservedDelModelKeys  []string
	reservedDelModels     map[string]*ReservedInfo
	insertModelCount      uint32
	writeEndTasks         []*Task
	CacheTTL              int64
	writable              bool
	delCacheModels        []interface{}
	priorityMap           map[string]int
	local                 bool
}

func NewModelManager(db *gorm.DB, redis *lib_redis.Client, writable bool) *ModelManager {
	priorityMap := map[string]int{}
	if writable {
		priorityMap = config.TablePriorityMap()
	}
	return &ModelManager{
		db:                    db,
		redis:                 redis,
		modelInfos:            map[string]*ModelInfo{},
		reservedSaveModelKeys: []string{},
		reservedSaveInfos:     map[string]*ReservedInfo{},
		reservedDelModelKeys:  []string{},
		reservedDelModels:     map[string]*ReservedInfo{},
		insertModelCount:      0,
		CacheTTL:              DefaultCacheTTL,
		writable:              writable,
		delCacheModels:        []interface{}{},
		priorityMap:           priorityMap,
		local:                 config.GetIsLocal(),
	}

}
func (mm *ModelManager) log(format string, args ...interface{}) {
	if !mm.local {
		return
	}
	fmt.Println(fmt.Sprintf(format, args...))
}
func (mm *ModelManager) GetDB() *gorm.DB {
	return mm.db
}
func (mm *ModelManager) GetRedis() *lib_redis.Client {
	return mm.redis
}
func (mm *ModelManager) newStatement(model interface{}) *gorm.Statement {
	stmt := &gorm.Statement{DB: mm.db}
	stmt.Parse(model)
	return stmt
}
func (mm *ModelManager) getInfo(model interface{}) *ModelInfo {
	stmt := mm.newStatement(model)
	name := stmt.Quote(stmt.Table)
	if _, ok := mm.modelInfos[name]; !ok {
		mm.modelInfos[name] = NewModelInfo(mm.db, model)
	}
	return mm.modelInfos[name]
}
func (mm *ModelManager) GetTableName(model interface{}) string {
	return mm.getInfo(model).TableName
}
func (mm *ModelManager) getPrimaryKey(model interface{}) []reflect.Value {
	info := mm.getInfo(model)
	modelV := reflect.ValueOf(model).Elem()
	if modelV.Kind() == reflect.Ptr {
		modelV = reflect.Indirect(modelV)
	}
	keys := make([]reflect.Value, len(info.PrimaryFields))
	for i, field := range info.PrimaryFields {
		keys[i] = modelV.FieldByName(field.Name)
	}
	return keys
}

//	Truy vấn với khóa: Hỗ trợ cả khóa đơn và khóa phức hợp
//
// : Lấy giá trị khóa chính của model
func (mm *ModelManager) GetPrimaryKey(model interface{}) interface{} {
	keys := mm.getPrimaryKey(model)
	if len(keys) == 1 {
		return keys[0].Interface()
	}
	keyIs := make([]interface{}, len(keys))
	for i, v := range keys {
		keyIs[i] = v.Interface()
	}
	return keyIs
}

//	Truy vấn với khóa: Hỗ trợ cả khóa đơn và khóa phức hợp
//
// : Kiểm tra xem khóa chính có giá trị zero không
func (mm *ModelManager) PrimaryKeyZero(model interface{}) bool {
	keys := mm.getPrimaryKey(model)
	if len(keys) == 0 {
		return true
	}
	info := mm.getInfo(model)
	zero := false
	for i, key := range keys {
		if !info.PrimaryFields[i].AutoIncrement {
			continue
		} else if isBlank(key) {
			zero = true
		}
	}
	return zero
}
func (mm *ModelManager) GetFields(model interface{}) map[string]*schema.Field {
	info := mm.getInfo(model)
	return info.StructFields
}
func (mm *ModelManager) GetModelName(model interface{}) string {
	v := reflect.ValueOf(model)
	t := v.Type()
	for t.Kind() != reflect.Struct {
		t = t.Elem()
	}
	return t.Name()
}
func (mm *ModelManager) Select(db *gorm.DB, model interface{}) *gorm.DB {
	info := mm.getInfo(model)
	return info.Select(db)
}
func (mm *ModelManager) makeKey(model interface{}) string {
	info := mm.getInfo(model)
	var pkey string
	if mm.PrimaryKeyZero(model) {
		pkey = fmt.Sprintf("NewModel:%v", mm.insertModelCount)
		mm.insertModelCount++
	} else {
		pkey = info.MakePrimaryKeyStringByModel(model, 0)
	}
	return fmt.Sprintf("{%s}%s:%s", info.TableName, info.TableName, pkey)
}

//	Truy vấn với cache: Kiểm tra cache trước khi truy vấn database
//
// : Lấy một model theo khóa chính
func (mm *ModelManager) GetModel(dest interface{}, pkey interface{}) (interface{}, error) {
	return mm.getModel(dest, pkey, false)
}

// Truy vấn với cache: Kiểm tra cache trước khi truy vấn database
// : Lấy nhiều model theo danh sách khóa chính
func (mm *ModelManager) GetModels(dest interface{}, pkeys interface{}) error {
	return mm.getModels(dest, pkeys, false)
}

func (mm *ModelManager) getModels(dest interface{}, pkeys interface{}, forUpdate bool) error {
	destV := reflect.ValueOf(dest)
	elem := destV.Elem()
	sl := reflect.MakeSlice(elem.Type(), 0, 0)
	info := mm.getInfo(dest)
	mm.log(info.TableName)
	keyLen := info.KeyLen()
	// プライマリキーの有無確認用にmapを用意.
	keyValues := reflect.ValueOf(pkeys)
	keyCnt := keyValues.Len()
	keyStrings := make(map[string]bool, keyCnt)
	for i := 0; i < keyCnt; i++ {
		keyString := info.MakePrimaryKeyString(keyValues.Index(i).Interface())
		keyStrings[keyString] = false
	}
	// すでに取得済みのモデルを取得.
	cachedModels := info.CacheStore().GetModels(pkeys)
	for _, cachedModel := range cachedModels {
		if forUpdate && !info.GetIsForUpdate(cachedModel) {
			key := mm.makeKey(cachedModel)
			if _, danger := mm.reservedSaveInfos[key]; danger {
				return fmt.Errorf("for updateの前にモデルを保存していて危険です:%s", key)
			}
			// ロックして取得し直すためにこれは含めない.
			continue
		}
		for i := 0; i < keyLen; i++ {
			keyString := info.MakePrimaryKeyStringByModel(cachedModel, i+1)
			if _, ok := keyStrings[keyString]; ok {
				keyStrings[keyString] = true
			}
		}
		sl = reflect.Append(sl, reflect.ValueOf(cachedModel))
	}
	// 未取得のキーを集める.
	notExists := map[int][]interface{}{}
	for i := 0; i < keyCnt; i++ {
		key := keyValues.Index(i).Interface()
		keyValue := reflect.ValueOf(key)
		keyString := info.MakePrimaryKeyString(key)
		if keyStrings[keyString] {
			// 取得済み.
			continue
		}
		length := 1
		if keyValue.Kind() == reflect.Array || keyValue.Kind() == reflect.Slice {
			length = keyValue.Len()
		}
		if _, ok := notExists[length]; !ok {
			notExists[length] = []interface{}{}
		}
		notExists[length] = append(notExists[length], key)
	}
	if !forUpdate && 0 < len(notExists) {
		mm.log("Get from cache...%v", len(notExists))
		if info.GetIsMaster(dest) {
			// マスターデータはlocalcacheから取得.
			mKeys := notExists[keyLen]
			if 0 < len(mKeys) {
				cacheStrings := make([]string, len(mKeys))
				for i, mKey := range mKeys {
					cacheStrings[i] = fmt.Sprintf("%s##%s", info.TableName, info.MakePrimaryKeyString(mKey))
				}
				tmpSlicePtr := reflect.New(elem.Type()).Interface()
				lib_cache.GetLocalCacheMany(cacheStrings, tmpSlicePtr)
				tmpSlice := reflect.ValueOf(tmpSlicePtr).Elem()
				exists := make(map[string]bool, tmpSlice.Len())
				for i := 0; i < tmpSlice.Len(); i++ {
					si := tmpSlice.Index(i)
					if si.IsNil() {
						continue
					}
					sl = reflect.Append(sl, si)
					exists[info.MakePrimaryKeyStringByModel(si.Interface(), 0)] = true
					info.CacheStore().Save(si.Interface())
				}
				notExists[keyLen] = []interface{}{}
				for _, mKey := range mKeys {
					ks := info.MakePrimaryKeyString(mKey)
					if exists[ks] {
						// localcacheから取得できた.
						continue
					}
					notExists[keyLen] = append(notExists[keyLen], mKey)
				}
			}
		} else if mm.redis != nil && !mm.writable {
			// トランデータはredisから取得.
			mKeys := notExists[keyLen]
			if 0 < len(mKeys) {
				mm.log("from redis...%v", mKeys)
				cacheKeys := make([]string, len(mKeys))
				for i, mKey := range mKeys {
					cacheKeys[i] = fmt.Sprintf("{%s}%s:%s", info.TableName, info.TableName, info.MakePrimaryKeyString(mKey))
				}
				tmpSlicePtr := reflect.New(elem.Type()).Interface()
				err := mm.redis.MGet(tmpSlicePtr, cacheKeys...)
				if err != nil {
					lib_debug.Error("MGet error keys=%v", cacheKeys)
					return lib_error.WrapError(err)
				}
				tmpSlice := reflect.ValueOf(tmpSlicePtr).Elem()
				notExists[keyLen] = []interface{}{}
				for i := 0; i < tmpSlice.Len(); i++ {
					vElem := tmpSlice.Index(i)
					if vElem.IsNil() {
						// キャッシュになかった.
						notExists[keyLen] = append(notExists[keyLen], mKeys[i])
						mm.log("not found:%v", mKeys[i])
						continue
					}
					sl = reflect.Append(sl, vElem)
					info.CacheStore().Save(vElem.Interface())
				}
			}
		}
	}
	if 0 < len(notExists) {
		db := mm.db
		if forUpdate {
			db = db.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		for length, keys := range notExists {
			if len(keys) == 0 {
				continue
			}
			mm.log("keys:%v:%v", keys, length)
			tmpSlicePtr := reflect.New(elem.Type()).Interface()
			scope := info.Select(info.Query(db, keys, length)).Find(tmpSlicePtr)
			if scope.Error != nil && !errors.Is(scope.Error, gorm.ErrRecordNotFound) {
				return lib_error.WrapError(scope.Error)
			}
			tmpSlice := reflect.ValueOf(tmpSlicePtr).Elem()
			if forUpdate {
				for i := 0; i < tmpSlice.Len(); i++ {
					err := info.setIsForUpdate(tmpSlice.Index(i).Interface(), true)
					if err != nil {
						return lib_error.WrapError(err)
					}
				}
			}
			info.CacheStore().Save(tmpSlice.Interface())
			if info.GetIsMaster(dest) {
				// localcacheに保存.
				for i := 0; i < tmpSlice.Len(); i++ {
					data := tmpSlice.Index(i).Interface()
					k := fmt.Sprintf("%s##%s", info.TableName, info.MakePrimaryKeyStringByModel(data, 0))
					err := lib_cache.SetLocalCache(k, data)
					if err != nil {
						return lib_error.WrapError(err)
					}
				}
			} else if mm.redis != nil && keyLen == length {
				// redisに保存.
				models := make([]interface{}, tmpSlice.Len())
				for i := 0; i < tmpSlice.Len(); i++ {
					models[i] = tmpSlice.Index(i).Interface()
				}
				err := mm.SaveModelsToCache(models, mm.CacheTTL, forUpdate)
				if err != nil {
					return lib_error.WrapError(err)
				}
			}
			sl = reflect.AppendSlice(sl, tmpSlice)
		}
	}
	elem.Set(sl)
	return nil
}
func (mm *ModelManager) GetModelForUpdate(dest interface{}, pkey interface{}) (interface{}, error) {
	return mm.getModel(dest, pkey, true)
}
func (mm *ModelManager) GetModelForUpdateWithCreate(dest interface{}, pkey interface{}, create func(interface{}, interface{}) (interface{}, error)) (interface{}, error) {
	ins, err := mm.getModel(dest, pkey, true)
	if err != nil || ins != nil || create == nil {
		return ins, err
	}
	ins, err = create(dest, pkey)
	if err != nil || ins == nil {
		return nil, err
	}
	info := mm.getInfo(dest)
	err = info.setIsForUpdate(ins, true)
	if err != nil {
		return nil, err
	}
	info.CacheStore().Save(ins)
	return ins, nil
}

func (mm *ModelManager) GetModelsForUpdate(dest interface{}, pkeys interface{}) error {
	return mm.getModels(dest, pkeys, true)
}

func (mm *ModelManager) SaveModelsToCache(models []interface{}, ttl int64, update bool) error {
	var err error = nil
	masterTables := map[string]bool{}
	masterDataMap := map[string]interface{}{}
	for _, model := range models {
		info := mm.getInfo(model)
		if info.GetIsMaster(model) {
			// マスターデータはlocalcache.
			masterTables[info.TableName] = true
			cacheKey := fmt.Sprintf("%s##%s", info.TableName, info.MakePrimaryKeyStringByModel(model, 0))
			masterDataMap[cacheKey] = model
			mm.log("SaveModelsToCache(Master):%s", cacheKey)
			continue
		} else if mm.redis == nil {
			return nil
		}
		// トランデータはredisへ.
		key := mm.makeKey(model)
		if update {
			// 更新の場合は上書き.
			err = mm.redis.Set(key, model)
		} else {
			// 更新ではない場合は無かったときだけ.
			err = mm.redis.SetNx(key, model)
		}
		if err == nil {
			err = mm.redis.Expire(key, ttl)
		}
		if err != nil {
			return lib_error.WrapError(err)
		}
		mm.log("SaveModelsToCache:%s", key)
	}
	if 0 < len(masterTables) {
		if update {
			// 更新の場合は削除.
			// マスターデータのキーを削除.
			if mm.redis != nil {
				cacheKeys := make([]string, len(masterTables))
				index := 0
				for tableName := range masterTables {
					cacheKeys[index] = fmt.Sprintf("mastermodel_keyset2::%s", tableName)
					index++
				}
				err = mm.redis.Del(cacheKeys...)
				if err != nil {
					return lib_error.WrapError(err)
				}
			}
			// ローカルキャッシュを削除.
			err = lib_cache.DeleteLocalCache(mm.redis)
			if err != nil {
				return lib_error.WrapError(err)
			}
		} else {
			// ローカルキャッシュに保存.
			err = lib_cache.SetLocalCacheMany(masterDataMap)
			if err != nil {
				return lib_error.WrapError(err)
			}
		}
	}
	return nil
}

func (mm *ModelManager) getModel(dest interface{}, pkey interface{}, forUpdate bool) (interface{}, error) {
	destV := reflect.ValueOf(dest)
	destSlicePtr := reflect.New(reflect.SliceOf(destV.Type()))
	keySliceValue := reflect.MakeSlice(reflect.SliceOf(reflect.ValueOf(pkey).Type()), 1, 1)
	keySliceValue.Index(0).Set(reflect.ValueOf(pkey))
	err := mm.getModels(destSlicePtr.Interface(), keySliceValue.Interface(), forUpdate)
	if err != nil {
		return false, err
	}
	elem := destV.Elem()
	if destSlicePtr.Elem().Len() == 0 {
		// 無かった.
		return nil, nil
	} else {
		result := destSlicePtr.Elem().Index(0)
		elem.Set(result.Elem())
		return result.Interface(), nil
	}
}

func (mm *ModelManager) RetrievePreviouslyAcquiredModels(model, pkeys interface{}) []interface{} {
	info := mm.getInfo(model)
	return info.CacheStore().GetModels(pkeys)
}
func (mm *ModelManager) SetGotModel(model interface{}) {
	info := mm.getInfo(model)
	info.CacheStore().Save(model)
}

type MasterModelAllCacheData struct {
	Version string
	Keys    []interface{}
}

func (mm *ModelManager) GetMasterModelAll(dest interface{}, reload bool, version string) error {
	var err error = nil
	info := mm.getInfo(dest)
	cacheKey := fmt.Sprintf("mastermodel_keyset2::%s", info.TableName)
	var cacheData *MasterModelAllCacheData = nil
	if mm.redis != nil && !reload {
		// キャッシュからIDを取得.
		exists, err := mm.redis.Exists(cacheKey)
		if err != nil {
			return err
		} else if exists {
			cacheData = &MasterModelAllCacheData{}
			_, err = mm.redis.Get(cacheData, cacheKey)
			if err != nil {
				return err
			} else if cacheData.Version != version {
				cacheData = nil
			}
		}
	}
	if cacheData != nil {
		// キャッシュあり.
		return mm.GetModels(dest, cacheData.Keys)
	}
	// 取り直し.
	err = mm.Select(mm.db, dest).Find(dest).Error
	if err != nil || mm.redis == nil {
		return err
	}
	// キャッシュに保存.
	destV := reflect.ValueOf(dest).Elem()
	keys := make([]interface{}, destV.Len())
	for i := range keys {
		destModel := destV.Index(i)
		if destModel.Kind() == reflect.Ptr {
			destModel = reflect.Indirect(destModel)
		}
		if info.KeyLen() == 1 {
			keys[i] = destModel.FieldByName(info.PrimaryFields[0].Name).Interface()
		} else {
			destModelKeys := make([]interface{}, info.KeyLen())
			for j, field := range info.PrimaryFields {
				destModelKeys[j] = destModel.FieldByName(field.Name).Interface()
			}
			keys[i] = destModelKeys
		}
		err = mm.SaveModelsToCache([]interface{}{destModel.Interface()}, mm.CacheTTL, false)
		if err != nil {
			return err
		}
	}
	err = mm.redis.Set(cacheKey, &MasterModelAllCacheData{
		Version: version,
		Keys:    keys,
	})
	if err == nil {
		err = mm.redis.Expire(cacheKey, mm.CacheTTL)
	}
	return err
}
func (mm *ModelManager) getReservedInfo(model interface{}) (*ReservedInfo, bool, error) {
	// 識別子.
	key := mm.makeKey(model)
	// 既に登録されている予約情報.
	info := mm.reservedSaveInfos[key]
	if info != nil {
		info.Model = model
		return info, false, nil
	}
	// 削除予約されていないかを確認.
	d := mm.reservedDelModels[key]
	if d != nil {
		return nil, false, lib_error.NewAppError(lib_error.DefaultErrorCode, "削除予約に入ってる:%s", key)
	}
	// 新規作成.
	info = &ReservedInfo{
		Model:       model,
		fieldMap:    map[string]bool{},
		ForceInsert: false,
		ForceUpdate: false,
		SavedTasks:  []*Task{},
		Priority:    mm.priorityMap[mm.getInfo(model).TableName],
	}
	mm.reservedSaveInfos[key] = info
	mm.reservedSaveModelKeys = append(mm.reservedSaveModelKeys, key)
	return info, true, nil
}
func (mm *ModelManager) GetSaveModelCount() int {
	return len(mm.reservedSaveModelKeys)
}
func (mm *ModelManager) GetDeleteModelCount() int {
	return len(mm.reservedDelModelKeys)
}
func (mm *ModelManager) SetSave(model interface{}, opt *SaveOptions) error {
	if opt == nil {
		opt = &SaveOptions{}
	} else if opt.ForceInsert && opt.ForceUpdate {
		return lib_error.NewAppError(lib_error.DefaultErrorCode, "ForceInsertとForceUpdateは両方をtrueに出来ません")
	}
	if opt.Fields == nil {
		opt.Fields = []string{}
	}
	info, isNew, err := mm.getReservedInfo(model)
	if err != nil {
		return err
	} else if 0 < len(info.SaveFunctions) && opt.SaveFunction == nil {
		return lib_error.NewAppError(lib_error.DefaultErrorCode, "SaveFunctionを使用する場合は通常の更新はできません.既にSaveFunctionを登録済みです")
	} else if !isNew && len(info.SaveFunctions) == 0 && opt.SaveFunction != nil {
		return lib_error.NewAppError(lib_error.DefaultErrorCode, "SaveFunctionを使用する場合は通常の更新はできません.既に保存予約がされています")
	}
	if opt.SaveFunction != nil {
		info.SaveFunctions = append(info.SaveFunctions, opt.SaveFunction)
	}
	info.ForceInsert = info.ForceInsert || opt.ForceInsert
	info.ForceUpdate = info.ForceUpdate || opt.ForceUpdate
	if info.ForceInsert && info.ForceUpdate {
		return lib_error.NewAppError(lib_error.DefaultErrorCode, "ForceInsertとForceUpdateは両方をtrueで指定していないけど、複数回呼ばれた結果そうなっています")
	}
	if len(opt.Fields) == 0 || info.Fields == nil {
		// 全部更新.
		info.Fields = opt.Fields
		info.fieldMap = map[string]bool{}
		for _, field := range opt.Fields {
			info.fieldMap[field] = true
		}
	} else if 0 < len(info.Fields) {
		// 更新するフィールドを追加.
		for _, field := range opt.Fields {
			if _, exists := info.fieldMap[field]; exists {
				continue
			}
			info.Fields = append(info.Fields, field)
			info.fieldMap[field] = true
		}
	}
	if opt.PrepareTask != nil {
		info.PrepareTasks = append(info.PrepareTasks, opt.PrepareTask)
	}
	if opt.SavedTask != nil {
		info.SavedTasks = append(info.SavedTasks, opt.SavedTask)
	}
	return nil
}
func (mm *ModelManager) SetSaveSupportEmptyKey(model interface{}) error {
	var opt *SaveOptions = nil
	pkey := mm.GetPrimaryKey(model)
	if fmt.Sprintf("%v", pkey) == "0" {
		// プライマリキーが0.
		modelInfo := mm.getInfo(model)
		var cnt int64 = 0
		err := mm.db.Model(model).Where(fmt.Sprintf("`%s` = 0", modelInfo.PrimaryFields[0].DBName)).Count(&cnt).Error
		if err != nil {
			return lib_error.WrapError(err)
		} else if cnt != 0 {
			// これは普通にSaveするとduplicate entryになってしまう.
			opt = &SaveOptions{
				ForceUpdate: true,
			}
		}
	}
	return mm.SetSave(model, opt)
}
func (mm *ModelManager) SetDelete(model interface{}) error {
	// 識別子.
	key := mm.makeKey(model)
	if d := mm.reservedDelModels[key]; d != nil {
		// 設定済み.
		return nil
	}
	// 保存設定されていないかを確認.
	info := mm.reservedSaveInfos[key]
	if info != nil {
		return lib_error.NewAppError(lib_error.DefaultErrorCode, "保存予約に入ってる:%s", key)
	}
	// 削除設定.
	mm.reservedDelModels[key] = &ReservedInfo{
		Model:    model,
		Priority: mm.priorityMap[mm.getInfo(model).TableName],
	}
	mm.reservedDelModelKeys = append(mm.reservedDelModelKeys, key)
	return nil
}
func (mm *ModelManager) SetForceDelete(model interface{}) error {
	// 識別子.
	key := mm.makeKey(model)
	// 保存設定をキャンセル.
	info := mm.reservedSaveInfos[key]
	if info != nil {
		index := -1
		for i, k := range mm.reservedSaveModelKeys {
			if k == key {
				index = i
				break
			}
		}
		if index != -1 {
			mm.reservedSaveModelKeys = append(mm.reservedSaveModelKeys[:index], mm.reservedSaveModelKeys[index+1:]...)
		}
		delete(mm.reservedSaveInfos, key)
	}
	// 削除設定.
	return mm.SetDelete(model)
}
func (mm *ModelManager) AddWriteEndTask(f func(...interface{}) error, args ...interface{}) {
	mm.writeEndTasks = append(mm.writeEndTasks, &Task{
		F:    f,
		Args: args,
	})
}
func (mm *ModelManager) WriteAll() error {
	var err error = nil
	if !mm.writable {
		return lib_error.WrapError(fmt.Errorf("書き込み不可状態です"))
	}
	// すべてのフィールドを保存したモデル.
	savedModels := []interface{}{}
	// フィールド指定して更新したモデル.
	updatedModels := []interface{}{}
	// 削除したモデル.
	deletedModels := []interface{}{}
	// 優先順位をみて並び替え.
	keys := append(mm.reservedSaveModelKeys, mm.reservedDelModelKeys...)
	sort.Slice(keys, func(i, j int) bool {
		k0 := keys[i]
		k1 := keys[j]
		info0 := mm.reservedSaveInfos[k0]
		info1 := mm.reservedSaveInfos[k1]
		if info0 == nil {
			info0 = mm.reservedDelModels[k0]
		}
		if info1 == nil {
			info1 = mm.reservedDelModels[k1]
		}
		if info0 == nil {
			return false
		} else if info1 == nil {
			return true
		} else if info0.Priority == info1.Priority {
			// 優先度が同じ場合は文字列でソート.
			return k0 < k1
		}
		// 優先度が高い順にソート.
		return info1.Priority < info0.Priority
	})
	// 保存.
	for _, key := range keys {
		info := mm.reservedSaveInfos[key]
		if info != nil {
			// 事前準備のタスク.
			for _, task := range info.PrepareTasks {
				err = task.F(task.Args...)
				if err != nil {
					return lib_error.WrapError(err)
				}
			}
			// 保存.
			modelInfo := mm.getInfo(info.Model)
			if 0 < len(info.SaveFunctions) {
				// 自前保存タスク.
				for _, task := range info.SaveFunctions {
					err = task.F(mm.db, task.Args...)
					if err != nil {
						return lib_error.WrapError(err)
					}
				}
				updatedModels = append(updatedModels, info.Model)
			} else if info.ForceInsert {
				// insert.
				err = lib_error.WrapError(mm.db.Create(info.Model).Error)
				if modelInfo.GetIsMaster(info.Model) {
					savedModels = append(savedModels, info.Model)
				}
			} else if 0 < len(info.Fields) || info.ForceUpdate {
				if info.ForceUpdate && len(info.Fields) == 0 {
					// プライマリキー以外のすべてのカラムを指定.
					for k, field := range modelInfo.StructFields {
						if field.PrimaryKey {
							continue
						}
						info.Fields = append(info.Fields, k)
					}
				}
				// フィールドを指定してupdate.
				db := modelInfo.QueryByModel(mm.db, info.Model)
				modelValue := reflect.ValueOf(info.Model)
				if modelValue.Kind() == reflect.Ptr {
					modelValue = reflect.Indirect(modelValue)
				}
				updates := make(map[string]interface{}, len(info.Fields))
				for _, fieldName := range info.Fields {
					field := modelInfo.StructFields[fieldName]
					fieldV := modelValue.FieldByName(fieldName)
					if !fieldV.IsValid() {
						return fmt.Errorf("invalid field : %s.%s", modelInfo.TableName, fieldName)
					}
					updates[field.DBName] = fieldV.Interface()
				}
				err = db.Updates(updates).Error
				if err != nil {
					err = lib_error.WrapError(fmt.Errorf("%s:%s", modelInfo.TableName, err.Error()))
				}
				updatedModels = append(updatedModels, info.Model)
			} else if modelInfo.HasCustomTypes() {
				// DBに存在しているかを確認.
				db := modelInfo.QueryByModel(mm.db, info.Model)
				var cnt int64 = 0
				err = db.Count(&cnt).Error
				if err != nil {
					return lib_error.WrapError(err)
				} else if cnt == 0 {
					// insert.
					err = lib_error.WrapError(mm.db.Create(info.Model).Error)
				} else {
					// update.
					pfMap := make(map[string]bool, len(modelInfo.PrimaryFields))
					for _, pf := range modelInfo.PrimaryFields {
						pfMap[pf.DBName] = true
					}
					modelValue := reflect.Indirect(reflect.ValueOf(info.Model))
					updates := make(map[string]interface{}, len(info.Fields))
					for _, field := range modelInfo.StructFields {
						if isPf := pfMap[field.DBName]; isPf {
							continue
						}
						updates[field.DBName] = modelValue.FieldByName(field.Name).Interface()
					}
					err = lib_error.WrapError(db.Updates(updates).Error)
				}
				savedModels = append(savedModels, info.Model)
			} else {
				// すべてのフィールドを更新.
				err = lib_error.WrapError(mm.db.Save(info.Model).Error)
				savedModels = append(savedModels, info.Model)
			}
			if err != nil {
				if strings.Contains(err.Error(), "Deadlock") {
					// デッドロック.
					sqlDB, _ := mm.db.DB()
					if sqlDB != nil {
						t := ""
						n := ""
						s := ""
						err = sqlDB.QueryRow("SHOW ENGINE INNODB STATUS").Scan(&t, &n, &s)
						if err != nil {
							s = err.Error()
						}
						return lib_error.WrapError(fmt.Errorf("Deadlock!!key=%s\n%s", key, s))
					}
					return lib_error.WrapError(fmt.Errorf("Deadlock!!key=%s", key))
				}
				appErr := lib_error.WrapError(err).(*lib_error.AppError)
				appErr.Message = fmt.Sprintf("key=%s,err=%s", key, appErr.Message)
				return appErr
			}
			// 書き込み後のタスク.
			for _, task := range info.SavedTasks {
				err = task.F(task.Args...)
				if err != nil {
					return lib_error.WrapError(err)
				}
			}
		} else {
			// 削除.
			info = mm.reservedDelModels[key]
			if info == nil || info.Model == nil {
				continue
			}
			pkey := mm.GetPrimaryKey(info.Model)
			lib_debug.Debug("Delete:%s...%v", mm.getInfo(info.Model).TableName, pkey)
			if fmt.Sprintf("%v", pkey) == "0" {
				modelInfo := mm.getInfo(info.Model)
				err = mm.db.Where(fmt.Sprintf("`%s` = 0", modelInfo.PrimaryFields[0].DBName)).Delete(info.Model).Error
			} else {
				err = mm.db.Delete(info.Model).Error
			}
			if err != nil {
				return lib_error.WrapError(err)
			}
			deletedModels = append(deletedModels, info.Model)
		}
	}
	mm.reservedSaveModelKeys = []string{}
	mm.reservedSaveInfos = map[string]*ReservedInfo{}
	mm.reservedDelModelKeys = []string{}
	mm.reservedDelModels = map[string]*ReservedInfo{}

	// ModelManagerがガメているモデルを破棄.
	mm.modelInfos = map[string]*ModelInfo{}
	// 書き込み後のタスク.
	for _, task := range mm.writeEndTasks {
		err = task.F(task.Args...)
		if err != nil {
			return lib_error.WrapError(err)
		}
	}
	// キャッシュを操作.
	err = mm.SaveModelsToCache(savedModels, mm.CacheTTL, true)
	if err != nil {
		return lib_error.WrapError(err)
	}
	err = mm.DeleteModelsFromCache(updatedModels)
	if err != nil {
		return lib_error.WrapError(err)
	}
	err = mm.DeleteModelsFromCache(deletedModels)
	if err != nil {
		return lib_error.WrapError(err)
	}
	mm.delCacheModels = append(mm.delCacheModels, updatedModels...)
	mm.delCacheModels = append(mm.delCacheModels, deletedModels...)
	return nil
}
func (mm *ModelManager) WriteEnd() error {
	err := mm.DeleteModelsFromCache(mm.delCacheModels)
	return lib_error.WrapError(err)
}

func (mm *ModelManager) DeleteModelsFromCache(models []interface{}) error {
	var err error = nil
	masterTables := map[string]bool{}
	keys := []string{}
	for _, model := range models {
		info := mm.getInfo(model)
		if info.GetIsMaster(model) {
			// マスターデータはlocalcacheを削除.
			masterTables[info.TableName] = true
			continue
		} else if mm.redis == nil {
			return nil
		}
		// トランデータはredisにある.
		keys = append(keys, mm.makeKey(model))
	}
	if 0 < len(keys) {
		err = mm.redis.Del(keys...)
		if err != nil {
			return lib_error.WrapError(err)
		}
		mm.log("DeleteModelsFromCache:%v", keys)
	}
	if 0 < len(masterTables) {
		// マスターデータのキーを削除.
		if mm.redis != nil {
			cacheKeys := make([]string, len(masterTables))
			index := 0
			for tableName := range masterTables {
				cacheKeys[index] = fmt.Sprintf("mastermodel_keyset2::%s", tableName)
				index++
			}
			err = mm.redis.Del(cacheKeys...)
			if err != nil {
				return lib_error.WrapError(err)
			}
		}
		// ローカルキャッシュを削除.
		err = lib_cache.DeleteLocalCache(mm.redis)
		if err != nil {
			return lib_error.WrapError(err)
		}
	}
	return nil
}
func isBlank(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.String:
		return value.Len() == 0
	case reflect.Bool:
		return !value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return value.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return value.IsNil()
	}
	return reflect.DeepEqual(value.Interface(), reflect.Zero(value.Type()).Interface())
}
