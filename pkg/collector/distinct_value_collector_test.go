package collector

import (
	"fmt"
	"sort"
	"strconv"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func TestDistinctValueCollectorDiff(t *testing.T) {
	d := NewDistinctValueWithDiff[string](0, func(s string) int { return len(s) })

	d.Collect("123")
	d.Collect("4567")

	stringsSlicesEqual(t, []string{"123", "4567"}, d.Diff())
	stringsSlicesEqual(t, []string{}, d.Diff())

	d.Collect("123")
	d.Collect("890")

	stringsSlicesEqual(t, []string{"890"}, d.Diff())
	stringsSlicesEqual(t, []string{}, d.Diff())
}

func stringsSlicesEqual(t *testing.T, a, b []string) {
	sort.Strings(a)
	sort.Strings(b)
	require.Equal(t, a, b)
}

func BenchmarkCollect(b *testing.B) {
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
		ingesterTagValues[i] = tagValues
	}

	limits := []int{
		0,          // no limit
		100_000,    // 100KB
		1_000_000,  // 1MB
		10_000_000, // 10MB
	}

	b.ResetTimer() // to exclude the setup time for generating tag values
	for _, lim := range limits {
		b.Run("limit:"+strconv.Itoa(lim), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				// NewDistinctValue is collecting tag values without diff support
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
	}
}
