package collector

import (
	"fmt"
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScopedDistinct(t *testing.T) {
	tcs := []struct {
		in       map[string][]string
		expected map[string][]string
		limit    int
		exceeded bool
	}{
		{
			in: map[string][]string{
				"scope1": {"val1", "val2"},
				"scope2": {"val1", "val2"},
			},
			expected: map[string][]string{
				"scope1": {"val1", "val2"},
				"scope2": {"val1", "val2"},
			},
		},
		{
			in: map[string][]string{
				"scope1": {"val1", "val2", "val1"},
				"scope2": {"val1", "val2", "val2"},
			},
			expected: map[string][]string{
				"scope1": {"val1", "val2"},
				"scope2": {"val1", "val2"},
			},
		},
		{
			in: map[string][]string{
				"scope1": {"val1", "val2"},
				"scope2": {"val1", "val2"},
			},
			expected: map[string][]string{
				"scope1": {"val1", "val2"},
				"scope2": {"val1"},
			},
			limit:    13,
			exceeded: true,
		},
	}

	for _, tc := range tcs {
		c := NewScopedDistinctString(tc.limit)

		// get and sort keys so we can deterministically add values
		keys := []string{}
		for k := range tc.in {
			keys = append(keys, k)
		}
		slices.Sort(keys)

		for _, k := range keys {
			v := tc.in[k]
			for _, val := range v {
				c.Collect(k, val)
			}
		}

		require.Equal(t, tc.exceeded, c.Exceeded())

		actual := c.Strings()
		assertMaps(t, tc.expected, actual)
	}
}

func TestScopedDistinctDiff(t *testing.T) {
	c := NewScopedDistinctStringWithDiff(0)

	c.Collect("scope1", "val1")
	expected := map[string][]string{
		"scope1": {"val1"},
	}
	assertMaps(t, expected, c.Diff())

	// no diff
	c.Collect("scope1", "val1")
	expected = map[string][]string{}
	assertMaps(t, expected, c.Diff())
	assertMaps(t, map[string][]string{}, c.Diff())

	// new value
	c.Collect("scope1", "val2")
	expected = map[string][]string{
		"scope1": {"val2"},
	}
	assertMaps(t, expected, c.Diff())
	assertMaps(t, map[string][]string{}, c.Diff())

	// new scope
	c.Collect("scope2", "val1")
	expected = map[string][]string{
		"scope2": {"val1"},
	}
	assertMaps(t, expected, c.Diff())
	assertMaps(t, map[string][]string{}, c.Diff())

	// all
	c.Collect("scope2", "val1")
	c.Collect("scope2", "val2")
	c.Collect("scope1", "val3")
	expected = map[string][]string{
		"scope1": {"val3"},
		"scope2": {"val2"},
	}
	assertMaps(t, expected, c.Diff())
	assertMaps(t, map[string][]string{}, c.Diff())
}

func assertMaps(t *testing.T, expected, actual map[string][]string) {
	require.Equal(t, len(expected), len(actual))

	for k, v := range expected {
		require.Equal(t, v, actual[k])
	}
}

func TestScopedDistinctStringCollectorIsSafe(t *testing.T) {
	d := NewScopedDistinctString(0) // no limit

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				d.Collect(fmt.Sprintf("scope-%d", id), fmt.Sprintf("goroutine-%d-string-%d", id, j))
			}
		}(i)
	}

	wg.Wait()

	totalStrings := 0
	for _, strings := range d.Strings() {
		totalStrings += len(strings)
	}
	require.Equal(t, totalStrings, 10*100)
	require.False(t, d.Exceeded())
}

func BenchmarkScopedDistinctStringCollect(b *testing.B) {
	// simulate 100 ingesters, each returning 10_000 tags with various scopes
	numIngesters := 100
	numTagsPerIngester := 10_000
	ingesterTags := make([]map[string][]string, numIngesters)
	scopeTypes := []string{"resource", "span", "event", "instrumentation"}

	for i := 0; i < numIngesters; i++ {
		tags := make(map[string][]string)
		for j := 0; j < numTagsPerIngester; j++ {
			scope := scopeTypes[j%len(scopeTypes)]
			value := fmt.Sprintf("tag_%d_%d", i, j)
			tags[scope] = append(tags[scope], value)
		}
		ingesterTags[i] = tags
	}

	limits := []int{
		0,          // no limit
		100_000,    // 100KB
		1_000_000,  // 1MB
		10_000_000, // 10MB
	}

	b.ResetTimer() // to exclude the setup time for generating tags
	for _, lim := range limits {
		b.Run("uniques_limit:"+strconv.Itoa(lim), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				scopedDistinctStrings := NewScopedDistinctString(lim)
				for _, tags := range ingesterTags {
					for scope, values := range tags {
						for _, v := range values {
							scopedDistinctStrings.Collect(scope, v)
							if scopedDistinctStrings.Exceeded() {
								break // stop early if limit is reached
							}
						}
					}
				}
			}
		})

		b.Run("duplicates_limit:"+strconv.Itoa(lim), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				scopedDistinctStrings := NewScopedDistinctString(lim)
				for i := 0; i < numIngesters; i++ {
					for scope := range ingesterTags[i] {
						// collect first item to simulate duplicates
						scopedDistinctStrings.Collect(scope, ingesterTags[i][scope][0])
						if scopedDistinctStrings.Exceeded() {
							break // stop early if limit is reached
						}
					}
				}
			}
		})
	}
}
