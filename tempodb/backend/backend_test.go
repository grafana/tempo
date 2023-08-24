package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateRanges(t *testing.T) {
	tests := []struct {
		name           string
		shardSize      int
		expectedRanges []struct{ Start, End rune }
	}{
		{
			name:           "shard size 1",
			shardSize:      1,
			expectedRanges: []struct{ Start, End rune }{{'0', 'z'}},
		},
		{
			name:      "shard size 3",
			shardSize: 3,
			expectedRanges: []struct{ Start, End rune }{
				{'0', 'J'},
				{'K', 'd'},
				{'e', 'z'},
			},
		},
		{
			name:      "shard size 7",
			shardSize: 7,
			expectedRanges: []struct{ Start, End rune }{
				{'0', '7'},
				{'8', 'F'},
				{'G', 'N'},
				{'O', 'V'},
				{'W', 'd'},
				{'e', 'l'},
				{'m', 'z'},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results := CalculateRanges(tc.shardSize)

			assert.Equal(t, len(tc.expectedRanges), len(results))

			for i, r := range results {
				assert.Equal(t, string(tc.expectedRanges[i].Start), string(r.Start))
				assert.Equal(t, string(tc.expectedRanges[i].End), string(r.End))
			}
		})
	}
}
