package engine

import (
	"fmt"
	"time"
)

func IsMapToInterface(arg interface{}) (map[string]interface{}, error) {
	switch args := arg.(type) {
	case map[string]interface{}:
		return args, nil
	default:
		return nil, fmt.Errorf("type of interface is not map[string]interface{}")
	}
}

func IsArray(arg interface{}) ([]interface{}, error) {
	switch args := arg.(type) {
	case []interface{}:
		return args, nil
	default:
		return nil, fmt.Errorf("type of interface is not []interface{}")
	}
}

func GetMapValueFromStringKey(arg map[string]interface{}, key string) (interface{}, error) {
	if value, ok := arg[key]; ok {
		return value, nil
	}

	return nil, fmt.Errorf("no such value")
}

func CheckIfFloat64IsInt(value float64) interface{} {
	intVal := int64(value)
	if (value - float64(intVal)) == 0 {
		return int64(value)
	}
	return value
}

func CheckIfFloat32IsInt(value float32) interface{} {
	intVal := int64(value)
	if (value - float32(intVal)) == 0 {
		return int64(value)
	}
	return value
}

func GetNow() string {
	now := time.Now()
	return now.String()
}
