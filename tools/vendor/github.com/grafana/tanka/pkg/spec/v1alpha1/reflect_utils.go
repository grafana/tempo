package v1alpha1

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

func getDeepFieldAsString(obj interface{}, keyPath []string) (string, error) {
	if !isSupportedType(obj, []reflect.Kind{reflect.Struct, reflect.Pointer, reflect.Map}) {
		return "", errors.New("intermediary objects must be object types")
	}

	objValue := reflectValue(obj)
	objType := objValue.Type()

	var nextFieldValue reflect.Value

	switch objType.Kind() {
	case reflect.Struct, reflect.Pointer:
		fieldsCount := objType.NumField()

		for i := 0; i < fieldsCount; i++ {
			candidateType := objType.Field(i)
			candidateValue := objValue.Field(i)
			jsonTag := candidateType.Tag.Get("json")

			if strings.Split(jsonTag, ",")[0] == keyPath[0] {
				nextFieldValue = candidateValue
				break
			}
		}

	case reflect.Map:
		for _, key := range objValue.MapKeys() {
			nextFieldValue = objValue.MapIndex(key)
		}
	}

	if len(keyPath) == 1 {
		return getReflectValueAsString(nextFieldValue)
	}

	if nextFieldValue.Type().Kind() == reflect.Pointer {
		nextFieldValue = nextFieldValue.Elem()
	}

	return getDeepFieldAsString(nextFieldValue.Interface(), keyPath[1:])
}

func getReflectValueAsString(val reflect.Value) (string, error) {
	switch val.Type().Kind() {
	case reflect.String:
		return val.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10), nil
	case reflect.Float32:
		return strconv.FormatFloat(val.Float(), 'f', -1, 32), nil
	case reflect.Float64:
		return strconv.FormatFloat(val.Float(), 'f', -1, 64), nil
	case reflect.Bool:
		return strconv.FormatBool(val.Bool()), nil
	default:
		return "", errors.New("unsupported value type")
	}
}

func reflectValue(obj interface{}) reflect.Value {
	var val reflect.Value

	if reflect.TypeOf(obj).Kind() == reflect.Pointer {
		val = reflect.ValueOf(obj).Elem()
	} else {
		val = reflect.ValueOf(obj)
	}

	return val
}

func isSupportedType(obj interface{}, types []reflect.Kind) bool {
	for _, t := range types {
		if reflect.TypeOf(obj).Kind() == t {
			return true
		}
	}

	return false
}
