//go:build go1.18

package parquet

import (
	"math/bits"
	"reflect"
	"unsafe"

	"github.com/segmentio/parquet-go/deprecated"
	"github.com/segmentio/parquet-go/internal/unsafecast"
	"github.com/segmentio/parquet-go/sparse"
)

// writeRowsFunc is the type of functions that apply rows to a set of column
// buffers.
//
// - columns is the array of column buffer where the rows are written.
//
// - rows is the array of Go values to write to the column buffers.
//
//   - levels is used to track the column index, repetition and definition levels
//     of values when writing optional or repeated columns.
type writeRowsFunc func(columns []ColumnBuffer, rows sparse.Array, levels columnLevels) error

// writeRowsFuncOf generates a writeRowsFunc function for the given Go type and
// parquet schema. The column path indicates the column that the function is
// being generated for in the parquet schema.
func writeRowsFuncOf(t reflect.Type, schema *Schema, path columnPath) writeRowsFunc {
	switch t {
	case reflect.TypeOf(deprecated.Int96{}):
		return writeRowsFuncOfRequired(t, schema, path)
	}

	switch t.Kind() {
	case reflect.Bool,
		reflect.Int,
		reflect.Uint,
		reflect.Int32,
		reflect.Uint32,
		reflect.Int64,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64,
		reflect.String:
		return writeRowsFuncOfRequired(t, schema, path)

	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return writeRowsFuncOfRequired(t, schema, path)
		} else {
			return writeRowsFuncOfSlice(t, schema, path)
		}

	case reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return writeRowsFuncOfRequired(t, schema, path)
		}

	case reflect.Pointer:
		return writeRowsFuncOfPointer(t, schema, path)

	case reflect.Struct:
		return writeRowsFuncOfStruct(t, schema, path)

	case reflect.Map:
		return writeRowsFuncOfMap(t, schema, path)
	}

	panic("cannot convert Go values of type " + typeNameOf(t) + " to parquet value")
}

func writeRowsFuncOfRequired(t reflect.Type, schema *Schema, path columnPath) writeRowsFunc {
	column := schema.mapping.lookup(path)
	columnIndex := column.columnIndex
	return func(columns []ColumnBuffer, rows sparse.Array, levels columnLevels) error {
		columns[columnIndex].writeValues(rows, levels)
		return nil
	}
}

func writeRowsFuncOfOptional(t reflect.Type, schema *Schema, path columnPath, writeRows writeRowsFunc) writeRowsFunc {
	nullIndex := nullIndexFuncOf(t)
	return func(columns []ColumnBuffer, rows sparse.Array, levels columnLevels) error {
		if rows.Len() == 0 {
			return writeRows(columns, rows, levels)
		}

		nulls := acquireBitmap(rows.Len())
		defer releaseBitmap(nulls)
		nullIndex(nulls.bits, rows)

		nullLevels := levels
		levels.definitionLevel++
		// In this function, we are dealing with optional values which are
		// neither pointers nor slices; for example, a int32 field marked
		// "optional" in its parent struct.
		//
		// We need to find zero values, which should be represented as nulls
		// in the parquet column. In order to minimize the calls to writeRows
		// and maximize throughput, we use the nullIndex and nonNullIndex
		// functions, which are type-specific implementations of the algorithm.
		//
		// Sections of the input that are contiguous nulls or non-nulls can be
		// sent to a single call to writeRows to be written to the underlying
		// buffer since they share the same definition level.
		//
		// This optimization is defeated by inputs alternating null and non-null
		// sequences of single values, we do not expect this condition to be a
		// common case.
		for i := 0; i < rows.Len(); {
			j := 0
			x := i / 64
			y := i % 64

			if y != 0 {
				if b := nulls.bits[x] >> uint(y); b == 0 {
					x++
					y = 0
				} else {
					y += bits.TrailingZeros64(b)
					goto writeNulls
				}
			}

			for x < len(nulls.bits) && nulls.bits[x] == 0 {
				x++
			}

			if x < len(nulls.bits) {
				y = bits.TrailingZeros64(nulls.bits[x]) % 64
			}

		writeNulls:
			if j = x*64 + y; j > rows.Len() {
				j = rows.Len()
			}

			if i < j {
				if err := writeRows(columns, rows.Slice(i, j), nullLevels); err != nil {
					return err
				}
				i = j
			}

			if y != 0 {
				if b := nulls.bits[x] >> uint(y); b == (1<<uint64(y))-1 {
					x++
					y = 0
				} else {
					y += bits.TrailingZeros64(^b)
					goto writeNonNulls
				}
			}

			for x < len(nulls.bits) && nulls.bits[x] == ^uint64(0) {
				x++
			}

			if x < len(nulls.bits) {
				y = bits.TrailingZeros64(^nulls.bits[x]) % 64
			}

		writeNonNulls:
			if j = x*64 + y; j > rows.Len() {
				j = rows.Len()
			}

			if i < j {
				if err := writeRows(columns, rows.Slice(i, j), levels); err != nil {
					return err
				}
				i = j
			}
		}

		return nil
	}
}

