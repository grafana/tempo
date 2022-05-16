//go:build go1.18

package parquet

import (
	"fmt"
	"io"

	"github.com/segmentio/parquet-go/encoding"
)

type dictionary[T primitive] struct {
	page[T]
	typ   Type
	index map[T]int32
}

func newDictionary[T primitive](typ Type, columnIndex int16, bufferSize int, class *class[T]) *dictionary[T] {
	return &dictionary[T]{
		typ: typ,
		page: page[T]{
			class:       class,
			values:      make([]T, 0, dictCap(bufferSize, sizeof[T]())),
			columnIndex: ^columnIndex,
		},
	}
}

func readDictionary[T primitive](typ Type, columnIndex int16, numValues int, decoder encoding.Decoder, class *class[T]) (*dictionary[T], error) {
	d := &dictionary[T]{
		typ: typ,
		page: page[T]{
			class:       class,
			values:      make([]T, 0, atLeastOne(numValues)),
			columnIndex: ^columnIndex,
		},
	}

	for {
		if len(d.values) == cap(d.values) {
			newValues := make([]T, len(d.values), 2*cap(d.values))
			copy(newValues, d.values)
			d.values = newValues
		}

		n, err := d.class.decode(decoder, d.values[len(d.values):cap(d.values)])
		if n > 0 {
			d.values = d.values[:len(d.values)+n]
		}

		if err != nil {
			if err == io.EOF {
				err = nil
			} else {
				err = fmt.Errorf("reading parquet dictionary of %s values: %w", d.class.name, err)
			}
			return d, err
		}
	}
}

func (d *dictionary[T]) Type() Type { return newIndexedType(d.typ, d) }

func (d *dictionary[T]) Len() int { return len(d.values) }

func (d *dictionary[T]) Index(index int32) Value {
	return d.class.makeValue(d.values[index])
}

func (d *dictionary[T]) Insert(indexes []int32, values []Value) {
	_ = indexes[:len(values)]

	if d.index == nil {
		d.index = make(map[T]int32, cap(d.values))
		for i, v := range d.values {
			d.index[v] = int32(i)
		}
	}

	for i, v := range values {
		value := d.class.value(v)

		index, exists := d.index[value]
		if !exists {
			index = int32(len(d.values))
			d.values = append(d.values, value)
			d.index[value] = index
		}

		indexes[i] = index
	}
}

func (d *dictionary[T]) Lookup(indexes []int32, values []Value) {
	for i, j := range indexes {
		values[i] = d.Index(j)
	}
}

func (d *dictionary[T]) Bounds(indexes []int32) (min, max Value) {
	if len(indexes) > 0 {
		minValue := d.values[indexes[0]]
		maxValue := minValue
		less := d.class.less

		for _, i := range indexes[1:] {
			value := d.values[i]
			switch {
			case less(value, minValue):
				minValue = value
			case less(maxValue, value):
				maxValue = value
			}
		}

		makeValue := d.class.makeValue
		min = makeValue(minValue)
		max = makeValue(maxValue)
	}
	return min, max
}

func (d *dictionary[T]) Reset() {
	d.values = d.values[:0]
	d.index = nil
}

func (d *dictionary[T]) Page() BufferedPage {
	return &d.page
}
