package tempofb

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchDataMap(t *testing.T) {
	testCases := []struct {
		name string
		impl SearchDataMap
	}{
		{"SearchDataMapSmall", &SearchDataMapSmall{}},
		{"SearchDataMapLarge", &SearchDataMapLarge{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			searchDataMap := tc.impl

			assert.False(t, searchDataMap.Contains("key-1", "value-1-2"))

			searchDataMap.Add("key-1", "value-1-1")

			assert.False(t, searchDataMap.Contains("key-1", "value-1-2"))

			searchDataMap.Add("key-1", "value-1-2")
			searchDataMap.Add("key-2", "value-2-1")

			assert.True(t, searchDataMap.Contains("key-1", "value-1-2"))
			assert.False(t, searchDataMap.Contains("key-2", "value-1-2"))

			type Pair struct {
				k string
				v string
			}
			var pairs []Pair
			capturePairFn := func(k, v string) {
				pairs = append(pairs, Pair{k, v})
			}

			searchDataMap.Range(capturePairFn)
			assert.ElementsMatch(t, []Pair{{"key-1", "value-1-1"}, {"key-1", "value-1-2"}, {"key-2", "value-2-1"}}, pairs)

			var strs []string
			captureSliceFn := func(value string) {
				strs = append(strs, value)
			}

			searchDataMap.RangeKeys(captureSliceFn)
			assert.ElementsMatch(t, []string{"key-1", "key-2"}, strs)
			strs = nil

			searchDataMap.RangeKeyValues("key-1", captureSliceFn)
			assert.ElementsMatch(t, []string{"value-1-1", "value-1-2"}, strs)
			strs = nil

			searchDataMap.RangeKeyValues("key-2", captureSliceFn)
			assert.ElementsMatch(t, []string{"value-2-1"}, strs)
			strs = nil

			searchDataMap.RangeKeyValues("does-not-exist", captureSliceFn)
			assert.ElementsMatch(t, []string{}, strs)
			strs = nil
		})
	}
}

func BenchmarkSearchDataMapAdd(b *testing.B) {
	intfs := []struct {
		name string
		f    func() SearchDataMap
	}{
		{"SearchDataMapSmall", func() SearchDataMap { return make(SearchDataMapSmall, 10) }},
		{"SearchDataMapLarge", func() SearchDataMap { return make(SearchDataMapLarge, 10) }},
	}

	testCases := []struct {
		name    string
		values  int
		repeats int
	}{
		{"inserts", 1, 0},
		{"inserts", 10, 0},
		{"inserts", 100, 0},
		{"repeats", 10, 100},
		{"repeats", 10, 1000},
		{"repeats", 100, 100},
		{"repeats", 100, 1000},
	}

	for _, tc := range testCases {
		for _, intf := range intfs {
			b.Run(fmt.Sprint(tc.name, "/", tc.values, "x value/", tc.repeats, "x repeat", "/", intf.name), func(b *testing.B) {

				var k []string
				for i := 0; i < b.N; i++ {
					k = append(k, fmt.Sprintf("key%d", i))
				}

				var v []string
				for i := 0; i < tc.values; i++ {
					v = append(v, fmt.Sprintf("value%d", i))
				}

				s := intf.f()
				insert := func() {
					for i := 0; i < len(k); i++ {
						for j := 0; j < len(v); j++ {
							s.Add(k[i], v[j])
						}
					}
				}

				// insert
				b.ResetTimer()
				insert()

				// reinsert?
				if tc.repeats > 0 {
					b.ResetTimer()
					for i := 0; i < tc.repeats; i++ {
						insert()
					}
				}
			})
		}
	}

}
