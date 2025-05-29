package util

import (
	"bytes"
	"crypto/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashForNoCollisions(t *testing.T) {
	// Verify cases of known collisions under TokenFor don't collide in HashForTraceID.
	pairs := [][2]string{
		{"fd5980503add11f09f80f77608c1b2da", "091ea7803ade11f0998a055186ee1243"},
		{"9e0d446036dc11f09ac04988d2097052", "a61ed97036dc11f0883771db3b51b1ec"},
		{"6b27f5501eda11f09e99db1b2c23c542", "6b4149b01eda11f0b0e2a966cf7ebbc8"},
		{"3e9582202f9a11f0afb01b7c06024bd6", "370db6802f9a11f0a9a212dff3125239"},
		{"978d70802a7311f0991f350653ef0ab4", "9b66da202a7311f09d292db17ccfd31a"},
		{"de567f703bb711f0b8c377682d1667e6", "dc2d0fc03bb711f091de732fcf93048c"},
	}
	for _, pair := range pairs {
		b1, _ := HexStringToTraceID(pair[0])
		b2, _ := HexStringToTraceID(pair[1])

		t1 := HashForTraceID(b1)
		t2 := HashForTraceID(b2)

		require.NotEqual(t, t1, t2)
	}
}

// Verify HashForTraceID doesn't collide within reasonable numbers, and estimate the hash collision rate if it does.
func TestHashForCollisionRate(t *testing.T) {
	var (
		n      = 1_000_000
		tokens = map[uint64]struct{}{}
		IDs    = make([][]byte, 0, n)
	)

	for i := 0; i < n; i++ {
		traceID := make([]byte, 16)
		_, err := rand.Read(traceID)
		require.NoError(t, err)

		IDs = append(IDs, traceID)
		tokens[HashForTraceID(traceID)] = struct{}{}
	}

	// Ensure no duplicate span IDs accidentally generated
	sort.Slice(IDs, func(i, j int) bool {
		return bytes.Compare(IDs[i], IDs[j]) == -1
	})
	for i := 1; i < len(IDs); i++ {
		if bytes.Equal(IDs[i-1], IDs[i]) {
			panic("same trace ID was generated, oops")
		}
	}

	missing := n - len(tokens)
	require.Zerof(t, missing, "missing 1 out of every %.2f trace ids", float32(n)/float32(missing))
}
