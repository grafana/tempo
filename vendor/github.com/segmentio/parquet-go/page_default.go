//go:build !go1.18

package parquet

import (
	"io"

	"github.com/segmentio/parquet-go/deprecated"
	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/internal/bits"
)

type booleanPage struct {
	values      []bool
	columnIndex int16
}

func (page *booleanPage) Column() int { return int(^page.columnIndex) }

func (page *booleanPage) Dictionary() Dictionary { return nil }

func (page *booleanPage) NumRows() int64 { return int64(len(page.values)) }

func (page *booleanPage) NumValues() int64 { return int64(len(page.values)) }

func (page *booleanPage) NumNulls() int64 { return 0 }

func (page *booleanPage) min() bool {
	for _, value := range page.values {
		if !value {
			return false
		}
	}
	return len(page.values) > 0
}

func (page *booleanPage) max() bool {
	for _, value := range page.values {
		if value {
			return true
		}
	}
	return false
}

func (page *booleanPage) bounds() (min, max bool) {
	hasFalse, hasTrue := false, false

	for _, value := range page.values {
		if value {
			hasTrue = true
		} else {
			hasFalse = true
		}
		if hasTrue && hasFalse {
			break
		}
	}

	if !hasFalse {
		min = true
	}
	if hasTrue {
		max = true
	}
	return min, max
}

func (page *booleanPage) Bounds() (min, max Value, ok bool) {
	if ok = len(page.values) > 0; ok {
		minBool, maxBool := page.bounds()
		min = makeValueBoolean(minBool)
		max = makeValueBoolean(maxBool)
	}
	return min, max, ok
}

func (page *booleanPage) Clone() BufferedPage {
	return &booleanPage{
		values:      append([]bool{}, page.values...),
		columnIndex: page.columnIndex,
	}
}

func (page *booleanPage) Slice(i, j int64) BufferedPage {
	return &booleanPage{
		values:      page.values[i:j],
		columnIndex: page.columnIndex,
	}
}

func (page *booleanPage) Size() int64 { return sizeOfBool(page.values) }

func (page *booleanPage) RepetitionLevels() []int8 { return nil }

func (page *booleanPage) DefinitionLevels() []int8 { return nil }

func (page *booleanPage) WriteTo(e encoding.Encoder) error { return e.EncodeBoolean(page.values) }

func (page *booleanPage) Values() ValueReader { return &booleanPageReader{page: page} }

func (page *booleanPage) Buffer() BufferedPage { return page }

type booleanPageReader struct {
	page   *booleanPage
	offset int
}

func (r *booleanPageReader) Read(b []byte) (n int, err error) {
	return r.ReadBooleans(bits.BytesToBool(b))
}

func (r *booleanPageReader) ReadBooleans(values []bool) (n int, err error) {
	n = copy(values, r.page.values[r.offset:])
	r.offset += n
	if r.offset == len(r.page.values) {
		err = io.EOF
	}
	return n, err
}

func (r *booleanPageReader) ReadValues(values []Value) (n int, err error) {
	for n < len(values) && r.offset < len(r.page.values) {
		values[n] = makeValueBoolean(r.page.values[r.offset])
		values[n].columnIndex = r.page.columnIndex
		r.offset++
		n++
	}
	if r.offset == len(r.page.values) {
		err = io.EOF
	}
	return n, err
}

type int32Page struct {
	values      []int32
	columnIndex int16
}

func (page *int32Page) Column() int { return int(^page.columnIndex) }

func (page *int32Page) Dictionary() Dictionary { return nil }

func (page *int32Page) NumRows() int64 { return int64(len(page.values)) }

func (page *int32Page) NumValues() int64 { return int64(len(page.values)) }

func (page *int32Page) NumNulls() int64 { return 0 }

func (page *int32Page) min() int32 { return bits.MinInt32(page.values) }

func (page *int32Page) max() int32 { return bits.MaxInt32(page.values) }

func (page *int32Page) bounds() (min, max int32) { return bits.MinMaxInt32(page.values) }

