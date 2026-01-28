package funk

import (
	"reflect"
)

// Union returns the union between two collections.
func Union(collections ...interface{}) interface{} {
	// shortcut zero/single argument
	if len(collections) == 0 {
		return nil
	} else if len(collections) == 1 {
		return collections[0]
	}

	if !IsIteratee(collections[0]) {
		panic("Parameter must be a collection")
	}

	cType := reflect.TypeOf(collections[0])
	zLen := 0

	for i, x := range collections {
		xValue := reflect.ValueOf(x)
		xType := xValue.Type()
		if i > 0 && NotEqual(cType, xType) {
			panic("Parameters must have the same type")
		}

		zLen += xValue.Len()
	}

	if cType.Kind() == reflect.Map {
		zType := reflect.MapOf(cType.Key(), cType.Elem())
		zMap := reflect.MakeMap(zType)

		for _, x := range collections {
			xIter := reflect.ValueOf(x).MapRange()
			for xIter.Next() {
				zMap.SetMapIndex(xIter.Key(), xIter.Value())
			}
		}

		return zMap.Interface()
	} else {
		zType := reflect.SliceOf(cType.Elem())
		zSlice := reflect.MakeSlice(zType, 0, 0)

		for _, x := range collections {
			xValue := reflect.ValueOf(x)
			zSlice = reflect.AppendSlice(zSlice, xValue)
		}

		return zSlice.Interface()
	}
}

// UnionStringMap returns the union between multiple string maps
func UnionStringMap(x ...map[string]string) map[string]string {
	zMap := map[string]string{}
	for _, xMap := range x {
		for k, v := range xMap {
			zMap[k] = v
		}
	}
	return zMap
}
