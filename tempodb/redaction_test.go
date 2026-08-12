package tempodb

import (
	"context"
	"math"
	"sort"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
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
//
// Controlling the span timestamps is the whole point. MakeBatchWithAttributes leaves span times at
// whatever MakeSpan produces, so every trace built on it lands at effectively the same instant and no
// redaction window can separate them — a window test using that fixture passes whether or not the
// window is applied.
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

// survivingTraceIDs returns the hex IDs of traces still present in a block and matching a query,
// scanned with no time bound.
//
// This is a query-scoped census, not a block census: a surviving trace that does not satisfy the query
// is not reported. Callers therefore pair it with an assertion on newMeta.TotalObjects, so a fixture
// containing a non-matching trace cannot hide that trace being over-deleted.
//
// Window tests must assert WHICH trace remains, not how many. With two candidate traces that both
// satisfy the query and differ only in time, "one was deleted" holds whichever of them went — so a
// count-only assertion passes even when the window is applied backwards. Transposing the two bounds
// inside fetchBounds is a one-character mistake, and on a redaction the wrong outcome cannot be undone.
func survivingTraceIDs(t *testing.T, r Reader, meta *backend.BlockMeta, query string) []string {
	t.Helper()

	_, _, filter, req, err := traceql.Compile(query)
	require.NoError(t, err)

	req.SecondPassConditions = traceql.SearchMetaConditionsWithout(req.Conditions, req.AllConditions)
	req.SecondPass = func(inSS *traceql.Spanset) ([]*traceql.Spanset, error) {
		if inSS == nil || len(inSS.Spans) == 0 {
			return nil, nil
		}
		return filter([]*traceql.Spanset{inSS})
	}

	ctx := context.Background()
	resp, err := r.Fetch(ctx, meta, *req, common.DefaultSearchOptions())
	require.NoError(t, err)
	defer resp.Results.Close()

	var ids []string
	for {
		ss, err := resp.Results.Next(ctx)
		require.NoError(t, err)
		if ss == nil {
			break
		}
		ids = append(ids, util.TraceIDToHexString(ss.TraceID))
		if ss.ReleaseFn != nil {
			ss.ReleaseFn(ss)
		}
	}

	sort.Strings(ids)
	return ids
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

		rewrote, found, newMeta, err := c.RedactBlock(ctx, meta, testTenantID, nil, query, tempopb.RedactionMode_REDACTION_MODE_APPLY, RedactionWindow{})
		require.NoError(t, err)
		require.True(t, rewrote)
		require.Equal(t, 1, found, "exactly one trace matches the query")
		require.NotNil(t, newMeta)
		require.Equal(t, int64(1), newMeta.TotalObjects, "the non-matching trace must survive")
	})

	t.Run("dry-run counts without rewriting", func(t *testing.T) {
		blk := cutTestBlockWithTraces(t, w, data)
		meta := blk.BlockMeta()

		rewrote, found, newMeta, err := c.RedactBlock(ctx, meta, testTenantID, nil, query, tempopb.RedactionMode_REDACTION_MODE_DRY_RUN, RedactionWindow{})
		require.NoError(t, err)
		require.False(t, rewrote, "dry-run must not rewrite")
		require.Equal(t, 1, found, "dry-run still reports the match count")
		require.Nil(t, newMeta)
	})
}

