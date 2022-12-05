package parquet

import (
	"encoding/binary"

	"github.com/segmentio/parquet-go/deprecated"
)

// CompareDescending constructs a comparison function which inverses the order
// of values.
//
//go:noinline
func CompareDescending(cmp func(Value, Value) int) func(Value, Value) int {
	return func(a, b Value) int { return -cmp(a, b) }
}

// CompareNullsFirst constructs a comparison function which assumes that null
// values are smaller than all other values.
//
//go:noinline
func CompareNullsFirst(cmp func(Value, Value) int) func(Value, Value) int {
	return func(a, b Value) int {
		switch {
		case a.IsNull():
			if b.IsNull() {
				return 0
			}
			return -1
		case b.IsNull():
			return +1
		default:
			return cmp(a, b)
		}
	}
}

// CompareNullsLast constructs a comparison function which assumes that null
// values are greater than all other values.
//
//go:noinline
func CompareNullsLast(cmp func(Value, Value) int) func(Value, Value) int {
	return func(a, b Value) int {
		switch {
		case a.IsNull():
			if b.IsNull() {
				return 0
			}
			return +1
		case b.IsNull():
			return -1
		default:
			return cmp(a, b)
		}
	}
}

func compareBool(v1, v2 bool) int {
	switch {
	case !v1 && v2:
		return -1
	case v1 && !v2:
		return +1
	default:
		return 0
	}
}

func compareInt32(v1, v2 int32) int {
	switch {
	case v1 < v2:
		return -1
	case v1 > v2:
		return +1
	default:
		return 0
	}
}

func compareInt64(v1, v2 int64) int {
	switch {
	case v1 < v2:
		return -1
	case v1 > v2:
		return +1
	default:
		return 0
	}
}

func compareInt96(v1, v2 deprecated.Int96) int {
	switch {
	case v1.Less(v2):
		return -1
	case v2.Less(v1):
		return +1
	default:
		return 0
	}
}

func compareFloat32(v1, v2 float32) int {
	switch {
	case v1 < v2:
		return -1
	case v1 > v2:
		return +1
	default:
		return 0
	}
}

func compareFloat64(v1, v2 float64) int {
	switch {
	case v1 < v2:
		return -1
	case v1 > v2:
		return +1
	default:
		return 0
	}
}

func compareUint32(v1, v2 uint32) int {
	switch {
	case v1 < v2:
		return -1
	case v1 > v2:
		return +1
	default:
		return 0
	}
}

func compareUint64(v1, v2 uint64) int {
	switch {
	case v1 < v2:
		return -1
	case v1 > v2:
		return +1
	default:
		return 0
	}
}

func compareBE128(v1, v2 *[16]byte) int {
	x := binary.BigEndian.Uint64(v1[:8])
	y := binary.BigEndian.Uint64(v2[:8])
	switch {
	case x < y:
		return -1
	case x > y:
		return +1
	}
	x = binary.BigEndian.Uint64(v1[8:])
	y = binary.BigEndian.Uint64(v2[8:])
	switch {
	case x < y:
		return -1
	case x > y:
		return +1
	default:
		return 0
	}
}

func lessBE128(v1, v2 *[16]byte) bool {
	x := binary.BigEndian.Uint64(v1[:8])
	y := binary.BigEndian.Uint64(v2[:8])
	switch {
	case x < y:
		return true
	case x > y:
		return false
	}
	x = binary.BigEndian.Uint64(v1[8:])
	y = binary.BigEndian.Uint64(v2[8:])
	return x < y
}

func compareRowsFuncOf(schema *Schema, sortingColumns []SortingColumn) func(Row, Row) int {
	compareFuncs := make([]func(Row, Row) int, 0, len(sortingColumns))
	direct := true

	for _, column := range schema.Columns() {
		leaf, _ := schema.Lookup(column...)
		if leaf.MaxRepetitionLevel > 0 {
			direct = false
		}

		for _, sortingColumn := range sortingColumns {
			path1 := columnPath(column)
			path2 := columnPath(sortingColumn.Path())

			if path1.equal(path2) {
				descending := sortingColumn.Descending()
				optional := leaf.MaxDefinitionLevel > 0
				sortFunc := (func(Row, Row) int)(nil)

				if direct && !optional {
					// This is an optimization for the common case where rows
					// are sorted by non-optional, non-repeated columns.
					//
					// The sort function can make the assumption that it will
					// find the column value at the current column index, and
					// does not need to scan the rows looking for values with
					// a matching column index.
					//
					// A second optimization consists in passing the column type
					// directly to the sort function instead of an intermediary
					// closure, which removes an indirection layer and improves
					// throughput by ~20% in BenchmarkSortRowBuffer.
					typ := leaf.Node.Type()
					if descending {
						sortFunc = compareRowsFuncOfIndexDescending(leaf.ColumnIndex, typ)
					} else {
						sortFunc = compareRowsFuncOfIndexAscending(leaf.ColumnIndex, typ)
					}
				} else {
					compare := leaf.Node.Type().Compare

					if descending {
						compare = CompareDescending(compare)
					}

					if optional {
						if sortingColumn.NullsFirst() {
							compare = CompareNullsFirst(compare)
						} else {
							compare = CompareNullsLast(compare)
						}
					}

					sortFunc = compareRowsFuncOfScan(leaf.ColumnIndex, compare)
				}

				compareFuncs = append(compareFuncs, sortFunc)
			}
		}
	}

	// For the common case where rows are sorted by a single column, we can skip
	// looping over the list of sort functions.
	switch len(compareFuncs) {
	case 0:
		return compareRowsUnordered
	case 1:
		return compareFuncs[0]
	default:
		return compareRowsFuncOfColumns(compareFuncs)
	}
}

func compareRowsUnordered(Row, Row) int { return 0 }

//go:noinline
func compareRowsFuncOfColumns(compareFuncs []func(Row, Row) int) func(Row, Row) int {
	return func(row1, row2 Row) int {
		for _, compare := range compareFuncs {
			if cmp := compare(row1, row2); cmp != 0 {
				return cmp
			}
		}
		return 0
	}
}

//go:noinline
func compareRowsFuncOfIndexAscending(columnIndex int, typ Type) func(Row, Row) int {
	return func(row1, row2 Row) int { return typ.Compare(row1[columnIndex], row2[columnIndex]) }
}

//go:noinline
func compareRowsFuncOfIndexDescending(columnIndex int, typ Type) func(Row, Row) int {
	return func(row1, row2 Row) int { return -typ.Compare(row1[columnIndex], row2[columnIndex]) }
}

//go:noinline
func compareRowsFuncOfScan(columnIndex int, compare func(Value, Value) int) func(Row, Row) int {
	columnIndexOfValues := ^int16(columnIndex)
	return func(row1, row2 Row) int {
		i1 := 0
		i2 := 0

		for {
			for i1 < len(row1) && row1[i1].columnIndex != columnIndexOfValues {
				i1++
			}

			for i2 < len(row2) && row2[i2].columnIndex != columnIndexOfValues {
				i2++
			}

			end1 := i1 == len(row1)
			end2 := i2 == len(row2)

			if end1 && end2 {
				return 0
			} else if end1 {
				return -1
			} else if end2 {
				return +1
			} else if cmp := compare(row1[i1], row2[i2]); cmp != 0 {
				return cmp
			}

			i1++
			i2++
		}
	}
}
