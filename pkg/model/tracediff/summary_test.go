package tracediff

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummarize(t *testing.T) {
	base, compare := summaryFixtureTraces()

	got, err := Summarize(base, compare)
	require.NoError(t, err)

	assert.Equal(t, VersionTraceSummaryV0Native, got.Version)
	assert.Equal(t, Signals{
		TraceLatency:    "increased",
		SumSpanDuration: "increased",
		Errors:          "increased",
		SpanCount:       "unchanged",
		Structure:       "unchanged",
	}, got.Signals)
	assert.Equal(t, 1.0, got.Trust.MatchedSpanRatio)
	assert.Equal(t, 1.0, got.Trust.StructureOverlap)
	assert.Equal(t, SummaryStats{
		TraceLatencyDeltaMs:    10,
		SumSpanDurationDeltaMs: 12,
		SpanCountDelta:         0,
		ErrorSpanDelta:         1,
		MatchedSpans:           3,
		ModifiedSpans:          2,
		FieldChanges:           1,
		AttributeChanges:       1,
	}, got.Stats)

	assert.Equal(t, []string{"checkout"}, got.ChangedServices)
	assert.Equal(t, []ServiceRollup{
		{Name: "checkout", SumSpanDurationDeltaMs: 12, Modified: 2, NewErrors: 1},
	}, got.Services)
}

func TestSummarizeNativeJSONShape(t *testing.T) {
	base, compare := summaryFixtureTraces()

	got, err := Summarize(base, compare)
	require.NoError(t, err)
	data, err := json.MarshalIndent(got, "", "  ")
	require.NoError(t, err)

	assert.Equal(t, `{
  "version": "trace-summary-v0-native",
  "base": {
    "traceId": "74726163652d69642d30303030303031",
    "rootService": "checkout",
    "rootName": "GET /checkout",
    "spanCount": 3,
    "errorSpanCount": 0,
    "durationMs": 100,
    "sumSpanDurationMs": 140
  },
  "compare": {
    "traceId": "74726163652d69642d30303030303031",
    "rootService": "checkout",
    "rootName": "GET /checkout",
    "spanCount": 3,
    "errorSpanCount": 1,
    "durationMs": 110,
    "sumSpanDurationMs": 152
  },
  "signals": {
    "traceLatency": "increased",
    "sumSpanDuration": "increased",
    "errors": "increased",
    "spanCount": "unchanged",
    "structure": "unchanged"
  },
  "trust": {
    "matchedSpanRatio": 1,
    "structureOverlap": 1
  },
  "stats": {
    "traceLatencyDeltaMs": 10,
    "sumSpanDurationDeltaMs": 12,
    "spanCountDelta": 0,
    "errorSpanDelta": 1,
    "matchedSpans": 3,
    "modifiedSpans": 2,
    "addedSpans": 0,
    "removedSpans": 0,
    "fieldChanges": 1,
    "attributeChanges": 1
  },
  "changedServices": [
    "checkout"
  ],
  "services": [
    {
      "name": "checkout",
      "sumSpanDurationDeltaMs": 12,
      "modified": 2,
      "added": 0,
      "removed": 0,
      "newErrors": 1,
      "resolvedErrors": 0
    }
  ]
}`, string(data))
}

func TestSummarizeStructureSignalDetectsTopologyChange(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	base := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "auth", "root", "auth", "authorize", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "db", "root", "db", "SELECT user", tracev1.Span_SPAN_KIND_CLIENT, 20, 30, tracev1.Status_STATUS_CODE_OK),
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "auth", "root", "auth", "authorize", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "db", "auth", "db", "SELECT user", tracev1.Span_SPAN_KIND_CLIENT, 20, 30, tracev1.Status_STATUS_CODE_OK),
	)

	got, err := Summarize(base, compare)
	require.NoError(t, err)
	assert.Equal(t, 0, got.Stats.AddedSpans)
	assert.Equal(t, 0, got.Stats.RemovedSpans)
	assert.Equal(t, "changed", got.Signals.Structure)
	assert.Equal(t, []string{"db"}, got.ChangedServices)
	assert.Empty(t, got.Services)
}

func TestSummarizePureReparentJoinsChangedServicesOnly(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	base := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "auth", "root", "auth", "authorize", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "db", "auth", "db", "SELECT user", tracev1.Span_SPAN_KIND_CLIENT, 20, 30, tracev1.Status_STATUS_CODE_OK),
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "auth", "root", "auth", "authorize", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "db", "root", "db", "SELECT user", tracev1.Span_SPAN_KIND_CLIENT, 20, 30, tracev1.Status_STATUS_CODE_OK),
	)

	got, err := Summarize(base, compare)
	require.NoError(t, err)
	assert.Equal(t, "changed", got.Signals.Structure)
	assert.Equal(t, 0, got.Stats.ModifiedSpans)
	assert.Equal(t, []string{"db"}, got.ChangedServices)
	assert.Empty(t, got.Services)
}

