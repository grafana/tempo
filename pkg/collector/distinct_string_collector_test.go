package collector

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDistinctStringCollector(t *testing.T) {
	d := NewDistinctString(12, 0, 0)

	require.True(t, d.Collect("123"))
	require.True(t, d.Collect("4567"))
	require.True(t, d.Collect("890"))
	require.False(t, d.Collect("11"))

	require.True(t, d.Exceeded())
	stringsSlicesEqual(t, []string{"123", "4567", "890"}, d.Strings())

	// diff fails when diff is not enabled
	res, err := d.Diff()
	require.Nil(t, res)
	require.Error(t, err, errDiffNotEnabled)
}

func TestDistinctStringCollectorWithMaxItemsLimit(t *testing.T) {
	d := NewDistinctString(0, 3, 0)

	require.True(t, d.Collect("123"))
	require.True(t, d.Collect("4567"))
	require.True(t, d.Collect("890"))
	require.False(t, d.Collect("11"))

	require.True(t, d.Exceeded())
	stringsSlicesEqual(t, []string{"123", "4567", "890"}, d.Strings())

	// diff fails when diff is not enabled
	res, err := d.Diff()
	require.Nil(t, res)
	require.Error(t, err, errDiffNotEnabled)
}

func TestDistinctStringCollectorWitCacheHitsLimit(t *testing.T) {
	d := NewDistinctString(0, 0, 3)

	require.True(t, d.Collect("123"))
	require.True(t, d.Collect("4567"))
	require.True(t, d.Collect("890"))
	require.False(t, d.Collect("890"))
	require.True(t, d.Collect("11")) // The counter resets with every new value
	require.False(t, d.Collect("890"))
	require.False(t, d.Collect("890"))
	require.False(t, d.Collect("890"))
	require.False(t, d.Collect("12"))

	require.True(t, d.Exceeded())
	stringsSlicesEqual(t, []string{"123", "4567", "890", "11"}, d.Strings())

	// diff fails when diff is not enabled
	res, err := d.Diff()
	require.Nil(t, res)
	require.Error(t, err, errDiffNotEnabled)
}

func TestDistinctStringCollectorDiff(t *testing.T) {
	d := NewDistinctStringWithDiff(0, 0, 0)

	require.True(t, d.Collect("123"))
	require.True(t, d.Collect("4567"))

	stringsSlicesEqual(t, []string{"123", "4567"}, readDistinctStringDiff(t, d))
	stringsSlicesEqual(t, []string{}, readDistinctStringDiff(t, d))

	require.False(t, d.Collect("123"))
	require.True(t, d.Collect("890"))

	stringsSlicesEqual(t, []string{"890"}, readDistinctStringDiff(t, d))
	stringsSlicesEqual(t, []string{}, readDistinctStringDiff(t, d))
}

func readDistinctStringDiff(t *testing.T, d *DistinctString) []string {
	res, err := d.Diff()
	require.NoError(t, err)
	return res
}

func TestDistinctStringCollectorIsSafe(t *testing.T) {
	d := NewDistinctString(0, 0, 0) // no limit
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				d.Collect(fmt.Sprintf("goroutine-%d-string-%d", id, j))
			}
		}(i)
	}
	wg.Wait()

	require.Equal(t, len(d.Strings()), 10*100)
	require.False(t, d.Exceeded())
}

func BenchmarkDistinctStringCollect(b *testing.B) {
	// simulate 100 ingesters, each returning 10_000 tag values
	numIngesters := 100
	numTagValuesPerIngester := 10_000
	ingesterStrings := make([][]string, numIngesters)
	for i := 0; i < numIngesters; i++ {
		strings := make([]string, numTagValuesPerIngester)
		for j := 0; j < numTagValuesPerIngester; j++ {
			strings[j] = fmt.Sprintf("string_%d_%d", i, j)
		}
		ingesterStrings[i] = strings
	}

	limits := []int{
		0,          // no limit
		100_000,    // 100KB
		1_000_000,  // 1MB
		10_000_000, // 10MB
	}

	b.ResetTimer() // to exclude the setup time for generating tag values
	for _, lim := range limits {
		b.Run("uniques_limit:"+strconv.Itoa(lim), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				distinctStrings := NewDistinctString(lim, 0, 0)
				for _, values := range ingesterStrings {
					for _, v := range values {
						if distinctStrings.Collect(v) {
							break // stop early if limit is reached
						}
					}
				}
			}
		})

		b.Run("duplicates_limit:"+strconv.Itoa(lim), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				distinctStrings := NewDistinctString(lim, 0, 0)
				for i := 0; i < numIngesters; i++ {
					for j := 0; j < numTagValuesPerIngester; j++ {
						// collect first item to simulate duplicates
						if distinctStrings.Collect(ingesterStrings[i][0]) {
							break // stop early if limit is reached
						}
					}
				}
			}
		})
	}
}
