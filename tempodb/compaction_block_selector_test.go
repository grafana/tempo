package tempodb

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/stretchr/testify/assert"
)

func TestTimeWindowBlockSelectorBlocksToCompact(t *testing.T) {
	now := time.Now()
	timeWindow := 12 * time.Hour
	tenantID := ""

	tests := []struct {
		name           string
		blocklist      []*encoding.BlockMeta
		minInputBlocks int // optional, defaults to global const
		maxInputBlocks int // optional, defaults to global const
		expected       []*encoding.BlockMeta
		expectedHash   string
		expectedSecond []*encoding.BlockMeta
		expectedHash2  string
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
					EndTime: now,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
			},
			expected: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v", tenantID, 0, now.Unix()),
		},
		{
			name: "choose smallest two",
			blocklist: []*encoding.BlockMeta{
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 0,
					EndTime:      now,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					TotalObjects: 1,
					EndTime:      now,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 0,
					EndTime:      now,
				},
			},
			maxInputBlocks: 2,
			expected: []*encoding.BlockMeta{
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 0,
					EndTime:      now,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 0,
					EndTime:      now,
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v", tenantID, 0, now.Unix()),
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
			expectedHash: fmt.Sprintf("%v-%v-%v", tenantID, 0, now.Unix()),
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
			expectedHash2: fmt.Sprintf("%v-%v-%v", tenantID, 0, now.Add(-timeWindow).Unix()),
		},
		{
			name: "different sizes",
			blocklist: []*encoding.BlockMeta{
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:      now,
					TotalObjects: 15,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:      now,
					TotalObjects: 1,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime:      now,
					TotalObjects: 3,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:      now,
					TotalObjects: 12,
				},
			},
			maxInputBlocks: 2,
			expected: []*encoding.BlockMeta{
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:      now,
					TotalObjects: 1,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime:      now,
					TotalObjects: 3,
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v", tenantID, 0, now.Unix()),
			expectedSecond: []*encoding.BlockMeta{
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:      now,
					TotalObjects: 12,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:      now,
					TotalObjects: 15,
				},
			},
			expectedHash2: fmt.Sprintf("%v-%v-%v", tenantID, 0, now.Unix()),
		},
		{
			name: "different compaction lvls",
			blocklist: []*encoding.BlockMeta{
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
			},
			expected: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v", tenantID, 0, now.Unix()),
			expectedSecond: []*encoding.BlockMeta{
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime:         now,
					CompactionLevel: 1,
				},
			},
			expectedHash2: fmt.Sprintf("%v-%v-%v", tenantID, 1, now.Unix()),
		},
		{
			name: "active time window vs not",
			blocklist: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime:         now.Add(-activeWindowDuration - time.Minute),
					CompactionLevel: 1,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:         now.Add(-activeWindowDuration - time.Minute),
					CompactionLevel: 0,
				},
			},
			expected: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v", tenantID, 0, now.Unix()),
			expectedSecond: []*encoding.BlockMeta{
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:         now.Add(-activeWindowDuration - time.Minute),
					CompactionLevel: 0,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime:         now.Add(-activeWindowDuration - time.Minute),
					CompactionLevel: 1,
				},
			},
			expectedHash2: fmt.Sprintf("%v-%v", tenantID, now.Add(-activeWindowDuration-time.Minute).Unix()),
		},
		{
			name: "choose lowest compaction level",
			blocklist: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime: now.Add(-timeWindow),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000005"),
					EndTime: now.Add(-timeWindow),
				},
			},
			expected: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v", tenantID, 0, now.Unix()),
			expectedSecond: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime: now.Add(-timeWindow),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000005"),
					EndTime: now.Add(-timeWindow),
				},
			},
			expectedHash2: fmt.Sprintf("%v-%v-%v", tenantID, 0, now.Add(-timeWindow).Unix()),
		},
		{
			name: "doesn't choose across time windows",
			blocklist: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now.Add(-timeWindow),
				},
			},
			expected:       nil,
			expectedHash:   "",
			expectedSecond: nil,
			expectedHash2:  "",
		},
		{
			// First compaction gets 3 blocks, second compaction gets 2 more
			name:           "choose more than 2 blocks",
			maxInputBlocks: 3,
			blocklist: []*encoding.BlockMeta{
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:      now,
					TotalObjects: 1,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:      now,
					TotalObjects: 2,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime:      now,
					TotalObjects: 3,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:      now,
					TotalObjects: 4,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000005"),
					EndTime:      now,
					TotalObjects: 5,
				},
			},
			expected: []*encoding.BlockMeta{
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:      now,
					TotalObjects: 1,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:      now,
					TotalObjects: 2,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime:      now,
					TotalObjects: 3,
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v", tenantID, 0, now.Unix()),
			expectedSecond: []*encoding.BlockMeta{
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:      now,
					TotalObjects: 4,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000005"),
					EndTime:      now,
					TotalObjects: 5,
				},
			},
			expectedHash2: fmt.Sprintf("%v-%v-%v", tenantID, 0, now.Unix()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			min := defaultMinInputBlocks
			if tt.minInputBlocks > 0 {
				min = tt.minInputBlocks
			}

			max := defaultMaxInputBlocks
			if tt.maxInputBlocks > 0 {
				max = tt.maxInputBlocks
			}

			selector := newTimeWindowBlockSelector(tt.blocklist, time.Second, 100, min, max)

			actual, hash := selector.BlocksToCompact()
			assert.Equal(t, tt.expected, actual)
			assert.Equal(t, tt.expectedHash, hash)

			actual, hash = selector.BlocksToCompact()
			assert.Equal(t, tt.expectedSecond, actual)
			assert.Equal(t, tt.expectedHash2, hash)
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
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					CompactionLevel: 1,
					EndTime:         now,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := newTimeWindowBlockSelector(tt.blocklist, timeWindow, 100, defaultMinInputBlocks, defaultMaxInputBlocks)
			actual := selector.(*timeWindowBlockSelector).blocklist
			assert.Equal(t, tt.expected, actual)
		})
	}
}
