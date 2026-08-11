package tracesummary

import (
	"sort"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
)

var testTraceID = []byte("trace-id-0000001")

func stringAttribute(key, value string) *commonv1.KeyValue {
	return &commonv1.KeyValue{
		Key:   key,
		Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: value}},
	}
}

func testSpan(spanID, parentID, name string, kind tracev1.Span_SpanKind, start, end uint64, status tracev1.Status_StatusCode, statusMsg string) *tracev1.Span {
	return &tracev1.Span{
		TraceId:           testTraceID,
		SpanId:            []byte(spanID),
		ParentSpanId:      []byte(parentID),
		Name:              name,
		Kind:              kind,
		StartTimeUnixNano: start,
		EndTimeUnixNano:   end,
		Status:            &tracev1.Status{Code: status, Message: statusMsg},
	}
}

func resourceSpans(service string, spans ...*tracev1.Span) *tracev1.ResourceSpans {
	return &tracev1.ResourceSpans{
		Resource: &resourcev1.Resource{
			Attributes: []*commonv1.KeyValue{stringAttribute("service.name", service)},
		},
		ScopeSpans: []*tracev1.ScopeSpans{
			{Spans: spans},
		},
	}
}

func TestSummarize_SimpleMultiLevelTrace(t *testing.T) {
	trace := &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			resourceSpans(
				"checkout",
				// Root's own span ends before some of its descendants (e.g. an
				// async downstream call outliving the parent's response) so the
				// critical path is a genuine multi-hop walk, not just the root.
				testSpan("root", "", "POST /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 60, tracev1.Status_STATUS_CODE_OK, ""),
			),
			resourceSpans(
				"inventory",
				testSpan("inventory", "root", "reserve inventory", tracev1.Span_SPAN_KIND_CLIENT, 20, 90, tracev1.Status_STATUS_CODE_OK, ""),
				testSpan("reserve", "inventory", "reserve", tracev1.Span_SPAN_KIND_SERVER, 25, 100, tracev1.Status_STATUS_CODE_OK, ""),
			),
			resourceSpans(
				"payment",
				testSpan("payment", "root", "charge", tracev1.Span_SPAN_KIND_CLIENT, 30, 45, tracev1.Status_STATUS_CODE_OK, ""),
			),
		},
	}

	got, err := Summarize(trace)
	require.NoError(t, err)

	require.Equal(t, "checkout", got.RootService)
	require.Equal(t, "POST /checkout", got.RootSpanName)
	require.Equal(t, int64(100), got.DurationNanos)
	require.Equal(t, 4, got.SpanCount)
	require.Equal(t, 0, got.ErrorCount)

	var pathNames []string
	for _, p := range got.CriticalPath {
		pathNames = append(pathNames, p.Name)
	}
	require.Equal(t, []string{"POST /checkout", "reserve inventory", "reserve"}, pathNames)

	require.Len(t, got.Services, 3)
	byService := map[string]ServiceBreakdown{}
	for _, s := range got.Services {
		byService[s.Service] = s
	}
	require.Equal(t, ServiceBreakdown{Service: "checkout", SpanCount: 1, ErrorCount: 0, DurationNanos: 60}, byService["checkout"])
	require.Equal(t, ServiceBreakdown{Service: "inventory", SpanCount: 2, ErrorCount: 0, DurationNanos: 145}, byService["inventory"])
	require.Equal(t, ServiceBreakdown{Service: "payment", SpanCount: 1, ErrorCount: 0, DurationNanos: 15}, byService["payment"])
}

func TestSummarize_NormalNestedTraceFollowsLastFinishingBranch(t *testing.T) {
	// Fully-synchronous instrumentation: every span's end time is <= its
	// parent's end time (the common case). The critical path must still be
	// a multi-hop walk, following whichever direct child finishes latest at
	// each level.
	trace := &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			resourceSpans(
				"svc",
				testSpan("root", "", "root-op", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK, ""),
				// branch-a finishes later than branch-b, so the walk descends
				// into branch-a at the root level.
				testSpan("branch-a", "root", "branch-a", tracev1.Span_SPAN_KIND_CLIENT, 10, 90, tracev1.Status_STATUS_CODE_OK, ""),
				testSpan("branch-b", "root", "branch-b", tracev1.Span_SPAN_KIND_CLIENT, 20, 70, tracev1.Status_STATUS_CODE_OK, ""),
				// leaf-a1 finishes later than leaf-a2, so the walk descends
				// into leaf-a1 at the branch-a level.
				testSpan("leaf-a1", "branch-a", "leaf-a1", tracev1.Span_SPAN_KIND_INTERNAL, 15, 85, tracev1.Status_STATUS_CODE_OK, ""),
				testSpan("leaf-a2", "branch-a", "leaf-a2", tracev1.Span_SPAN_KIND_INTERNAL, 30, 60, tracev1.Status_STATUS_CODE_OK, ""),
			),
		},
	}

	got, err := Summarize(trace)
	require.NoError(t, err)

	var pathNames []string
	for _, p := range got.CriticalPath {
		pathNames = append(pathNames, p.Name)
	}
	require.Greater(t, len(pathNames), 1)
	require.Equal(t, []string{"root-op", "branch-a", "leaf-a1"}, pathNames)
}

