package blockselector

import (
	"fmt"
	"testing"
	"time"

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
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Unix(), 0),
		},
		{
			name: "choose smallest two",
			blocklist: []*backend.BlockMeta{
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 0,
					EndTime:      now,
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000000"),
					TotalObjects: 1,
					EndTime:      now,
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 0,
					EndTime:      now,
				},
			},
			maxInputBlocks: 2,
			expected: []*backend.BlockMeta{
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 0,
					EndTime:      now,
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 0,
					EndTime:      now,
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Unix(), 0),
		},
		{
			name: "different windows",
			blocklist: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now.Add(-timeWindow),
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime: now.Add(-timeWindow),
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Unix(), 0),
			expectedSecond: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now.Add(-timeWindow),
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime: now.Add(-timeWindow),
				},
			},
			expectedHash2: fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Add(-timeWindow).Unix(), 0),
		},
		{
			name: "different sizes",
			blocklist: []*backend.BlockMeta{
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:      now,
					TotalObjects: 15,
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:      now,
					TotalObjects: 1,
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime:      now,
					TotalObjects: 3,
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:      now,
					TotalObjects: 12,
				},
			},
			maxInputBlocks: 2,
			expected: []*backend.BlockMeta{
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:      now,
					TotalObjects: 1,
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime:      now,
					TotalObjects: 3,
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Unix(), 0),
			expectedSecond: []*backend.BlockMeta{
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:      now,
					TotalObjects: 12,
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:      now,
					TotalObjects: 15,
				},
			},
			expectedHash2: fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Unix(), 0),
		},
		{
			name: "different compaction lvls",
			blocklist: []*backend.BlockMeta{
				{
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
				},
				{
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Unix(), 0),
			expectedSecond: []*backend.BlockMeta{
				{
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime:         now,
					CompactionLevel: 1,
				},
			},
			expectedHash2: fmt.Sprintf("%v-%v-%v-%v", tenantID, 1, now.Unix(), 0),
		},
		{
			name: "active time window vs not",
			blocklist: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
				},
				{
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime:         now.Add(-activeWindowDuration - time.Minute),
					CompactionLevel: 1,
				},
				{
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:         now.Add(-activeWindowDuration - time.Minute),
					CompactionLevel: 0,
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Unix(), 0),
			expectedSecond: []*backend.BlockMeta{
				{
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:         now.Add(-activeWindowDuration - time.Minute),
					CompactionLevel: 0,
				},
				{
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime:         now.Add(-activeWindowDuration - time.Minute),
					CompactionLevel: 1,
				},
			},
			expectedHash2: fmt.Sprintf("%v-%v-%v", tenantID, now.Add(-activeWindowDuration-time.Minute).Unix(), 0),
		},
		{
			name: "choose lowest compaction level",
			blocklist: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
				},
				{
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime: now.Add(-timeWindow),
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000005"),
					EndTime: now.Add(-timeWindow),
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime: now,
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Unix(), 0),
			expectedSecond: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime: now.Add(-timeWindow),
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000005"),
					EndTime: now.Add(-timeWindow),
				},
			},
			expectedHash2: fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Add(-timeWindow).Unix(), 0),
		},
		{
			name: "doesn't choose across time windows",
			blocklist: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000002"),
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
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 99,
					EndTime:      now,
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000002"),
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
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					Size_:   50,
					EndTime: now,
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000002"),
					Size_:   51,
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
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 50,
					EndTime:      now,
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 50,
					EndTime:      now,
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000003"),
					TotalObjects: 50,
					EndTime:      now,
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 50,
					EndTime:      now,
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 50,
					EndTime:      now,
				},
			},
			expectedHash:   fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Unix(), 0),
			expectedSecond: nil,
			expectedHash2:  "",
		},
		{
			name:          "Returns as many blocks as possible without exceeding max block size",
			maxBlockBytes: 100,
			blocklist: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					Size_:   50,
					EndTime: now,
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000002"),
					Size_:   50,
					EndTime: now,
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000003"),
					Size_:   1,
					EndTime: now,
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					Size_:   50,
					EndTime: now,
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000002"),
					Size_:   50,
					EndTime: now,
				},
			},
			expectedHash:   fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Unix(), 0),
			expectedSecond: nil,
			expectedHash2:  "",
		},
		{
			// First compaction gets 3 blocks, second compaction gets 2 more
			name:           "choose more than 2 blocks",
			maxInputBlocks: 3,
			blocklist: []*backend.BlockMeta{
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:      now,
					TotalObjects: 1,
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:      now,
					TotalObjects: 2,
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime:      now,
					TotalObjects: 3,
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:      now,
					TotalObjects: 4,
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000005"),
					EndTime:      now,
					TotalObjects: 5,
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:      now,
					TotalObjects: 1,
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:      now,
					TotalObjects: 2,
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime:      now,
					TotalObjects: 3,
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Unix(), 0),
			expectedSecond: []*backend.BlockMeta{
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:      now,
					TotalObjects: 4,
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000005"),
					EndTime:      now,
					TotalObjects: 5,
				},
			},
			expectedHash2: fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Unix(), 0),
		},
		{
			name: "honors minimum block count",
			blocklist: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000002"),
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
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:         now,
					CompactionLevel: 0,
				},
				{
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:         now,
					CompactionLevel: 1,
				},
			},
			minInputBlocks: 3,
			maxInputBlocks: 3,
			expected: []*backend.BlockMeta{
				{
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime:         now,
					CompactionLevel: 1,
				},
				{
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:         now,
					CompactionLevel: 1,
				},
			},
			expectedHash:   fmt.Sprintf("%v-%v-%v-%v", tenantID, 1, now.Unix(), 0),
			expectedSecond: nil,
			expectedHash2:  "",
		},
		{
			name: "doesn't select blocks in last active window",
			blocklist: []*backend.BlockMeta{
				{
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:         now.Add(-activeWindowDuration),
					CompactionLevel: 0,
				},
				{
					BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:         now.Add(-activeWindowDuration),
					CompactionLevel: 0,
				},
			},
		},
		{
			name: "don't compact across dataEncodings",
			blocklist: []*backend.BlockMeta{
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000000"),
					EndTime:      now,
					DataEncoding: "bar",
				},
				{
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000001"),
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
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
					Version: "v2",
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
					Version: "vParquet3",
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
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
					Version: "v2",
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
					Version: "vParquet3",
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime: now,
					Version: "v2",
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime: now,
					Version: "vParquet3",
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
					Version: "v2",
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime: now,
					Version: "v2",
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Unix(), 0),
			expectedSecond: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
					Version: "vParquet3",
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime: now,
					Version: "vParquet3",
				},
			},
			expectedHash2: fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Unix(), 0),
		},
		{
			name: "blocks with different dedicated columns are not selected together",
			blocklist: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "span", Name: "foo", Type: "int"},
					},
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "span", Name: "foo", Type: "string"},
					},
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime: now,
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "span", Name: "foo", Type: "int"},
					},
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime: now,
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "span", Name: "foo", Type: "string"},
					},
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime: now,
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "span", Name: "foo", Type: "int"},
					},
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime: now,
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "span", Name: "foo", Type: "int"},
					},
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Unix(), 0),
			expectedSecond: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime: now,
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "span", Name: "foo", Type: "string"},
					},
				},
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime: now,
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "span", Name: "foo", Type: "string"},
					},
				},
			},
			expectedHash2: fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Unix(), 0),
		},
		{
			name: "blocks are grouped by replication factor",
			blocklist: []*backend.BlockMeta{
				{
					BlockID:           backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:           now,
					ReplicationFactor: 1,
				},
				{
					BlockID:           backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:           now,
					ReplicationFactor: 3,
				},
				{
					BlockID:           backend.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime:           now,
					ReplicationFactor: 1,
				},
				{
					BlockID:           backend.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:           now,
					ReplicationFactor: 3,
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID:           backend.MustParse("00000000-0000-0000-0000-000000000001"),
					EndTime:           now,
					ReplicationFactor: 1,
				},
				{
					BlockID:           backend.MustParse("00000000-0000-0000-0000-000000000003"),
					EndTime:           now,
					ReplicationFactor: 1,
				},
			},
			expectedHash: fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Unix(), 1),
			expectedSecond: []*backend.BlockMeta{
				{
					BlockID:           backend.MustParse("00000000-0000-0000-0000-000000000002"),
					EndTime:           now,
					ReplicationFactor: 3,
				},
				{
					BlockID:           backend.MustParse("00000000-0000-0000-0000-000000000004"),
					EndTime:           now,
					ReplicationFactor: 3,
				},
			},
			expectedHash2: fmt.Sprintf("%v-%v-%v-%v", tenantID, 0, now.Unix(), 3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minBlocks := DefaultMinInputBlocks
			if tt.minInputBlocks > 0 {
				minBlocks = tt.minInputBlocks
			}

			maxBlocks := DefaultMaxInputBlocks
			if tt.maxInputBlocks > 0 {
				maxBlocks = tt.maxInputBlocks
			}

			maxSize := uint64(1024 * 1024)
			if tt.maxBlockBytes > 0 {
				maxSize = tt.maxBlockBytes
			}

			selector := NewTimeWindowBlockSelector(tt.blocklist, time.Second, 100, maxSize, minBlocks, maxBlocks)

			actual, hash := selector.BlocksToCompact()
			assert.Equal(t, tt.expected, actual)
			assert.Equal(t, tt.expectedHash, hash)

			actual, hash = selector.BlocksToCompact()
			assert.Equal(t, tt.expectedSecond, actual)
			assert.Equal(t, tt.expectedHash2, hash)
		})
	}
}
