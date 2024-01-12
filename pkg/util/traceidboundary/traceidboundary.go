package traceidboundary

import (
	"bytes"

	"github.com/grafana/tempo/pkg/blockboundary"
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
//   - Technically 8-byte IDs are only 63 bits, so we account for this
//   - The boundaries are inclusive/exclusive: [min, max), except the max of the last shard
//     is the valid ID FFFFF... and inclusive/inclusive.
func Pairs(shard, of uint32) (boundaries []Boundary, upperInclusive bool) {
	// First pair is 63-bit IDs left-padded with zeroes to make 16-byte divisions
	// that matches the 16-byte layout in the block.
	// To create 63-bit boundaries we create twice as many as needed,
	// then only use the first half.  i.e. shaving off the top-most bit.
	int63bounds := blockboundary.CreateBlockBoundaries(int(of * 2))

	// Adjust last boundary to be inclusive so it matches the other pair.
	int63bounds[of] = []byte{0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	boundaries = append(boundaries, Boundary{
		Min: append([]byte{0, 0, 0, 0, 0, 0, 0, 0}, int63bounds[shard-1][0:8]...),
		Max: append([]byte{0, 0, 0, 0, 0, 0, 0, 0}, int63bounds[shard][0:8]...),
	})

	// Second pair is normal full precision 16-byte IDs.
	int128bounds := blockboundary.CreateBlockBoundaries(int(of))

	// However there is one caveat - We adjust the very first boundary to ensure it doesn't
	// overlap with the 63-bit precision ones. I.e. a minimum of 0x0000.... would
	// unintentionally include all 63-bit IDs.
	// The first 64-bit ID starts here:
	int128bounds[0] = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0x80, 0, 0, 0, 0, 0, 0, 0}

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
	pairs, upperInclusive := Pairs(shard, of)

	upper := -1
	if upperInclusive {
		upper = 0
	}

	isMatch := func(id []byte) bool {
		for _, p := range pairs {
			if bytes.Compare(p.Min, id) <= 0 && bytes.Compare(id, p.Max) <= upper {
				// fmt.Printf("TraceID: %16X true\n", id)
				return true
			}
		}
		// fmt.Printf("TraceID: %16X false\n", id)
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
