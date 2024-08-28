package util

import (
	"strings"
	"testing"
	"time"

	thrift "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceInfo(t *testing.T) {
	writeBackoff := 1 * time.Second
	longWriteBackoff := 5 * time.Second

	seed := time.Unix(1632146180, 0)
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

	result := info.generateRandomTags()

	for _, k := range result {
		assertStandardVultureKey(t, k)
	}
}

func TestGenerateRandomString(t *testing.T) {
	seed := time.Unix(1632146180, 0)
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
	seed := time.Unix(1632146180, 0)
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
	seed := time.Unix(1632146180, 0)
	info := NewTraceInfo(seed, "")

	result, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)
	assert.Equal(t, 7, len(result.ResourceSpans))

	result2, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	require.Equal(t, result, result2)
}

func assertStandardVultureKey(t *testing.T, tag *thrift.Tag) {
	if !strings.HasPrefix(tag.Key, "vulture-") {
		t.Errorf("prefix vulture- is wanted, have: %s", tag.Key)
	}

	require.NotNil(t, tag.VStr)
	require.GreaterOrEqual(t, len(tag.VType.String()), 5)
	require.LessOrEqual(t, len(tag.VType.String()), 20)
}
