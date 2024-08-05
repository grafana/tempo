package traceql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBucketSet(t *testing.T) {
	buckets := 50
	s := newBucketSet(buckets)

	// Add two to each bucket
	for i := 0; i < buckets; i++ {
		require.False(t, s.addAndTest(i))
		require.False(t, s.addAndTest(i))
	}
	require.Equal(t, buckets*2, s.len())

	// Should be full and reject new adds
	require.True(t, s.testTotal())
	require.True(t, s.addAndTest(0))
}
