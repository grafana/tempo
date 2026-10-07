package backend

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/tempopb"
)

func TestTraceByIDBlocksRoundTrip(t *testing.T) {
	start := time.Unix(1700000000, 0).UTC()
	metas := []*BlockMeta{
		{
			Version:           "vParquet5",
			BlockID:           MustParse("00000000-0000-0000-0000-000000000001"),
			TenantID:          "tenant",
			StartTime:         start,
			EndTime:           start.Add(time.Hour),
			TotalObjects:      10,
			Size_:             1024,
			CompactionLevel:   2,
			IndexPageSize:     4,
			TotalRecords:      5,
			BloomShardCount:   3,
			FooterSize:        100,
			ReplicationFactor: 1,
			DedicatedColumns: DedicatedColumns{
				{Scope: DedicatedColumnScopeSpan, Name: "http.method", Type: DedicatedColumnTypeString},
			},
		},
		{
			Version:  "vParquet4",
			BlockID:  MustParse("ffffffff-0000-0000-0000-000000000002"),
			TenantID: "tenant",
		},
		{
			Version:  "vParquet5",
			BlockID:  MustParse("ffffffff-0000-0000-0000-000000000003"),
			TenantID: "tenant",
			DedicatedColumns: DedicatedColumns{
				{Scope: DedicatedColumnScopeResource, Name: "k8s.namespace", Type: DedicatedColumnTypeString},
			},
		},
		{
			Version:  "vParquet5",
			BlockID:  MustParse("ffffffff-0000-0000-0000-000000000004"),
			TenantID: "tenant",
			DedicatedColumns: DedicatedColumns{
				{Scope: DedicatedColumnScopeSpan, Name: "http.method", Type: DedicatedColumnTypeString},
			},
		},
	}

	blocks, err := TraceByIDBlocksFromMetas(metas)
	require.NoError(t, err)
	// blocks 0 and 3 share dedicated columns
	require.Len(t, blocks.DedicatedColumns, 2)

	actual, err := MetasFromTraceByIDBlocks(blocks, "tenant")
	require.NoError(t, err)
	require.Equal(t, metas, actual)
}

func TestMetasFromTraceByIDBlocksTakesTenantFromCaller(t *testing.T) {
	blocks, err := TraceByIDBlocksFromMetas([]*BlockMeta{{BlockID: MustParse("00000000-0000-0000-0000-000000000001"), TenantID: "other"}})
	require.NoError(t, err)

	metas, err := MetasFromTraceByIDBlocks(blocks, "tenant")
	require.NoError(t, err)
	require.Equal(t, "tenant", metas[0].TenantID)
}

func TestMetasFromTraceByIDBlocksInvalid(t *testing.T) {
	blockID := MustParse("00000000-0000-0000-0000-000000000001")

	tests := []struct {
		name   string
		blocks *tempopb.TraceByIDBlocks
	}{
		{
			name:   "short block id",
			blocks: &tempopb.TraceByIDBlocks{Blocks: []*tempopb.TraceByIDBlock{{BlockID: []byte{1}}}},
		},
		{
			name:   "dedicated columns index past the sets",
			blocks: &tempopb.TraceByIDBlocks{Blocks: []*tempopb.TraceByIDBlock{{BlockID: blockID[:], DedicatedColumnsIndex: 1}}},
		},
		{
			name: "invalid dedicated column scope",
			blocks: &tempopb.TraceByIDBlocks{
				DedicatedColumns: []*tempopb.TraceByIDDedicatedColumns{{Columns: []*tempopb.DedicatedColumn{{Name: "a", Scope: 99}}}},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := MetasFromTraceByIDBlocks(tc.blocks, "tenant")
			require.Error(t, err)
		})
	}
}
