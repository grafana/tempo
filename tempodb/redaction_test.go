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

// windowTestNamespace is the resource.namespace value the window fixtures carry and the window tests
// query on; the tests distinguish traces by TIME, not by attribute.
const windowTestNamespace = "checkout"

// traceAtTime builds a trace whose spans sit at a chosen instant, carrying windowTestNamespace.
// Controlling the span timestamps is the whole point: MakeBatchWithAttributes leaves them at whatever
// MakeSpan produces, so a fixture built on it cannot distinguish traces by time at all, so a
// test can distinguish traces by time as well as by attribute. MakeBatchWithAttributes leaves span
// times at their defaults, which no redaction window can separate.
func traceAtTime(id []byte, startNano, endNano uint64) *tempopb.Trace {
	attrs := []*v1_common.KeyValue{{
		Key:   "namespace",
		Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: windowTestNamespace}},
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
		{idOld, traceAtTime(idOld, oldNano, oldNano), uint32(oldNano / 1e9), uint32(oldNano / 1e9)},
		{idRecent, traceAtTime(idRecent, recentNano, recentNano), uint32(recentNano / 1e9), uint32(recentNano / 1e9)},
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

// TestRedactBlockQueryDisjunction checks that a query joining conditions with || matches traces
// satisfying EITHER side. validateRedactionQuery explicitly permits ||, and an operator redacting on
// several attribute values in one pass is the natural use — so an under-match here means traces the
// operator asked to delete survive, which is the failure they cannot detect.
func TestRedactBlockQueryDisjunction(t *testing.T) {
	_, w, c, _ := testConfig(t, 0)
	ctx := context.Background()

	idA := test.ValidTraceID(nil)
	idB := test.ValidTraceID(nil)
	idKeep := test.ValidTraceID(nil)
	now := uint32(time.Now().Unix())

	data := []testData{
		{idA, traceWithResourceAttr(idA, "namespace", "checkout"), now, now},
		{idB, traceWithResourceAttr(idB, "namespace", "payments"), now, now},
		{idKeep, traceWithResourceAttr(idKeep, "namespace", "keep"), now, now},
	}
	query := `{resource.namespace = "checkout" || resource.namespace = "payments"}`

	blk := cutTestBlockWithTraces(t, w, data)
	meta := blk.BlockMeta()
	require.Equal(t, int64(3), meta.TotalObjects)

	rewrote, found, newMeta, err := c.RedactBlock(ctx, meta, testTenantID, nil, query, tempopb.RedactionMode_REDACTION_MODE_APPLY, 0, 0)
	require.NoError(t, err)
	require.True(t, rewrote)
	require.Equal(t, 2, found, "both sides of the || must match")
	require.NotNil(t, newMeta)
	require.Equal(t, int64(1), newMeta.TotalObjects, "only the non-matching trace survives")
}

// traceWithSpanAttr puts the attribute on the span rather than the resource, so a test can build a
// query that mixes resource.* and span.* scopes — both of which validateRedactionQuery permits.
func traceWithSpanAttr(id []byte, key, val string) *tempopb.Trace {
	sp := test.MakeSpan(id)
	sp.Attributes = append(sp.Attributes, test.MakeAttribute(key, val))
	return &tempopb.Trace{ResourceSpans: []*v1_trace.ResourceSpans{{
		Resource: &v1_resource.Resource{Attributes: []*v1_common.KeyValue{
			test.MakeAttribute("service.name", "test-service"),
		}},
		ScopeSpans: []*v1_trace.ScopeSpans{{
			Scope: &v1_common.InstrumentationScope{Name: "redaction scope test"},
			Spans: []*v1_trace.Span{sp},
		}},
	}}}
}

// TestRedactBlockQueryDisjunctionAcrossScopes checks a || that mixes a resource attribute with a span
// attribute — both permitted by validateRedactionQuery, and the natural shape when an operator redacts
// on more than one signal in a single pass. Each trace satisfies exactly one side.
func TestRedactBlockQueryDisjunctionAcrossScopes(t *testing.T) {
	_, w, c, _ := testConfig(t, 0)
	ctx := context.Background()

	idRes := test.ValidTraceID(nil)
	idSpan := test.ValidTraceID(nil)
	idKeep := test.ValidTraceID(nil)
	now := uint32(time.Now().Unix())

	data := []testData{
		{idRes, traceWithResourceAttr(idRes, "namespace", "checkout"), now, now},
		{idSpan, traceWithSpanAttr(idSpan, "tenantref", "acme"), now, now},
		{idKeep, traceWithResourceAttr(idKeep, "namespace", "keep"), now, now},
	}
	query := `{resource.namespace = "checkout" || span.tenantref = "acme"}`

	blk := cutTestBlockWithTraces(t, w, data)
	meta := blk.BlockMeta()
	require.Equal(t, int64(3), meta.TotalObjects)

	_, found, newMeta, err := c.RedactBlock(ctx, meta, testTenantID, nil, query, tempopb.RedactionMode_REDACTION_MODE_APPLY, 0, 0)
	require.NoError(t, err)
	require.Equal(t, 2, found, "a || across resource and span scopes must match both sides")
	require.NotNil(t, newMeta)
	require.Equal(t, int64(1), newMeta.TotalObjects, "only the non-matching trace survives")
}

// TestRedactBlockOneSidedWindowStillBounds verifies a window with only one bound set still bounds the
// per-block scan.
//
// vparquet{3,4,5} install the trace-time predicate only when both bounds are non-zero, so a half-set
// window removes the filter rather than narrowing it — scanning the block in full and dropping every
// query match regardless of timestamp. SubmitRedaction now rejects one-sided windows, but that guard
// runs at submit: a batch persisted by an older scheduler, or any future caller, can still deliver one
// here. Normalising the open side at the point of use keeps the scan bounded whatever the source.
func TestRedactBlockOneSidedWindowStillBounds(t *testing.T) {
	_, w, c, _ := testConfig(t, 0)
	ctx := context.Background()

	idOld := test.ValidTraceID(nil)
	idRecent := test.ValidTraceID(nil)
	oldNano := uint64(time.Now().Add(-72 * time.Hour).UnixNano())
	recentNano := uint64(time.Now().Add(-1 * time.Hour).UnixNano())

	data := []testData{
		{idOld, traceAtTime(idOld, oldNano, oldNano), uint32(oldNano / 1e9), uint32(oldNano / 1e9)},
		{idRecent, traceAtTime(idRecent, recentNano, recentNano), uint32(recentNano / 1e9), uint32(recentNano / 1e9)},
	}
	query := `{resource.namespace = "checkout"}`

	t.Run("start only excludes older traces", func(t *testing.T) {
		blk := cutTestBlockWithTraces(t, w, data)
		_, found, newMeta, err := c.RedactBlock(ctx, blk.BlockMeta(), testTenantID, nil, query,
			tempopb.RedactionMode_REDACTION_MODE_APPLY, time.Now().Add(-2*time.Hour).UnixNano(), 0)
		require.NoError(t, err)
		require.Equal(t, 1, found, "a start-only window must still exclude the out-of-window trace")
		require.Equal(t, int64(1), newMeta.TotalObjects)
	})

	t.Run("end only excludes newer traces", func(t *testing.T) {
		blk := cutTestBlockWithTraces(t, w, data)
		_, found, newMeta, err := c.RedactBlock(ctx, blk.BlockMeta(), testTenantID, nil, query,
			tempopb.RedactionMode_REDACTION_MODE_APPLY, 0, time.Now().Add(-48*time.Hour).UnixNano())
		require.NoError(t, err)
		require.Equal(t, 1, found, "an end-only window must still exclude the out-of-window trace")
		require.Equal(t, int64(1), newMeta.TotalObjects)
	})
}
