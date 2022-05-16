package parquet

import (
	"fmt"
	"io"

	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/encoding/plain"
	"github.com/segmentio/parquet-go/internal/bits"
)

// ColumnReader is an interface implemented by types which support reading
// columns of values. The interface extends ValueReader to work on top of
// parquet encodings.
//
// Implementations of ColumnReader may also provide extensions that the
// application can detect using type assertions. For example, readers for
// columns of INT32 values may implement the parquet.Int32Reader interface
// as a mechanism to provide a type safe and more efficient access to the
// column values.
type ColumnReader interface {
	ValueReader

	// Returns the type of values read.
	Type() Type

	// Returns the column number of values read.
	Column() int

	// Resets the reader state to read numValues values from the given decoder.
	//
	// Column readers created from parquet types are initialized to an empty
	// state and will return io.EOF on every read until a decoder is installed
	// via a call to Reset.
	Reset(numValues int, decoder encoding.Decoder)
}

type fileColumnReader struct {
	remain             int
	numValues          int
	maxRepetitionLevel int8
	maxDefinitionLevel int8
	repetitions        levelReader
	definitions        levelReader
	values             ColumnReader
}

func newFileColumnReader(values ColumnReader, maxRepetitionLevel, maxDefinitionLevel int8, bufferSize int) *fileColumnReader {
	repetitionBufferSize := 0
	definitionBufferSize := 0

	switch {
	case maxRepetitionLevel > 0 && maxDefinitionLevel > 0:
		repetitionBufferSize = bufferSize / 2
		definitionBufferSize = bufferSize / 2

	case maxRepetitionLevel > 0:
		repetitionBufferSize = bufferSize

	case maxDefinitionLevel > 0:
		definitionBufferSize = bufferSize
	}

	return &fileColumnReader{
		maxRepetitionLevel: maxRepetitionLevel,
		maxDefinitionLevel: maxDefinitionLevel,
		repetitions:        makeLevelReader(repetitionBufferSize),
		definitions:        makeLevelReader(definitionBufferSize),
		values:             values,
	}
}

func (r *fileColumnReader) Type() Type { return r.values.Type() }

func (r *fileColumnReader) Column() int { return r.values.Column() }

func (r *fileColumnReader) ReadValues(values []Value) (int, error) {
	if r.values == nil {
		return 0, io.EOF
	}
	read := 0
	columnIndex := ^int16(r.Column())

	for r.remain > 0 && len(values) > 0 {
		var err error
		var repetitionLevels []int8
		var definitionLevels []int8
		var numValues = r.remain

		if len(values) < numValues {
			numValues = len(values)
		}

		if r.maxRepetitionLevel > 0 {
			repetitionLevels, err = r.repetitions.peekLevels()
			if err != nil {
				return read, fmt.Errorf("decoding repetition level from data page of column %d: %w", r.Column(), err)
			}
			if len(repetitionLevels) < numValues {
				numValues = len(repetitionLevels)
			}
		}

		if r.maxDefinitionLevel > 0 {
			definitionLevels, err = r.definitions.peekLevels()
			if err != nil {
				return read, fmt.Errorf("decoding definition level from data page of column %d: %w", r.Column(), err)
			}
			if len(definitionLevels) < numValues {
				numValues = len(definitionLevels)
			}
		}

		if len(repetitionLevels) > 0 {
			repetitionLevels = repetitionLevels[:numValues]
		}
		if len(definitionLevels) > 0 {
			definitionLevels = definitionLevels[:numValues]
		}
		numNulls := countLevelsNotEqual(definitionLevels, r.maxDefinitionLevel)
		wantRead := numValues - numNulls
		n, err := r.values.ReadValues(values[:wantRead])
		if n < wantRead && err != nil {
			return read, fmt.Errorf("read error after decoding %d/%d values from data page of column %d: %w", r.numValues-r.remain, r.numValues, r.Column(), err)
		}

		for i, j := n-1, len(definitionLevels)-1; j >= 0; j-- {
			if definitionLevels[j] != r.maxDefinitionLevel {
				values[j] = Value{columnIndex: columnIndex}
			} else {
				values[j] = values[i]
				i--
			}
		}

		for i, lvl := range repetitionLevels {
			values[i].repetitionLevel = lvl
		}

		for i, lvl := range definitionLevels {
			values[i].definitionLevel = lvl
		}

		values = values[numValues:]
		r.repetitions.discardLevels(len(repetitionLevels))
		r.definitions.discardLevels(len(definitionLevels))
		r.remain -= numValues
		read += numValues
	}

	if r.remain == 0 {
		return read, io.EOF
	}

	return read, nil
}

