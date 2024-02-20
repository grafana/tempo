package intern

import (
	"sync"
	"unsafe"

	pq "github.com/parquet-go/parquet-go"
)

type Interner struct {
	mtx sync.RWMutex
	m   map[string][]byte
}

func New() *Interner {
	return &Interner{m: make(map[string][]byte)}
}

func (i *Interner) UnsafeClone(v *pq.Value) pq.Value {
	a := *(*aValue)(unsafe.Pointer(v))
	switch v.Kind() {
	case pq.ByteArray, pq.FixedLenByteArray:
		a.ptr = addressOfBytes(i.internBytes(v.ByteArray()))
		return *(*pq.Value)(unsafe.Pointer(&a))
	default:
		return *v
	}
}

func (i *Interner) internBytes(b []byte) []byte {
	s := bytesToString(b)
	i.mtx.RLock()
	if x, ok := i.m[s]; ok {
		i.mtx.RUnlock()
		return x
	}
	i.mtx.RUnlock()
	i.mtx.Lock()
	defer i.mtx.Unlock()
	clone := make([]byte, len(b))
	copy(clone, b)
	i.m[s] = clone
	return clone
}

func (i *Interner) Reset() {
	i.mtx.Lock()
	i.m = make(map[string][]byte)
	i.mtx.Unlock()
}

// bytesToString converts a byte slice to a string.
func bytesToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

//go:linkname addressOfBytes github.com/parquet-go/parquet-go/internal/unsafecast.AddressOfBytes
func addressOfBytes(data []byte) *byte

type aValue struct {
	// data
	ptr *byte
	u64 uint64
	// type
	kind int8 // XOR(Kind) so the zero-value is <null>
	// levels
	definitionLevel byte
	repetitionLevel byte
	columnIndex     int16 // XOR so the zero-value is -1
}
