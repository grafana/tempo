package traceql

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBucketSet_Bucket(t *testing.T) {
	testCases := []struct {
		name     string
		start    uint64
		end      uint64
		ts       uint64
		expected int
	}{
		{
			name:     "timestamp at start",
			start:    100,
			end:      200,
			ts:       100,
			expected: 0,
		},
		{
			name:     "timestamp at end",
			start:    100,
			end:      200,
			ts:       200,
			expected: 49, // Should be last bucket (50 buckets, 0-49)
		},
		{
			name:     "timestamp in middle",
			start:    100,
			end:      200,
			ts:       150,
			expected: 25, // Middle bucket
		},
		{
			name:     "timestamp just past middle",
			start:    100,
			end:      200,
			ts:       151,
			expected: 25, // Should be in the same bucket as 150
		},
		{
			name:     "start equals end",
			start:    100,
			end:      100,
			ts:       100,
			expected: 0, // Should be the first and only bucket
		},
		{
			name:     "timestamp almost at end",
			start:    100,
			end:      200,
			ts:       199,
			expected: 49, // Should be in the last bucket
		},
		{
			name:     "large range",
			start:    0,
			end:      1000000,
			ts:       500000,
			expected: 25, // Middle bucket
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bs := newBucketSet(maxExemplars, tc.start*uint64(time.Second.Nanoseconds()), tc.end*uint64(time.Second.Nanoseconds())) //nolint: gosec // G115
			actual := bs.bucket(tc.ts * uint64(time.Second.Milliseconds()))                                                        //nolint: gosec // G115
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestBucketSet_Instant(t *testing.T) {
	start := uint64(10)
	end := uint64(1010)
	step := end - start

	bs := newExemplarBucketSet(maxExemplars, start, end, step)
	assert.True(t, bs.testTotal())
	assert.True(t, bs.addAndTest(start-1))
	assert.True(t, bs.addAndTest(start))
	assert.True(t, bs.addAndTest(start+1))
}

func TestBucketSet(t *testing.T) {
	s := newBucketSet(maxExemplars, uint64(100*time.Second.Nanoseconds()), uint64(199*time.Second.Nanoseconds())) //nolint: gosec // G115

	// Add two to each bucket
	for ts := uint64(100); ts <= 199; ts += 2 { // 100 in total
		tsMilli := uint64(int64(ts) * time.Second.Milliseconds()) //nolint: gosec // G115

		assert.False(t, s.addAndTest(tsMilli), "ts=%d should be added to bucket", ts)
		assert.False(t, s.addAndTest(tsMilli), "ts=%d should be added to bucket", ts)
	}
	assert.Equal(t, 50*2, s.len())

	// Should be full and reject new adds
	assert.True(t, s.testTotal())
	for ts := uint64(100); ts <= 199; ts += 2 {
		tsMilli := uint64(int64(ts) * time.Second.Milliseconds()) //nolint: gosec // G115
		assert.True(t, s.addAndTest(tsMilli), "ts=%d should be added to bucket", ts)
		assert.True(t, s.addAndTest(tsMilli), "ts=%d should be added to bucket", ts)
	}
	assert.True(t, s.addAndTest(0))
}

func TestBucketSetSingleExemplar(t *testing.T) {
	s := newBucketSet(1, uint64(100*time.Second.Nanoseconds()), uint64(199*time.Second.Nanoseconds())) //nolint: gosec // G115
	tsMilli := uint64(100 * time.Second.Milliseconds())                                                //nolint: gosec // G115
	assert.False(t, s.addAndTest(tsMilli), "ts=%d should be added to bucket", 100)
	assert.True(t, s.addAndTest(tsMilli), "ts=%d should not be added to bucket", 100)
}
