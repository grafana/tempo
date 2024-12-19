// Forked with love from: https://github.com/prometheus/prometheus/tree/c954cd9d1d4e3530be2939d39d8633c38b70913f/util/pool

package pool

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPoolGet(t *testing.T) {
	testPool := New(5, 2, 7)
	cases := []struct {
		size        int
		expectedCap int
		tooLarge    bool
	}{
		{ // under the smallest pool size, should return an unaligned slice
			size:        3,
			expectedCap: 3,
		},
		{ // minBucket is exclusive. 5 is technically an unaligned slice
			size:        5,
			expectedCap: 5,
		},
		{
			size:        6,
			expectedCap: 12,
		},
		{
			size:        12,
			expectedCap: 12,
		},
		{
			size:        15,
			expectedCap: 19,
		},
		{ // over the largest pool size, should return an unaligned slice
			size:        20,
			expectedCap: 20,
			tooLarge:    true,
		},
	}
	for _, c := range cases {
		for i := 0; i < 10; i++ {
			ret := testPool.Get(c.size)
			require.Equal(t, c.expectedCap, cap(ret))
			putBucket := testPool.Put(ret)

			if !c.tooLarge {
				require.Equal(t, testPool.bucketFor(cap(ret)), putBucket)
			}
		}
	}
}

func TestPoolSlicesAreAlwaysLargeEnough(t *testing.T) {
	testPool := New(100, 200, 5)

	for i := 0; i < 10000; i++ {
		size := rand.Intn(1000)
		externalSlice := make([]byte, 0, size)
		testPool.Put(externalSlice)

		size = rand.Intn(1000)
		ret := testPool.Get(size)

		require.True(t, cap(ret) >= size, "cap: %d, size: %d", cap(ret), size)

		testPool.Put(ret)
	}
}

func TestBucketFor(t *testing.T) {
	testPool := New(5, 10, 5)
	cases := []struct {
		size     int
		expected int
	}{
		{
			size:     0,
			expected: -1,
		},
		{
			size:     5,
			expected: -1,
		},
		{
			size:     6,
			expected: 0,
		},
		{
			size:     10,
			expected: 0,
		},
		{
			size:     11,
			expected: 1,
		},
		{
			size:     15,
			expected: 1,
		},
		{
			size:     16,
			expected: 2,
		},
	}
	for _, c := range cases {
		ret := testPool.bucketFor(c.size)
		require.Equal(t, c.expected, ret, "size: %d", c.size)
	}
}
