package encoding

import (
	"sort"
)

// ByteArrayList is a container similar to [][]byte with a smaller memory
// overhead. Where using a byte slices introduces ~24 bytes of overhead per
// element, ByteArrayList requires only 8 bytes per element. Extra efficiency
// also comes from reducing GC pressure by using contiguous areas of memory
// instead of allocating individual slices for each element. For lists with
// many small-size elements, the memory footprint can be reduced by 40-80%.
type ByteArrayList struct {
	slices []slice
	values []byte
}

type slice struct{ i, j uint32 }

func (s slice) len() int { return int(s.j - s.i) }

func MakeByteArrayList(capacity int) ByteArrayList {
	return ByteArrayList{
		slices: make([]slice, 0, capacity),
		values: make([]byte, 0, 8*capacity),
	}
}

func (list *ByteArrayList) Clone() ByteArrayList {
	size := 0
	for _, s := range list.slices {
		size += s.len()
	}
	clone := ByteArrayList{
		slices: make([]slice, 0, len(list.slices)),
		values: make([]byte, 0, size),
	}
	for _, s := range list.slices {
		clone.Push(list.slice(s))
	}
	return clone
}

func (list *ByteArrayList) Split() [][]byte {
	clone := ByteArrayList{
		slices: list.slices,
		values: make([]byte, len(list.values)),
	}
	copy(clone.values, list.values)
	split := make([][]byte, clone.Len())
	for i := range split {
		split[i] = clone.Index(i)
	}
	return split
}

func (list *ByteArrayList) Slice(i, j int) ByteArrayList {
	return ByteArrayList{
		slices: list.slices[i:j:j],
		values: list.values,
	}
}

func (list *ByteArrayList) Grow(n int) {
	if n > (cap(list.slices) - len(list.slices)) {
		newCap := 2 * cap(list.slices)
		newLen := len(list.slices) + n
		for newCap < newLen {
			newCap *= 2
		}
		newSlices := make([]slice, len(list.slices), newCap)
		copy(newSlices, list.slices)
		list.slices = newSlices
	}
}

func (list *ByteArrayList) Reset() {
	list.slices = list.slices[:0]
	list.values = list.values[:0]
}

func (list *ByteArrayList) Push(v []byte) {
	list.slices = append(list.slices, slice{
		i: uint32(len(list.values)),
		j: uint32(len(list.values) + len(v)),
	})
	list.values = append(list.values, v...)
}

func (list *ByteArrayList) PushSize(n int) []byte {
	i := len(list.values)
	j := len(list.values) + n

	list.slices = append(list.slices, slice{
		i: uint32(i),
		j: uint32(j),
	})

	if j <= cap(list.values) {
		list.values = list.values[:j]
	} else {
		newCap := 2 * cap(list.values)
		newLen := j
		for newCap < newLen {
			newCap *= 2
		}
		newValues := make([]byte, newLen, newCap)
		copy(newValues, list.values)
		list.values = newValues
	}

	return list.values[i:j:j]
}

func (list *ByteArrayList) Index(i int) []byte {
	return list.slice(list.slices[i])
}

func (list *ByteArrayList) Range(f func([]byte) bool) {
	for _, s := range list.slices {
		if !f(list.slice(s)) {
			break
		}
	}
}

func (list *ByteArrayList) Size() int64 {
	size := int64(0)
	for _, s := range list.slices {
		size += 8 + int64(s.len())
	}
	return size
}

func (list *ByteArrayList) Cap() int {
	return cap(list.slices)
}

func (list *ByteArrayList) Len() int {
	return len(list.slices)
}

func (list *ByteArrayList) Less(i, j int) bool {
	return string(list.Index(i)) < string(list.Index(j))
}

func (list *ByteArrayList) Swap(i, j int) {
	list.slices[i], list.slices[j] = list.slices[j], list.slices[i]
}

func (list *ByteArrayList) slice(s slice) []byte {
	return list.values[s.i:s.j:s.j]
}

var (
	_ sort.Interface = (*ByteArrayList)(nil)
)
