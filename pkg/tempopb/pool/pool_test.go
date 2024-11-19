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
	testPool := New(20, 4, makeFunc)
	cases := []struct {
		size        int
		expectedCap int
	}{
		{
			size:        -5,
			expectedCap: 4,
		},
		{
			size:        0,
			expectedCap: 4,
		},
		{
			size:        3,
			expectedCap: 4,
		},
		{
			size:        10,
			expectedCap: 12,
		},
		{
			size:        23,
			expectedCap: 23,
		},
	}
	for _, c := range cases {
		ret := testPool.Get(c.size)
		require.Equal(t, c.expectedCap, cap(ret))
		testPool.Put(ret)
	}
}

func TestPoolSlicesAreAlwaysLargeEnough(t *testing.T) {
	testPool := New(1025, 5, makeFunc)

	for i := 0; i < 10000; i++ {
		size := rand.Intn(1000)
		externalSlice := make([]byte, 0, size)
		testPool.Put(externalSlice)

		size = rand.Intn(1000)
		ret := testPool.Get(size)

		require.True(t, cap(ret) >= size)

		testPool.Put(ret)
	}
}