func (page *int32Page) Bounds() (min, max Value, ok bool) {
	if ok = len(page.values) > 0; ok {
		minInt32, maxInt32 := page.bounds()
		min = makeValueInt32(minInt32)
		max = makeValueInt32(maxInt32)
	}
	return min, max, ok
}

func (page *int32Page) Clone() BufferedPage {
	return &int32Page{
		values:      append([]int32{}, page.values...),
		columnIndex: page.columnIndex,
	}
}

func (page *int32Page) Slice(i, j int64) BufferedPage {
	return &int32Page{
		values:      page.values[i:j],
		columnIndex: page.columnIndex,
	}
}

func (page *int32Page) Size() int64 { return sizeOfInt32(page.values) }

func (page *int32Page) RepetitionLevels() []int8 { return nil }

func (page *int32Page) DefinitionLevels() []int8 { return nil }

func (page *int32Page) WriteTo(e encoding.Encoder) error { return e.EncodeInt32(page.values) }

func (page *int32Page) Values() ValueReader { return &int32PageReader{page: page} }

func (page *int32Page) Buffer() BufferedPage { return page }

type int32PageReader struct {
	page   *int32Page
	offset int
}

func (r *int32PageReader) Read(b []byte) (n int, err error) {
	n, err = r.ReadInt32s(bits.BytesToInt32(b))
	return 4 * n, err
}

func (r *int32PageReader) ReadInt32s(values []int32) (n int, err error) {
	n = copy(values, r.page.values[r.offset:])
	r.offset += n
	if r.offset == len(r.page.values) {
		err = io.EOF
	}
	return n, err
}

func (r *int32PageReader) ReadValues(values []Value) (n int, err error) {
	for n < len(values) && r.offset < len(r.page.values) {
		values[n] = makeValueInt32(r.page.values[r.offset])
		values[n].columnIndex = r.page.columnIndex
		r.offset++
		n++
	}
	if r.offset == len(r.page.values) {
		err = io.EOF
	}
	return n, err
}

type int64Page struct {
	values      []int64
	columnIndex int16
}

func (page *int64Page) Column() int { return int(^page.columnIndex) }

func (page *int64Page) Dictionary() Dictionary { return nil }

func (page *int64Page) NumRows() int64 { return int64(len(page.values)) }

func (page *int64Page) NumValues() int64 { return int64(len(page.values)) }

func (page *int64Page) NumNulls() int64 { return 0 }

func (page *int64Page) min() int64 { return bits.MinInt64(page.values) }

func (page *int64Page) max() int64 { return bits.MaxInt64(page.values) }

func (page *int64Page) bounds() (min, max int64) { return bits.MinMaxInt64(page.values) }

func (page *int64Page) Bounds() (min, max Value, ok bool) {
	if ok = len(page.values) > 0; ok {
		minInt64, maxInt64 := page.bounds()
		min = makeValueInt64(minInt64)
		max = makeValueInt64(maxInt64)
	}
	return min, max, ok
}

func (page *int64Page) Clone() BufferedPage {
	return &int64Page{
		values:      append([]int64{}, page.values...),
		columnIndex: page.columnIndex,
	}
}

func (page *int64Page) Slice(i, j int64) BufferedPage {
	return &int64Page{
		values:      page.values[i:j],
		columnIndex: page.columnIndex,
	}
}

func (page *int64Page) Size() int64 { return sizeOfInt64(page.values) }

func (page *int64Page) RepetitionLevels() []int8 { return nil }

func (page *int64Page) DefinitionLevels() []int8 { return nil }

func (page *int64Page) WriteTo(e encoding.Encoder) error { return e.EncodeInt64(page.values) }

func (page *int64Page) Values() ValueReader { return &int64PageReader{page: page} }

func (page *int64Page) Buffer() BufferedPage { return page }

type int64PageReader struct {
	page   *int64Page
	offset int
}

func (r *int64PageReader) Read(b []byte) (n int, err error) {
	n, err = r.ReadInt64s(bits.BytesToInt64(b))
	return 8 * n, err
}