func writeRowsFuncOfPointer(t reflect.Type, schema *Schema, path columnPath) writeRowsFunc {
	elemType := t.Elem()
	elemSize := uintptr(elemType.Size())
	writeRows := writeRowsFuncOf(elemType, schema, path)

	if len(path) == 0 {
		// This code path is taken when generating a writeRowsFunc for a pointer
		// type. In this case, we do not need to increase the definition level
		// since we are not deailng with an optional field but a pointer to the
		// row type.
		return func(columns []ColumnBuffer, rows sparse.Array, levels columnLevels) error {
			if rows.Len() == 0 {
				return writeRows(columns, rows, levels)
			}

			for i := 0; i < rows.Len(); i++ {
				p := *(*unsafe.Pointer)(rows.Index(i))
				a := sparse.Array{}
				if p != nil {
					a = makeArray(p, 1, elemSize)
				}
				if err := writeRows(columns, a, levels); err != nil {
					return err
				}
			}

			return nil
		}
	}

	return func(columns []ColumnBuffer, rows sparse.Array, levels columnLevels) error {
		if rows.Len() == 0 {
			return writeRows(columns, rows, levels)
		}

		for i := 0; i < rows.Len(); i++ {
			p := *(*unsafe.Pointer)(rows.Index(i))
			a := sparse.Array{}
			elemLevels := levels
			if p != nil {
				a = makeArray(p, 1, elemSize)
				elemLevels.definitionLevel++
			}
			if err := writeRows(columns, a, elemLevels); err != nil {
				return err
			}
		}

		return nil
	}
}

func writeRowsFuncOfSlice(t reflect.Type, schema *Schema, path columnPath) writeRowsFunc {
	elemType := t.Elem()
	elemSize := uintptr(elemType.Size())
	writeRows := writeRowsFuncOf(elemType, schema, path)

	// When the element is a pointer type, the writeRows function will be an
	// instance returned by writeRowsFuncOfPointer, which handles incrementing
	// the definition level if the pointer value is not nil.
	definitionLevelIncrement := byte(0)
	if elemType.Kind() != reflect.Ptr {
		definitionLevelIncrement = 1
	}

	return func(columns []ColumnBuffer, rows sparse.Array, levels columnLevels) error {
		if rows.Len() == 0 {
			return writeRows(columns, rows, levels)
		}

		levels.repetitionDepth++

		for i := 0; i < rows.Len(); i++ {
			p := (*sliceHeader)(rows.Index(i))
			a := makeArray(p.base, p.len, elemSize)
			b := sparse.Array{}

			elemLevels := levels
			if a.Len() > 0 {
				b = a.Slice(0, 1)
				elemLevels.definitionLevel += definitionLevelIncrement
			}

			if err := writeRows(columns, b, elemLevels); err != nil {
				return err
			}

			if a.Len() > 1 {
				elemLevels.repetitionLevel = elemLevels.repetitionDepth

				if err := writeRows(columns, a.Slice(1, a.Len()), elemLevels); err != nil {
					return err
				}
			}
		}

		return nil
	}
}

func writeRowsFuncOfStruct(t reflect.Type, schema *Schema, path columnPath) writeRowsFunc {
	type column struct {
		offset    uintptr
		writeRows writeRowsFunc
	}

	fields := structFieldsOf(t)
	columns := make([]column, len(fields))

	for i, f := range fields {
		optional := false
		columnPath := path.append(f.Name)
		forEachStructTagOption(f, func(_ reflect.Type, option, _ string) {
			switch option {
			case "list":
				columnPath = columnPath.append("list", "element")
			case "optional":
				optional = true
			}
		})

		writeRows := writeRowsFuncOf(f.Type, schema, columnPath)
		if optional {
			switch f.Type.Kind() {
			case reflect.Pointer, reflect.Slice:
			default:
				writeRows = writeRowsFuncOfOptional(f.Type, schema, columnPath, writeRows)
			}
		}

		columns[i] = column{
			offset:    f.Offset,
			writeRows: writeRows,
		}
	}

	return func(buffers []ColumnBuffer, rows sparse.Array, levels columnLevels) error {
		if rows.Len() == 0 {
			for _, column := range columns {
				if err := column.writeRows(buffers, rows, levels); err != nil {
					return err
				}
			}
		} else {
			for _, column := range columns {
				if err := column.writeRows(buffers, rows.Offset(column.offset), levels); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func writeRowsFuncOfMap(t reflect.Type, schema *Schema, path columnPath) writeRowsFunc {
	keyPath := path.append("key_value", "key")
	keyType := t.Key()
	keySize := uintptr(keyType.Size())
	writeKeys := writeRowsFuncOf(keyType, schema, keyPath)

	valuePath := path.append("key_value", "value")
	valueType := t.Elem()
	valueSize := uintptr(valueType.Size())
	writeValues := writeRowsFuncOf(valueType, schema, valuePath)

	writeKeyValues := func(columns []ColumnBuffer, keys, values sparse.Array, levels columnLevels) error {
		if err := writeKeys(columns, keys, levels); err != nil {
			return err
		}
		if err := writeValues(columns, values, levels); err != nil {
			return err
		}
		return nil
	}

	return func(columns []ColumnBuffer, rows sparse.Array, levels columnLevels) error {
		if rows.Len() == 0 {
			return writeKeyValues(columns, rows, rows, levels)
		}

		levels.repetitionDepth++
		mapKey := reflect.New(keyType).Elem()
		mapValue := reflect.New(valueType).Elem()

		for i := 0; i < rows.Len(); i++ {
			m := reflect.NewAt(t, rows.Index(i)).Elem()

			if m.Len() == 0 {
				empty := sparse.Array{}
				if err := writeKeyValues(columns, empty, empty, levels); err != nil {
					return err
				}
			} else {
				elemLevels := levels
				elemLevels.definitionLevel++

				for it := m.MapRange(); it.Next(); {
					mapKey.SetIterKey(it)
					mapValue.SetIterValue(it)

					k := makeArray(unsafecast.PointerOfValue(mapKey), 1, keySize)
					v := makeArray(unsafecast.PointerOfValue(mapValue), 1, valueSize)

					if err := writeKeyValues(columns, k, v, elemLevels); err != nil {
						return err
					}

					elemLevels.repetitionLevel = elemLevels.repetitionDepth
				}
			}
		}

		return nil
	}
}
