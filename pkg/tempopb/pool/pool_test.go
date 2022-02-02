// Forked with love from: https://github.com/prometheus/prometheus/tree/c954cd9d1d4e3530be2939d39d8633c38b70913f/util/pool

package pool

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func makeFunc(size int) []byte {
	return make([]byte, 0, size)
}

func TestPool(t *testing.T) {
	testPool := New(1, 8, 2, makeFunc)
	cases := []struct {
		size        int
		expectedCap int
	}{
		{
			size:        -1,
			expectedCap: 1,
		},
		{
			size:        3,
			expectedCap: 4,
		},
		{
			size:        10,
			expectedCap: 10,
		},
	}
	for _, c := range cases {
		ret := testPool.Get(c.size)
		require.Equal(t, c.expectedCap, cap(ret))
		testPool.Put(ret)
	}
}

func TestPoolSlicesAreAlwaysLargeEnough(t *testing.T) {
	testPool := New(1, 1024, 2, makeFunc)

	for i := 0; i < 10000; i++ {
		size := rand.Intn(1000)
		externalSlice := make([]byte, 0, size)
		testPool.Put(externalSlice)

		size = rand.Intn(1000)
		ret := testPool.Get(size)

		require.True(t, cap(ret) >= size)
	}
}

// TestPoolReusesSlice checks to make sure if a slice is reused in the pool. Since this depends on
// the underlying sync.Pool implementation it's a bad-ish test and should be removed if it gets flakey.
// Added to confirm that the Get/Put []byte methods choose the same buckets.
func TestPoolReusesSlice(t *testing.T) {
	testPool := New(1, 1024, 2, makeFunc)
	size := 33
	mark := byte(0x42)
	markPos := 5

	// get a slice and mark it so we can check its reused
	s := testPool.Get(size)[:size]
	s[markPos] = mark
	testPool.Put(s)

	// request a slice of the same size
	s = testPool.Get(size)[:size]
	require.Equal(t, mark, s[markPos])
}