func TestSummarize_SingleSpanTrace(t *testing.T) {
	trace := &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			resourceSpans(
				"solo",
				testSpan("only", "", "do-thing", tracev1.Span_SPAN_KIND_INTERNAL, 5, 15, tracev1.Status_STATUS_CODE_OK, ""),
			),
		},
	}

	got, err := Summarize(trace)
	require.NoError(t, err)

	require.Equal(t, "solo", got.RootService)
	require.Equal(t, "do-thing", got.RootSpanName)
	require.Equal(t, int64(10), got.DurationNanos)
	require.Equal(t, 1, got.SpanCount)
	require.Len(t, got.CriticalPath, 1)
	require.Equal(t, "do-thing", got.CriticalPath[0].Name)
	require.Len(t, got.SlowestSpans, 1)
	require.Empty(t, got.ErrorSpans)
}

func TestSummarize_NoErrors(t *testing.T) {
	trace := &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			resourceSpans(
				"svc",
				testSpan("root", "", "root-op", tracev1.Span_SPAN_KIND_SERVER, 0, 10, tracev1.Status_STATUS_CODE_OK, ""),
				testSpan("child", "root", "child-op", tracev1.Span_SPAN_KIND_CLIENT, 1, 9, tracev1.Status_STATUS_CODE_UNSET, ""),
			),
		},
	}

	got, err := Summarize(trace)
	require.NoError(t, err)
	require.Equal(t, 0, got.ErrorCount)
	require.Empty(t, got.ErrorSpans)
}

func TestSummarize_TruncatesToFive(t *testing.T) {
	var spans []*tracev1.Span
	spans = append(spans, testSpan("root", "", "root-op", tracev1.Span_SPAN_KIND_SERVER, 0, 1000, tracev1.Status_STATUS_CODE_OK, ""))
	for i := 0; i < 8; i++ {
		id := string([]byte{byte('a' + i)})
		start := uint64(i * 10)
		end := start + uint64(100+i)
		spans = append(spans, testSpan("slow-"+id, "root", "op-"+id, tracev1.Span_SPAN_KIND_CLIENT, start, end, tracev1.Status_STATUS_CODE_ERROR, "boom"))
	}
	trace := &tempopb.Trace{ResourceSpans: []*tracev1.ResourceSpans{resourceSpans("svc", spans...)}}

	got, err := Summarize(trace)
	require.NoError(t, err)

	require.Len(t, got.SlowestSpans, 5)
	require.Len(t, got.ErrorSpans, 5)
	require.Equal(t, 8, got.ErrorCount)

	for i := 1; i < len(got.SlowestSpans); i++ {
		require.GreaterOrEqual(t, got.SlowestSpans[i-1].DurationNanos, got.SlowestSpans[i].DurationNanos)
	}
	for i := 1; i < len(got.ErrorSpans); i++ {
		require.LessOrEqual(t, got.ErrorSpans[i-1].StartTimeUnixNano, got.ErrorSpans[i].StartTimeUnixNano)
	}
}

