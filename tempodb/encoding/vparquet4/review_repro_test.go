package vparquet4

import (
	"context"
	"testing"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	tempo_io "github.com/grafana/tempo/v3/pkg/io"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/require"
)

func reviewWriteBlock(t *testing.T, r backend.Reader, w backend.Writer, legacy bool) *backend.BlockMeta {
	t.Helper()
	ctx := context.Background()
	meta := backend.NewBlockMeta("disabled-tenant", uuid.New(), VersionString)
	meta.TotalObjects = 1
	cfg := &common.BlockConfig{BloomFP: .01, BloomShardSizeBytes: 100 * 1024, RowGroupSizeBytes: 1 << 20, PersistServiceGraphEdges: false}
	sb, stored := newStreamingBlock(ctx, cfg, meta, r, w, tempo_io.NewBufferedWriter)
	if legacy {
		// Simulate a real pre-PR block, without the new edge column group.
		sb.pw = parquet.NewGenericWriter[*Trace](sb.bw, parquet.StructTag(`parquet:"-"`, "ServiceGraphEdges"))
	}
	id := make([]byte, 16)
	id[15] = 1
	parentID, childID := make([]byte, 8), make([]byte, 8)
	parentID[7], childID[7] = 1, 2
	tr := &Trace{TraceID: id, RootServiceName: "A", RootSpanName: "root", ResourceSpans: []ResourceSpans{
		{Resource: Resource{ServiceName: "A"}, ScopeSpans: []ScopeSpans{{Spans: []Span{{SpanID: parentID, Kind: 1, StartTimeUnixNano: 1, DurationNano: 1}}}}},
		{Resource: Resource{ServiceName: "B"}, ScopeSpans: []ScopeSpans{{Spans: []Span{{SpanID: childID, ParentSpanID: parentID, Kind: 1, StartTimeUnixNano: 1, DurationNano: 1}}}}},
	}}
	require.NoError(t, sb.Add(tr, 1, 2))
	_, err := sb.Complete()
	require.NoError(t, err)
	return stored
}

// Upgrading readers must not require rewriting pre-feature trace blocks.
func TestReviewLegacyBlockQuery(t *testing.T) {
	rawR, rawW, _, err := local.New(&local.Config{Path: t.TempDir()})
	require.NoError(t, err)
	r, w := backend.NewReader(rawR), backend.NewWriter(rawW)
	meta := reviewWriteBlock(t, r, w, true)
	id := make([]byte, 16)
	id[15] = 1
	found, err := newBackendBlock(meta, r).FindTraceByID(context.Background(), id, common.DefaultSearchOptions())
	require.NoError(t, err)
	require.NotNil(t, found.Trace)
	require.Len(t, found.Trace.ResourceSpans, 2)
	require.Empty(t, found.Trace.ServiceGraphEdges)
}

// A global opt-in must not backfill edge records for a tenant that explicitly
// disabled persistence when its original blocks were written.
func TestReviewCompactionHonorsTenantOptOut(t *testing.T) {
	rawR, rawW, _, err := local.New(&local.Config{Path: t.TempDir()})
	require.NoError(t, err)
	r, w := backend.NewReader(rawR), backend.NewWriter(rawW)
	inputs := []*backend.BlockMeta{reviewWriteBlock(t, r, w, false), reviewWriteBlock(t, r, w, false)}
	cfg := common.BlockConfig{BloomFP: .01, BloomShardSizeBytes: 100 * 1024, RowGroupSizeBytes: 1 << 20, PersistServiceGraphEdges: true}
	c := NewCompactor(common.CompactionOptions{
		BlockConfig: cfg, OutputBlocks: 1,
		DedupedSpans: func(int, int) {}, DisconnectedTrace: func() {}, RootlessTrace: func() {},
		ObjectsCombined: func(int, int) {}, ObjectsWritten: func(int, int) {}, BytesWritten: func(int, int) {}, TraceWritten: func(int, int) {},
	})
	output, err := c.Compact(context.Background(), log.NewNopLogger(), r, w, inputs)
	require.NoError(t, err)
	require.Len(t, output, 1)
	id := make([]byte, 16)
	id[15] = 1
	found, err := newBackendBlock(output[0], r).FindTraceByID(context.Background(), id, common.DefaultSearchOptions())
	require.NoError(t, err)
	require.Empty(t, found.Trace.ServiceGraphEdges, "compaction must preserve the tenant's explicit opt-out")
}
