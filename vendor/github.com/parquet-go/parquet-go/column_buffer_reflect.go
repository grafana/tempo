package parquet

import (
	"cmp"
	"fmt"
	"math/bits"
	"reflect"
	"sort"
	"time"
	"unsafe"

	"github.com/parquet-go/parquet-go/deprecated"
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
	return makeArrayFromSlice(m.keys), makeArray(m.vals.UnsafePointer(), m.Len(), m.size)
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

// writeValueFunc is a function that writes a single reflect.Value to a set of
// column buffers.
// Panics if the value cannot be written (similar to reflect package behavior).
type writeValueFunc func([]ColumnBuffer, columnLevels, reflect.Value)

// timeOfDayNanos returns the nanoseconds since midnight for the given time.
func timeOfDayNanos(t time.Time) int64 {
	m := nearestMidnightLessThan(t)
	return t.Sub(m).Nanoseconds()
}

func writeTime(col ColumnBuffer, levels columnLevels, t time.Time, node Node) {
	typ := node.Type()

	if logicalType := typ.LogicalType(); logicalType != nil {
		switch {
		case logicalType.Timestamp != nil:
			// TIMESTAMP logical type -> write to int64
			unit := logicalType.Timestamp.Unit
			var val int64
			switch {
			case unit.Millis != nil:
				val = t.UnixMilli()
			case unit.Micros != nil:
				val = t.UnixMicro()
			default:
				val = t.UnixNano()
			}
			col.writeInt64(levels, val)
			return

		case logicalType.Date != nil:
			// DATE logical type -> write to int32
			col.writeInt32(levels, int32(daysSinceUnixEpoch(t)))
			return

		case logicalType.Time != nil:
			// TIME logical type -> write time of day
			unit := logicalType.Time.Unit
			nanos := timeOfDayNanos(t)
			switch {
			case unit.Millis != nil:
				col.writeInt32(levels, int32(nanos/1e6))
			case unit.Micros != nil:
				col.writeInt64(levels, nanos/1e3)
			default:
				col.writeInt64(levels, nanos)
			}
			return
		}
	}

	// No time logical type - use physical type
	switch typ.Kind() {
	case Int32:
		// int32 without logical type -> days since epoch
		col.writeInt32(levels, int32(daysSinceUnixEpoch(t)))
	case Int64:
		// int64 without logical type -> nanoseconds since epoch
		col.writeInt64(levels, t.UnixNano())
	case Float:
		// float -> fractional seconds since epoch
		col.writeFloat(levels, float32(float64(t.UnixNano())/1e9))
	case Double:
		// double -> fractional seconds since epoch
		col.writeDouble(levels, float64(t.UnixNano())/1e9)
	case ByteArray:
		// byte array -> RFC3339Nano
		s := t.Format(time.RFC3339Nano)
		col.writeByteArray(levels, unsafe.Slice(unsafe.StringData(s), len(s)))
	default:
		panic(fmt.Sprintf("cannot write time.Time to column with physical type %v", typ))
	}
}

func writeDuration(col ColumnBuffer, levels columnLevels, d time.Duration, node Node) {
	typ := node.Type()

	if logicalType := typ.LogicalType(); logicalType != nil && logicalType.Time != nil {
		// TIME logical type
		unit := logicalType.Time.Unit
		switch {
		case unit.Millis != nil:
			col.writeInt32(levels, int32(d.Milliseconds()))
		case unit.Micros != nil:
			col.writeInt64(levels, d.Microseconds())
		default:
			col.writeInt64(levels, d.Nanoseconds())
		}
		return
	}

	// No TIME logical type - use physical type
	switch typ.Kind() {
	case Int32:
		panic("cannot write time.Duration to int32 column without TIME logical type")
	case Int64:
		// int64 -> nanoseconds
		col.writeInt64(levels, d.Nanoseconds())
	case Float:
		// float -> seconds
		col.writeFloat(levels, float32(d.Seconds()))
	case Double:
		// double -> seconds
		col.writeDouble(levels, d.Seconds())
	case ByteArray:
		// byte array -> String()
		s := d.String()
		col.writeByteArray(levels, unsafe.Slice(unsafe.StringData(s), len(s)))
	default:
		panic(fmt.Sprintf("cannot write time.Duration to column with physical type %v", typ))
	}
}

// writeValueFuncOf constructs a function that writes reflect.Values to column buffers.
// It follows the deconstructFuncOf pattern, recursively building functions for the schema tree.
// Returns (nextColumnIndex, writeFunc).
func writeValueFuncOf(columnIndex int16, node Node) (int16, writeValueFunc) {
	switch {
	case node.Optional():
		return writeValueFuncOfOptional(columnIndex, node)
	case node.Repeated():
		return writeValueFuncOfRepeated(columnIndex, node)
	case isList(node):
		return writeValueFuncOfList(columnIndex, node)
	case isMap(node):
		return writeValueFuncOfMap(columnIndex, node)
	default:
		return writeValueFuncOfRequired(columnIndex, node)
	}
}

func writeValueFuncOfOptional(columnIndex int16, node Node) (int16, writeValueFunc) {
	nextColumnIndex, writeValue := writeValueFuncOf(columnIndex, Required(node))
	return nextColumnIndex, func(columns []ColumnBuffer, levels columnLevels, value reflect.Value) {
		if !value.IsValid() {
			writeValue(columns, levels, value)
			return
		}
		if value.IsZero() {
			writeValue(columns, levels, value)
			return
		}
		levels.definitionLevel++
		writeValue(columns, levels, value)
	}
}

func writeValueFuncOfRepeated(columnIndex int16, node Node) (int16, writeValueFunc) {
	nextColumnIndex, writeValue := writeValueFuncOf(columnIndex, Required(node))
	return nextColumnIndex, func(columns []ColumnBuffer, levels columnLevels, value reflect.Value) {
		for {
			if !value.IsValid() {
				writeValue(columns, levels, reflect.Value{})
				return
			}
			kind := value.Kind()
			if kind != reflect.Interface && kind != reflect.Pointer {
				break
			}
			if value.IsNil() {
				writeValue(columns, levels, reflect.Value{})
				break
			}
			value = value.Elem()
		}

		switch value.Kind() {
		case reflect.Slice, reflect.Array:
			if value.Len() == 0 {
				writeValue(columns, levels, reflect.Value{})
				return
			}

			levels.repetitionDepth++
			levels.definitionLevel++
			writeValue(columns, levels, value.Index(0))

			if n := value.Len(); n > 1 {
				levels.repetitionLevel = levels.repetitionDepth
				for i := 1; i < n; i++ {
					writeValue(columns, levels, value.Index(i))
				}
			}
		default:
			levels.repetitionDepth++
			levels.definitionLevel++

			// If this is a repeated group with a single field, and the value is a scalar,
			// wrap the scalar into a struct with that field name.
			if !node.Leaf() && value.IsValid() && value.Kind() != reflect.Struct && value.Kind() != reflect.Map {
				fields := Required(node).Fields()
				if len(fields) == 1 {
					field := fields[0]
					fieldType := field.GoType()
					fieldName := field.Name()

					if value.Type().AssignableTo(fieldType) || value.Type().ConvertibleTo(fieldType) {
						structType := reflect.StructOf([]reflect.StructField{
							{Name: fieldName, Type: fieldType},
						})
						wrappedValue := reflect.New(structType).Elem()

						if value.Type().AssignableTo(fieldType) {
							wrappedValue.Field(0).Set(value)
						} else {
							wrappedValue.Field(0).Set(value.Convert(fieldType))
						}

						value = wrappedValue
					}
				}
			}

			writeValue(columns, levels, value)
		}
	}
}

func writeValueFuncOfRequired(columnIndex int16, node Node) (int16, writeValueFunc) {
	switch {
	case node.Leaf():
		return writeValueFuncOfLeaf(columnIndex, node)
	default:
		return writeValueFuncOfGroup(columnIndex, node)
	}
}

func writeValueFuncOfList(columnIndex int16, node Node) (int16, writeValueFunc) {
	return writeValueFuncOf(columnIndex, Repeated(listElementOf(node)))
}

func writeValueFuncOfMap(columnIndex int16, node Node) (int16, writeValueFunc) {
	keyValue := mapKeyValueOf(node)
	keyValueType := keyValue.GoType()
	keyValueElem := keyValueType.Elem()
	keyType := keyValueElem.Field(0).Type
	valueType := keyValueElem.Field(1).Type
	nextColumnIndex, writeValue := writeValueFuncOf(columnIndex, schemaOf(keyValueElem))

	return nextColumnIndex, func(columns []ColumnBuffer, levels columnLevels, mapValue reflect.Value) {
		if mapValue.Len() == 0 {
			writeValue(columns, levels, reflect.Zero(keyValueElem))
			return
		}

		levels.repetitionDepth++
		levels.definitionLevel++

		mapType := mapValue.Type()
		mapKey := reflect.New(mapType.Key()).Elem()
		mapElem := reflect.New(mapType.Elem()).Elem()

		elem := reflect.New(keyValueElem).Elem()
		k := elem.Field(0)
		v := elem.Field(1)

		for it := mapValue.MapRange(); it.Next(); {
			mapKey.SetIterKey(it)
			mapElem.SetIterValue(it)
			k.Set(mapKey.Convert(keyType))
			v.Set(mapElem.Convert(valueType))
			writeValue(columns, levels, elem)
			levels.repetitionLevel = levels.repetitionDepth
		}
	}
}

func writeValueFuncOfGroup(columnIndex int16, node Node) (int16, writeValueFunc) {
	type fieldWriter struct {
		fieldName  string
		writeValue writeValueFunc
	}

	fields := node.Fields()
	writers := make([]fieldWriter, len(fields))
	for i, field := range fields {
		writers[i].fieldName = field.Name()
		columnIndex, writers[i].writeValue = writeValueFuncOf(columnIndex, field)
	}

	// Pre-compute type information for common map types
	mapStringStringType := reflect.TypeOf((map[string]string)(nil))
	mapStringAnyType := reflect.TypeOf((map[string]any)(nil))

	return columnIndex, func(columns []ColumnBuffer, levels columnLevels, value reflect.Value) {
		for {
			if !value.IsValid() {
				for i := range writers {
					w := &writers[i]
					w.writeValue(columns, levels, reflect.Value{})
				}
				return
			}

			switch t := value.Type(); t.Kind() {
			case reflect.Map:
				switch {
				case t.ConvertibleTo(mapStringStringType):
					m := value.Convert(mapStringStringType).Interface().(map[string]string)
					v := new(string)
					for i := range writers {
						w := &writers[i]
						*v = m[w.fieldName]
						w.writeValue(columns, levels, reflect.ValueOf(v).Elem())
					}

				case t.ConvertibleTo(mapStringAnyType):
					m := value.Convert(mapStringAnyType).Interface().(map[string]any)
					for i := range writers {
						w := &writers[i]
						v := m[w.fieldName]
						w.writeValue(columns, levels, reflect.ValueOf(v))
					}

				default:
					for i := range writers {
						w := &writers[i]
						fieldName := reflect.ValueOf(&w.fieldName).Elem()
						fieldValue := value.MapIndex(fieldName)
						w.writeValue(columns, levels, fieldValue)
					}
				}
				return

			case reflect.Struct:
				for i := range writers {
					w := &writers[i]
					fieldValue := value.FieldByName(w.fieldName)
					w.writeValue(columns, levels, fieldValue)
				}
				return

			case reflect.Pointer, reflect.Interface:
				if value.IsNil() {
					value = reflect.Value{}
				} else {
					value = value.Elem()
				}

			default:
				value = reflect.Value{}
			}
		}
	}
}

func writeValueFuncOfLeaf(columnIndex int16, node Node) (int16, writeValueFunc) {
	if columnIndex > MaxColumnIndex {
		panic("row cannot be written because it has more than 127 columns")
	}
	return columnIndex + 1, func(columns []ColumnBuffer, levels columnLevels, value reflect.Value) {
		col := columns[columnIndex]
		for {
			if !value.IsValid() {
				col.writeNull(levels)
				return
			}
			switch value.Kind() {
			case reflect.Pointer, reflect.Interface:
				if value.IsNil() {
					col.writeNull(levels)
					return
				}
				value = value.Elem()
				continue
			case reflect.Bool:
				col.writeBoolean(levels, value.Bool())
			case reflect.Int8, reflect.Int16, reflect.Int32:
				col.writeInt32(levels, int32(value.Int()))
			case reflect.Int:
				col.writeInt64(levels, value.Int())
			case reflect.Int64:
				if value.Type() == reflect.TypeFor[time.Duration]() {
					writeDuration(col, levels, time.Duration(value.Int()), node)
				} else {
					col.writeInt64(levels, value.Int())
				}
			case reflect.Uint8, reflect.Uint16, reflect.Uint32:
				col.writeInt32(levels, int32(value.Uint()))
			case reflect.Uint, reflect.Uint64:
				col.writeInt64(levels, int64(value.Uint()))
			case reflect.Float32:
				col.writeFloat(levels, float32(value.Float()))
			case reflect.Float64:
				col.writeDouble(levels, value.Float())
			case reflect.String:
				s := value.String()
				col.writeByteArray(levels, unsafe.Slice(unsafe.StringData(s), len(s)))
			case reflect.Slice, reflect.Array:
				col.writeByteArray(levels, value.Bytes())
			case reflect.Struct:
				switch v := value.Interface().(type) {
				case time.Time:
					writeTime(col, levels, v, node)
				case deprecated.Int96:
					col.writeInt96(levels, v)
				default:
					goto unsupported
				}
			default:
				goto unsupported
			}
			return
		unsupported:
			panic(fmt.Sprintf("cannot write value of type %s to leaf column", value.Type()))
		}
	}
}
