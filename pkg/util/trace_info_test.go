package util

import (
	"strings"
	"testing"
	"time"

	thrift "github.com/jaegertracing/jaeger-idl/thrift-gen/jaeger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSeed = 1632146180

func TestTraceInfo(t *testing.T) {
	writeBackoff := 1 * time.Second
	longWriteBackoff := 5 * time.Second

	seed := time.Unix(0, testSeed)
	info := NewTraceInfo(seed, "")
	assert.False(t, info.Ready(seed, writeBackoff, longWriteBackoff))
	assert.False(t, info.Ready(seed.Add(longWriteBackoff), writeBackoff, longWriteBackoff))
	assert.False(t, info.Ready(seed.Add(longWriteBackoff).Add(1*time.Second), writeBackoff, longWriteBackoff))
	assert.True(t, info.Ready(seed.Add(2*longWriteBackoff), writeBackoff, longWriteBackoff))
}

func TestGenerateRandomLogs(t *testing.T) {
	now := time.Now()
	info := NewTraceInfo(now, "")
	result := info.generateRandomLogs()

	for _, l := range result {
		require.NotNil(t, l.Timestamp)
		for _, f := range l.Fields {
			assertStandardVultureKey(t, f)
		}
	}
}

func TestGenerateRandomTags(t *testing.T) {
	now := time.Now()
	info := NewTraceInfo(now, "")

	result := info.generateRandomTagsWithPrefix("vulture")

	for _, k := range result {
		assertStandardVultureKey(t, k)
	}
}

func TestGenerateRandomString(t *testing.T) {
	seed := time.Unix(0, testSeed)
	info := NewTraceInfo(seed, "")

	strings := []string{
		"XqaIBSJMJVGkEg",
	}

	for _, s := range strings {
		result := info.generateRandomString()
		require.Equal(t, s, result)
	}
}

func TestGenerateRandomInt(t *testing.T) {
	seed := time.Unix(0, testSeed)
	info := NewTraceInfo(seed, "")

	cases := []struct {
		min    int64
		max    int64
		result int64
	}{
		{
			min:    1,
			max:    5,
			result: 3,
		},
		{
			min:    10,
			max:    50,
			result: 17,
		},
		{
			min:    1,
			max:    3,
			result: 2,
		},
	}

	for _, tc := range cases {
		result := info.generateRandomInt(tc.min, tc.max)
		require.Equal(t, tc.result, result)
	}
}

func TestConstructTraceFromEpoch(t *testing.T) {
	seed := time.Unix(0, testSeed)
	info := NewTraceInfo(seed, "")

	result, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)
	// Batch count is deterministic per seed; adding well-known attributes changes RNG sequence (now 8 batches for testSeed).
	assert.Equal(t, 8, len(result.ResourceSpans))

	result2, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	require.Equal(t, result, result2)
}

// assertStandardVultureKey checks that a tag has the expected type and value constraints.
// All vulture-generated keys use a "vulture" prefix (e.g. vulture-0, vulture-process-string-01, vulture-event-blob-01).
func assertStandardVultureKey(t *testing.T, tag *thrift.Tag) {
	require.NotEmpty(t, tag.Key)
	switch {
	case strings.Contains(tag.Key, "-blob-"):
		require.NotNil(t, tag.VStr, "blob tag %s should have VStr", tag.Key)
		require.Len(t, *tag.VStr, vultureBlobSize, "blob %s should be exactly %d bytes", tag.Key, vultureBlobSize)
	case strings.Contains(tag.Key, "-int-"):
		require.NotNil(t, tag.VLong, "int tag %s should have VLong", tag.Key)
	case strings.Contains(tag.Key, "-string-") || strings.HasPrefix(tag.Key, "vulture"):
		// Fixed string attrs (vulture-string-01, etc.) and random tags (vulture-0, vulture-process-0, vulture-event-0)
		require.NotNil(t, tag.VStr, "string tag %s should have VStr", tag.Key)
		require.GreaterOrEqual(t, len(*tag.VStr), 5, "string %s length", tag.Key)
		require.LessOrEqual(t, len(*tag.VStr), 20, "string %s length", tag.Key)
	default:
		// well-known keys (cluster, http.method, etc.)
	}
}