func TestSummarizeWarnsOnReparentBetweenDuplicateLogicalParents(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func(childParent string) *tempopb.Trace {
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "root", "", "gateway", "GET /", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "parent-a", "root", "worker", "process", tracev1.Span_SPAN_KIND_INTERNAL, 10, 40, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "parent-b", "root", "worker", "process", tracev1.Span_SPAN_KIND_INTERNAL, 50, 80, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "child", childParent, "db", "query", tracev1.Span_SPAN_KIND_CLIENT, 20, 30, tracev1.Status_STATUS_CODE_OK),
		)
	}

	got, err := Summarize(makeTrace("parent-a"), makeTrace("parent-b"))
	require.NoError(t, err)
	assert.Zero(t, got.Stats.ModifiedSpans)
	assert.Zero(t, got.Stats.AddedSpans)
	assert.Zero(t, got.Stats.RemovedSpans)
	assert.Equal(t, "unchanged", got.Signals.Structure)
	assert.Contains(t, got.Warnings, Warning{
		Code:    WarningAmbiguousSpanMatch,
		Message: "1 duplicate logical span group(s) may match ambiguously; matching minimizes changes, but instance-level transitions may not identify the same physical operation",
	})
}

func TestSummarizeDuplicateLogicalParentReorderIsStable(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func(parentAStart, parentBStart uint64) *tempopb.Trace {
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "root", "", "gateway", "GET /", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "parent-a", "root", "worker", "process", tracev1.Span_SPAN_KIND_INTERNAL, parentAStart, parentAStart+30, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "parent-b", "root", "worker", "process", tracev1.Span_SPAN_KIND_INTERNAL, parentBStart, parentBStart+30, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "child-a", "parent-a", "db", "query", tracev1.Span_SPAN_KIND_CLIENT, parentAStart+10, parentAStart+20, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "child-b", "parent-b", "db", "query", tracev1.Span_SPAN_KIND_CLIENT, parentBStart+10, parentBStart+20, tracev1.Status_STATUS_CODE_OK),
		)
	}

	got, err := Summarize(makeTrace(10, 50), makeTrace(50, 10))
	require.NoError(t, err)
	assert.Equal(t, "unchanged", got.Signals.Structure)
	assert.Contains(t, got.Warnings, Warning{
		Code:    WarningAmbiguousSpanMatch,
		Message: "2 duplicate logical span group(s) may match ambiguously; matching minimizes changes, but instance-level transitions may not identify the same physical operation",
	})
}

func TestSummarizeMatchingCoverageRatiosDoNotRoundNearPerfectToOne(t *testing.T) {
	assert.NotEqual(t, 1.0, matchedSpanRatio(10_000, 10_000, 9_999))
	assert.Equal(t, float64(9_999)/float64(10_000), matchedSpanRatio(10_000, 10_000, 9_999))
	assert.Equal(t, 1.0, matchedSpanRatio(10_000, 10_000, 10_000))
}

func TestSummarizeTreatsIdenticalNonFiniteAttributesAsUnchanged(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func() *tempopb.Trace {
		root := spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK)
		root.Attributes = append(root.Attributes, doubleAttribute("score", math.NaN()))
		return traceWithNamedSpans(root)
	}

	got, err := Summarize(makeTrace(), makeTrace())
	require.NoError(t, err)
	assert.Equal(t, "unchanged", got.Signals.TraceLatency)
	assert.Equal(t, "unchanged", got.Signals.SumSpanDuration)
	assert.Equal(t, "unchanged", got.Signals.Errors)
	assert.Equal(t, 0, got.Stats.AttributeChanges)
	assert.Empty(t, got.Services)
}

func TestSummarizeSignalsReportIndependentDirections(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	// Fewer errors but higher latency: report both facts without declaring whether
	// the overall change is good or bad.
	base := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_ERROR),
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 150, tracev1.Status_STATUS_CODE_OK),
	)

	got, err := Summarize(base, compare)
	require.NoError(t, err)
	assert.Equal(t, "increased", got.Signals.TraceLatency)
	assert.Equal(t, "decreased", got.Signals.Errors)
}

