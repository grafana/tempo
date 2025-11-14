package parquet

import (
	"encoding/json"
	"fmt"
	"hash/maphash"
	"math/bits"
	"reflect"
	"slices"
	"sync"
	"time"
	"unsafe"

	"github.com/parquet-go/parquet-go/deprecated"
	"github.com/parquet-go/parquet-go/sparse"
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
func writeRowsFuncOf(t reflect.Type, schema *Schema, path columnPath, tagReplacements []StructTagOption) writeRowsFunc {
	if leaf, exists := schema.Lookup(path...); exists && leaf.Node.Type().LogicalType() != nil && leaf.Node.Type().LogicalType().Json != nil {
		return writeRowsFuncOfJSON(t, schema, path)
	}

	switch t {
	case reflect.TypeOf(deprecated.Int96{}):
		return writeRowsFuncOfRequired(t, schema, path)
	case reflect.TypeOf(time.Time{}):
		return writeRowsFuncOfTime(t, schema, path, tagReplacements)
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
			return writeRowsFuncOfSlice(t, schema, path, tagReplacements)
		}

	case reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return writeRowsFuncOfArray(t, schema, path)
		}

	case reflect.Pointer:
		return writeRowsFuncOfPointer(t, schema, path, tagReplacements)

	case reflect.Struct:
		return writeRowsFuncOfStruct(t, schema, path, tagReplacements)

	case reflect.Map:
		return writeRowsFuncOfMap(t, schema, path, tagReplacements)
	}

	panic("cannot convert Go values of type " + typeNameOf(t) + " to parquet value")
}

func writeRowsFuncOfRequired(t reflect.Type, schema *Schema, path columnPath) writeRowsFunc {
	column := schema.lazyLoadState().mapping.lookup(path)
	columnIndex := column.columnIndex
	if columnIndex < 0 {
		panic("parquet: column not found: " + path.String())
	}
	return func(columns []ColumnBuffer, rows sparse.Array, levels columnLevels) error {
		columns[columnIndex].writeValues(rows, levels)
		return nil
	}
}

