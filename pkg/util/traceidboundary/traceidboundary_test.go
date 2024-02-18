package traceidboundary

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPairs(t *testing.T) {
	testCases := []struct {
		shard, of     uint32
		bits          int
		expectedPairs []Boundary
		expectedUpper bool
	}{
		//---------------------------------------------
		// Simplest case, no sub-sharding,
		// 4 shards all at the top level.
		//---------------------------------------------
		{
			1, 4, 0, []Boundary{
				{
					[]byte{0x00, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					[]byte{0x40, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				},
			}, false,
		},
		{
			2, 4, 0, []Boundary{
				{
					[]byte{0x40, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					[]byte{0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				},
			}, false,
		},
		{
			3, 4, 0, []Boundary{
				{
					[]byte{0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					[]byte{0xC0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				},
			}, false,
		},
		{
			4, 4, 0, []Boundary{
				{
					[]byte{0xC0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
				},
			}, true,
		},

		//---------------------------------------------
		// Sub-sharding of 1 bit down into 64-bit IDs.
		//---------------------------------------------
		{
			1, 2, 1, []Boundary{
				{
					[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},    // Min value overall is always zero
					[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0xC0, 0, 0, 0, 0, 0, 0, 0}, // Half of 64-bit space (exlusive)
				},
				{
					[]byte{0, 0, 0, 0, 0, 0, 0, 0x01, 0, 0, 0, 0, 0, 0, 0, 0}, // Min 65-bit value (not overlapping with max 64-bit value)
					[]byte{0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, // Half of 128-bit space (exlusive)
				},
			}, false,
		},
		{
			2, 2, 1, []Boundary{
				{
					[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0xC0, 0, 0, 0, 0, 0, 0, 0},                      // Half of 64-bit space
					[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, // Max 64-bit space (inclusive)
				},
				{
					[]byte{0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},                                              // Half of 128-bit space (not overlapping with max 64-bit value)
					[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, // Max 128-bit value (inclusive)
				},
			}, true,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%d of %d", tc.shard, tc.of), func(t *testing.T) {
			pairs, upper := PairsWithBitSharding(tc.shard, tc.of, tc.bits)
			require.Equal(t, tc.expectedPairs, pairs)
			require.Equal(t, tc.expectedUpper, upper)
		})
	}
}