func TestSummarizeIgnoresUnsetSpanStartInDuration(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	// A child span has an unset (zero) start. It contributes zero while the valid
	// root still determines the trace envelope and work.
	makeTrace := func() *tempopb.Trace {
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
			&tracev1.Span{
				TraceId:           traceID,
				SpanId:            []byte("child"),
				ParentSpanId:      []byte("root"),
				Name:              "cache lookup",
				Kind:              tracev1.Span_SPAN_KIND_CLIENT,
				StartTimeUnixNano: 0, // unset
				EndTimeUnixNano:   (normalizeTestTimeOffsetMs + 50) * 1_000_000,
				Attributes:        []*commonv1.KeyValue{stringAttribute("service.name", "checkout")},
				Status:            &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK},
			},
		)
	}

	got, err := Summarize(makeTrace(), makeTrace())
	require.NoError(t, err)
	assert.Equal(t, int64(100), got.Base.DurationMs)
	assert.Equal(t, int64(100), got.Compare.DurationMs)
	assert.Zero(t, got.Stats.TraceLatencyDeltaMs)
	assert.Equal(t, int64(100), got.Base.SumSpanDurationMs)
	assert.Equal(t, int64(100), got.Compare.SumSpanDurationMs)
	assert.Zero(t, got.Stats.SumSpanDurationDeltaMs)
	assert.Equal(t, "unchanged", got.Signals.TraceLatency)
	assert.Equal(t, "unchanged", got.Signals.SumSpanDuration)
}

func TestSummarizeUnsetSpanStartDoesNotInflateSumSpanDuration(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	// Baseline: a root and a child, both with usable start times.
	base := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "child", "root", "checkout", "cache lookup", tracev1.Span_SPAN_KIND_CLIENT, 10, 60, tracev1.Status_STATUS_CODE_OK),
	)
	// Comparison: the child keeps a realistic end but loses its start and
	// contributes zero work.
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
		&tracev1.Span{
			TraceId:           traceID,
			SpanId:            []byte("child"),
			ParentSpanId:      []byte("root"),
			Name:              "cache lookup",
			Kind:              tracev1.Span_SPAN_KIND_CLIENT,
			StartTimeUnixNano: 0, // unset
			EndTimeUnixNano:   (normalizeTestTimeOffsetMs + 60) * 1_000_000,
			Attributes:        []*commonv1.KeyValue{stringAttribute("service.name", "checkout")},
			Status:            &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK},
		},
	)

	got, err := Summarize(base, compare)
	require.NoError(t, err)
	assert.Equal(t, int64(150), got.Base.SumSpanDurationMs)
	assert.Equal(t, int64(100), got.Compare.SumSpanDurationMs)
	assert.Equal(t, int64(-50), got.Stats.SumSpanDurationDeltaMs)
	assert.Equal(t, "decreased", got.Signals.SumSpanDuration)
}

func TestSummarizeStructuralSurfacesAddedAndRemovedErrors(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	base := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "gone", "root", "checkout", "dropped call", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, tracev1.Status_STATUS_CODE_ERROR),
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "new", "root", "checkout", "new call", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, tracev1.Status_STATUS_CODE_ERROR),
	)

	got, err := Summarize(base, compare)
	require.NoError(t, err)

	require.Len(t, got.Services, 1)
	assert.Equal(t, ServiceRollup{
		Name:                   "checkout",
		SumSpanDurationDeltaMs: 0,
		Added:                  1,
		Removed:                1,
		NewErrors:              1,
		ResolvedErrors:         1,
	}, got.Services[0])
}

func TestSummarizeInvalidDurationWarnsAndDoesNotInflateEnvelope(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func() *tempopb.Trace {
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
			&tracev1.Span{
				TraceId:           traceID,
				SpanId:            []byte("bad-child"),
				ParentSpanId:      []byte("root"),
				Name:              "bad duration",
				Kind:              tracev1.Span_SPAN_KIND_CLIENT,
				StartTimeUnixNano: 0,
				EndTimeUnixNano:   (normalizeTestTimeOffsetMs + 1000) * 1_000_000,
				Attributes:        []*commonv1.KeyValue{stringAttribute("service.name", "checkout")},
				Status:            &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK},
			},
		)
	}

	got, err := Summarize(makeTrace(), makeTrace())
	require.NoError(t, err)
	assert.Equal(t, int64(100), got.Base.DurationMs)
	assert.Equal(t, int64(100), got.Compare.DurationMs)
	require.Len(t, got.Warnings, 2)
	assert.Equal(t, WarningInvalidDuration, got.Warnings[0].Code)
	assert.Contains(t, got.Warnings[0].Message, "base trace")
	assert.Equal(t, WarningInvalidDuration, got.Warnings[1].Code)
	assert.Contains(t, got.Warnings[1].Message, "compare trace")
}

