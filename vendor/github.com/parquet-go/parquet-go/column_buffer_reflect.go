package parquet

import (
	"cmp"
	"math/bits"
	"reflect"
	"sort"
	"unsafe"

	"github.com/parquet-go/parquet-go/sparse"
)

type anymap interface {
	entries() (keys, values sparse.Array)
}

type gomap[K cmp.Ordered] struct {
	keys []K
	vals reflect.Value // slice
	swap func(int, int)
	size uintptr
}

func (m *gomap[K]) Len() int { return len(m.keys) }

func (m *gomap[K]) Less(i, j int) bool { return cmp.Compare(m.keys[i], m.keys[j]) < 0 }

func (m *gomap[K]) Swap(i, j int) {
	m.keys[i], m.keys[j] = m.keys[j], m.keys[i]
	m.swap(i, j)
}

func (m *gomap[K]) entries() (keys, values sparse.Array) {
	return makeArrayOf(m.keys), makeArray(m.vals.UnsafePointer(), m.Len(), m.size)
}

type reflectMap struct {
	keys    reflect.Value // slice
	vals    reflect.Value // slice
	numKeys int
	keySize uintptr
	valSize uintptr
}

func (m *reflectMap) entries() (keys, values sparse.Array) {
	return makeArray(m.keys.UnsafePointer(), m.numKeys, m.keySize), makeArray(m.vals.UnsafePointer(), m.numKeys, m.valSize)
}

func makeMapFuncOf(mapType reflect.Type) func(reflect.Value) anymap {
	switch mapType.Key().Kind() {
	case reflect.Int:
		return makeMapFunc[int](mapType)
	case reflect.Int8:
		return makeMapFunc[int8](mapType)
	case reflect.Int16:
		return makeMapFunc[int16](mapType)
	case reflect.Int32:
		return makeMapFunc[int32](mapType)
	case reflect.Int64:
		return makeMapFunc[int64](mapType)
	case reflect.Uint:
		return makeMapFunc[uint](mapType)
	case reflect.Uint8:
		return makeMapFunc[uint8](mapType)
	case reflect.Uint16:
		return makeMapFunc[uint16](mapType)
	case reflect.Uint32:
		return makeMapFunc[uint32](mapType)
	case reflect.Uint64:
		return makeMapFunc[uint64](mapType)
	case reflect.Uintptr:
		return makeMapFunc[uintptr](mapType)
	case reflect.Float32:
		return makeMapFunc[float32](mapType)
	case reflect.Float64:
		return makeMapFunc[float64](mapType)
	case reflect.String:
		return makeMapFunc[string](mapType)
	}

	keyType := mapType.Key()
	valType := mapType.Elem()

	mapBuffer := &reflectMap{
		keySize: keyType.Size(),
		valSize: valType.Size(),
	}

	keySliceType := reflect.SliceOf(keyType)
	valSliceType := reflect.SliceOf(valType)
	return func(mapValue reflect.Value) anymap {
		length := mapValue.Len()

		if !mapBuffer.keys.IsValid() || mapBuffer.keys.Len() < length {
			capacity := 1 << bits.Len(uint(length))
			mapBuffer.keys = reflect.MakeSlice(keySliceType, capacity, capacity)
			mapBuffer.vals = reflect.MakeSlice(valSliceType, capacity, capacity)
		}

		mapBuffer.numKeys = length
		for i, mapIter := 0, mapValue.MapRange(); mapIter.Next(); i++ {
			mapBuffer.keys.Index(i).SetIterKey(mapIter)
			mapBuffer.vals.Index(i).SetIterValue(mapIter)
		}

		return mapBuffer
	}
}

func makeMapFunc[K cmp.Ordered](mapType reflect.Type) func(reflect.Value) anymap {
	keyType := mapType.Key()
	valType := mapType.Elem()
	valSliceType := reflect.SliceOf(valType)
	mapBuffer := &gomap[K]{size: valType.Size()}
	return func(mapValue reflect.Value) anymap {
		length := mapValue.Len()

		if cap(mapBuffer.keys) < length {
			capacity := 1 << bits.Len(uint(length))
			mapBuffer.keys = make([]K, capacity)
			mapBuffer.vals = reflect.MakeSlice(valSliceType, capacity, capacity)
			mapBuffer.swap = reflect.Swapper(mapBuffer.vals.Interface())
		}

		mapBuffer.keys = mapBuffer.keys[:length]
		for i, mapIter := 0, mapValue.MapRange(); mapIter.Next(); i++ {
			reflect.NewAt(keyType, unsafe.Pointer(&mapBuffer.keys[i])).Elem().SetIterKey(mapIter)
			mapBuffer.vals.Index(i).SetIterValue(mapIter)
		}

		sort.Sort(mapBuffer)
		return mapBuffer
	}
}
