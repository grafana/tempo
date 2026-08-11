package tempodb

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/require"
)

// traceWithResourceAttr builds a single-batch trace carrying a specific resource
// attribute, so a TraceQL query can select it and leave others untouched.
func traceWithResourceAttr(id []byte, key, val string) *tempopb.Trace {
	attrs := []*v1_common.KeyValue{{
		Key:   key,
		Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: val}},
	}}
	return &tempopb.Trace{
		ResourceSpans: []*v1_trace.ResourceSpans{test.MakeBatchWithAttributes(2, id, attrs)},
	}
}

// traceWithResourceAttrAtTime is traceWithResourceAttr with control over the span timestamps, so a
// test can distinguish traces by time as well as by attribute. MakeBatchWithAttributes leaves span
// times at their defaults, which no redaction window can separate.
func traceWithResourceAttrAtTime(id []byte, key, val string, startNano, endNano uint64) *tempopb.Trace {
	attrs := []*v1_common.KeyValue{{
		Key:   key,
		Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: val}},
	}}
	return &tempopb.Trace{ResourceSpans: []*v1_trace.ResourceSpans{{
		Resource: &v1_resource.Resource{Attributes: attrs},
		ScopeSpans: []*v1_trace.ScopeSpans{{
			Scope: &v1_common.InstrumentationScope{Name: "redaction window test"},
			Spans: []*v1_trace.Span{test.MakeSpanWithTimeWindow(id, startNano, endNano)},
		}},
	}}}
}

// TestRedactBlockQuerySelector verifies the TraceQL query path of RedactBlock: exactly the
// traces matching the query are dropped (no over-deletion), and dry-run counts without
// rewriting.
func TestRedactBlockQuerySelector(t *testing.T) {
	_, w, c, _ := testConfig(t, 0)
	ctx := context.Background()

	idMatch := test.ValidTraceID(nil)
	idKeep := test.ValidTraceID(nil)
	now := uint32(time.Now().Unix())

	// Two traces distinguished by resource.namespace; the query targets only one.
	data := []testData{
		{idMatch, traceWithResourceAttr(idMatch, "namespace", "checkout"), now, now},
		{idKeep, traceWithResourceAttr(idKeep, "namespace", "keep"), now, now},
	}
	query := `{resource.namespace = "checkout"}`

	t.Run("apply drops only the matching trace", func(t *testing.T) {
		blk := cutTestBlockWithTraces(t, w, data)
		meta := blk.BlockMeta()
		require.Equal(t, int64(2), meta.TotalObjects)

		rewrote, found, newMeta, err := c.RedactBlock(ctx, meta, testTenantID, nil, query, tempopb.RedactionMode_REDACTION_MODE_APPLY, 0, 0)
		require.NoError(t, err)
		require.True(t, rewrote)
		require.Equal(t, 1, found, "exactly one trace matches the query")
		require.NotNil(t, newMeta)
		require.Equal(t, int64(1), newMeta.TotalObjects, "the non-matching trace must survive")
	})

	t.Run("dry-run counts without rewriting", func(t *testing.T) {
		blk := cutTestBlockWithTraces(t, w, data)
		meta := blk.BlockMeta()

		rewrote, found, newMeta, err := c.RedactBlock(ctx, meta, testTenantID, nil, query, tempopb.RedactionMode_REDACTION_MODE_DRY_RUN, 0, 0)
		require.NoError(t, err)
		require.False(t, rewrote, "dry-run must not rewrite")
		require.Equal(t, 1, found, "dry-run still reports the match count")
		require.Nil(t, newMeta)
	})
}

// TestRedactBlockWindowBoundsTheScan verifies the [start, end] window bounds which traces inside a
// block are matched, not only which blocks are selected.
//
// Both traces satisfy the query and differ only in their span timestamps. Without the window applied to
// the fetch, the out-of-window trace is matched and dropped too — over-deletion of data the operator did
// not ask to remove, on a path with no recovery.
func TestRedactBlockWindowBoundsTheScan(t *testing.T) {
	_, w, c, _ := testConfig(t, 0)
	ctx := context.Background()

	idOld := test.ValidTraceID(nil)
	idRecent := test.ValidTraceID(nil)

	oldNano := uint64(time.Now().Add(-72 * time.Hour).UnixNano())
	recentNano := uint64(time.Now().Add(-1 * time.Hour).UnixNano())

	data := []testData{
		{idOld, traceWithResourceAttrAtTime(idOld, "namespace", "checkout", oldNano, oldNano), uint32(oldNano / 1e9), uint32(oldNano / 1e9)},
		{idRecent, traceWithResourceAttrAtTime(idRecent, "namespace", "checkout", recentNano, recentNano), uint32(recentNano / 1e9), uint32(recentNano / 1e9)},
	}
	query := `{resource.namespace = "checkout"}`

	// A window covering only the recent trace.
	startNano := time.Now().Add(-2 * time.Hour).UnixNano()
	endNano := time.Now().UnixNano()

	blk := cutTestBlockWithTraces(t, w, data)
	meta := blk.BlockMeta()
	require.Equal(t, int64(2), meta.TotalObjects)

	rewrote, found, newMeta, err := c.RedactBlock(ctx, meta, testTenantID, nil, query, tempopb.RedactionMode_REDACTION_MODE_APPLY, startNano, endNano)
	require.NoError(t, err)
	require.True(t, rewrote)
	require.Equal(t, 1, found, "only the in-window trace is matched, even though both satisfy the query")
	require.NotNil(t, newMeta)
	require.Equal(t, int64(1), newMeta.TotalObjects, "the out-of-window trace must survive")
}
