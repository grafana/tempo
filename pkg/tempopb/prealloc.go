package tempopb

import "github.com/grafana/tempo/pkg/tempopb/pool"

var bytePool *pool.Pool

func init() {
	bktSize := 400
	numBuckets := 250
	minBucket := 0

	bytePool = pool.New(minBucket, numBuckets, bktSize, func(size int) []byte { return make([]byte, 0, size) })
}

// PreallocBytes is a (repeated bytes slices) which preallocs slices on Unmarshal.
type PreallocBytes struct {
	Slice []byte
}

// Unmarshal implements proto.Message.
func (r *PreallocBytes) Unmarshal(dAtA []byte) error {
	r.Slice = bytePool.Get(len(dAtA))
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

// ReuseByteSlices puts the byte slice back into bytePool for reuse.
func ReuseByteSlices(buffs [][]byte) {
	for _, b := range buffs {
		_ = bytePool.Put(b[:0])
	}
}
