package cache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProbabilisticCacheStore(t *testing.T) {
	testCases := []struct {
		name          string
		probability   float64
		storeAttempts int
		expectedCalls int
		delta         float64
	}{
		{
			name:          "zero never stores",
			probability:   0,
			storeAttempts: 100,
			expectedCalls: 0,
			delta:         0,
		},
		{
			name:          "one always stores",
			probability:   1,
			storeAttempts: 100,
			expectedCalls: 100,
			delta:         0,
		},
		{
			name:          "50%",
			probability:   0.5,
			storeAttempts: 1000,
			expectedCalls: 500,
			delta:         75, // probability of false positive: 1 every 600k runs
		},
		{
			name:          "75%",
			probability:   0.75,
			storeAttempts: 1000,
			expectedCalls: 750,
			delta:         75, // probability of false positive: 1 every 21M runs
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wrapped := &storeRecorder{}
			cache := NewProbabilistic(wrapped, tc.probability)

			keys := []string{"key"}
			bufs := [][]byte{[]byte("value")}
			for range tc.storeAttempts {
				cache.Store(context.Background(), keys, bufs)
			}

			require.InDelta(t, tc.expectedCalls, wrapped.storeCalls, tc.delta)

			if tc.expectedCalls > 0 {
				require.Equal(t, keys, wrapped.keys)
				require.Equal(t, bufs, wrapped.bufs)
			}
		})
	}
}

type storeRecorder struct {
	Cache

	storeCalls int
	keys       []string
	bufs       [][]byte
}

func (c *storeRecorder) Store(_ context.Context, keys []string, bufs [][]byte) {
	c.storeCalls++
	c.keys = keys
	c.bufs = bufs
}
