package tempofb

import (
	"fmt"
	"testing"
)

func TestEncodingSize(t *testing.T) {
	traceCount := 1000
	tagCounts := []int{1, 5, 10}
	valueCounts := []int{1, 5, 10}

	for _, tagCount := range tagCounts {
		for _, valueCount := range valueCounts {
			t.Run(fmt.Sprintf("%d/%d", tagCount, valueCount), func(t *testing.T) {

				b := NewBatchSearchDataBuilder()

				for i := 0; i < traceCount; i++ {
					sd := &SearchDataMutable{}
					for j := 0; j < tagCount; j++ {
						t := fmt.Sprintf("tag%d", j)
						for k := 0; k < valueCount; k++ {
							sd.AddTag(t, fmt.Sprintf("value%d", k))
						}
					}

					b.AddData(sd)
				}

				valCount := traceCount * tagCount * valueCount
				byteCount := len(b.Finish())
				fmt.Printf("Data size: %d bytes, %.2f bytes per tag value \n", byteCount, float32(byteCount)/float32(valCount))
			})
		}
	}
}
