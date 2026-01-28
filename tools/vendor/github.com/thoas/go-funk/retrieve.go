package funk

import (
	"reflect"
	"strings"
)

// Get retrieves the value from given path, retriever can be modified with available RetrieverOptions
func Get(out interface{}, path string, opts ...option) interface{} {
	options := newOptions(opts...)

	result := get(reflect.ValueOf(out), path, opts...)
	// valid kind and we can return a result.Interface() without panic
	if result.Kind() != reflect.Invalid && result.CanInterface() {
		// if we don't allow zero and the result is a zero value return nil
		if !options.allowZero && result.IsZero() {
			return nil
		}
		// if the result kind is a pointer and its nil return nil
		if result.Kind() == reflect.Ptr && result.IsNil() {
			return nil
		}
		// return the result interface (i.e the zero value of it)
		return result.Interface()
	}

	return nil
}

// GetOrElse retrieves the value of the pointer or default.
func GetOrElse(v interface{}, def interface{}) interface{} {
	val := reflect.ValueOf(v)
	if v == nil || (val.Kind() == reflect.Ptr && val.IsNil()) {
		return def
	} else if val.Kind() != reflect.Ptr {
		return v
	}
	return val.Elem().Interface()
}

func get(value reflect.Value, path string, opts ...option) reflect.Value {
	options := newOptions(opts...)

	if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		var resultSlice reflect.Value

		length := value.Len()

		if length == 0 {
			zeroElement := reflect.Zero(value.Type().Elem())
			pathValue := get(zeroElement, path)
			value = reflect.MakeSlice(reflect.SliceOf(pathValue.Type()), 0, 0)

			return value
		}

		for i := 0; i < length; i++ {
			item := value.Index(i)

			resultValue := get(item, path)

			if resultValue.Kind() == reflect.Invalid || (resultValue.IsZero() && !options.allowZero) {
				continue
			}

			resultType := resultValue.Type()

			if resultSlice.Kind() == reflect.Invalid {
				resultType := reflect.SliceOf(resultType)

				resultSlice = reflect.MakeSlice(resultType, 0, 0)
			}

			resultSlice = reflect.Append(resultSlice, resultValue)
		}

		// if the result is a slice of a slice, we need to flatten it
		if resultSlice.Kind() != reflect.Invalid && resultSlice.Type().Elem().Kind() == reflect.Slice {
			return flattenDeep(resultSlice)
		}

		return resultSlice
	}

	quoted := false
	parts := strings.FieldsFunc(path, func(r rune) bool {
		if r == '"' {
			quoted = !quoted
		}
		return !quoted && r == '.'
	})

	for i, part := range parts {
		parts[i] = strings.Trim(part, "\"")
	}

	for _, part := range parts {
		value = redirectValue(value)
		kind := value.Kind()

		switch kind {
		case reflect.Invalid:
			continue
		case reflect.Struct:
			if isNilIndirection(value, part) {
				return reflect.ValueOf(nil)
			}
			value = value.FieldByName(part)
		case reflect.Map:
			value = value.MapIndex(reflect.ValueOf(part))
		case reflect.Slice, reflect.Array:
			value = get(value, part)
		default:
			return reflect.ValueOf(nil)
		}
	}

	return value
}

func isNilIndirection(v reflect.Value, name string) bool {
	vType := v.Type()
	for i := 0; i < vType.NumField(); i++ {
		field := vType.Field(i)
		if !isEmbeddedStructPointerField(field) {
			return false
		}

		fieldType := field.Type.Elem()

		_, found := fieldType.FieldByName(name)
		if found {
			return v.Field(i).IsNil()
		}
	}

	return false
}

func isEmbeddedStructPointerField(field reflect.StructField) bool {
	if !field.Anonymous {
		return false
	}

	return field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct
}
