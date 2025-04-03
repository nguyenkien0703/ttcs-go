package lib_db_manager

import (
	"fmt"
	"reflect"
)

type ModelCacheStore struct {
	info  *ModelInfo
	store map[string]interface{}
}

func NewModelCacheStore(info *ModelInfo) *ModelCacheStore {
	return &ModelCacheStore{
		info:  info,
		store: map[string]interface{}{},
	}
}

func (mcs *ModelCacheStore) saveModel(model interface{}) {
	pKeyLen := mcs.info.KeyLen()
	modelValue := reflect.ValueOf(model)
	if modelValue.Kind() == reflect.Ptr {
		modelValue = reflect.Indirect(modelValue)
	} else {
		tmp := reflect.New(modelValue.Type())
		tmp.Elem().Set(modelValue)
		model = tmp.Interface()
	}
	mapAddr := &mcs.store
	for i, field := range mcs.info.PrimaryFields {
		modelField := modelValue.FieldByName(field.Name)
		key := fmt.Sprintf("%v", modelField.Interface())
		if i == (pKeyLen - 1) {
			(*mapAddr)[key] = model
		} else {
			if _, exists := (*mapAddr)[key]; !exists {
				(*mapAddr)[key] = &map[string]interface{}{}
			}
			mapAddr = (*mapAddr)[key].(*map[string]interface{})
		}
	}
}
func (mcs *ModelCacheStore) Save(model interface{}) {
	modelValue := reflect.ValueOf(model)
	switch modelValue.Kind() {
	case reflect.Array:
		fallthrough
	case reflect.Slice:
		// 複数.
		for i := 0; i < modelValue.Len(); i++ {
			mcs.saveModel(modelValue.Index(i).Interface())
		}
	default:
		// 単体.
		mcs.saveModel(model)
	}
}
func (mcs *ModelCacheStore) get(pkey interface{}) []interface{} {
	pkeyValue := reflect.ValueOf(pkey)
	var keys []string
	if pkeyValue.Kind() == reflect.Array || pkeyValue.Kind() == reflect.Slice {
		// 複数キー取得.
		keys = make([]string, pkeyValue.Len())
		for i := range keys {
			keys[i] = fmt.Sprintf("%v", pkeyValue.Index(i).Interface())
		}
	} else {
		// 単体キー取得.
		keys = []string{fmt.Sprintf("%v", pkey)}
	}
	mapAddr := &mcs.store
	for _, key := range keys {
		v, ok := (*mapAddr)[key]
		if !ok {
			// 存在しない.
			return []interface{}{}
		}
		switch v.(type) {
		case *map[string]interface{}:
			mapAddr = (*mapAddr)[key].(*map[string]interface{})
		default:
			// 探索終わり.
			return []interface{}{v}
		}
	}
	return aggregateStoreMapItems(mapAddr)
}

func (mcs *ModelCacheStore) GetModel(pkey interface{}) interface{} {
	v := reflect.ValueOf(mcs.get(pkey))
	if v.Len() == 0 {
		return nil
	}
	return v.Index(0).Interface()
}
func (mcs *ModelCacheStore) GetModels(pkeys interface{}) []interface{} {
	pkeyValues := reflect.ValueOf(pkeys)
	dest := []interface{}{}
	for i := 0; i < pkeyValues.Len(); i++ {
		arr := mcs.get(pkeyValues.Index(i).Interface())
		if 0 < len(arr) {
			dest = append(dest, arr...)
		}
	}
	return dest
}

func aggregateStoreMapItems(mapAddr *map[string]interface{}) []interface{} {
	dest := []interface{}{}
	for _, v := range *mapAddr {
		switch vv := v.(type) {
		case *map[string]interface{}:
			arr := aggregateStoreMapItems(vv)
			if 0 < len(arr) {
				dest = append(dest, arr...)
			}
		default:
			dest = append(dest, vv)
		}
	}
	return dest
}
