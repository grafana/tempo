package traceidboundary

import (
	"bytes"
	"encoding/binary"

	"github.com/grafana/tempo/pkg/blockboundary"
)

const defaultBitSharding = 12

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
func Pairs(shard, of uint32) (boundaries []Boundary, upperInclusive bool) {
	return PairsWithBitSharding(shard, of, defaultBitSharding)
}

// PairsWithBitSharding allows choosing a specific level of sub-sharding.
func PairsWithBitSharding(shard, of uint32, bits int) (boundaries []Boundary, upperInclusive bool) {
	if bits > 0 {
		boundaries = append(boundaries, complicatedShardingFor8ByteIDs(shard, of, bits)...)
	}

	// Final pair is the normal full precision 16-byte IDs.
	int128bounds := blockboundary.CreateBlockBoundaries(int(of))
	if bits > 0 {
		// Avoid overlap with the 64-bit precision ones. I.e. a minimum of 0x0000.... would
		// unintentionally include all 64-bit IDs. The first 65-bit ID starts here:
		int128bounds[0] = []byte{0, 0, 0, 0, 0, 0, 0, 0x01, 0, 0, 0, 0, 0, 0, 0, 0}
	}

	boundaries = append(boundaries, Boundary{
		Min: int128bounds[shard-1],
		Max: int128bounds[shard],
	})

	// Top most 0xFFFFF... boundary is inclusive
	upperInclusive = shard == of

	return
}

// Funcs returns helper functions that match trace IDs in the given shard.
func Funcs(shard, of uint32) (testSingle func([]byte) bool, testRange func([]byte, []byte) bool) {
	return FuncsWithBitSharding(shard, of, defaultBitSharding)
}

// FuncsWithBitSharding is like Funcs but allows choosing a specific level of sub-sharding.
func FuncsWithBitSharding(shard, of uint32, bits int) (testSingle func([]byte) bool, testRange func([]byte, []byte) bool) {
	pairs, upperInclusive := PairsWithBitSharding(shard, of, bits)

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
// the 64-bit space by the given number of bits. This seems like overkill but in practice
// 8-byte IDs are unevenly weighted towards lower values starting with zeros.  The benefit of
// this approach is better fairness across shards, and also *invariance* across workloads,
// no matter if your instrumentation is generating 8-byte or 16-byte trace IDs.
func complicatedShardingFor8ByteIDs(shard, of uint32, bits int) []Boundary {
	// This function takes a trace ID boundary and shifts it down to the
	// same space by the given number of bits.
	// For example shard 2 of 4 has the boundary:
	//		0x40	b0100
	//		0x80	b1000
	// Shifting by 1 bit gives shard 2 of 4 in 64-bit-only space:
	//				 |
	//				 v
	//		0xA0	b1010
	//		0xC0	b1100
	// Shifting by 2 bits gives shard 2 of 4 in 63-bit-only space:
	//				  |
	// 				  v
	//      0x50	b0101
	//      0x60	b0110
	// ... and so on
	cloneRotateAndSet := func(v []byte, right int) []byte {
		v2 := binary.BigEndian.Uint64(v)
		v2 >>= right
		v2 |= 0x01 << (64 - right)

		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, v2)
		return buf
	}

	var boundaries []Boundary
	original := blockboundary.CreateBlockBoundaries(int(of))

	for i := bits; i >= 1; i-- {
		min := cloneRotateAndSet(original[shard-1], i)
		max := cloneRotateAndSet(original[shard], i)

		if i == bits && shard == 1 {
			// We don't shard below this, so its minimum is absolute zero.
			clear(min)
		}
		boundaries = append(boundaries, Boundary{
			Min: append([]byte{0, 0, 0, 0, 0, 0, 0, 0}, min[0:8]...),
			Max: append([]byte{0, 0, 0, 0, 0, 0, 0, 0}, max[0:8]...),
		})
	}

	return boundaries
}
