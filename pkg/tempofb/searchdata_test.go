package tempofb

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func sizeWithCounts(traceCount, tagCount, valueCount int) int {
	b := NewSearchPageBuilder()

	for t := 0; t < traceCount; t++ {
		sd := &SearchEntryMutable{}
		for g := 0; g < tagCount; g++ {
			for v := 0; v < valueCount; v++ {
				sd.AddTag(fmt.Sprintf("tag%d", g), fmt.Sprintf("value%d", v))
			}
		}

		b.AddData(sd)
	}

	return len(b.Finish())
}

func TestSearchEntryMutableSetStartTimeUnixNano(t *testing.T) {

	testCases := []struct {
		name     string
		inputs   []uint64
		expected uint64
	}{
		{"save smallest", []uint64{2, 1, 3}, 1},
		{"don't save zeros", []uint64{1000, 0}, 1000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := &SearchEntryMutable{}

			for _, i := range tc.inputs {
				e.SetStartTimeUnixNano(i)
			}

			require.Equal(t, tc.expected, e.StartTimeUnixNano)
		})
	}
}

func TestSearchEntryMutableSetEndTimeUnixNano(t *testing.T) {

	testCases := []struct {
		name     string
		inputs   []uint64
		expected uint64
	}{
		{"save largest", []uint64{2, 3, 1}, 3},
		{"don't save zeros", []uint64{1000, 0}, 1000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := &SearchEntryMutable{}

			for _, i := range tc.inputs {
				e.SetEndTimeUnixNano(i)
			}

			require.Equal(t, tc.expected, e.EndTimeUnixNano)
		})
	}
}

func TestEncodingSize(t *testing.T) {
	delta := 1000

	batchBaseLine := sizeWithCounts(0, 0, 0)

	traceBaseLine := sizeWithCounts(1, 0, 0)
	traceLongTerm := sizeWithCounts(delta, 0, 0)

	tagValueBaseLine := sizeWithCounts(1, 1, 1)
	tagValueLongTermTags := sizeWithCounts(1, delta, 1)
	tagValueLongTermValues := sizeWithCounts(1, 1, delta)

	fmt.Printf("Data sizes:\n")
	fmt.Printf("- Batch:      %d bytes\n", batchBaseLine)
	fmt.Printf("- Trace:      %d bytes first, %.1f bytes after\n", traceBaseLine-batchBaseLine, float32(traceLongTerm-traceBaseLine)/float32(delta))
	fmt.Printf("- TagValue:   %d bytes first\n", tagValueBaseLine-traceBaseLine)
	fmt.Printf("  - Tag:      %.1f bytes after\n", float32(tagValueLongTermTags-tagValueBaseLine)/float32(delta))
	fmt.Printf("  - Value:    %.1f bytes after\n", float32(tagValueLongTermValues-tagValueBaseLine)/float32(delta))
}
