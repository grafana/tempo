package collector

import (
	"fmt"
	"sort"
	"strconv"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func TestDistinctValueCollector(t *testing.T) {
	d := NewDistinctValue[string](10, func(s string) int { return len(s) })

	var stop bool
	stop = d.Collect("123")
	require.False(t, stop)
	stop = d.Collect("4567")
	require.False(t, stop)
	stop = d.Collect("890")
	require.True(t, stop)

	require.True(t, d.Exceeded())
	require.Equal(t, stop, d.Exceeded()) // final stop should be same as Exceeded
	stringsSlicesEqual(t, []string{"123", "4567"}, d.Values())

	// diff fails when diff is not enabled
	res, err := d.Diff()
	require.Nil(t, res)
	require.Error(t, err, errDiffNotEnabled)
}

func TestDistinctValueCollectorDiff(t *testing.T) {
	d := NewDistinctValueWithDiff[string](0, func(s string) int { return len(s) })

	d.Collect("123")
	d.Collect("4567")

	stringsSlicesEqual(t, []string{"123", "4567"}, readDistinctValueDiff(t, d))
	stringsSlicesEqual(t, []string{}, readDistinctValueDiff(t, d))

	d.Collect("123")
	d.Collect("890")

	stringsSlicesEqual(t, []string{"890"}, readDistinctValueDiff(t, d))
	stringsSlicesEqual(t, []string{}, readDistinctValueDiff(t, d))
}

func readDistinctValueDiff(t *testing.T, d *DistinctValue[string]) []string {
	res, err := d.Diff()
	require.NoError(t, err)
	return res
}

func stringsSlicesEqual(t *testing.T, a, b []string) {
	sort.Strings(a)
	sort.Strings(b)
	require.Equal(t, a, b)
}

func BenchmarkDistinctValueCollect(b *testing.B) {
	// simulate 100 ingesters, each returning 10_000 tag values
	numIngesters := 100
	numTagValuesPerIngester := 10_000
	ingesterTagValues := make([][]tempopb.TagValue, numIngesters)
	for i := 0; i < numIngesters; i++ {
		tagValues := make([]tempopb.TagValue, numTagValuesPerIngester)
		for j := 0; j < numTagValuesPerIngester; j++ {
			tagValues[j] = tempopb.TagValue{
				Type:  "string",
				Value: fmt.Sprintf("value_%d_%d", i, j),
			}
		}
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
				distinctValues := NewDistinctValue(lim, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) })
				for _, tagValues := range ingesterTagValues {
					for _, v := range tagValues {
						if distinctValues.Collect(v) {
							break // stop early if limit is reached
						}
					}
				}
			}
		})

		b.Run("duplicates_limit:"+strconv.Itoa(lim), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				distinctValues := NewDistinctValue(lim, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) })
				for i := 0; i < numIngesters; i++ {
					for j := 0; j < numTagValuesPerIngester; j++ {
						// collect first item to simulate duplicates
						if distinctValues.Collect(ingesterTagValues[i][0]) {
							break // stop early if limit is reached
						}
					}
				}
			}
		})
	}
}