func (r *int64PageReader) ReadInt64s(values []int64) (n int, err error) {
	n = copy(values, r.page.values[r.offset:])
	r.offset += n
	if r.offset == len(r.page.values) {
		err = io.EOF
	}
	return n, err
}

func (r *int64PageReader) ReadValues(values []Value) (n int, err error) {
	for n < len(values) && r.offset < len(r.page.values) {
		values[n] = makeValueInt64(r.page.values[r.offset])
		values[n].columnIndex = r.page.columnIndex
		r.offset++
		n++
	}
	if r.offset == len(r.page.values) {
		err = io.EOF
	}
	return n, err
}

type int96Page struct {
	values      []deprecated.Int96
	columnIndex int16
}

func (page *int96Page) Column() int { return int(^page.columnIndex) }

func (page *int96Page) Dictionary() Dictionary { return nil }

func (page *int96Page) NumRows() int64 { return int64(len(page.values)) }

func (page *int96Page) NumValues() int64 { return int64(len(page.values)) }

func (page *int96Page) NumNulls() int64 { return 0 }

func (page *int96Page) min() deprecated.Int96 { return deprecated.MinInt96(page.values) }

func (page *int96Page) max() deprecated.Int96 { return deprecated.MaxInt96(page.values) }

func (page *int96Page) bounds() (min, max deprecated.Int96) {
	return deprecated.MinMaxInt96(page.values)
}

func (page *int96Page) Bounds() (min, max Value, ok bool) {
	if ok = len(page.values) > 0; ok {
		minInt96, maxInt96 := page.bounds()
		min = makeValueInt96(minInt96)
		max = makeValueInt96(maxInt96)
	}
	return min, max, ok
}

func (page *int96Page) Clone() BufferedPage {
	return &int96Page{
		values:      append([]deprecated.Int96{}, page.values...),
		columnIndex: page.columnIndex,
	}
}

func (page *int96Page) Slice(i, j int64) BufferedPage {
	return &int96Page{
		values:      page.values[i:j],
		columnIndex: page.columnIndex,
	}
}

func (page *int96Page) Size() int64 { return sizeOfInt96(page.values) }

func (page *int96Page) RepetitionLevels() []int8 { return nil }

func (page *int96Page) DefinitionLevels() []int8 { return nil }

func (page *int96Page) WriteTo(e encoding.Encoder) error { return e.EncodeInt96(page.values) }

func (page *int96Page) Values() ValueReader { return &int96PageReader{page: page} }

func (page *int96Page) Buffer() BufferedPage { return page }

type int96PageReader struct {
	page   *int96Page
	offset int
}

func (r *int96PageReader) Read(b []byte) (n int, err error) {
	n, err = r.ReadInt96s(deprecated.BytesToInt96(b))
	return 12 * n, err
}

func (r *int96PageReader) ReadInt96s(values []deprecated.Int96) (n int, err error) {
	n = copy(values, r.page.values[r.offset:])
	r.offset += n
	if r.offset == len(r.page.values) {
		err = io.EOF
	}
	return n, err
}

func (r *int96PageReader) ReadValues(values []Value) (n int, err error) {
	for n < len(values) && r.offset < len(r.page.values) {
		values[n] = makeValueInt96(r.page.values[r.offset])
		values[n].columnIndex = r.page.columnIndex
		r.offset++
		n++
	}
	if r.offset == len(r.page.values) {
		err = io.EOF
	}
	return n, err
}

type floatPage struct {
	values      []float32
	columnIndex int16
}

func (page *floatPage) Column() int { return int(^page.columnIndex) }

func (page *floatPage) Dictionary() Dictionary { return nil }

func (page *floatPage) NumRows() int64 { return int64(len(page.values)) }

func (page *floatPage) NumValues() int64 { return int64(len(page.values)) }

func (page *floatPage) NumNulls() int64 { return 0 }

func (page *floatPage) min() float32 { return bits.MinFloat32(page.values) }

func (page *floatPage) max() float32 { return bits.MaxFloat32(page.values) }

