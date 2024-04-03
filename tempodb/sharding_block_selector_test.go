package tempodb

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/require"
)

func TestShardingBlockSelector(t *testing.T) {
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
			expectedHash: fmt.Sprintf("%v-%v", tenantID, now.Unix()),
		},
		{
			name: "choose two with lowest trace ID",
			blocklist: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
					MinID:   []byte{2},
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
					MinID:   []byte{0},
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
					MinID:   []byte{1},
				},
			},
			maxInputBlocks: 2,
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
					MinID:   []byte{0},
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
					MinID:   []byte{1},
				},
			},
			expectedHash: fmt.Sprintf("%v-%v", tenantID, now.Unix()),
			expectedSecond: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
					MinID:   []byte{2},
				},
			},
			expectedHash2: fmt.Sprintf("%v-%v", tenantID, now.Unix()),
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
			expectedHash: fmt.Sprintf("%v-%v", tenantID, now.Unix()),
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
			expectedHash2: fmt.Sprintf("%v-%v", tenantID, now.Add(-timeWindow).Unix()),
		},
		{
			// All of these blocks fall within the same shard.
			// Therefore each pass it will choose the two with the next lowest trace IDs.
			name: "different minimum trace ids",
			blocklist: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime: now,
					MinID:   []byte{4},
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
					MinID:   []byte{2},
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
					MinID:   []byte{0},
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
					MinID:   []byte{1},
				},
			},
			maxInputBlocks: 2,
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
					MinID:   []byte{0},
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
					MinID:   []byte{1},
				},
			},
			expectedHash: fmt.Sprintf("%v-%v", tenantID, now.Unix()),
			expectedSecond: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
					MinID:   []byte{2},
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime: now,
					MinID:   []byte{4},
				},
			},
			expectedHash2: fmt.Sprintf("%v-%v", tenantID, now.Unix()),
		},
		{
			// The two blocks that are already compacted and within a single shard (min/max=0)
			// will be prioritized over the 2 new blocks with CompactionLevel=0
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
			expectedHash: fmt.Sprintf("%v-%v-%03d", tenantID, now.Unix(), 0), // CompactionLevel=1 and shard=0
			expectedSecond: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
			},
			expectedHash2: fmt.Sprintf("%v-%v", tenantID, now.Unix()),
		},
		{
			name: "doesn't choose across time windows for already sharded blocks",
			blocklist: []*backend.BlockMeta{
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:         now.Add(-timeWindow),
					CompactionLevel: 1,
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
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects:    99,
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects:    2,
					EndTime:         now,
					CompactionLevel: 1,
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
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Size:            50,
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Size:            51,
					EndTime:         now,
					CompactionLevel: 1,
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
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects:    50,
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects:    50,
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					TotalObjects:    50,
					EndTime:         now,
					CompactionLevel: 1,
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects:    50,
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects:    50,
					EndTime:         now,
					CompactionLevel: 1,
				},
			},
			expectedHash:   fmt.Sprintf("%v-%v-%03d", tenantID, now.Unix(), 0), // shard=0
			expectedSecond: nil,
			expectedHash2:  "",
		},
		{
			name:          "Returns as many blocks as possible without exceeding max block size",
			maxBlockBytes: 100,
			blocklist: []*backend.BlockMeta{
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Size:            50,
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Size:            50,
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Size:            1,
					EndTime:         now,
					CompactionLevel: 1,
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Size:            50,
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Size:            50,
					EndTime:         now,
					CompactionLevel: 1,
				},
			},
			expectedHash:   fmt.Sprintf("%v-%v-%03d", tenantID, now.Unix(), 0), // shard=0
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
			expectedHash: fmt.Sprintf("%v-%v", tenantID, now.Unix()),
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
			expectedHash2: fmt.Sprintf("%v-%v", tenantID, now.Unix()),
		},
		{
			name: "honors minimum block count for sharded blocks",
			blocklist: []*backend.BlockMeta{
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:         now,
					CompactionLevel: 1,
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
			name: "don't compact across dataEncodings",
			blocklist: []*backend.BlockMeta{
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime:         now,
					DataEncoding:    "bar",
					CompactionLevel: 1,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:         now,
					DataEncoding:    "foo",
					CompactionLevel: 1,
				},
			},
			expected: nil,
		},
		{
			name: "don't compact across versions",
			blocklist: []*backend.BlockMeta{
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:         now,
					Version:         "v2",
					CompactionLevel: 1,
				},
				{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:         now,
					Version:         "vParquet",
					CompactionLevel: 1,
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
			expectedHash: fmt.Sprintf("%v-%v", tenantID, now.Unix()),
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
			expectedHash2: fmt.Sprintf("%v-%v", tenantID, now.Unix()),
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
			expectedHash: fmt.Sprintf("%v-%v", tenantID, now.Unix()),
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
			expectedHash2: fmt.Sprintf("%v-%v", tenantID, now.Unix()),
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

			selector := newShardingBlockSelector(2, tt.blocklist, time.Second, 100, maxSize, min, max)

			actual := selector.BlocksToCompact()
			require.Equal(t, tt.expected, actual.Blocks())
			require.Equal(t, tt.expectedHash, actual.Ownership())

			actual = selector.BlocksToCompact()
			require.Equal(t, tt.expectedSecond, actual.Blocks())
			require.Equal(t, tt.expectedHash2, actual.Ownership())
		})
	}
}

/*
func TestRealIndex(t *testing.T) {
	x, err := os.ReadFile("/Users/marty/src/deployment_tools/index.json")
	require.NoError(t, err)

	i := backend.TenantIndex{}

	err = json.Unmarshal(x, &i)
	require.NoError(t, err)

	window := 8 * time.Minute
	maxObjs := 3_000_000
	maxSize := uint64(107374182400)
	selector := newShardingBlockSelector(8, i.Meta, window, maxObjs, maxSize, 2, 4)

	for {
		cmd := selector.BlocksToCompact()
		if len(cmd.Blocks()) == 0 {
			break
		}

		// fmt.Println("Compaction command:", blockIDs, cmd.Ownership())
	}
}
*/