func TestSummarizeInvalidDurationOnlyServiceKeepsRollupCounts(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func(status tracev1.Status_StatusCode) *tempopb.Trace {
		return traceWithNamedSpans(&tracev1.Span{
			TraceId:           traceID,
			SpanId:            []byte("invalid"),
			Name:              "invalid operation",
			Kind:              tracev1.Span_SPAN_KIND_INTERNAL,
			StartTimeUnixNano: 0,
			EndTimeUnixNano:   100_000_000,
			Attributes:        []*commonv1.KeyValue{stringAttribute("service.name", "broken")},
			Status:            &tracev1.Status{Code: status},
		})
	}

	got, err := Summarize(makeTrace(tracev1.Status_STATUS_CODE_OK), makeTrace(tracev1.Status_STATUS_CODE_ERROR))
	require.NoError(t, err)
	assert.Equal(t, []string{"broken"}, got.ChangedServices)
	assert.Equal(t, []ServiceRollup{{Name: "broken", Modified: 1, NewErrors: 1}}, got.Services)
}

func TestSummarizeZeroSpanTracesWarnAndStayUnchanged(t *testing.T) {
	got, err := Summarize(&tempopb.Trace{}, &tempopb.Trace{})
	require.NoError(t, err)

	require.Len(t, got.Warnings, 2)
	assert.Equal(t, WarningZeroSpanTrace, got.Warnings[0].Code)
	assert.Contains(t, got.Warnings[0].Message, "base trace")
	assert.Equal(t, WarningZeroSpanTrace, got.Warnings[1].Code)
	assert.Contains(t, got.Warnings[1].Message, "compare trace")
	// The (0,0) branch pins both coverage ratios at 1; the warning explains that
	// the vacuous ratios carry no matching evidence.
	assert.Equal(t, 1.0, got.Trust.MatchedSpanRatio)
	assert.Equal(t, 1.0, got.Trust.StructureOverlap)
	assert.Equal(t, Signals{
		TraceLatency:    "unchanged",
		SumSpanDuration: "unchanged",
		Errors:          "unchanged",
		SpanCount:       "unchanged",
		Structure:       "unchanged",
	}, got.Signals)
}

func TestSummarizeOneSideEmptyTraceZeroesMatchingCoverageRatios(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
	)

	got, err := Summarize(&tempopb.Trace{}, compare)
	require.NoError(t, err)
	assert.Equal(t, 0.0, got.Trust.MatchedSpanRatio)
	assert.Equal(t, 0.0, got.Trust.StructureOverlap)
	require.Len(t, got.Warnings, 1)
	assert.Equal(t, WarningZeroSpanTrace, got.Warnings[0].Code)
	assert.Contains(t, got.Warnings[0].Message, "base trace")
}

func TestSummarizeDuplicateSpanIDsWarn(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func() *tempopb.Trace {
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "dup", "root", "checkout", "op a", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "dup", "root", "checkout", "op b", tracev1.Span_SPAN_KIND_CLIENT, 30, 40, tracev1.Status_STATUS_CODE_OK),
		)
	}

	got, err := Summarize(makeTrace(), makeTrace())
	require.NoError(t, err)
	require.Len(t, got.Warnings, 2)
	assert.Equal(t, WarningDuplicateSpanID, got.Warnings[0].Code)
	assert.Contains(t, got.Warnings[0].Message, "base trace has 1 span(s) with duplicate span IDs")
	assert.Equal(t, WarningDuplicateSpanID, got.Warnings[1].Code)
	assert.Contains(t, got.Warnings[1].Message, "compare trace has 1 span(s) with duplicate span IDs")
}

