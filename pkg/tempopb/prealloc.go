package tempopb

import (
	"bytes"
	"os"
	"strconv"
)

var bytePool *Pool

func init() {
	bktSize := intFromEnv("PREALLOC_BKT_SIZE", 400)
	numBuckets := intFromEnv("PREALLOC_NUM_BUCKETS", 250)
	minBucket := intFromEnv("PREALLOC_MIN_BUCKET", 0)

	bytePool = NewPool("ingester_prealloc", minBucket, numBuckets, bktSize)
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
	copy(dAtA, r.Slice)
	return len(r.Slice), nil
}

// Size implements proto.Sizer.
func (r *PreallocBytes) Size() (n int) {
	if r == nil {
		return 0
	}
	return len(r.Slice)
}

// SizeWiresmith implements wiresmith's CustomMarshaler.
func (r *PreallocBytes) SizeWiresmith() int {
	if r == nil {
		return 0
	}
	return len(r.Slice)
}

// MarshalWiresmith implements wiresmith's CustomMarshaler. buf is sized to
// exactly SizeWiresmith() bytes by the generated envelope code.
func (r *PreallocBytes) MarshalWiresmith(buf []byte) (int, error) {
	copy(buf, r.Slice)
	return len(r.Slice), nil
}

// UnmarshalWiresmith implements wiresmith's CustomMarshaler. Like Unmarshal,
// it copies the payload into a pooled slice; call ReuseByteSlices to return
// the slice to the pool.
func (r *PreallocBytes) UnmarshalWiresmith(buf []byte) error {
	return r.Unmarshal(buf)
}

// EqualWiresmith implements wiresmith's CustomMarshaler.
func (r *PreallocBytes) EqualWiresmith(other any) bool {
	o, ok := other.(PreallocBytes)
	if !ok {
		po, pok := other.(*PreallocBytes)
		if !pok || po == nil {
			return false
		}
		o = *po
	}
	return bytes.Equal(r.Slice, o.Slice)
}

// CompareWiresmith implements wiresmith's CustomMarshaler. Returns -1 on a
// type mismatch so the generated Compare stays total.
func (r *PreallocBytes) CompareWiresmith(other any) int {
	o, ok := other.(PreallocBytes)
	if !ok {
		po, pok := other.(*PreallocBytes)
		if !pok || po == nil {
			return -1
		}
		o = *po
	}
	return bytes.Compare(r.Slice, o.Slice)
}

// ReuseByteSlices puts the byte slice back into bytePool for reuse.
func ReuseByteSlices(buffs [][]byte) {
	for _, b := range buffs {
		_ = bytePool.Put(b)
	}
}

func intFromEnv(env string, defaultValue int) int {
	// get the value from the environment
	val, ok := os.LookupEnv(env)
	if !ok {
		return defaultValue
	}

	// try to parse the value
	intVal, err := strconv.Atoi(val)
	if err != nil {
		panic("failed to parse " + env + " as int")
	}

	return intVal
}
