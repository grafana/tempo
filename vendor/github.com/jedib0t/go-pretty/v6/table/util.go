package table

import (
	"reflect"
)

// AutoIndexColumnID returns a unique Column ID/Name for the given Column Number.
// The functionality is similar to what you get in an Excel spreadsheet w.r.t.
// the Column ID/Name.
func AutoIndexColumnID(colIdx int) string {
	charIdx := colIdx % 26
	out := string(rune(65 + charIdx))
	colIdx = colIdx / 26
	if colIdx > 0 {
		return AutoIndexColumnID(colIdx-1) + out
	}
	return out
}

// WidthEnforcer is a function that helps enforce a width condition on a string.
type WidthEnforcer func(col string, maxLen int) string

// widthEnforcerNone returns the input string as is without any modifications.
func widthEnforcerNone(col string, _ int) string {
	return col
}

// isNumber returns true if the argument is a numeric type; false otherwise.
func isNumber(x interface{}) bool {
	if x == nil {
		return false
	}

	switch reflect.TypeOf(x).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

type mergedColumnIndices map[int]map[int]bool

func (m mergedColumnIndices) mergedLength(colIdx int, maxColumnLengths []int) int {
	mergedLength := maxColumnLengths[colIdx]
	for otherColIdx := range m[colIdx] {
		mergedLength += maxColumnLengths[otherColIdx]
	}
	return mergedLength
}

func (m mergedColumnIndices) len(colIdx int) int {
	return len(m[colIdx]) + 1
}

func (m mergedColumnIndices) safeAppend(colIdx, otherColIdx int) {
	// map
	if m[colIdx] == nil {
		m[colIdx] = make(map[int]bool)
	}
	m[colIdx][otherColIdx] = true

	// reverse map
	if m[otherColIdx] == nil {
		m[otherColIdx] = make(map[int]bool)
	}
	m[otherColIdx][colIdx] = true
}

func objAsSlice(in interface{}) []interface{} {
	var out []interface{}
	if in != nil {
		// dereference pointers
		val := reflect.ValueOf(in)
		if val.Kind() == reflect.Ptr && !val.IsNil() {
			in = val.Elem().Interface()
		}

		if objIsSlice(in) {
			v := reflect.ValueOf(in)
			for i := 0; i < v.Len(); i++ {
				// dereference pointers
				v2 := v.Index(i)
				if v2.Kind() == reflect.Ptr && !v2.IsNil() {
					v2 = reflect.ValueOf(v2.Elem().Interface())
				}

				out = append(out, v2.Interface())
			}
		}
	}

	// remove trailing nil pointers
	tailIdx := len(out)
	for i := len(out) - 1; i >= 0; i-- {
		val := reflect.ValueOf(out[i])
		if val.Kind() != reflect.Ptr || !val.IsNil() {
			break
		}
		tailIdx = i
	}
	return out[:tailIdx]
}

func objIsSlice(in interface{}) bool {
	if in == nil {
		return false
	}
	k := reflect.TypeOf(in).Kind()
	return k == reflect.Slice || k == reflect.Array
}