func TestSummarizeEmptyServicesEncodeAsEmptyArray(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func() *tempopb.Trace {
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
		)
	}

	got, err := Summarize(makeTrace(), makeTrace())
	require.NoError(t, err)
	data, err := json.Marshal(got)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"services":[]`)
}

func TestSummarizeNilTrace(t *testing.T) {
	got, err := Summarize(nil, &tempopb.Trace{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilTrace)
	assert.Nil(t, got)
}

func TestDriftSignificantBoundary(t *testing.T) {
	const maxInt64Value = int64(^uint64(0) >> 1)
	tests := []struct {
		name       string
		baseNanos  int64
		deltaNanos int64
		want       bool
	}{
		{name: "exactly the absolute floor", baseNanos: 0, deltaNanos: driftMinNanos, want: true},
		{name: "just below the absolute floor", baseNanos: 0, deltaNanos: driftMinNanos - 1, want: false},
		{name: "exactly the relative threshold", baseNanos: 100_000_000, deltaNanos: 5_000_000, want: true},
		{name: "just below the relative threshold", baseNanos: 100_000_000, deltaNanos: 4_999_999, want: false},
		{name: "negative delta at the relative threshold", baseNanos: 100_000_000, deltaNanos: -5_000_000, want: true},
		{name: "large value just below exact relative threshold", baseNanos: maxInt64Value, deltaNanos: maxInt64Value / 20, want: false},
		{name: "large value at exact relative threshold", baseNanos: maxInt64Value, deltaNanos: maxInt64Value/20 + 1, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, driftSignificant(tt.baseNanos, tt.deltaNanos))
		})
	}
}

func TestSummarizeSystemicDriftAttribution(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func(spanDurationMs uint64) *tempopb.Trace {
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "root", "", "frontend", "GET /", tracev1.Span_SPAN_KIND_SERVER, 0, spanDurationMs, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "backend", "root", "backend", "process", tracev1.Span_SPAN_KIND_CLIENT, 10, 10+spanDurationMs, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "db", "backend", "database", "SELECT orders", tracev1.Span_SPAN_KIND_CLIENT, 20, 20+spanDurationMs, tracev1.Status_STATUS_CODE_OK),
		)
	}

	// +13% per span is inside the differ's 20%/1ms duration tolerance, so the
	// patch is empty; only the per-service aggregates can attribute the drift.
	got, err := Summarize(makeTrace(100), makeTrace(113))
	require.NoError(t, err)
	assert.Equal(t, 0, got.Stats.ModifiedSpans)
	assert.Equal(t, 0, got.Stats.FieldChanges)
	assert.Equal(t, []ServiceRollup{
		{Name: "backend", SumSpanDurationDeltaMs: 13},
		{Name: "database", SumSpanDurationDeltaMs: 13},
		{Name: "frontend", SumSpanDurationDeltaMs: 13},
	}, got.Services)
}

func TestSummarizeNanosecondJitterNotFlagged(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func() *tempopb.Trace {
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "child", "root", "checkout", "cache lookup", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, tracev1.Status_STATUS_CODE_OK),
		)
	}

	tests := []struct {
		name       string
		endShiftNs int64
	}{
		{name: "one nanosecond slower", endShiftNs: 1},
		{name: "one nanosecond faster", endShiftNs: -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compare := makeTrace()
			child := compare.ResourceSpans[0].ScopeSpans[0].Spans[1]
			child.EndTimeUnixNano = uint64(int64(child.EndTimeUnixNano) + tt.endShiftNs)

			got, err := Summarize(makeTrace(), compare)
			require.NoError(t, err)
			assert.Equal(t, 0, got.Stats.ModifiedSpans)
			assert.Empty(t, got.Services)
			assert.Zero(t, got.Stats.SumSpanDurationDeltaMs)
			assert.Equal(t, "unchanged", got.Signals.SumSpanDuration)
		})
	}
}

func TestSummarizeSubMillisecondDriftAccumulatesInNanoseconds(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func() *tempopb.Trace {
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 5, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "a", "root", "checkout", "op a", tracev1.Span_SPAN_KIND_CLIENT, 10, 15, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "b", "root", "checkout", "op b", tracev1.Span_SPAN_KIND_CLIENT, 20, 25, tracev1.Status_STATUS_CODE_OK),
		)
	}

	// Every span slows by 0.4ms. Nanosecond accumulation must preserve the full
	// 1.2ms delta in both trace stats and the service rollup.
	compare := makeTrace()
	for _, span := range compare.ResourceSpans[0].ScopeSpans[0].Spans {
		span.EndTimeUnixNano += 400_000
	}

	got, err := Summarize(makeTrace(), compare)
	require.NoError(t, err)
	assert.Equal(t, 0, got.Stats.ModifiedSpans)
	assert.Equal(t, int64(15), got.Base.SumSpanDurationMs)
	assert.Equal(t, int64(16), got.Compare.SumSpanDurationMs)
	assert.Equal(t, int64(1), got.Stats.SumSpanDurationDeltaMs)
	assert.Equal(t, "increased", got.Signals.SumSpanDuration)
	require.Len(t, got.Services, 1)
	assert.Equal(t, int64(1), got.Services[0].SumSpanDurationDeltaMs)
}

func TestSummarizeErrorChurnFromMatcherCounts(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func(aStatus, bStatus tracev1.Status_StatusCode) *tempopb.Trace {
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "a", "root", "checkout", "op a", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, aStatus),
			spanForNormalizeTest(traceID, "b", "root", "checkout", "op b", tracev1.Span_SPAN_KIND_CLIENT, 30, 40, bStatus),
		)
	}

	got, err := Summarize(
		makeTrace(tracev1.Status_STATUS_CODE_OK, tracev1.Status_STATUS_CODE_ERROR),
		makeTrace(tracev1.Status_STATUS_CODE_ERROR, tracev1.Status_STATUS_CODE_OK),
	)
	require.NoError(t, err)
	// The net error delta cancels to zero; only the matcher-derived counts keep
	// the simultaneous churn visible.
	assert.Equal(t, 0, got.Stats.ErrorSpanDelta)
	require.Len(t, got.Services, 1)
	assert.Equal(t, 1, got.Services[0].NewErrors)
	assert.Equal(t, 1, got.Services[0].ResolvedErrors)
}

func TestSummarizeWarnsOnAmbiguousDuplicateErrorChurn(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func(first, second tracev1.Status_StatusCode) *tempopb.Trace {
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "root", "", "gateway", "GET /", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "first", "root", "backend", "duplicate", tracev1.Span_SPAN_KIND_INTERNAL, 10, 20, first),
			spanForNormalizeTest(traceID, "second", "root", "backend", "duplicate", tracev1.Span_SPAN_KIND_INTERNAL, 30, 40, second),
		)
	}

	got, err := Summarize(
		makeTrace(tracev1.Status_STATUS_CODE_OK, tracev1.Status_STATUS_CODE_ERROR),
		makeTrace(tracev1.Status_STATUS_CODE_ERROR, tracev1.Status_STATUS_CODE_OK),
	)
	require.NoError(t, err)
	assert.Zero(t, got.Stats.ModifiedSpans)
	assert.Empty(t, got.Services)
	assert.Contains(t, got.Warnings, Warning{
		Code:    WarningAmbiguousSpanMatch,
		Message: "1 duplicate logical span group(s) may match ambiguously; matching minimizes changes, but instance-level transitions may not identify the same physical operation",
	})
}

func TestSummarizeUsesCanonicalAmbiguousMatcherCounts(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	span := func(id, variant string, start uint64) *tracev1.Span {
		span := spanForNormalizeTest(traceID, id, "root", "backend", "duplicate", tracev1.Span_SPAN_KIND_INTERNAL, start, start+10, tracev1.Status_STATUS_CODE_OK)
		span.Attributes = append(span.Attributes, stringAttribute("variant", variant))
		return span
	}
	base := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "gateway", "GET /", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
		span("base-a", "a", 10),
		span("base-b", "b", 30),
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "gateway", "GET /", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
		span("compare-c", "c", 10),
		span("compare-a", "a", 30),
	)

	got, err := Summarize(base, compare)
	require.NoError(t, err)
	assert.Equal(t, 1, got.Stats.ModifiedSpans)
	assert.Equal(t, []ServiceRollup{{Name: "backend", Modified: 1}}, got.Services)
}

func TestSummarizeWarnsWhenDuplicateParentsAllChange(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func(parentAService, parentBService string) *tempopb.Trace {
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "root", "", "gateway", "GET /", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "parent-a", "root", parentAService, "parent", tracev1.Span_SPAN_KIND_INTERNAL, 10, 40, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "parent-b", "root", parentBService, "parent", tracev1.Span_SPAN_KIND_INTERNAL, 50, 80, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "first", "parent-a", "backend", "duplicate", tracev1.Span_SPAN_KIND_INTERNAL, 20, 30, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "second", "parent-b", "backend", "duplicate", tracev1.Span_SPAN_KIND_INTERNAL, 60, 70, tracev1.Status_STATUS_CODE_ERROR),
		)
	}

	got, err := Summarize(makeTrace("a", "b"), makeTrace("c", "d"))
	require.NoError(t, err)
	assert.Contains(t, got.Warnings, Warning{
		Code:    WarningAmbiguousSpanMatch,
		Message: "1 duplicate logical span group(s) may match ambiguously; matching minimizes changes, but instance-level transitions may not identify the same physical operation",
	})
}

func TestSummarizeKeepsOpposingTransitionsUnderDistinctParents(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func(first, second tracev1.Status_StatusCode) *tempopb.Trace {
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "root", "", "gateway", "GET /", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "parent-a", "root", "a", "parent a", tracev1.Span_SPAN_KIND_INTERNAL, 10, 40, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "parent-b", "root", "b", "parent b", tracev1.Span_SPAN_KIND_INTERNAL, 50, 80, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "first", "parent-a", "backend", "duplicate", tracev1.Span_SPAN_KIND_INTERNAL, 20, 30, first),
			spanForNormalizeTest(traceID, "second", "parent-b", "backend", "duplicate", tracev1.Span_SPAN_KIND_INTERNAL, 60, 70, second),
		)
	}

	got, err := Summarize(
		makeTrace(tracev1.Status_STATUS_CODE_OK, tracev1.Status_STATUS_CODE_ERROR),
		makeTrace(tracev1.Status_STATUS_CODE_ERROR, tracev1.Status_STATUS_CODE_OK),
	)
	require.NoError(t, err)
	require.Len(t, got.Services, 1)
	assert.Equal(t, 1, got.Services[0].NewErrors)
	assert.Equal(t, 1, got.Services[0].ResolvedErrors)
	for _, warning := range got.Warnings {
		assert.NotEqual(t, WarningAmbiguousSpanMatch, warning.Code)
	}
}

func TestSummarizeRootFallsBackForCycleOnlyTrace(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	// Every span's parent is present, forming a cycle with no structural root.
	// The summary must still report a deterministic root (earliest by start time,
	// then span ID) rather than leaving the root fields empty.
	makeTrace := func() *tempopb.Trace {
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "a", "b", "checkout", "span a", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "b", "a", "checkout", "span b", tracev1.Span_SPAN_KIND_CLIENT, 0, 30, tracev1.Status_STATUS_CODE_OK),
		)
	}

	got, err := Summarize(makeTrace(), makeTrace())
	require.NoError(t, err)
	assert.Equal(t, "checkout", got.Base.RootService)
	assert.Equal(t, "span b", got.Base.RootName)
}

func TestSummarizeRootHandlesDanglingParent(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	// The true root's parent span was not returned (a partial trace), so its
	// parent ID dangles. The root must still be identified.
	makeTrace := func() *tempopb.Trace {
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "root", "missing-parent", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "child", "root", "checkout", "cache lookup", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, tracev1.Status_STATUS_CODE_OK),
		)
	}

	got, err := Summarize(makeTrace(), makeTrace())
	require.NoError(t, err)
	assert.Equal(t, "checkout", got.Base.RootService)
	assert.Equal(t, "GET /checkout", got.Base.RootName)
	assert.Equal(t, "checkout", got.Compare.RootService)
	assert.Equal(t, "GET /checkout", got.Compare.RootName)
}

func TestSummarizeTraceLatencyDeltaUsesRawNanoseconds(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func(durationNanos uint64) *tempopb.Trace {
		span := spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 1, tracev1.Status_STATUS_CODE_OK)
		span.EndTimeUnixNano = span.StartTimeUnixNano + durationNanos
		return traceWithNamedSpans(span)
	}

	got, err := Summarize(makeTrace(1_999_999), makeTrace(2_000_000))
	require.NoError(t, err)
	assert.Equal(t, int64(1), got.Base.DurationMs)
	assert.Equal(t, int64(2), got.Compare.DurationMs)
	assert.Zero(t, got.Stats.TraceLatencyDeltaMs)
	assert.Equal(t, "unchanged", got.Signals.TraceLatency)
}

func TestSummarizeMissingServiceUsesReferenceUnknownFallback(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func(status tracev1.Status_StatusCode) *tempopb.Trace {
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "missing", "", "", "missing service", tracev1.Span_SPAN_KIND_INTERNAL, 0, 10, status),
			spanForNormalizeTest(traceID, "literal", "", "<unknown>", "literal service", tracev1.Span_SPAN_KIND_INTERNAL, 20, 30, status),
		)
	}

	got, err := Summarize(makeTrace(tracev1.Status_STATUS_CODE_OK), makeTrace(tracev1.Status_STATUS_CODE_ERROR))
	require.NoError(t, err)
	require.Len(t, got.Services, 1)
	assert.Equal(t, "<unknown>", got.Services[0].Name)
	assert.Equal(t, 2, got.Services[0].NewErrors)

	data, err := json.Marshal(got)
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"rootService"`)
	assert.Contains(t, string(data), `"name":"\u003cunknown\u003e"`)
}