func writeRowsFuncOfOptional(t reflect.Type, schema *Schema, path columnPath, writeRows writeRowsFunc) writeRowsFunc {
	if t.Kind() == reflect.Slice && t.Elem().Kind() != reflect.Uint8 { // assume nested list; []byte is scalar
		return func(columns []ColumnBuffer, rows sparse.Array, levels columnLevels) error {
			if rows.Len() == 0 {
				return writeRows(columns, rows, levels)
			}
			levels.definitionLevel++
			return writeRows(columns, rows, levels)
		}
	}
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

func writeRowsFuncOfArray(t reflect.Type, schema *Schema, path columnPath) writeRowsFunc {
	column := schema.lazyLoadState().mapping.lookup(path)
	arrayLen := t.Len()
	columnLen := column.node.Type().Length()
	if arrayLen != columnLen {
		panic(fmt.Sprintf("cannot convert Go values of type "+typeNameOf(t)+" to FIXED_LEN_BYTE_ARRAY(%d)", columnLen))
	}
	return writeRowsFuncOfRequired(t, schema, path)
}

func writeRowsFuncOfPointer(t reflect.Type, schema *Schema, path columnPath, tagReplacements []StructTagOption) writeRowsFunc {
	elemType := t.Elem()
	elemSize := uintptr(elemType.Size())
	writeRows := writeRowsFuncOf(elemType, schema, path, tagReplacements)

	if len(path) == 0 {
		// This code path is taken when generating a writeRowsFunc for a pointer
		// type. In this case, we do not need to increase the definition level
		// since we are not deailng with an optional field but a pointer to the
		// row type.
		return func(columns []ColumnBuffer, rows sparse.Array, levels columnLevels) error {
			if rows.Len() == 0 {
				return writeRows(columns, rows, levels)
			}

			for i := range rows.Len() {
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

		for i := range rows.Len() {
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

func writeRowsFuncOfSlice(t reflect.Type, schema *Schema, path columnPath, tagReplacements []StructTagOption) writeRowsFunc {
	elemType := t.Elem()
	elemSize := uintptr(elemType.Size())
	writeRows := writeRowsFuncOf(elemType, schema, path, tagReplacements)

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

		for i := range rows.Len() {
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

func writeRowsFuncOfStruct(t reflect.Type, schema *Schema, path columnPath, tagReplacements []StructTagOption) writeRowsFunc {
	type column struct {
		offset    uintptr
		writeRows writeRowsFunc
	}

	fields := structFieldsOf(path, t, tagReplacements)
	columns := make([]column, len(fields))

	for i, f := range fields {
		list, optional := false, false
		columnPath := path.append(f.Name)
		forEachStructTagOption(f, func(_ reflect.Type, option, _ string) {
			switch option {
			case "list":
				list = true
				columnPath = columnPath.append("list", "element")
			case "optional":
				optional = true
			}
		})

		writeRows := writeRowsFuncOf(f.Type, schema, columnPath, tagReplacements)
		if optional {
			kind := f.Type.Kind()
			switch {
			case kind == reflect.Pointer:
			case kind == reflect.Slice && !list && f.Type.Elem().Kind() != reflect.Uint8:
				// For slices other than []byte, optional applies to the element, not the list
			case f.Type == reflect.TypeOf(time.Time{}):
				// time.Time is a struct but has IsZero() method, so it needs special handling
				// Don't use writeRowsFuncOfOptional which relies on bitmap batching
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

var (
	mapStringStringType = reflect.TypeOf((map[string]string)(nil))
	mapStringAnyType    = reflect.TypeOf((map[string]any)(nil))
)

// writeRowsFuncOfMapToGroup handles writing a Go map to a Parquet GROUP schema
// (as opposed to a MAP logical type). This allows map[string]T to be written
// to schemas with named optional fields.
func writeRowsFuncOfMapToGroup(t reflect.Type, schema *Schema, path columnPath, groupNode Node, tagReplacements []StructTagOption) writeRowsFunc {
	if t.Key().Kind() != reflect.String {
		panic("map keys must be strings when writing to GROUP schema")
	}

	type fieldWriter struct {
		fieldName  string
		fieldPath  columnPath
		writeRows  writeRowsFunc // Writes null/empty value
		writeValue func([]ColumnBuffer, reflect.Value, columnLevels) error
	}

	// Get all fields from the GROUP and create write functions for each
	fields := groupNode.Fields()
	writers := make([]fieldWriter, len(fields))
	valueType := t.Elem()
	valueSize := uintptr(valueType.Size())

	// Check if the value type is interface{} - if so, we need runtime type handling
	// We split into two separate loops to avoid branching inside the loop
	if valueType.Kind() == reflect.Interface {
		// Interface{} path - need runtime type handling
		for i, field := range fields {
			fieldPath := path.append(field.Name())
			fieldNode := findByPath(schema, fieldPath)

			// For interface{} types, create a write function based on the SCHEMA type
			// This will be used when writing null values
			writeNull := writeRowsFuncOfSchemaNode(fieldNode, schema, fieldPath, field)

			// Capture variables for the closure
			writeValue := func(columns []ColumnBuffer, mapValue reflect.Value, levels columnLevels) error {
				actualValue := mapValue
				actualValueKind := actualValue.Kind()
				if actualValueKind == reflect.Interface && !actualValue.IsNil() {
					actualValue = actualValue.Elem()
					actualValueKind = actualValue.Kind()
				}
				if !actualValue.IsValid() || (actualValueKind == reflect.Pointer && actualValue.IsNil()) {
					// Nil interface or nil pointer - write null
					return writeNull(columns, sparse.Array{}, levels)
				}
				if actualValueKind == reflect.Pointer {
					actualValue = actualValue.Elem()
				}
				return writeInterfaceValue(columns, actualValue, field, schema, fieldPath, levels, tagReplacements)
			}

			writers[i] = fieldWriter{
				fieldName:  field.Name(),
				fieldPath:  fieldPath,
				writeRows:  writeNull,
				writeValue: writeValue,
			}
		}
	} else {
		// Concrete type path - can pre-create write functions
		for i, field := range fields {
			fieldPath := path.append(field.Name())

			// For concrete types, we can pre-create the write function
			writeRows := writeRowsFuncOf(valueType, schema, fieldPath, tagReplacements)

			// Check if the field is optional
			if field.Optional() {
				writeRows = writeRowsFuncOfOptional(valueType, schema, fieldPath, writeRows)
			}

			// Both null and value use the same function for concrete types
			writeValue := func(columns []ColumnBuffer, mapValue reflect.Value, levels columnLevels) error {
				valueArray := makeArray(reflectValuePointer(mapValue), 1, valueSize)
				return writeRows(columns, valueArray, levels)
			}

			writers[i] = fieldWriter{
				fieldName:  field.Name(),
				fieldPath:  fieldPath,
				writeRows:  writeRows,
				writeValue: writeValue,
			}
		}
	}

	// We make sepcial cases for the common types to avoid paying the cost of
	// reflection in calls like MapIndex which force the returned value to be
	// allocated on the heap.
	var writeMaps writeRowsFunc
	switch {
	case t.ConvertibleTo(mapStringStringType):
		writeMaps = func(columns []ColumnBuffer, rows sparse.Array, levels columnLevels) error {
			buffer, _ := stringArrayPool.Get().(*stringArray)
			if buffer == nil {
				buffer = new(stringArray)
			}
			numRows := rows.Len()
			numValues := len(writers) * numRows
			buffer.values = slices.Grow(buffer.values, numValues)[:numValues]
			defer stringArrayPool.Put(buffer)

			for i := range numRows {
				m := *(*map[string]string)(reflect.NewAt(t, rows.Index(i)).UnsafePointer())

				for j := range writers {
					buffer.values[j*numRows+i] = m[writers[j].fieldName]
				}
			}

			for j := range writers {
				a := sparse.MakeStringArray(buffer.values[j*numRows : (j+1)*numRows])
				if err := writers[j].writeRows(columns, a.UnsafeArray(), levels); err != nil {
					return err
				}
			}

			return nil
		}

	case t.ConvertibleTo(mapStringAnyType):
		writeMaps = func(columns []ColumnBuffer, rows sparse.Array, levels columnLevels) error {
			for i := range rows.Len() {
				m := *(*map[string]any)(reflect.NewAt(t, rows.Index(i)).UnsafePointer())

				for j := range writers {
					w := &writers[j]
					v, ok := m[w.fieldName]

					var err error
					if !ok {
						err = w.writeRows(columns, sparse.Array{}, levels)
					} else {
						err = w.writeValue(columns, reflect.ValueOf(v), levels)
					}
					if err != nil {
						return err
					}
				}
			}
			return nil
		}

	default:
		writeMaps = func(columns []ColumnBuffer, rows sparse.Array, levels columnLevels) error {
			for i := range rows.Len() {
				m := reflect.NewAt(t, rows.Index(i)).Elem()

				for j := range writers {
					w := &writers[j]
					keyValue := reflect.ValueOf(&w.fieldName).Elem()
					mapValue := m.MapIndex(keyValue)

					var err error
					if !mapValue.IsValid() {
						err = w.writeRows(columns, sparse.Array{}, levels)
					} else {
						err = w.writeValue(columns, mapValue, levels)
					}
					if err != nil {
						return err
					}
				}
			}
			return nil
		}
	}

	return func(columns []ColumnBuffer, rows sparse.Array, levels columnLevels) error {
		if rows.Len() == 0 {
			// Write empty values for all fields
			for _, w := range writers {
				if err := w.writeRows(columns, sparse.Array{}, levels); err != nil {
					return err
				}
			}
			return nil
		}
		return writeMaps(columns, rows, levels)
	}
}

type stringArray struct{ values []string }

var stringArrayPool sync.Pool // *stringArray

// writeRowsFuncOfSchemaNode creates a write function based on the schema node type
// rather than a Go type. This is used for interface{} values where we need to write
// nulls based on the schema structure.
func writeRowsFuncOfSchemaNode(node Node, schema *Schema, path columnPath, field Node) writeRowsFunc {
	if node == nil {
		panic(fmt.Sprintf("schema node not found at path: %v", path))
	}

	// Check if this is a leaf or a group
	if len(node.Fields()) == 0 {
		// It's a leaf node - create a simple write function
		return func(columns []ColumnBuffer, rows sparse.Array, levels columnLevels) error {
			leaf, ok := schema.Lookup(path...)
			if !ok {
				return fmt.Errorf("leaf not found: %v", path)
			}

			// For optional fields with no data, we need to write at the parent's definition level
			// For non-optional or when there's data, increment the definition level
			if rows.Len() == 0 && field.Optional() {
				// Write null - don't increment definition level
				columns[leaf.ColumnIndex].writeValues(rows, levels)
			} else if field.Optional() {
				// Write value - increment definition level
				levels.definitionLevel++
				columns[leaf.ColumnIndex].writeValues(rows, levels)
				levels.definitionLevel--
			} else {
				// Required field
				columns[leaf.ColumnIndex].writeValues(rows, levels)
			}
			return nil
		}
	}

	// It's a group - recursively create write functions for all children
	type childWriter struct {
		writeRows writeRowsFunc
	}

	fields := node.Fields()
	children := make([]childWriter, len(fields))

	for i, childField := range fields {
		childPath := path.append(childField.Name())
		childNode := findByPath(schema, childPath)
		children[i] = childWriter{
			writeRows: writeRowsFuncOfSchemaNode(childNode, schema, childPath, childField),
		}
	}

	return func(columns []ColumnBuffer, rows sparse.Array, levels columnLevels) error {
		// For groups, we need to write to all child columns
		for _, child := range children {
			if err := child.writeRows(columns, rows, levels); err != nil {
				return err
			}
		}
		return nil
	}
}

// writeInterfaceValue writes an interface{} value at runtime, determining the appropriate
// write function based on the actual type.
func writeInterfaceValue(columns []ColumnBuffer, value reflect.Value, field Node, schema *Schema, path columnPath, levels columnLevels, tagReplacements []StructTagOption) error {
	actualType := value.Type()
	schemaCache := schema.lazyLoadCache()

	hash := maphash.Hash{}
	hash.SetSeed(schemaCache.hashSeed)

	for _, name := range path {
		hash.WriteString(name)
		hash.WriteByte(0)
	}

	writeRowsKey := writeRowsCacheKey{
		gotype: actualType,
		column: hash.Sum64(),
	}

	writeRows := schemaCache.writeRows.load(writeRowsKey, func() writeRowsFunc {
		return writeRowsFuncOf(actualType, schema, path, tagReplacements)
	})

	// Handle optional fields
	if field.Optional() {
		// For optional fields with actual values, we need to increment definition level
		levels.definitionLevel++
		defer func() { levels.definitionLevel-- }()
	}

	valueArray := makeArray(reflectValuePointer(value), 1, actualType.Size())
	return writeRows(columns, valueArray, levels)
}

func writeRowsFuncOfMap(t reflect.Type, schema *Schema, path columnPath, tagReplacements []StructTagOption) writeRowsFunc {
	// Check if the schema at this path is a MAP or a GROUP.
	node := findByPath(schema, path)
	if node != nil && !isMap(node) {
		// The schema is a GROUP (not a MAP), so we need to handle it differently.
		// Instead of using key_value structure, we iterate through the GROUP's fields
		// and look up corresponding map keys.
		return writeRowsFuncOfMapToGroup(t, schema, path, node, tagReplacements)
	}

	// Standard MAP logical type handling
	keyPath := path.append("key_value", "key")
	keyType := t.Key()
	writeKeys := writeRowsFuncOf(keyType, schema, keyPath, tagReplacements)

	valuePath := path.append("key_value", "value")
	valueType := t.Elem()
	writeValues := writeRowsFuncOf(valueType, schema, valuePath, tagReplacements)

	return func(columns []ColumnBuffer, rows sparse.Array, levels columnLevels) error {
		if rows.Len() == 0 {
			if err := writeKeys(columns, rows, levels); err != nil {
				return err
			}
			if err := writeValues(columns, rows, levels); err != nil {
				return err
			}
			return nil
		}

		levels.repetitionDepth++
		makeMap := makeMapFuncOf(t)

		for i := range rows.Len() {
			m := reflect.NewAt(t, rows.Index(i)).Elem()
			n := m.Len()

			if n == 0 {
				empty := sparse.Array{}
				if err := writeKeys(columns, empty, levels); err != nil {
					return err
				}
				if err := writeValues(columns, empty, levels); err != nil {
					return err
				}
				continue
			}

			elemLevels := levels
			elemLevels.definitionLevel++

			keys, values := makeMap(m).entries()
			if err := writeKeys(columns, keys.Slice(0, 1), elemLevels); err != nil {
				return err
			}
			if err := writeValues(columns, values.Slice(0, 1), elemLevels); err != nil {
				return err
			}
			if n > 1 {
				elemLevels.repetitionLevel = elemLevels.repetitionDepth
				if err := writeKeys(columns, keys.Slice(1, n), elemLevels); err != nil {
					return err
				}
				if err := writeValues(columns, values.Slice(1, n), elemLevels); err != nil {
					return err
				}
			}
		}

		return nil
	}
}

func writeRowsFuncOfJSON(t reflect.Type, schema *Schema, path columnPath) writeRowsFunc {
	// If this is a string or a byte array write directly.
	switch t.Kind() {
	case reflect.String:
		return writeRowsFuncOfRequired(t, schema, path)
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return writeRowsFuncOfRequired(t, schema, path)
		}
	}

	// Otherwise handle with a json.Marshal
	asStrT := reflect.TypeOf(string(""))
	writer := writeRowsFuncOfRequired(asStrT, schema, path)

	return func(columns []ColumnBuffer, rows sparse.Array, levels columnLevels) error {
		if rows.Len() == 0 {
			return writer(columns, rows, levels)
		}
		for i := range rows.Len() {
			val := reflect.NewAt(t, rows.Index(i))
			asI := val.Interface()

			b, err := json.Marshal(asI)
			if err != nil {
				return err
			}

			asStr := string(b)
			a := sparse.MakeStringArray([]string{asStr})
			if err := writer(columns, a.UnsafeArray(), levels); err != nil {
				return err
			}
		}
		return nil
	}
}

func writeRowsFuncOfTime(_ reflect.Type, schema *Schema, path columnPath, tagReplacements []StructTagOption) writeRowsFunc {
	t := reflect.TypeOf(int64(0))
	elemSize := uintptr(t.Size())
	writeRows := writeRowsFuncOf(t, schema, path, tagReplacements)

	col, _ := schema.Lookup(path...)
	unit := Nanosecond.TimeUnit()
	lt := col.Node.Type().LogicalType()
	if lt != nil && lt.Timestamp != nil {
		unit = lt.Timestamp.Unit
	}

	// Check if the column is optional
	isOptional := col.Node.Optional()

	return func(columns []ColumnBuffer, rows sparse.Array, levels columnLevels) error {
		if rows.Len() == 0 {
			return writeRows(columns, rows, levels)
		}

		// If we're optional and the current definition level is already > 0,
		// then we're in a pointer/nested context where writeRowsFuncOfPointer already handles optionality.
		// Don't double-handle it here. For simple optional fields, definitionLevel starts at 0.
		alreadyHandled := isOptional && levels.definitionLevel > 0

		times := rows.TimeArray()
		for i := range times.Len() {
			t := times.Index(i)

			// For optional fields, check if the value is zero (unless already handled by pointer wrapper)
			elemLevels := levels
			if isOptional && !alreadyHandled && t.IsZero() {
				// Write as NULL (don't increment definition level)
				empty := sparse.Array{}
				if err := writeRows(columns, empty, elemLevels); err != nil {
					return err
				}
				continue
			}

			// For optional non-zero values, increment definition level (unless already handled)
			if isOptional && !alreadyHandled {
				elemLevels.definitionLevel++
			}

			var val int64
			switch {
			case unit.Millis != nil:
				val = t.UnixMilli()
			case unit.Micros != nil:
				val = t.UnixMicro()
			default:
				val = t.UnixNano()
			}

			a := makeArray(reflectValueData(reflect.ValueOf(val)), 1, elemSize)
			if err := writeRows(columns, a, elemLevels); err != nil {
				return err
			}
		}

		return nil
	}
}
