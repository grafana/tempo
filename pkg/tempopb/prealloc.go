package tempopb

import (
	"github.com/prometheus/prometheus/pkg/pool"
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

// ReuseRequest puts the byte slice back into bytePool for reuse.
func ReuseRequest(req *PushBytesRequest) {
	for _, r := range req.Requests { // deprecated
		// We want to preserve the underlying allocated memory, [:0] helps us retains the cap() of the slice
		bytePool.Put(r.Slice[:0])
	}
	for _, t := range req.Traces { // current
		bytePool.Put(t.Slice[:0])
	}
	for _, i := range req.Ids {
		bytePool.Put(i.Slice[:0])
	}
}