// TestRedactBlockTwoSidedWindowBoundsTheScan verifies a fully-specified window bounds which traces inside a
// block are matched, not only which blocks are selected.
//
// Both traces satisfy the query and differ only in their span timestamps. Without the window applied to
// the fetch, the out-of-window trace is matched and dropped too — over-deletion of data the operator did
// not ask to remove, on a path with no recovery.
func TestRedactBlockTwoSidedWindowBoundsTheScan(t *testing.T) {
	r, w, c, _ := testConfig(t, 0)
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

	rewrote, found, newMeta, err := c.RedactBlock(ctx, meta, testTenantID, nil, query, tempopb.RedactionMode_REDACTION_MODE_APPLY, RedactionWindow{StartNano: startNano, EndNano: endNano})
	require.NoError(t, err)
	require.True(t, rewrote)
	require.Equal(t, 1, found, "only the in-window trace is matched, even though both satisfy the query")
	require.NotNil(t, newMeta)
	require.Equal(t, int64(1), newMeta.TotalObjects, "the out-of-window trace must survive")
	require.Equal(t, []string{util.TraceIDToHexString(idOld)}, survivingTraceIDs(t, r, newMeta, query),
		"the OLD trace must be the one left behind; a count alone cannot tell the window from its inverse")
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

	rewrote, found, newMeta, err := c.RedactBlock(ctx, meta, testTenantID, nil, query, tempopb.RedactionMode_REDACTION_MODE_APPLY, RedactionWindow{})
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

	_, found, newMeta, err := c.RedactBlock(ctx, meta, testTenantID, nil, query, tempopb.RedactionMode_REDACTION_MODE_APPLY, RedactionWindow{})
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
	r, w, c, _ := testConfig(t, 0)
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
			tempopb.RedactionMode_REDACTION_MODE_APPLY, RedactionWindow{StartNano: time.Now().Add(-2 * time.Hour).UnixNano()})
		require.NoError(t, err)
		require.Equal(t, 1, found, "a start-only window must still exclude the out-of-window trace")
		require.Equal(t, int64(1), newMeta.TotalObjects)
		require.Equal(t, []string{util.TraceIDToHexString(idOld)}, survivingTraceIDs(t, r, newMeta, query),
			"a start-only window must delete the RECENT trace and spare the older one")
	})

	t.Run("end only excludes newer traces", func(t *testing.T) {
		blk := cutTestBlockWithTraces(t, w, data)
		_, found, newMeta, err := c.RedactBlock(ctx, blk.BlockMeta(), testTenantID, nil, query,
			tempopb.RedactionMode_REDACTION_MODE_APPLY, RedactionWindow{EndNano: time.Now().Add(-48 * time.Hour).UnixNano()})
		require.NoError(t, err)
		require.Equal(t, 1, found, "an end-only window must still exclude the out-of-window trace")
		require.Equal(t, int64(1), newMeta.TotalObjects)
		require.Equal(t, []string{util.TraceIDToHexString(idRecent)}, survivingTraceIDs(t, r, newMeta, query),
			"an end-only window must delete the OLD trace and spare the recent one")
	})
}