func (r *fileColumnReader) reset(numValues int, repetitions, definitions, values encoding.Decoder) {
	if repetitions != nil {
		repetitions.SetBitWidth(bits.Len8(r.maxRepetitionLevel))
	}
	if definitions != nil {
		definitions.SetBitWidth(bits.Len8(r.maxDefinitionLevel))
	}
	r.remain = numValues
	r.numValues = numValues
	r.repetitions.reset(repetitions)
	r.definitions.reset(definitions)
	r.values.Reset(numValues, values)
}

func (r *fileColumnReader) Reset(int, encoding.Decoder) {
	panic("BUG: parquet.fileColumnReader.Reset must not be called")
}

type levelReader struct {
	decoder encoding.Decoder
	levels  []int8
	offset  int
	count   int
}

func makeLevelReader(bufferSize int) levelReader {
	return levelReader{
		levels: make([]int8, 0, bufferSize),
	}
}

func (r *levelReader) readLevel() (int8, error) {
	for {
		if r.offset < len(r.levels) {
			lvl := r.levels[r.offset]
			r.offset++
			return lvl, nil
		}
		if err := r.decodeLevels(); err != nil {
			return -1, err
		}
	}
}

func (r *levelReader) peekLevels() ([]int8, error) {
	if r.offset == len(r.levels) {
		if err := r.decodeLevels(); err != nil {
			return nil, err
		}
	}
	return r.levels[r.offset:], nil
}

func (r *levelReader) discardLevels(n int) {
	remain := len(r.levels) - r.offset
	switch {
	case n > remain:
		panic("BUG: cannot discard more levels than buffered")
	case n == remain:
		r.levels = r.levels[:0]
		r.offset = 0
	default:
		r.offset += n
	}
}

func (r *levelReader) decodeLevels() error {
	n, err := r.decoder.DecodeInt8(r.levels[:cap(r.levels)])
	if n == 0 {
		return err
	}
	r.levels = r.levels[:n]
	r.offset = 0
	r.count += n
	return nil
}

func (r *levelReader) reset(decoder encoding.Decoder) {
	r.decoder = decoder
	r.levels = r.levels[:0]
	r.offset = 0
	r.count = 0
}

type byteArrayColumnReader struct {
	typ         Type
	decoder     encoding.Decoder
	buffer      encoding.ByteArrayList
	offset      int
	remain      int
	columnIndex int16
}

func newByteArrayColumnReader(typ Type, columnIndex int16, bufferSize int) *byteArrayColumnReader {
	return &byteArrayColumnReader{
		typ:         typ,
		buffer:      encoding.MakeByteArrayList(atLeastOne(bufferSize / 16)),
		columnIndex: ^columnIndex,
	}
}

func (r *byteArrayColumnReader) Type() Type { return r.typ }

func (r *byteArrayColumnReader) Column() int { return int(^r.columnIndex) }

func (r *byteArrayColumnReader) readByteArrays(do func([]byte) bool) (n int, err error) {
	for {
		for r.remain > 0 && r.offset < r.buffer.Len() {
			if !do(r.buffer.Index(r.offset)) {
				return n, nil
			}
			r.offset++
			r.remain--
			n++
		}

		if r.remain == 0 || r.decoder == nil {
			return n, io.EOF
		}

		r.buffer.Reset()
		r.offset = 0

		d, err := r.decoder.DecodeByteArray(&r.buffer)
		if d == 0 {
			return n, err
		}
	}
}

func (r *byteArrayColumnReader) ReadRequired(values []byte) (int, error) {
	return r.ReadByteArrays(values)
}

func (r *byteArrayColumnReader) ReadByteArrays(values []byte) (int, error) {
	i := 0
	n, err := r.readByteArrays(func(b []byte) bool {
		k := plain.ByteArrayLengthSize + len(b)
		if k > (len(values) - i) {
			return false
		}
		plain.PutByteArrayLength(values[i:], len(b))
		copy(values[i+plain.ByteArrayLengthSize:], b)
		i += k
		return true
	})
	if i == 0 && err == nil {
		err = io.ErrShortBuffer
	}
	return n, err
}

