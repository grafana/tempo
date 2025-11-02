package parquet

import (
	"bytes"
	"io"

	"github.com/parquet-go/bitpack/unsafecast"
	"github.com/parquet-go/parquet-go/encoding/plain"
	"github.com/parquet-go/parquet-go/sparse"
)

type byteArrayColumnBuffer struct {
	byteArrayPage
	lengths []uint32
	scratch []byte
}

func newByteArrayColumnBuffer(typ Type, columnIndex int16, numValues int32) *byteArrayColumnBuffer {
	return &byteArrayColumnBuffer{
		byteArrayPage: byteArrayPage{
			typ:         typ,
			values:      make([]byte, 0, typ.EstimateSize(int(numValues))),
			offsets:     make([]uint32, 0, numValues+1),
			columnIndex: ^columnIndex,
		},
		lengths: make([]uint32, 0, numValues),
	}
}

func (col *byteArrayColumnBuffer) Clone() ColumnBuffer {
	return &byteArrayColumnBuffer{
		byteArrayPage: byteArrayPage{
			typ:         col.typ,
			values:      col.cloneValues(),
			offsets:     col.cloneOffsets(),
			columnIndex: col.columnIndex,
		},
		lengths: col.cloneLengths(),
	}
}

func (col *byteArrayColumnBuffer) cloneLengths() []uint32 {
	lengths := make([]uint32, len(col.lengths))
	copy(lengths, col.lengths)
	return lengths
}

func (col *byteArrayColumnBuffer) ColumnIndex() (ColumnIndex, error) {
	return byteArrayColumnIndex{col.page()}, nil
}

func (col *byteArrayColumnBuffer) OffsetIndex() (OffsetIndex, error) {
	return byteArrayOffsetIndex{col.page()}, nil
}

func (col *byteArrayColumnBuffer) BloomFilter() BloomFilter { return nil }

func (col *byteArrayColumnBuffer) Dictionary() Dictionary { return nil }

func (col *byteArrayColumnBuffer) Pages() Pages { return onePage(col.Page()) }

func (col *byteArrayColumnBuffer) page() *byteArrayPage {
	if len(col.lengths) > 0 && orderOfUint32(col.offsets) < 1 { // unordered?
		if cap(col.scratch) < len(col.values) {
			col.scratch = make([]byte, 0, cap(col.values))
		} else {
			col.scratch = col.scratch[:0]
		}

		for i := range col.lengths {
			n := len(col.scratch)
			col.scratch = append(col.scratch, col.index(i)...)
			col.offsets[i] = uint32(n)
		}

		col.values, col.scratch = col.scratch, col.values
	}
	col.offsets = append(col.offsets[:len(col.lengths)], uint32(len(col.values)))
	return &col.byteArrayPage
}

func (col *byteArrayColumnBuffer) Page() Page {
	return col.page()
}

func (col *byteArrayColumnBuffer) Reset() {
	col.values = col.values[:0]
	col.offsets = col.offsets[:0]
	col.lengths = col.lengths[:0]
}

func (col *byteArrayColumnBuffer) NumRows() int64 { return int64(col.Len()) }

func (col *byteArrayColumnBuffer) NumValues() int64 { return int64(col.Len()) }

func (col *byteArrayColumnBuffer) Cap() int { return cap(col.lengths) }

func (col *byteArrayColumnBuffer) Len() int { return len(col.lengths) }

func (col *byteArrayColumnBuffer) Less(i, j int) bool {
	return bytes.Compare(col.index(i), col.index(j)) < 0
}

func (col *byteArrayColumnBuffer) Swap(i, j int) {
	col.offsets[i], col.offsets[j] = col.offsets[j], col.offsets[i]
	col.lengths[i], col.lengths[j] = col.lengths[j], col.lengths[i]
}

func (col *byteArrayColumnBuffer) Write(b []byte) (int, error) {
	_, n, err := col.writeByteArrays(b)
	return n, err
}

func (col *byteArrayColumnBuffer) WriteByteArrays(values []byte) (int, error) {
	n, _, err := col.writeByteArrays(values)
	return n, err
}

func (col *byteArrayColumnBuffer) writeByteArrays(values []byte) (count, bytes int, err error) {
	baseCount := len(col.lengths)
	baseBytes := len(col.values) + (plain.ByteArrayLengthSize * len(col.lengths))

	err = plain.RangeByteArray(values, func(value []byte) error {
		col.append(unsafecast.String(value))
		return nil
	})

	count = len(col.lengths) - baseCount
	bytes = (len(col.values) - baseBytes) + (plain.ByteArrayLengthSize * count)
	return count, bytes, err
}

func (col *byteArrayColumnBuffer) WriteValues(values []Value) (int, error) {
	col.writeValues(makeArrayValue(values, offsetOfPtr), columnLevels{})
	return len(values), nil
}

func (col *byteArrayColumnBuffer) writeValues(rows sparse.Array, _ columnLevels) {
	for i := range rows.Len() {
		p := rows.Index(i)
		col.append(*(*string)(p))
	}
}

func (col *byteArrayColumnBuffer) ReadValuesAt(values []Value, offset int64) (n int, err error) {
	i := int(offset)
	switch {
	case i < 0:
		return 0, errRowIndexOutOfBounds(offset, int64(len(col.lengths)))
	case i >= len(col.lengths):
		return 0, io.EOF
	default:
		for n < len(values) && i < len(col.lengths) {
			values[n] = col.makeValueBytes(col.index(i))
			n++
			i++
		}
		if n < len(values) {
			err = io.EOF
		}
		return n, err
	}
}

func (col *byteArrayColumnBuffer) append(value string) {
	col.offsets = append(col.offsets, uint32(len(col.values)))
	col.lengths = append(col.lengths, uint32(len(value)))
	col.values = append(col.values, value...)
}

func (col *byteArrayColumnBuffer) index(i int) []byte {
	offset := col.offsets[i]
	length := col.lengths[i]
	end := offset + length
	return col.values[offset:end:end]
}