// TestRedactionWindowValidate covers the window predicate exhaustively: which windows RedactBlock will
// honour and which it refuses.
//
// The accept cases matter as much as the reject cases. A one-sided window must stay ACCEPTED here — a
// batch persisted by an older scheduler can still deliver one, and fetchBounds materialises the open
// side to keep the scan bounded. A guard that rejected one-sided windows would turn those in-flight
// batches into permanently failing jobs.
func TestRedactionWindowValidate(t *testing.T) {
	now := time.Now().UnixNano()
	hour := int64(time.Hour)

	for _, tc := range []struct {
		name    string
		window  RedactionWindow
		wantErr string
	}{
		{name: "unbounded", window: RedactionWindow{}},
		{name: "start only", window: RedactionWindow{StartNano: now}},
		{name: "end only", window: RedactionWindow{EndNano: now}},
		{name: "ordered two-sided", window: RedactionWindow{StartNano: now - hour, EndNano: now}},
		{
			name:    "transposed bounds",
			window:  RedactionWindow{StartNano: now, EndNano: now - hour},
			wantErr: "must be before",
		},
		{
			name:    "zero width",
			window:  RedactionWindow{StartNano: now, EndNano: now},
			wantErr: "must be before",
		},
		{
			name:    "negative start",
			window:  RedactionWindow{StartNano: -1, EndNano: now},
			wantErr: "non-negative",
		},
		{
			name:    "negative end",
			window:  RedactionWindow{EndNano: -1},
			wantErr: "non-negative",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.window.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err, "this window must remain usable")
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

// TestRedactBlockRejectsUnusableWindow verifies RedactBlock refuses a window it cannot honour instead
// of scanning with it.
//
// A transposed window selects no traces at all. Without this guard the job completes, reports zero
// found, and the batch advances — the operator is told the block was processed when nothing was
// removed. That is the redaction failure with no external signal: over-deletion is visible in the
// data, under-deletion reported as success is not.
func TestRedactBlockRejectsUnusableWindow(t *testing.T) {
	_, w, c, _ := testConfig(t, 0)
	ctx := context.Background()

	id := test.ValidTraceID(nil)
	nano := uint64(time.Now().Add(-time.Hour).UnixNano())
	data := []testData{{id, traceAtTime(id, nano, nano), uint32(nano / 1e9), uint32(nano / 1e9)}}

	blk := cutTestBlockWithTraces(t, w, data)
	now := time.Now().UnixNano()

	rewrote, found, newMeta, err := c.RedactBlock(ctx, blk.BlockMeta(), testTenantID, nil,
		`{resource.namespace = "checkout"}`, tempopb.RedactionMode_REDACTION_MODE_APPLY,
		RedactionWindow{StartNano: now, EndNano: now - int64(time.Hour)})

	require.ErrorContains(t, err, "must be before")
	require.False(t, rewrote, "a refused window must not rewrite the block")
	require.Zero(t, found)
	require.Nil(t, newMeta)
}

// TestRedactionWindowFetchBounds covers the resolution of a window into block-fetch bounds directly.
//
// The unbounded case is the one that matters most and is invisible at the block level. If the zero
// window stopped reporting "no bound" and instead resolved to [1ns, MaxInt64], the vparquet trace-time
// predicate would be INSTALLED on every whole-tenant redaction — and any trace whose recorded trace-time
// is zero would then be excluded and silently survive. Blocks completed from a replayed WAL carry no
// times at all, which is exactly the population blockOverlapsWindow's indeterminate branch exists for,
// so that is a live under-deletion path reported as success.
func TestRedactionWindowFetchBounds(t *testing.T) {
	const day = int64(24 * time.Hour)

	for _, tc := range []struct {
		name           string
		window         RedactionWindow
		wantOK         bool
		wantLo, wantHi uint64
	}{
		{
			name:   "unbounded installs no predicate at all",
			window: RedactionWindow{},
			wantOK: false,
		},
		{
			name:   "both bounds pass through unchanged",
			window: RedactionWindow{StartNano: day, EndNano: 2 * day},
			wantOK: true, wantLo: uint64(day), wantHi: uint64(2 * day),
		},
		{
			// The open side is materialised rather than left at 0: vparquet gates the predicate on
			// start > 0 && end > 0, so a zero bound would remove the filter instead of widening it.
			name:   "start only materialises the upper bound",
			window: RedactionWindow{StartNano: day},
			wantOK: true, wantLo: uint64(day), wantHi: math.MaxInt64,
		},
		{
			name:   "end only materialises the lower bound at 1ns, not 0",
			window: RedactionWindow{EndNano: day},
			wantOK: true, wantLo: 1, wantHi: uint64(day),
		},
		{
			// Unreachable through RedactBlock, which validates first; kept because the failure is silent.
			name:   "an inverted window installs nothing rather than a predicate matching nothing",
			window: RedactionWindow{StartNano: 2 * day, EndNano: day},
			wantOK: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lo, hi, ok := tc.window.fetchBounds()
			require.Equal(t, tc.wantOK, ok, "ok decides whether a trace-time predicate is installed at all")
			if !tc.wantOK {
				require.Zero(t, lo)
				require.Zero(t, hi)
				return
			}
			require.Equal(t, tc.wantLo, lo)
			require.Equal(t, tc.wantHi, hi)
			require.Positive(t, lo, "a zero lower bound would disable the predicate instead of narrowing it")
			require.Less(t, lo, hi, "bounds must describe a non-empty range")
		})
	}
}

// TestRedactBlockRefusesWindowWithTraceIDs verifies a window alongside an explicit trace-ID list is an
// error rather than a validated-then-ignored parameter.
//
// The ID path resolves each trace with FindTraceByID, which takes no time bound, so the window cannot
// scope it. Accepting the pair deletes each listed trace from every selected block regardless of when its
// spans occurred, while the caller believes the window constrained the operation — under-deletion of the
// rest of the trace and over-deletion within these blocks, both unrecoverable and both reported as
// success. SubmitRedaction refuses the pair too; this is the layer that would do the deleting.
func TestRedactBlockRefusesWindowWithTraceIDs(t *testing.T) {
	_, w, c, _ := testConfig(t, 0)
	ctx := context.Background()

	id := test.ValidTraceID(nil)
	nano := uint64(time.Now().Add(-time.Hour).UnixNano())
	blk := cutTestBlockWithTraces(t, w, []testData{
		{id, traceAtTime(id, nano, nano), uint32(nano / 1e9), uint32(nano / 1e9)},
	})

	window := RedactionWindow{
		StartNano: time.Now().Add(-2 * time.Hour).UnixNano(),
		EndNano:   time.Now().UnixNano(),
	}

	rewrote, found, newMeta, err := c.RedactBlock(ctx, blk.BlockMeta(), testTenantID,
		[]common.ID{id}, "", tempopb.RedactionMode_REDACTION_MODE_APPLY, window)

	require.ErrorContains(t, err, "cannot be combined")
	require.False(t, rewrote, "a refused request must not rewrite the block")
	require.Zero(t, found)
	require.Nil(t, newMeta)

	// The same list without a window still works, so the guard is specific to the combination.
	rewrote, found, _, err = c.RedactBlock(ctx, blk.BlockMeta(), testTenantID,
		[]common.ID{id}, "", tempopb.RedactionMode_REDACTION_MODE_APPLY, RedactionWindow{})
	require.NoError(t, err)
	require.True(t, rewrote)
	require.Equal(t, 1, found)
}
