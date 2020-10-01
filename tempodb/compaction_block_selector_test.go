package tempodb

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/stretchr/testify/assert"
)

func TestTimeWindowBlockSelectorBlocksToCompact(t *testing.T) {
	now := time.Now()
	timeWindow := 12 * time.Hour

	tests := []struct {
		name           string
		blocklist      []*encoding.BlockMeta
		expected       []*encoding.BlockMeta
		expectedSecond []*encoding.BlockMeta
	}{
		{
			name:      "nil - nil",
			blocklist: nil,
			expected:  nil,
		},
		{
			name:      "empty - nil",
			blocklist: []*encoding.BlockMeta{},
			expected:  nil,
		},
		{
			name: "only two",
			blocklist: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
			expected: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
		},
		{
			name: "choose smallest two",
			blocklist: []*encoding.BlockMeta{
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 0,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					TotalObjects: 1,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 0,
				},
			},
			expected: []*encoding.BlockMeta{
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 0,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 0,
				},
			},
		},
		{
			name: "different windows",
			blocklist: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now.Add(-timeWindow),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime: now.Add(-timeWindow),
				},
			},
			expected: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
			},
			expectedSecond: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now.Add(-timeWindow),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime: now.Add(-timeWindow),
				},
			},
		},
		{
			name: "different sizes",
			blocklist: []*encoding.BlockMeta{
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					TotalObjects: 15,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 1,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					TotalObjects: 3,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 12,
				},
			},
			expected: []*encoding.BlockMeta{
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 1,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					TotalObjects: 3,
				},
			},
			expectedSecond: []*encoding.BlockMeta{
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 12,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					TotalObjects: 15,
				},
			},
		},
		{
			name: "different compaction lvls",
			blocklist: []*encoding.BlockMeta{
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					CompactionLevel: 1,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					CompactionLevel: 1,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
			expected: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
			expectedSecond: []*encoding.BlockMeta{
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					CompactionLevel: 1,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					CompactionLevel: 1,
				},
			},
		},
		// {
		// 	name: "active time window vs not",
		// 	blocklist: []*encoding.BlockMeta{
		// 		{
		// 			BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		// 			EndTime: now,
		// 		},
		// 		{
		// 			BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
		// 			EndTime: now,
		// 		},
		// 		{
		// 			BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000004"),
		// 			EndTime:         now,
		// 			CompactionLevel: 1,
		// 		},
		// 		{
		// 			BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		// 			EndTime:         now.Add(-activeWindowDuration),
		// 			CompactionLevel: 1,
		// 		},
		// 		{
		// 			BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		// 			EndTime:         now.Add(-activeWindowDuration),
		// 			CompactionLevel: 0,
		// 		},
		// 	},
		// 	expected: []*encoding.BlockMeta{
		// 		{
		// 			BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		// 			EndTime: now,
		// 		},
		// 		{
		// 			BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
		// 			EndTime: now,
		// 		},
		// 	},
		// 	expectedSecond: []*encoding.BlockMeta{
		// 		{
		// 			BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		// 			EndTime:         now.Add(-activeWindowDuration),
		// 			CompactionLevel: 1,
		// 		},
		// 		{
		// 			BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		// 			EndTime:         now.Add(-activeWindowDuration),
		// 			CompactionLevel: 0,
		// 		},
		// 	},
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := newTimeWindowBlockSelector(tt.blocklist, time.Second, 100)

			actual, _ := selector.BlocksToCompact()
			assert.Equal(t, tt.expected, actual)

			actual, _ = selector.BlocksToCompact()
			assert.Equal(t, tt.expectedSecond, actual)
		})
	}
}

func TestTimeWindowBlockSelectorSort(t *testing.T) {
	now := time.Now()
	timeWindow := 12 * time.Hour

	tests := []struct {
		name      string
		blocklist []*encoding.BlockMeta
		expected  []*encoding.BlockMeta
	}{
		{
			name:      "nil - nil",
			blocklist: nil,
			expected:  nil,
		},
		{
			name:      "empty - nil",
			blocklist: []*encoding.BlockMeta{},
			expected:  nil,
		},
		{
			name: "different time windows",
			blocklist: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now.Add(-2 * timeWindow),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now.Add(-timeWindow),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
			},
			expected: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now.Add(-timeWindow),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now.Add(-2 * timeWindow),
				},
			},
		},
		{
			name: "different compaction lvls",
			blocklist: []*encoding.BlockMeta{
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					CompactionLevel: 1,
					EndTime:         now,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					CompactionLevel: 0,
					EndTime:         now,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					CompactionLevel: 2,
					EndTime:         now,
				},
			},
			expected: []*encoding.BlockMeta{
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					CompactionLevel: 0,
					EndTime:         now,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					CompactionLevel: 1,
					EndTime:         now,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					CompactionLevel: 2,
					EndTime:         now,
				},
			},
		},
		{
			name: "different sizes",
			blocklist: []*encoding.BlockMeta{
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime:      now,
					TotalObjects: 2,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:      now,
					TotalObjects: 1,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:      now,
					TotalObjects: 0,
				},
			},
			expected: []*encoding.BlockMeta{
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:      now,
					TotalObjects: 0,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:      now,
					TotalObjects: 1,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime:      now,
					TotalObjects: 2,
				},
			},
		},
		{
			name: "all things",
			blocklist: []*encoding.BlockMeta{
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					CompactionLevel: 1,
					EndTime:         now,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					CompactionLevel: 0,
					EndTime:         now.Add(-timeWindow),
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					CompactionLevel: 0,
					EndTime:         now.Add(-timeWindow),
					TotalObjects:    1,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					CompactionLevel: 0,
					EndTime:         now,
				},
			},
			expected: []*encoding.BlockMeta{
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					CompactionLevel: 0,
					EndTime:         now,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					CompactionLevel: 1,
					EndTime:         now,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					CompactionLevel: 0,
					EndTime:         now.Add(-timeWindow),
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					CompactionLevel: 0,
					EndTime:         now.Add(-timeWindow),
					TotalObjects:    1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := newTimeWindowBlockSelector(tt.blocklist, timeWindow, 100)
			actual := selector.(*timeWindowBlockSelector).blocklist
			assert.Equal(t, tt.expected, actual)
		})
	}
}
