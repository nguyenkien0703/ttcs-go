package lib_db_manager

import (
	lib_db_fields "application/src/lib/db/fields"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"reflect"
	"strings"
)

type ModelInfo struct {
	TableName          string
	PrimaryFields      []*schema.Field
	StructFields       map[string]*schema.Field
	customTypes        map[string]lib_db_fields.CustomType
	customSelectClause string
	cacheStore         *ModelCacheStore
	dbNameMap          map[string]string
}

func NewModelInfo(db *gorm.DB, model interface{}) *ModelInfo {
	stmt := gorm.Statement{DB: db}
	stmt.Parse(model)
	info := &ModelInfo{
		TableName:     stmt.Table,
		PrimaryFields: stmt.Schema.PrimaryFields,
		StructFields:  make(map[string]*schema.Field, len(stmt.Schema.Fields)),
		customTypes:   map[string]lib_db_fields.CustomType{},
		cacheStore:    nil,
		dbNameMap:     make(map[string]string, len(stmt.Schema.Fields)),
	}
	for _, field := range stmt.Schema.Fields {
		info.StructFields[field.Name] = field
		// customTypes.
		indirectType := field.FieldType
		for indirectType.Kind() == reflect.Ptr {
			indirectType = indirectType.Elem()
		}
		//fieldValue := reflect.New(indirectType).Interface()
		//switch v := fieldValue.(type) {
		//case lib_db_fields.Point:
		//	info.customTypes[field.Name] = &v
		//case *lib_db_fields.Point:
		//	info.customTypes[field.Name] = v
		//}
		info.dbNameMap[field.DBName] = field.Name
	}
	return info
}
func (mi *ModelInfo) KeyLen() int {
	return len(mi.PrimaryFields)
}
func (mi *ModelInfo) CacheStore() *ModelCacheStore {
	if mi.cacheStore == nil {
		mi.cacheStore = NewModelCacheStore(mi)
	}
	return mi.cacheStore
}

func (mi *ModelInfo) MakePrimaryKeyString(pkey interface{}) string {
	var keys []string
	switch pk := pkey.(type) {
	case []string:
		keys = pk
	default:
		keyValue := reflect.ValueOf(pk)
		switch keyValue.Kind() {
		case reflect.Array:
			fallthrough
		case reflect.Slice:
			keys = make([]string, keyValue.Len())
			for i := 0; i < keyValue.Len(); i++ {
				keys[i] = fmt.Sprintf("%v", keyValue.Index(i).Interface())
			}
		default:
			keys = []string{fmt.Sprintf("%v", pkey)}
		}
	}
	return strings.Join(keys, "#&#")
}
func (mi *ModelInfo) MakePrimaryKeyStringByModel(model interface{}, length int) string {
	modelValue := reflect.ValueOf(model)
	if modelValue.Kind() == reflect.Ptr {
		modelValue = reflect.Indirect(modelValue)
	}
	keyLen := mi.KeyLen()
	if length < 1 || keyLen < length {
		length = keyLen
	}
	pkeyStrings := make([]string, length)
	for i := 0; i < length; i++ {
		field := mi.PrimaryFields[i]
		pkeyStrings[i] = fmt.Sprintf("%v", modelValue.FieldByName(field.Name).Interface())
	}
	return mi.MakePrimaryKeyString(pkeyStrings)
}
func (mi *ModelInfo) getMethodByName(model interface{}, name string) reflect.Value {
	modelValue := reflect.ValueOf(model)
	tmp := modelValue.MethodByName(name)
	if tmp.IsValid() {
		return tmp
	}
	modelValue = reflect.Indirect(modelValue)
	if modelValue.Kind() == reflect.Array || modelValue.Kind() == reflect.Slice {
		tmp := reflect.Indirect(reflect.New(modelValue.Type().Elem()))
		if tmp.Kind() == reflect.Ptr {
			tmp = reflect.Indirect(reflect.New(tmp.Type().Elem()))
		}
		modelValue = tmp
	}
	return modelValue.MethodByName(name)
}
func (mi *ModelInfo) GetIsMaster(model interface{}) bool {
	getIsMaster := mi.getMethodByName(model, "GetIsMaster")
	if !getIsMaster.IsValid() {
		return false
	}
	return getIsMaster.Call([]reflect.Value{})[0].Interface().(bool)
}

