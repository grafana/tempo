package tempofb

import (
	"fmt"
	"testing"
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