// The bounded top-K selection must pick exactly what a full sort over every
// span would have picked, including on the tie-break paths.
func TestSummarize_TopKSelectionMatchesFullSort(t *testing.T) {
	durations := []uint64{50, 10, 50, 30, 10, 90, 30, 90, 50, 70, 10, 90}

	spans := []*tracev1.Span{
		testSpan("root", "", "root-op", tracev1.Span_SPAN_KIND_SERVER, 0, 1000, tracev1.Status_STATUS_CODE_OK, ""),
	}
	for i, d := range durations {
		id := string([]byte{byte('a' + i)})
		start := uint64((i % 3) * 10)
		spans = append(spans, testSpan(id, "root", "op-"+id, tracev1.Span_SPAN_KIND_CLIENT, start, start+d, tracev1.Status_STATUS_CODE_ERROR, "boom"))
	}
	trace := &tempopb.Trace{ResourceSpans: []*tracev1.ResourceSpans{resourceSpans("svc", spans...)}}

	got, err := Summarize(trace)
	require.NoError(t, err)

	all := flattenSpans(trace)

	bySlowest := make([]*resolvedSpan, len(all))
	copy(bySlowest, all)
	sort.Slice(bySlowest, func(i, j int) bool { return longerDuration(bySlowest[i].span, bySlowest[j].span) })

	errored := make([]*resolvedSpan, 0, len(all))
	for _, s := range all {
		if isError(s.span) {
			errored = append(errored, s)
		}
	}
	sort.Slice(errored, func(i, j int) bool { return lessByStartThenID(errored[i].span, errored[j].span) })

	gotSlowest := make([]string, len(got.SlowestSpans))
	for i, s := range got.SlowestSpans {
		gotSlowest[i] = s.SpanID
	}
	gotErrors := make([]string, len(got.ErrorSpans))
	for i, s := range got.ErrorSpans {
		gotErrors[i] = s.SpanID
	}

	require.Equal(t, expectedSpanIDs(bySlowest, maxSlowestSpans), gotSlowest)
	require.Equal(t, expectedSpanIDs(errored, maxErrorSpans), gotErrors)
}

func expectedSpanIDs(spans []*resolvedSpan, k int) []string {
	out := make([]string, 0, k)
	for _, s := range spans[:k] {
		out = append(out, util.SpanIDToHexString(s.span.GetSpanId()))
	}
	return out
}

func TestSummarize_FewerThanFive(t *testing.T) {
	trace := &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			resourceSpans(
				"svc",
				testSpan("root", "", "root-op", tracev1.Span_SPAN_KIND_SERVER, 0, 30, tracev1.Status_STATUS_CODE_OK, ""),
				testSpan("child", "root", "child-op", tracev1.Span_SPAN_KIND_CLIENT, 1, 20, tracev1.Status_STATUS_CODE_ERROR, "oops"),
			),
		},
	}

	require.NotPanics(t, func() {
		got, err := Summarize(trace)
		require.NoError(t, err)
		require.Len(t, got.SlowestSpans, 2)
		require.Len(t, got.ErrorSpans, 1)
	})
}

func TestSummarize_DisconnectedTraceFallsBackToEarliestSpan(t *testing.T) {
	// Neither span has an empty parent id pointing nowhere in the trace
	// AND both look like orphans (parent ids that don't resolve to any
	// span present); this exercises the zero-root-candidate fallback only
	// when both spans additionally have non-empty parent ids so neither
	// qualifies as a normal root.
	trace := &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			resourceSpans(
				"orphan-a",
				testSpan("a", "missing-1", "op-a", tracev1.Span_SPAN_KIND_INTERNAL, 50, 60, tracev1.Status_STATUS_CODE_OK, ""),
			),
			resourceSpans(
				"orphan-b",
				testSpan("b", "missing-2", "op-b", tracev1.Span_SPAN_KIND_INTERNAL, 10, 90, tracev1.Status_STATUS_CODE_OK, ""),
			),
		},
	}

	got, err := Summarize(trace)
	require.NoError(t, err)

	// Fallback picks the globally earliest-starting span as synthetic root.
	require.Equal(t, "orphan-b", got.RootService)
	require.Equal(t, "op-b", got.RootSpanName)
	require.Equal(t, int64(80), got.DurationNanos)
}

func TestSummarize_DuplicateSpanIDZipkinPattern(t *testing.T) {
	// This exercises disambiguateParent's own defense-in-depth logic on raw,
	// un-deduped input. It does NOT reflect real /summary endpoint behavior:
	// on that path, combiner.NewTraceByIDV2's deduper resolves zipkin
	// dual-ID pairs before Summarize ever runs (see TestTraceSummaryHandler_
	// ZipkinDualIDTrace_GoesThroughDeduperBeforeSummarize in the frontend
	// package for the real, deduped path).
	//
	// Client and server spans share a span ID (zipkin pattern), across two
	// different ResourceSpans batches, both parented under a real root. A
	// fourth span is the true child of the server-kind half of the pair and
	// must resolve its parent without panicking or misattributing to the
	// client-kind half.
	trace := &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			resourceSpans(
				"root-svc",
				testSpan("root", "", "root-op", tracev1.Span_SPAN_KIND_SERVER, 0, 120, tracev1.Status_STATUS_CODE_OK, ""),
			),
			resourceSpans(
				"caller",
				testSpan("shared", "root", "call", tracev1.Span_SPAN_KIND_CLIENT, 10, 90, tracev1.Status_STATUS_CODE_OK, ""),
			),
			resourceSpans(
				"callee",
				// Ends later than the client-kind half so the top-down walk
				// descends into it, not "call".
				testSpan("shared", "root", "handle", tracev1.Span_SPAN_KIND_SERVER, 15, 95, tracev1.Status_STATUS_CODE_OK, ""),
			),
			resourceSpans(
				"downstream",
				testSpan("grandchild", "shared", "do-work", tracev1.Span_SPAN_KIND_INTERNAL, 20, 80, tracev1.Status_STATUS_CODE_OK, ""),
			),
		},
	}

	var got *Summary
	var err error
	require.NotPanics(t, func() {
		got, err = Summarize(trace)
	})
	require.NoError(t, err)
	require.Equal(t, 4, got.SpanCount)

	var pathNames []string
	for _, p := range got.CriticalPath {
		pathNames = append(pathNames, p.Name)
	}
	// At the root, "handle" ends later than "call" so the walk descends into
	// it. grandchild's parent (kindWant=SERVER, since grandchild is INTERNAL)
	// then correctly resolves to the SERVER-kind "handle", not "call".
	require.Equal(t, []string{"root-op", "handle", "do-work"}, pathNames)
}

