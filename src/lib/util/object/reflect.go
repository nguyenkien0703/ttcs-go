package lib_util_object

import (
	"reflect"
)

func StructToMap(data interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	elem := reflect.ValueOf(data).Elem()
	size := elem.NumField()
	for i := 0; i < size; i++ {
		field := elem.Type().Field(i).Name
		value := elem.Field(i).Interface()
		result[field] = value
	}
	return result
}

func StructToIntMap(data interface{}) map[string]int {
	tmp := StructToMap(data)
	result := map[string]int{}
	for k, v := range tmp {
		result[k] = v.(int)
	}
	return result
}

func StructToUint8Map(data interface{}) map[string]uint8 {
	tmp := StructToMap(data)
	result := map[string]uint8{}
	for k, v := range tmp {
		result[k] = v.(uint8)
	}
	return result
}

func StructToUint16Map(data interface{}) map[string]uint16 {
	tmp := StructToMap(data)
	result := map[string]uint16{}
	for k, v := range tmp {
		result[k] = v.(uint16)
	}
	return result
}
