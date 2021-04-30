package tempopb

import (
	"github.com/prometheus/prometheus/pkg/pool"
)

var (
	// buckets: [1KiB, 2KiB, 4KiB, 8KiB, 16KiB]
	bytePool = pool.New(1_000, 16_000, 2, func(size int) interface{} { return make([]byte, 0, size) })
)

// PreallocRequest is a (repeated bytes requests) which preallocs slices on Unmarshal.
type PreallocRequest struct {
	Request []byte
}

// Unmarshal implements proto.Message.
func (r *PreallocRequest) Unmarshal(dAtA []byte) error {
	r.Request = bytePool.Get(len(dAtA)).([]byte)
	r.Request = r.Request[:len(dAtA)]
	copy(r.Request, dAtA)
	return nil
}

// MarshalTo implements proto.Marshaller.
// returned int is not used
func (r *PreallocRequest) MarshalTo(dAtA []byte) (int, error) {
	copy(dAtA[:], r.Request[:])
	return len(r.Request), nil
}

// Size implements proto.Sizer.
func (r *PreallocRequest) Size() (n int) {
	if r == nil {
		return 0
	}
	return len(r.Request)
}

// ReuseRequest puts the byte slice back into bytePool for reuse.
func ReuseRequest(req *PushBytesRequest) {
	for i := range req.Requests {
		// We want to preserve the underlying allocated memory, [:0] helps us retains the cap() of the slice
		bytePool.Put(req.Requests[i].Request[:0])
	}
}