func (r *byteArrayColumnReader) ReadValues(values []Value) (int, error) {
	i := 0
	return r.readByteArrays(func(b []byte) (ok bool) {
		if ok = i < len(values); ok {
			values[i] = makeValueBytes(ByteArray, copyBytes(b))
			values[i].columnIndex = r.columnIndex
			i++
		}
		return ok
	})
}

func (r *byteArrayColumnReader) Reset(numValues int, decoder encoding.Decoder) {
	r.decoder = decoder
	r.buffer.Reset()
	r.offset = 0
	r.remain = numValues
}

type fixedLenByteArrayColumnReader struct {
	typ         Type
	decoder     encoding.Decoder
	buffer      []byte
	offset      int
	remain      int
	size        int
	bufferSize  int
	columnIndex int16
}

func newFixedLenByteArrayColumnReader(typ Type, columnIndex int16, bufferSize int) *fixedLenByteArrayColumnReader {
	return &fixedLenByteArrayColumnReader{
		typ:         typ,
		size:        typ.Length(),
		bufferSize:  bufferSize,
		columnIndex: ^columnIndex,
	}
}

func (r *fixedLenByteArrayColumnReader) Type() Type { return r.typ }

func (r *fixedLenByteArrayColumnReader) Column() int { return int(^r.columnIndex) }

func (r *fixedLenByteArrayColumnReader) ReadRequired(values []byte) (int, error) {
	return r.ReadFixedLenByteArrays(values)
}

func (r *fixedLenByteArrayColumnReader) ReadFixedLenByteArrays(values []byte) (n int, err error) {
	if (len(values) % r.size) != 0 {
		return 0, fmt.Errorf("cannot read FIXED_LEN_BYTE_ARRAY values of size %d into buffer of size %d", r.size, len(values))
	}
	if r.offset < len(r.buffer) {
		i := copy(values, r.buffer[r.offset:])
		n = i / r.size
		r.offset += i
		r.remain -= i
		values = values[i:]
	}
	if r.decoder == nil {
		err = io.EOF
	} else {
		var d int
		values = values[:min(r.remain, len(values))]
		d, err = r.decoder.DecodeFixedLenByteArray(r.size, values)
		n += d
		r.remain -= d
		if r.remain == 0 && err == nil {
			err = io.EOF
		}
	}
	return n, err
}

func (r *fixedLenByteArrayColumnReader) ReadValues(values []Value) (n int, err error) {
	if cap(r.buffer) == 0 {
		r.buffer = make([]byte, 0, atLeast((r.bufferSize/r.size)*r.size, r.size))
	}

	for {
		for (r.offset+r.size) <= len(r.buffer) && n < len(values) {
			values[n] = makeValueBytes(FixedLenByteArray, copyBytes(r.buffer[r.offset:r.offset+r.size]))
			values[n].columnIndex = r.columnIndex
			r.offset += r.size
			r.remain -= r.size
			n++
		}

		if r.remain == 0 || r.decoder == nil {
			return n, io.EOF
		}
		if n == len(values) {
			return n, nil
		}

		length := min(r.remain, cap(r.buffer))
		buffer := r.buffer[:length]
		d, err := r.decoder.DecodeFixedLenByteArray(r.size, buffer)
		if d == 0 {
			return n, err
		}

		r.buffer = buffer[:d*r.size]
		r.offset = 0
	}
}

func (r *fixedLenByteArrayColumnReader) Reset(numValues int, decoder encoding.Decoder) {
	r.decoder = decoder
	r.buffer = r.buffer[:0]
	r.offset = 0
	r.remain = r.size * numValues
}

type nullColumnReader struct {
	typ         Type
	remain      int
	columnIndex int16
}

func newNullColumnReader(typ Type, columnIndex int16) *nullColumnReader {
	return &nullColumnReader{
		typ:         typ,
		columnIndex: ^columnIndex,
	}
}

func (r *nullColumnReader) Type() Type {
	return r.typ
}

func (r *nullColumnReader) Column() int {
	return int(^r.columnIndex)
}

func (r *nullColumnReader) Reset(numValues int, decoder encoding.Decoder) {
	r.remain = numValues
}

func (r *nullColumnReader) ReadValues(values []Value) (n int, err error) {
	values = values[:min(r.remain, len(values))]
	for i := range values {
		values[i] = Value{columnIndex: r.columnIndex}
	}
	r.remain -= len(values)
	if r.remain == 0 {
		err = io.EOF
	}
	return len(values), err
}
