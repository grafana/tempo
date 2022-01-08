package tempopb

import (
	"github.com/prometheus/prometheus/util/pool"
)

var (
	// buckets: [0.5KiB, 1KiB, 2KiB, 4KiB, 8KiB, 16KiB]
	bytePool = pool.New(500, 16_000, 2, func(size int) interface{} { return make([]byte, 0, size) })
)

// PreallocBytes is a (repeated bytes slices) which preallocs slices on Unmarshal.
type PreallocBytes struct {
	Slice []byte
}

// Unmarshal implements proto.Message.
func (r *PreallocBytes) Unmarshal(dAtA []byte) error {
	r.Slice = bytePool.Get(len(dAtA)).([]byte)
	r.Slice = r.Slice[:len(dAtA)]
	copy(r.Slice, dAtA)
	return nil
}

// MarshalTo implements proto.Marshaller.
// returned int is not used
func (r *PreallocBytes) MarshalTo(dAtA []byte) (int, error) {
	copy(dAtA[:], r.Slice[:])
	return len(r.Slice), nil
}

// Size implements proto.Sizer.
func (r *PreallocBytes) Size() (n int) {
	if r == nil {
		return 0
	}
	return len(r.Slice)
}

// ReuseTraceBytes puts the byte slice back into bytePool for reuse.
func ReuseTraceBytes(trace *TraceBytes) {
	for _, t := range trace.Traces {
		bytePool.Put(t[:0])
	}
}

// SliceFromBytePool gets a slice from the byte pool
func SliceFromBytePool(size int) []byte {
	return bytePool.Get(size).([]byte)[:size]
}
