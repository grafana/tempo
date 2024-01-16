package traceidboundary

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPairs(t *testing.T) {
	testCases := []struct {
		shard, of     uint32
		expectedPairs []Boundary
		expectedUpper bool
	}{
		{
			1, 2,
			[]Boundary{
				{
					[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},    // Min 63-bit value
					[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0x40, 0, 0, 0, 0, 0, 0, 0}, // Half of 63-bit space (exlusive)
				},
				{
					[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0x80, 0, 0, 0, 0, 0, 0, 0}, // Min 64-bit value (not overlapping with max 63-bit value)
					[]byte{0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, // Half of 128-bit space (exlusive)
				},
			},
			false,
		},
		{
			2, 2, []Boundary{
				{
					[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0x40, 0, 0, 0, 0, 0, 0, 0},                      // Half of 63-bit space
					[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, // Max 63-bit space (inclusive)
				},
				{
					[]byte{0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},                                              // Half of 128-bit space (not overlapping with max 63-bit value)
					[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, // Max 128-bit value (inclusive)
				},
			}, true,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%d of %d", tc.shard, tc.of), func(t *testing.T) {
			pairs, upper := Pairs(tc.shard, tc.of)
			require.Equal(t, tc.expectedPairs, pairs)
			require.Equal(t, tc.expectedUpper, upper)
		})
	}
}
