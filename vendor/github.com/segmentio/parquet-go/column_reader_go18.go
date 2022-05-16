//go:build go1.18

package parquet

import (
	"io"

	"github.com/segmentio/parquet-go/encoding"
)

type columnReader[T primitive] struct {
	class       *class[T]
	typ         Type
	decoder     encoding.Decoder
	buffer      []T
	offset      int
	remain      int
	bufferSize  int
	columnIndex int16
}

func newColumnReader[T primitive](typ Type, columnIndex int16, bufferSize int, class *class[T]) *columnReader[T] {
	return &columnReader[T]{
		class:       class,
		typ:         typ,
		bufferSize:  bufferSize,
		columnIndex: ^columnIndex,
	}
}

func (r *columnReader[T]) Type() Type { return r.typ }

func (r *columnReader[T]) Column() int { return int(^r.columnIndex) }

func (r *columnReader[T]) ReadRequired(values []T) (n int, err error) {
	if r.offset < len(r.buffer) {
		n = copy(values, r.buffer[r.offset:])
		r.offset += n
		r.remain -= n
		values = values[n:]
	}
	if r.remain == 0 || r.decoder == nil {
		return n, io.EOF
	}
	d, err := r.class.decode(r.decoder, values)
	r.remain -= d
	if r.remain == 0 && err == nil {
		err = io.EOF
	}
	return n + d, err
}

func (r *columnReader[T]) ReadValues(values []Value) (n int, err error) {
	if cap(r.buffer) == 0 {
		r.buffer = make([]T, 0, atLeastOne(r.bufferSize))
	}

	makeValue := r.class.makeValue
	columnIndex := r.columnIndex
	for {
		for r.offset < len(r.buffer) && n < len(values) {
			values[n] = makeValue(r.buffer[r.offset])
			values[n].columnIndex = columnIndex
			r.offset++
			r.remain--
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
		d, err := r.class.decode(r.decoder, buffer)
		if d == 0 {
			return n, err
		}

		r.buffer = buffer[:d]
		r.offset = 0
	}
}

func (r *columnReader[T]) Reset(numValues int, decoder encoding.Decoder) {
	r.decoder = decoder
	r.buffer = r.buffer[:0]
	r.offset = 0
	r.remain = numValues
}
