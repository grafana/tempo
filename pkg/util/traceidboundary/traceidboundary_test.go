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
		//---------------------------------------------
		// Simplest case, 2 shards
		//---------------------------------------------
		{1, 2, []Boundary{
			{
				// First half of upper-byte = 0 (between 0x0000 and 0x00FF)
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0x80, 0, 0, 0, 0, 0, 0},
			},
			{
				// First half of upper-nibble = 0 (between 0x01 and 0x0F)
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0x01, 0, 0, 0, 0, 0, 0, 0},
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0x08, 0x80, 0, 0, 0, 0, 0, 0},
			},
			{
				// First half of upper-bit = 0 (between 0x10 and 0x7F)
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0x10, 0, 0, 0, 0, 0, 0, 0},
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0x48, 0, 0, 0, 0, 0, 0, 0},
			},
			{
				// First half of full 8-byte ids (between 0x80 and 0xFF)
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0x80, 0, 0, 0, 0, 0, 0, 0},
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0xC0, 0, 0, 0, 0, 0, 0, 0},
			},
			{
				// First half of full 16-byte ids (between 0x00 and 0x80)
				[]byte{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0},
				[]byte{0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			},
		}, false},
		{2, 2, []Boundary{
			{
				// Second half of upper-byte = 0 (between 0x0000 and 0x00FF)
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0x80, 0, 0, 0, 0, 0, 0},
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			},
			{
				// Second half of upper-nibble = 0 (between 0x01 and 0x0F)
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0x08, 0x80, 0, 0, 0, 0, 0, 0},
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0x0F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			},
			{
				// Second half of upper-bit = 0 (between 0x10 and 0x7F)
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0x48, 0, 0, 0, 0, 0, 0, 0},
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			},
			{
				// Second half of full 8-byte ids (between 0x80 and 0xFF)
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0xC0, 0, 0, 0, 0, 0, 0, 0},
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			},
			{
				// Second half of full 16-byte ids (between 0x80 and 0xFF)
				[]byte{0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			},
		}, true},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%d of %d", tc.shard, tc.of), func(t *testing.T) {
			pairs, upper := Pairs(tc.shard, tc.of)
			require.Equal(t, tc.expectedPairs, pairs)
			require.Equal(t, tc.expectedUpper, upper)
		})
	}
}