func (page *floatPage) bounds() (min, max float32) { return bits.MinMaxFloat32(page.values) }

func (page *floatPage) Bounds() (min, max Value, ok bool) {
	if ok = len(page.values) > 0; ok {
		minFloat32, maxFloat32 := page.bounds()
		min = makeValueFloat(minFloat32)
		max = makeValueFloat(maxFloat32)
	}
	return min, max, ok
}

func (page *floatPage) Clone() BufferedPage {
	return &floatPage{
		values:      append([]float32{}, page.values...),
		columnIndex: page.columnIndex,
	}
}

func (page *floatPage) Slice(i, j int64) BufferedPage {
	return &floatPage{
		values:      page.values[i:j],
		columnIndex: page.columnIndex,
	}
}

func (page *floatPage) Size() int64 { return sizeOfFloat32(page.values) }

func (page *floatPage) RepetitionLevels() []int8 { return nil }

func (page *floatPage) DefinitionLevels() []int8 { return nil }

func (page *floatPage) WriteTo(e encoding.Encoder) error { return e.EncodeFloat(page.values) }

func (page *floatPage) Values() ValueReader { return &floatPageReader{page: page} }

func (page *floatPage) Buffer() BufferedPage { return page }

type floatPageReader struct {
	page   *floatPage
	offset int
}

func (r *floatPageReader) Read(b []byte) (n int, err error) {
	n, err = r.ReadFloats(bits.BytesToFloat32(b))
	return 4 * n, err
}

func (r *floatPageReader) ReadFloats(values []float32) (n int, err error) {
	n = copy(values, r.page.values[r.offset:])
	r.offset += n
	if r.offset == len(r.page.values) {
		err = io.EOF
	}
	return n, err
}

func (r *floatPageReader) ReadValues(values []Value) (n int, err error) {
	for n < len(values) && r.offset < len(r.page.values) {
		values[n] = makeValueFloat(r.page.values[r.offset])
		values[n].columnIndex = r.page.columnIndex
		r.offset++
		n++
	}
	if r.offset == len(r.page.values) {
		err = io.EOF
	}
	return n, err
}

type doublePage struct {
	values      []float64
	columnIndex int16
}

func (page *doublePage) Column() int { return int(^page.columnIndex) }

func (page *doublePage) Dictionary() Dictionary { return nil }

func (page *doublePage) NumRows() int64 { return int64(len(page.values)) }

func (page *doublePage) NumValues() int64 { return int64(len(page.values)) }

func (page *doublePage) NumNulls() int64 { return 0 }

func (page *doublePage) min() float64 { return bits.MinFloat64(page.values) }

func (page *doublePage) max() float64 { return bits.MaxFloat64(page.values) }

func (page *doublePage) bounds() (min, max float64) { return bits.MinMaxFloat64(page.values) }

func (page *doublePage) Bounds() (min, max Value, ok bool) {
	if ok = len(page.values) > 0; ok {
		minFloat64, maxFloat64 := page.bounds()
		min = makeValueDouble(minFloat64)
		max = makeValueDouble(maxFloat64)
	}
	return min, max, ok
}

func (page *doublePage) Clone() BufferedPage {
	return &doublePage{
		values:      append([]float64{}, page.values...),
		columnIndex: page.columnIndex,
	}
}

func (page *doublePage) Slice(i, j int64) BufferedPage {
	return &doublePage{
		values:      page.values[i:j],
		columnIndex: page.columnIndex,
	}
}

func (page *doublePage) Size() int64 { return sizeOfFloat64(page.values) }

func (page *doublePage) RepetitionLevels() []int8 { return nil }

func (page *doublePage) DefinitionLevels() []int8 { return nil }

func (page *doublePage) WriteTo(e encoding.Encoder) error { return e.EncodeDouble(page.values) }

func (page *doublePage) Values() ValueReader { return &doublePageReader{page: page} }

func (page *doublePage) Buffer() BufferedPage { return page }

type doublePageReader struct {
	page   *doublePage
	offset int
}