func TestSummarize_RootCandidateExcludedByChildOfLink(t *testing.T) {
	linked := testSpan("linked-root", "", "linked-op", tracev1.Span_SPAN_KIND_SERVER, 0, 50, tracev1.Status_STATUS_CODE_OK, "")
	linked.Links = []*tracev1.Span_Link{
		{
			TraceId: testTraceID,
			SpanId:  []byte("elsewhere"),
			Attributes: []*commonv1.KeyValue{
				stringAttribute("opentracing.ref_type", "child_of"),
			},
		},
	}
	realRoot := testSpan("real-root", "", "real-op", tracev1.Span_SPAN_KIND_SERVER, 10, 20, tracev1.Status_STATUS_CODE_OK, "")

	trace := &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			resourceSpans("linked-svc", linked),
			resourceSpans("real-svc", realRoot),
		},
	}

	got, err := Summarize(trace)
	require.NoError(t, err)

	require.Equal(t, "real-svc", got.RootService)
	require.Equal(t, "real-op", got.RootSpanName)
}

func TestCriticalPath_CycleGuardTerminates(t *testing.T) {
	// Malformed data: A's parent is B and B's parent is A, a mutual (2-node)
	// cycle with no true root. findRoot falls back to the earliest-starting
	// span. The critical path walk must still terminate via the visited-map
	// guard rather than looping forever, and must produce a sane, bounded
	// result (both cycle members visited once each).
	trace := &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			resourceSpans(
				"svc",
				testSpan("a", "b", "op-a", tracev1.Span_SPAN_KIND_INTERNAL, 0, 100, tracev1.Status_STATUS_CODE_OK, ""),
				testSpan("b", "a", "op-b", tracev1.Span_SPAN_KIND_INTERNAL, 10, 90, tracev1.Status_STATUS_CODE_OK, ""),
			),
		},
	}

	var got *Summary
	var err error
	require.NotPanics(t, func() {
		got, err = Summarize(trace)
	})
	require.NoError(t, err)

	var pathNames []string
	for _, p := range got.CriticalPath {
		pathNames = append(pathNames, p.Name)
	}
	require.Equal(t, []string{"op-a", "op-b"}, pathNames)
}

func TestCriticalPath_SelfReferentialSpanTerminates(t *testing.T) {
	// A single span whose ParentSpanId equals its own SpanId. No span in the
	// trace has an empty parent, so findRoot falls back to the (only) span,
	// which is then also its own child in the children index. The walk must
	// terminate immediately rather than looping on itself forever.
	trace := &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			resourceSpans(
				"svc",
				testSpan("loopy", "loopy", "op-loopy", tracev1.Span_SPAN_KIND_INTERNAL, 0, 100, tracev1.Status_STATUS_CODE_OK, ""),
			),
		},
	}

	var got *Summary
	var err error
	require.NotPanics(t, func() {
		got, err = Summarize(trace)
	})
	require.NoError(t, err)

	var pathNames []string
	for _, p := range got.CriticalPath {
		pathNames = append(pathNames, p.Name)
	}
	require.Equal(t, []string{"op-loopy"}, pathNames)
}

func TestSummarize_NilTraceReturnsError(t *testing.T) {
	got, err := Summarize(nil)
	require.Error(t, err)
	require.Nil(t, got)
}

func TestSummarize_EmptyTraceReturnsZeroSummary(t *testing.T) {
	got, err := Summarize(&tempopb.Trace{})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, 0, got.SpanCount)
	require.Empty(t, got.CriticalPath)
}
