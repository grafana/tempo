package traceidboundary

import (
	"bytes"
	"encoding/binary"
	"math"
)

type Boundary struct {
	Min, Max []byte
}

// Pairs returns the boundaries that match trace IDs in that shard.  Internally this is
// similar to how queriers divide the block ID-space, but here it's trace IDs instead.
// The inputs are 1-based because it seems more readable: shard 1 of 10.  Most boundaries
// are [,) lower inclusive, upper exclusive. However the last boundary that ends in the
// max value 0xFFFF... is [,] inclusive/inclusive and indicated when the return value
// upperInclusive is set.
// Of course there are some caveats:
//   - Trace IDs can be 16 or 8 bytes.  If we naively sharded only in 16-byte space it would
//     be unbalanced because all 8-byte IDs would land in the first shard. Therefore we
//     divide in both 16- and 8-byte spaces and a single shard covers a range in each.
//   - There are different regimes of 8-byte ID generation and they are not uniformly
//     distributed. So there are several sub-shards within 8-byte space to compensate.
func Pairs(shard, of uint32) (boundaries []Boundary, upperInclusive bool) {
	boundaries = append(boundaries, complicatedShardingFor8ByteIDs(shard, of)...)

	// Final pair is the normal full precision 16-byte IDs.
	b := bounds(of, 0, math.MaxUint64, 0)

	// Avoid overlap with the 64-bit precision ones. I.e. a minimum of 0x0000.... would
	// unintentionally include all 64-bit IDs. The first 65-bit ID starts here:
	b[0] = []byte{0, 0, 0, 0, 0, 0, 0, 0x01, 0, 0, 0, 0, 0, 0, 0, 0}

	// Adjust max to be full 16-byte max
	b[of] = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	boundaries = append(boundaries, Boundary{
		Min: b[shard-1],
		Max: b[shard],
	})

	// Top most 0xFFFFF... boundary is inclusive
	upperInclusive = shard == of

	return
}

// Funcs returns helper functions that match trace IDs in the given shard.
func Funcs(shard, of uint32) (testSingle func([]byte) bool, testRange func([]byte, []byte) bool) {
	pairs, upperInclusive := Pairs(shard, of)

	upper := -1
	if upperInclusive {
		upper = 0
	}

	isMatch := func(id []byte) bool {
		for _, p := range pairs {
			if bytes.Compare(p.Min, id) <= 0 && bytes.Compare(id, p.Max) <= upper {
				return true
			}
		}
		return false
	}

	withinRange := func(min []byte, max []byte) bool {
		for _, p := range pairs {
			if bytes.Compare(p.Min, max) <= 0 && bytes.Compare(min, p.Max) <= upper {
				return true
			}
		}
		return false
	}

	return isMatch, withinRange
}

// complicatedShardingFor8ByteIDs generates a list of trace ID boundaries that is subdividing
// the 64-bit space. Not optimal or universal rules. This seems like overkill but in practice
// 8-byte IDs are unevenly weighted towards lower values starting with zeros.  The benefit of
// this approach is better fairness across shards, and also *invariance* across workloads,
// no matter if your instrumentation is generating 8-byte or 16-byte trace IDs.
func complicatedShardingFor8ByteIDs(shard, of uint32) []Boundary {
	var results []Boundary

	regions := []struct {
		min, max uint64
	}{
		{0x0000000000000000, 0x00FFFFFFFFFFFFFF}, // Region with upper byte = 0
		{0x0100000000000000, 0x0FFFFFFFFFFFFFFF}, // Region with upper nibble = 0
		{0x1000000000000000, 0x7FFFFFFFFFFFFFFF}, // Region for 63-bit IDs (upper bit = 0)
		{0x8000000000000000, 0xFFFFFFFFFFFFFFFF}, // Region for true 64-bit IDs
	}

	for _, r := range regions {
		b := bounds(of, r.min, r.max, 8)
		results = append(results, Boundary{
			Min: b[shard-1],
			Max: b[shard],
		})
	}

	return results
}

func bounds(shards uint32, min, max uint64, dest int) [][]byte {
	if shards == 0 {
		return nil
	}

	bounds := make([][]byte, shards+1)
	for i := uint32(0); i < shards+1; i++ {
		bounds[i] = make([]byte, 16)
	}

	bucketSz := (max - min) / uint64(shards)

	// numLarger is the number of buckets that have to be bumped by 1
	numLarger := max % bucketSz

	boundary := min
	for i := uint32(0); i < shards; i++ {
		binary.BigEndian.PutUint64(bounds[i][dest:], boundary)

		boundary += bucketSz
		if numLarger != 0 {
			numLarger--
			boundary++
		}
	}

	binary.BigEndian.PutUint64(bounds[shards][dest:], max)

	return bounds
}
