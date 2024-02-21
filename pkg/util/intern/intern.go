package intern

import (
	"sync"
	"unsafe"

	pq "github.com/parquet-go/parquet-go"
)

var mapPool = sync.Pool{
	New: func() interface{} {
		return make(map[string][]byte)
	},
}

type Interner struct {
	mtx sync.RWMutex
	m   map[string][]byte
}

func New() *Interner {
	return &Interner{m: mapPool.Get().(map[string][]byte)}
}

func (i *Interner) UnsafeClone(v *pq.Value) pq.Value {
	switch v.Kind() {
	case pq.ByteArray, pq.FixedLenByteArray:
		// Look away, this is unsafe.
		a := *(*pqValue)(unsafe.Pointer(v))
		a.ptr = addressOfBytes(i.internBytes(a.byteArray()))
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
	clear(i.m)
	i.mtx.Unlock()
}

func (i *Interner) Close() {
	i.mtx.Lock()
	clear(i.m)
	mapPool.Put(i.m)
	i.m = nil
	i.mtx.Unlock()
}

// bytesToString converts a byte slice to a string.
func bytesToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

//go:linkname addressOfBytes github.com/parquet-go/parquet-go/internal/unsafecast.AddressOfBytes
func addressOfBytes(data []byte) *byte

//go:linkname bytes github.com/parquet-go/parquet-go/internal/unsafecast.Bytes
func bytes(data *byte, size int) []byte

// pqValue is a slimmer version of parquet-go's pq.Value.
type pqValue struct {
	// data
	ptr *byte
	u64 uint64
	// type
	kind int8 // XOR(Kind) so the zero-value is <null>
}

func (v *pqValue) byteArray() []byte {
	return bytes(v.ptr, int(v.u64))
}