func TestSummarizeKeepsCompleteServiceSections(t *testing.T) {
	const serviceCount = 60
	traceID := []byte("trace-id-0000001")
	makeTrace := func(status tracev1.Status_StatusCode) *tempopb.Trace {
		spans := make([]*tracev1.Span, 0, serviceCount)
		for i := 0; i < serviceCount; i++ {
			spans = append(spans, spanForNormalizeTest(
				traceID,
				fmt.Sprintf("span-%02d", i),
				"",
				fmt.Sprintf("service-%02d", i),
				"op",
				tracev1.Span_SPAN_KIND_INTERNAL,
				uint64(i*2),
				uint64(i*2+1),
				status,
			))
		}
		return traceWithNamedSpans(spans...)
	}

	got, err := Summarize(makeTrace(tracev1.Status_STATUS_CODE_OK), makeTrace(tracev1.Status_STATUS_CODE_ERROR))
	require.NoError(t, err)
	assert.Len(t, got.ChangedServices, serviceCount)
	assert.Len(t, got.Services, serviceCount)
}

func TestBuildServiceSectionsRetainsLargestDurationSumChange(t *testing.T) {
	const serviceCount = 50
	baseDurationSums := make(map[string]int64, serviceCount+1)
	compareDurationSums := make(map[string]int64, serviceCount+1)
	patch := &Result{}
	for i := 0; i < serviceCount; i++ {
		service := fmt.Sprintf("error-%02d", i)
		baseDurationSums[service] = 10_000_000
		compareDurationSums[service] = 10_000_000
		patch.Modified = append(patch.Modified, ModifiedSpan{
			Span: SpanRef{Service: service},
			Changes: []Change{{
				Op:     OperationModify,
				Target: Target{Type: TargetField, Name: FieldStatus},
				Before: "ok",
				After:  "error",
			}},
		})
	}
	baseDurationSums["regressor"] = 10_000_000
	compareDurationSums["regressor"] = 1_010_000_000

	_, services := buildServiceSections(patch, baseDurationSums, compareDurationSums, nil)
	require.Len(t, services, serviceCount+1)
	assert.Equal(t, "regressor", services[0].Name)
}

