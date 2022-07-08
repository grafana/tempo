package tempofb

import (
	"fmt"
	"strings"
	"testing"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSearchDataMap(t *testing.T) {
	searchDataMap := NewSearchDataMap()

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
}

func TestSearchDataMapMaxBufferLen(t *testing.T) {
	// Verify we don't get a panic when
	// writing more data than can fit in a flatbuffer.

	m := NewSearchDataMap()

	// This generates roughly 960MB of data:
	// 1M entries, 960 bytes each
	for i := 0; i < 1024; i++ {
		k := uuid.New().String()
		for j := 0; j < 1024; j++ {
			v := strings.Repeat(uuid.New().String(), 30) // 32*30=960
			m.Add(k, v)
		}
	}

	fmt.Println("generated map")

	// Try to write more than 2GB of data
	// Start with 512 MB buffer to reduce buffer copying
	// and make the test quicker
	b := flatbuffers.NewBuilder(512 * 1024 * 1024)
	for i := 0; i < 5; i++ {
		WriteSearchDataMap(b, m, nil)
	}
}

func BenchmarkSearchDataMapAdd(b *testing.B) {
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

		b.Run(fmt.Sprint(tc.name, "/", tc.values, "x value/", tc.repeats, "x repeat"), func(b *testing.B) {

			var k []string
			for i := 0; i < b.N; i++ {
				k = append(k, fmt.Sprintf("key%d", i))
			}

			var v []string
			for i := 0; i < tc.values; i++ {
				v = append(v, fmt.Sprintf("value%d", i))
			}

			s := NewSearchDataMap()
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
