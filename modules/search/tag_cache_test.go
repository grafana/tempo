package search

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSearchTagCacheGetNames(t *testing.T) {
	c := NewTagCache()
	c.setEntry(0, "k1", "v1")
	c.setEntry(0, "k1", "v2")
	require.Equal(t, []string{"k1"}, c.GetNames())
}

func TestSearchTagCacheMaxValuesPerTag(t *testing.T) {
	c := NewTagCache()

	for i := 0; i < maxValuesPerTag+1; i++ {
		c.setEntry(int64(i), "k", fmt.Sprintf("v%02d", i))
	}

	vals := c.GetValues("k")

	require.Len(t, vals, maxValuesPerTag)
	require.Equal(t, "v01", vals[0]) // oldest v0 was evicted
	require.Equal(t, fmt.Sprintf("v%02d", maxValuesPerTag), vals[maxValuesPerTag-1])
}

func TestSearchTagCachePurge(t *testing.T) {
	c := NewTagCache()

	oneMinuteAgo := time.Now().Add(-1 * time.Minute)
	twoMinutesAgo := time.Now().Add(-2 * time.Minute)

	c.setEntry(twoMinutesAgo.Unix(), "j", "a")
	c.setEntry(twoMinutesAgo.Unix(), "k", "a")
	c.setEntry(oneMinuteAgo.Unix(), "k", "b")

	c.PurgeExpired(oneMinuteAgo)

	require.Equal(t, []string{"k"}, c.GetNames())     // Empty tags purged
	require.Equal(t, []string{"b"}, c.GetValues("k")) // Old values purged
}

func BenchmarkSearchTagCacheSetEntry(b *testing.B) {
	c := NewTagCache()

	for i := 0; i < b.N; i++ {
		c.setEntry(int64(i), "k", strconv.Itoa(b.N))
	}
}
