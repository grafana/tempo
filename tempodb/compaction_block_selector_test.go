package tempodb

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
)

func TestTimeWindowBlockSelectorBlocksToCompact(t *testing.T) {
	now := time.Now()
	timeWindow := 12 * time.Hour
	tenantID := ""

	tests := []struct {
		name           string
		blocklist      []*backend.BlockMeta
		minInputBlocks int    // optional, defaults to global const
		maxInputBlocks int    // optional, defaults to global const
		maxBlockBytes  uint64 // optional, defaults to ???
		expected       []*backend.BlockMeta
		expectedHash   string
		expectedSecond []*backend.BlockMeta
		expectedHash2  string
	}{
		{
			name:      "nil - nil",
			blocklist: nil,
			expected:  nil,
		},
		{
			name:      "empty - nil",
			blocklist: []*backend.BlockMeta{},
			expected:  nil,
		},
		{
			name: "only two",
			blocklist: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
			},
			expected: []*backend.BlockMeta{
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
			blocklist: []*backend.BlockMeta{
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
			expected: []*backend.BlockMeta{
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
			blocklist: []*backend.BlockMeta{
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
			expected: []*backend.BlockMeta{
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
			expectedSecond: []*backend.BlockMeta{
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
			blocklist: []*backend.BlockMeta{
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
			expected: []*backend.BlockMeta{
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
			expectedSecond: []*backend.BlockMeta{
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
			blocklist: []*backend.BlockMeta{
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
			expected: []*backend.BlockMeta{
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
			expectedSecond: []*backend.BlockMeta{
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
			blocklist: []*backend.BlockMeta{
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
			expected: []*backend.BlockMeta{
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
			expectedSecond: []*backend.BlockMeta{
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
			blocklist: []*backend.BlockMeta{
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
			expected: []*backend.BlockMeta{
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
			expectedSecond: []*backend.BlockMeta{
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
			blocklist: []*backend.BlockMeta{
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
			name: "doesn't exceed max compaction objects",
			blocklist: []*backend.BlockMeta{
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 99,
					EndTime:      now,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 2,
					EndTime:      now,
				},
			},
			expected:       nil,
			expectedHash:   "",
			expectedSecond: nil,
			expectedHash2:  "",
		},
		{
			name: "doesn't exceed max block size",
			blocklist: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Size:    50,
					EndTime: now,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Size:    51,
					EndTime: now,
				},
			},
			maxBlockBytes:  100,
			expected:       nil,
			expectedHash:   "",
			expectedSecond: nil,
			expectedHash2:  "",
		},
		{
			name: "Returns as many blocks as possible without exceeding max compaction objects",
			blocklist: []*backend.BlockMeta{
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 50,
					EndTime:      now,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 50,
					EndTime:      now,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					TotalObjects: 50,
					EndTime:      now,
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 50,
					EndTime:      now,
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 50,
					EndTime:      now,
				},
			},
			expectedHash:   fmt.Sprintf("%v-%v-%v", tenantID, 0, now.Unix()),
			expectedSecond: nil,
			expectedHash2:  "",
		},
		{
			name:          "Returns as many blocks as possible without exceeding max block size",
			maxBlockBytes: 100,
			blocklist: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Size:    50,
					EndTime: now,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Size:    50,
					EndTime: now,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Size:    1,
					EndTime: now,
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Size:    50,
					EndTime: now,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Size:    50,
					EndTime: now,
				},
			},
			expectedHash:   fmt.Sprintf("%v-%v-%v", tenantID, 0, now.Unix()),
			expectedSecond: nil,
			expectedHash2:  "",
		},
		{
			// First compaction gets 3 blocks, second compaction gets 2 more
			name:           "choose more than 2 blocks",
			maxInputBlocks: 3,
			blocklist: []*backend.BlockMeta{
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
			expected: []*backend.BlockMeta{
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
			expectedSecond: []*backend.BlockMeta{
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
		{
			name: "honors minimum block count",
			blocklist: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
				},
			},
			minInputBlocks: 3,
			maxInputBlocks: 3,
			expected:       nil,
			expectedHash:   "",
			expectedSecond: nil,
			expectedHash2:  "",
		},
		{
			name: "can choose blocks not at the lowest compaction level",
			blocklist: []*backend.BlockMeta{
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:         now,
					CompactionLevel: 0,
				},
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
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:         now,
					CompactionLevel: 1,
				},
			},
			minInputBlocks: 3,
			maxInputBlocks: 3,
			expected: []*backend.BlockMeta{
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
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:         now,
					CompactionLevel: 1,
				},
			},
			expectedHash:   fmt.Sprintf("%v-%v-%v", tenantID, 1, now.Unix()),
			expectedSecond: nil,
			expectedHash2:  "",
		},
		{
			name: "doesn't select blocks in last active window",
			blocklist: []*backend.BlockMeta{
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:         now.Add(-activeWindowDuration),
					CompactionLevel: 0,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:         now.Add(-activeWindowDuration),
					CompactionLevel: 0,
				},
			},
		},
		{
			name: "don't compact across dataEncodings",
			blocklist: []*backend.BlockMeta{
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime:      now,
					DataEncoding: "bar",
				},
				{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:      now,
					DataEncoding: "foo",
				},
			},
			expected: nil,
		},
		{
			name: "ensures blocks of different versions are not compacted",
			blocklist: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
					Version: "v2",
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
					Version: "vParquet",
				},
			},
			expected:       nil,
			expectedHash:   "",
			expectedSecond: nil,
			expectedHash2:  "",
		},
		{
			name: "ensures blocks of the same version are compacted",
			blocklist: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
					Version: "v2",
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
					Version: "vParquet",
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime: now,
					Version: "v2",
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime: now,
					Version: "vParquet",
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
					Version: "v2",
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime: now,
					Version: "v2",
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v", tenantID, 0, now.Unix()),
			expectedSecond: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
					Version: "vParquet",
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime: now,
					Version: "vParquet",
				},
			},
			expectedHash2: fmt.Sprintf("%v-%v-%v", tenantID, 0, now.Unix()),
		},
		{
			name: "blocks with different dedicated columns are not selected together",
			blocklist: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "span", Name: "foo", Type: "int"},
					},
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "span", Name: "foo", Type: "string"},
					},
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime: now,
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "span", Name: "foo", Type: "int"},
					},
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime: now,
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "span", Name: "foo", Type: "string"},
					},
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "span", Name: "foo", Type: "int"},
					},
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime: now,
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "span", Name: "foo", Type: "int"},
					},
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v", tenantID, 0, now.Unix()),
			expectedSecond: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "span", Name: "foo", Type: "string"},
					},
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime: now,
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "span", Name: "foo", Type: "string"},
					},
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

			maxSize := uint64(1024 * 1024)
			if tt.maxBlockBytes > 0 {
				maxSize = tt.maxBlockBytes
			}

			selector := newTimeWindowBlockSelector(tt.blocklist, time.Second, 100, maxSize, min, max)

			actual := selector.BlocksToCompact()
			assert.Equal(t, tt.expected, actual.Blocks())
			assert.Equal(t, tt.expectedHash, actual.Ownership())

			actual = selector.BlocksToCompact()
			assert.Equal(t, tt.expectedSecond, actual.Blocks())
			assert.Equal(t, tt.expectedHash2, actual.Ownership())
		})
	}
}