func (r *doublePageReader) Read(b []byte) (n int, err error) {
	n, err = r.ReadDoubles(bits.BytesToFloat64(b))
	return 8 * n, err
}

func (r *doublePageReader) ReadDoubles(values []float64) (n int, err error) {
	n = copy(values, r.page.values[r.offset:])
	r.offset += n
	if r.offset == len(r.page.values) {
		err = io.EOF
	}
	return n, err
}

func (r *doublePageReader) ReadValues(values []Value) (n int, err error) {
	for n < len(values) && r.offset < len(r.page.values) {
		values[n] = makeValueDouble(r.page.values[r.offset])
		values[n].columnIndex = r.page.columnIndex
		r.offset++
		n++
	}
	if r.offset == len(r.page.values) {
		err = io.EOF
	}
	return n, err
}

// The following two specializations for unsigned integer types are needed to
// apply an unsigned comparison when looking up the min and max page values.

type uint32Page struct{ *int32Page }

func (page uint32Page) min() uint32 { return bits.MinUint32(bits.Int32ToUint32(page.values)) }

func (page uint32Page) max() uint32 { return bits.MaxUint32(bits.Int32ToUint32(page.values)) }

func (page uint32Page) bounds() (min, max uint32) {
	return bits.MinMaxUint32(bits.Int32ToUint32(page.values))
}

func (page uint32Page) Bounds() (min, max Value, ok bool) {
	if ok = len(page.values) > 0; ok {
		minUint32, maxUint32 := page.bounds()
		min = makeValueInt32(int32(minUint32))
		max = makeValueInt32(int32(maxUint32))
	}
	return min, max, ok
}

func (page uint32Page) Clone() BufferedPage {
	return uint32Page{page.int32Page.Clone().(*int32Page)}
}

func (page uint32Page) Slice(i, j int64) BufferedPage {
	return uint32Page{page.int32Page.Slice(i, j).(*int32Page)}
}

func (page uint32Page) Buffer() BufferedPage { return page }

type uint64Page struct{ *int64Page }

func (page uint64Page) min() uint64 { return bits.MinUint64(bits.Int64ToUint64(page.values)) }

func (page uint64Page) max() uint64 { return bits.MaxUint64(bits.Int64ToUint64(page.values)) }

func (page uint64Page) bounds() (min, max uint64) {
	return bits.MinMaxUint64(bits.Int64ToUint64(page.values))
}

func (page uint64Page) Bounds() (min, max Value, ok bool) {
	if ok = len(page.values) > 0; ok {
		minUint64, maxUint64 := page.bounds()
		min = makeValueInt64(int64(minUint64))
		max = makeValueInt64(int64(maxUint64))
	}
	return min, max, ok
}

func (page uint64Page) Clone() BufferedPage {
	return uint64Page{page.int64Page.Clone().(*int64Page)}
}

func (page uint64Page) Slice(i, j int64) BufferedPage {
	return uint64Page{page.int64Page.Slice(i, j).(*int64Page)}
}

func (page uint64Page) Buffer() BufferedPage { return page }

var (
	_ io.Reader = (*booleanPageReader)(nil)
	_ io.Reader = (*int32PageReader)(nil)
	_ io.Reader = (*int64PageReader)(nil)
	_ io.Reader = (*int96PageReader)(nil)
	_ io.Reader = (*floatPageReader)(nil)
	_ io.Reader = (*doublePageReader)(nil)
	_ io.Reader = (*byteArrayPageReader)(nil)
	_ io.Reader = (*fixedLenByteArrayPageReader)(nil)

	_ BooleanReader           = (*booleanPageReader)(nil)
	_ Int32Reader             = (*int32PageReader)(nil)
	_ Int64Reader             = (*int64PageReader)(nil)
	_ Int96Reader             = (*int96PageReader)(nil)
	_ FloatReader             = (*floatPageReader)(nil)
	_ DoubleReader            = (*doublePageReader)(nil)
	_ ByteArrayReader         = (*byteArrayPageReader)(nil)
	_ FixedLenByteArrayReader = (*fixedLenByteArrayPageReader)(nil)
)