func TestBuildServiceSectionsDoesNotLetJitterDisplaceErrors(t *testing.T) {
	const serviceCount = 50
	baseDurationSums := make(map[string]int64, serviceCount*2)
	compareDurationSums := make(map[string]int64, serviceCount*2)
	patch := &Result{}
	for i := 0; i < serviceCount; i++ {
		service := fmt.Sprintf("error-%02d", i)
		baseDurationSums[service] = 10_000_000
		compareDurationSums[service] = 10_000_000
		patch.Modified = append(patch.Modified, ModifiedSpan{
			Span: SpanRef{Service: service},
			Changes: []Change{{
				Op:     OperationModify,
				Target: Target{Type: TargetField, Name: FieldStatus},
				Before: "ok",
				After:  "error",
			}},
		})
	}
	for i := 0; i < serviceCount; i++ {
		service := fmt.Sprintf("jitter-%02d", i)
		baseDurationSums[service] = 10_000_000
		compareDurationSums[service] = 10_000_001
	}

	_, services := buildServiceSections(patch, baseDurationSums, compareDurationSums, nil)
	require.Len(t, services, serviceCount)
	for _, service := range services {
		assert.Equal(t, 1, service.NewErrors)
	}
}

func doubleAttribute(key string, value float64) *commonv1.KeyValue {
	return &commonv1.KeyValue{
		Key:   key,
		Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_DoubleValue{DoubleValue: value}},
	}
}

func summaryFixtureTraces() (*tempopb.Trace, *tempopb.Trace) {
	traceID := []byte("trace-id-0000001")
	baseRoot := spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK)
	baseRoot.Attributes = append(baseRoot.Attributes, stringAttribute("service.version", "v1"))
	compareRoot := spanForNormalizeTest(traceID, "root2", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 110, tracev1.Status_STATUS_CODE_OK)
	compareRoot.Attributes = append(compareRoot.Attributes, stringAttribute("service.version", "v2"))

	return traceWithNamedSpans(
			baseRoot,
			spanForNormalizeTest(traceID, "cache", "root", "checkout", "cache lookup", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "db", "root", "checkout", "db query", tracev1.Span_SPAN_KIND_CLIENT, 20, 50, tracev1.Status_STATUS_CODE_OK),
		), traceWithNamedSpans(
			compareRoot,
			spanForNormalizeTest(traceID, "cache2", "root2", "checkout", "cache lookup", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, tracev1.Status_STATUS_CODE_ERROR),
			spanForNormalizeTest(traceID, "db2", "root2", "checkout", "db query", tracev1.Span_SPAN_KIND_CLIENT, 20, 52, tracev1.Status_STATUS_CODE_OK),
		)
}