func (mi *ModelInfo) GetIsForUpdate(model interface{}) bool {
	getIsForUpdate := mi.getMethodByName(model, "GetIsForUpdate")
	if !getIsForUpdate.IsValid() {
		return false
	}
	return getIsForUpdate.Call([]reflect.Value{})[0].Interface().(bool)
}
func (mi *ModelInfo) setIsForUpdate(model interface{}, flag bool) error {
	setIsForUpdate := mi.getMethodByName(model, "SetIsForUpdate")
	if !setIsForUpdate.IsValid() {
		return fmt.Errorf("for update設定できないモデルです")
	}
	setIsForUpdate.Call([]reflect.Value{reflect.ValueOf(flag)})
	return nil
}
func (mi *ModelInfo) HasCustomTypes() bool {
	return 0 < len(mi.customTypes)
}
func (mi *ModelInfo) GetCustomType(name string) lib_db_fields.CustomType {
	customType := mi.customTypes[name]
	if customType != nil {
		return customType
	}
	name = mi.dbNameMap[name]
	customType = mi.customTypes[name]
	return customType
}
func (mi *ModelInfo) getCustomSelectClause() string {
	if 0 < len(mi.customSelectClause) {
		return mi.customSelectClause
	}
	columns := make([]string, len(mi.StructFields))
	i := 0
	for fieldName, field := range mi.StructFields {
		customType := mi.customTypes[fieldName]
		if customType != nil {
			columns[i] = customType.Select(field.DBName)
		} else {
			columns[i] = fmt.Sprintf("`%s`", field.DBName)
		}
		i++
	}
	mi.customSelectClause = strings.Join(columns, ",")
	return mi.customSelectClause
}
func (mi *ModelInfo) Select(db *gorm.DB) *gorm.DB {
	if 0 < len(mi.customTypes) {
		db = db.Select(mi.getCustomSelectClause())
	}
	return db
}
func (mi *ModelInfo) Query(db *gorm.DB, keys []interface{}, keyLength int) *gorm.DB {
	db = db.Table(mi.TableName)
	if mi.KeyLen() < keyLength {
		keyLength = mi.KeyLen()
	}
	if keyLength == 1 {
		// キーの長さが1の場合はシンプル.
		if len(keys) == 1 {
			return db.Where(fmt.Sprintf("`%s` = ?", mi.PrimaryFields[0].DBName), keys[0])
		} else {
			return db.Where(fmt.Sprintf("`%s` in (?)", mi.PrimaryFields[0].DBName), keys)
		}
	}
	sarr := make([]string, keyLength)
	for i := range sarr {
		sarr[i] = mi.PrimaryFields[i].DBName
	}
	varr := make([]string, len(keys))
	for i := range varr {
		varr[i] = "(?)"
	}
	query := fmt.Sprintf("(`%s`) in (%s)", strings.Join(sarr, "`,`"), strings.Join(varr, ","))
	return db.Where(query, keys...)
}
func (mi *ModelInfo) QueryByModel(db *gorm.DB, models interface{}) *gorm.DB {
	modelsValue := reflect.ValueOf(models)
	if modelsValue.Kind() != reflect.Array && modelsValue.Kind() != reflect.Slice {
		tmp := reflect.New(reflect.SliceOf(modelsValue.Type())).Elem()
		tmp = reflect.Append(tmp, modelsValue)
		modelsValue = tmp
	}
	keys := make([]interface{}, modelsValue.Len())
	for i := 0; i < modelsValue.Len(); i++ {
		modelValue := modelsValue.Index(i)
		if modelValue.Kind() == reflect.Ptr {
			modelValue = reflect.Indirect(modelValue)
		}
		pkeys := make([]interface{}, mi.KeyLen())
		for j, field := range mi.PrimaryFields {
			pkeys[j] = modelValue.FieldByName(field.Name).Interface()
		}
		keys[i] = pkeys
	}
	return mi.Query(db, keys, mi.KeyLen()).Model(models)
}
