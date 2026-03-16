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
			bs := newBucketSet(100, tc.start*uint64(time.Second.Nanoseconds()), tc.end*uint64(time.Second.Nanoseconds())) //nolint: gosec // G115
			actual := bs.bucket(tc.ts * uint64(time.Second.Milliseconds()))                                               //nolint: gosec // G115
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestBucketSet_Instant(t *testing.T) {
	start := uint64(10)
	end := uint64(1010)
	step := end - start

	bs := newExemplarBucketSet(100, start, end, step, true)
	assert.True(t, bs.testTotal())
	assert.True(t, bs.addAndTest(start-1))
	assert.True(t, bs.addAndTest(start))
	assert.True(t, bs.addAndTest(start+1))
}

func TestBucketSet(t *testing.T) {
	s := newBucketSet(100, uint64(100*time.Second.Nanoseconds()), uint64(199*time.Second.Nanoseconds())) //nolint: gosec // G115

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

func TestBucketSetLargeExemplarsShortRange(t *testing.T) {
	// exemplars=10000 → buckets=5000, but the range is only 1 second (1000ms interval).
	// Without the guard, bucketWidth=0 causes a divide-by-zero in bucket().
	s := newBucketSet(10000, 0, uint64(time.Second.Nanoseconds())) //nolint: gosec // G115
	assert.NotPanics(t, func() {
		s.addAndTest(500) // 500ms into the range
	})
	assert.False(t, s.testTotal(), "should not be full after one exemplar")
}

func TestBucketSetZeroExemplars(t *testing.T) {
	// exemplars=0 means collection is disabled: testTotal() should always return true
	// and no exemplars should ever be accepted.
	s := newBucketSet(0, uint64(100*time.Second.Nanoseconds()), uint64(200*time.Second.Nanoseconds())) //nolint: gosec // G115
	assert.True(t, s.testTotal(), "bucket set with 0 exemplars should always report full")
	tsMilli := uint64(150 * time.Second.Milliseconds()) //nolint: gosec // G115
	assert.True(t, s.addAndTest(tsMilli), "adding to a 0-exemplar bucket set should be rejected")
	assert.Equal(t, 0, s.len())
}
