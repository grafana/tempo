package tracediff

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// summaryTestStartMs is a realistic wall-clock start in milliseconds. The shared
// span helper adds its own non-zero offset, so passing this value keeps fixtures
// safely away from the unset 0 epoch while preserving requested durations.
const summaryTestStartMs = 1_700_000_000_000

func TestSummarize(t *testing.T) {
	base, compare := summaryFixtureTraces()

	got, err := Summarize(base, compare, SummaryOptions{TopN: 1})
	require.NoError(t, err)

	assert.Equal(t, VersionTraceSummaryV0Ranked, got.Version)
	assert.Equal(t, Signals{
		TraceLatency: "increased",
		SpanWork:     "increased",
		Errors:       "increased",
		SpanCount:    "unchanged",
		Structure:    "unchanged",
	}, got.Signals)
	assert.Equal(t, 1.0, got.Trust.MatchedSpanRatio)
	assert.Equal(t, 1.0, got.Trust.StructureOverlap)
	assert.Equal(t, SummaryStats{
		TraceLatencyDeltaMs: 10,
		SpanWorkDeltaMs:     12,
		SpanCountDelta:      0,
		ErrorSpanDelta:      1,
		MatchedSpans:        3,
		ModifiedSpans:       2,
		FieldChanges:        1,
		AttributeChanges:    1,
	}, got.Stats)
	require.NotNil(t, got.TopChanges)

	assert.Empty(t, got.TopChanges.Regressions)

	require.Len(t, got.TopChanges.Status, 1)
	assert.Equal(t, "cache lookup", got.TopChanges.Status[0].Span.Name)
	assert.Equal(t, "ok", got.TopChanges.Status[0].StatusBefore)
	assert.Equal(t, "error", got.TopChanges.Status[0].StatusAfter)

	require.Len(t, got.TopChanges.Attributes, 1)
	assert.Equal(t, "service.version", got.TopChanges.Attributes[0].Key)
	assert.Equal(t, OperationModify, got.TopChanges.Attributes[0].Op)
	assert.Empty(t, got.Groups)
}

func TestSummarizeStructureSignalDetectsTopologyChange(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	base := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, summaryTestStartMs, summaryTestStartMs+100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "auth", "root", "auth", "authorize", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+10, summaryTestStartMs+20, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "db", "root", "db", "SELECT user", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+20, summaryTestStartMs+30, tracev1.Status_STATUS_CODE_OK),
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, summaryTestStartMs, summaryTestStartMs+100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "auth", "root", "auth", "authorize", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+10, summaryTestStartMs+20, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "db", "auth", "db", "SELECT user", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+20, summaryTestStartMs+30, tracev1.Status_STATUS_CODE_OK),
	)

	got, err := Summarize(base, compare, SummaryOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, got.Stats.AddedSpans)
	assert.Equal(t, 0, got.Stats.RemovedSpans)
	assert.Equal(t, "changed", got.Signals.Structure)
}

func TestSummarizeTrustRatiosDoNotRoundNearPerfectToOne(t *testing.T) {
	assert.NotEqual(t, 1.0, matchedSpanRatio(10_000, 10_000, 9_999))
	assert.Equal(t, float64(9_999)/float64(10_000), matchedSpanRatio(10_000, 10_000, 9_999))
	assert.Equal(t, 1.0, matchedSpanRatio(10_000, 10_000, 10_000))
}

func TestSummarizeRanksLargeStructuralChangesByAbsoluteDuration(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	base := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, summaryTestStartMs, summaryTestStartMs+100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "removed", "root", "cache", "large removed", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs, summaryTestStartMs+90, tracev1.Status_STATUS_CODE_OK),
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root2", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, summaryTestStartMs, summaryTestStartMs+100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "added", "root2", "cache", "small added", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs, summaryTestStartMs+1, tracev1.Status_STATUS_CODE_OK),
	)

	got, err := Summarize(base, compare, SummaryOptions{TopN: 1})
	require.NoError(t, err)
	require.Len(t, got.TopChanges.Structural, 1)
	assert.Equal(t, "large removed", got.TopChanges.Structural[0].Span.Name)
}

func TestSummarizeRankedEncodesNonFiniteAttributes(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	baseRoot := spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK)
	baseRoot.Attributes = append(baseRoot.Attributes, doubleAttribute("score", 1.5))
	compareRoot := spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK)
	compareRoot.Attributes = append(compareRoot.Attributes, doubleAttribute("score", math.NaN()))

	got, err := Summarize(traceWithNamedSpans(baseRoot), traceWithNamedSpans(compareRoot), SummaryOptions{})
	require.NoError(t, err)
	require.NotNil(t, got.TopChanges)
	require.Len(t, got.TopChanges.Attributes, 1)
	assert.Equal(t, "score", got.TopChanges.Attributes[0].Key)
	assert.Equal(t, 1.5, got.TopChanges.Attributes[0].Before)
	assert.Equal(t, "NaN", got.TopChanges.Attributes[0].After)

	// The default encoder rejects non-finite floats; the sanitized result must
	// still encode successfully.
	data, err := json.Marshal(got)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"after":"NaN"`)
}

func TestSummarizeTreatsIdenticalNonFiniteAttributesAsUnchanged(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func() *tempopb.Trace {
		root := spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK)
		root.Attributes = append(root.Attributes, doubleAttribute("score", math.NaN()))
		return traceWithNamedSpans(root)
	}

	got, err := Summarize(makeTrace(), makeTrace(), SummaryOptions{})
	require.NoError(t, err)
	assert.Equal(t, "unchanged", got.Signals.TraceLatency)
	assert.Equal(t, "unchanged", got.Signals.SpanWork)
	assert.Equal(t, "unchanged", got.Signals.Errors)
	assert.Equal(t, 0, got.Stats.AttributeChanges)
	if got.TopChanges != nil {
		assert.Empty(t, got.TopChanges.Attributes)
	}
}

func TestSummarizeSignalsReportIndependentDirections(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	// Fewer errors but higher latency: report both facts without declaring whether
	// the overall change is good or bad.
	base := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, summaryTestStartMs, summaryTestStartMs+100, tracev1.Status_STATUS_CODE_ERROR),
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, summaryTestStartMs, summaryTestStartMs+150, tracev1.Status_STATUS_CODE_OK),
	)

	got, err := Summarize(base, compare, SummaryOptions{})
	require.NoError(t, err)
	assert.Equal(t, "increased", got.Signals.TraceLatency)
	assert.Equal(t, "decreased", got.Signals.Errors)
}

func TestSummarizeIgnoresUnsetSpanStartInDuration(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	// A child span has an unset (zero) start while the root has a real start, as
	// can happen with buggy instrumentation. The trace envelope must use the
	// root's start, not the epoch, so the duration is the real window and not a
	// nonsensical span of tens of thousands of years.
	makeTrace := func() *tempopb.Trace {
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, summaryTestStartMs, summaryTestStartMs+100, tracev1.Status_STATUS_CODE_OK),
			&tracev1.Span{
				TraceId:           traceID,
				SpanId:            []byte("child"),
				ParentSpanId:      []byte("root"),
				Name:              "cache lookup",
				Kind:              tracev1.Span_SPAN_KIND_CLIENT,
				StartTimeUnixNano: 0, // unset
				EndTimeUnixNano:   uint64(summaryTestStartMs+50) * 1_000_000,
				Attributes:        []*commonv1.KeyValue{stringAttribute("service.name", "checkout")},
				Status:            &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK},
			},
		)
	}

	got, err := Summarize(makeTrace(), makeTrace(), SummaryOptions{})
	require.NoError(t, err)
	assert.Equal(t, int64(100), got.Base.DurationMs)
	assert.Equal(t, int64(100), got.Compare.DurationMs)
	assert.Equal(t, int64(0), got.Stats.TraceLatencyDeltaMs)
	// The unset-start child adds no work: only the root's 100ms counts.
	assert.Equal(t, int64(100), got.Base.SpanWorkMs)
	assert.Equal(t, int64(100), got.Compare.SpanWorkMs)
	assert.Equal(t, int64(0), got.Stats.SpanWorkDeltaMs)
}

func TestSummarizeUnsetSpanStartDoesNotInflateSpanWork(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	// Baseline: a root and a child, both with usable start times.
	base := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, summaryTestStartMs, summaryTestStartMs+100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "child", "root", "checkout", "cache lookup", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+10, summaryTestStartMs+60, tracev1.Status_STATUS_CODE_OK),
	)
	// Comparison: the child keeps a realistic end but loses its start (buggy
	// instrumentation). Its duration must count as 0, not ~decades.
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, summaryTestStartMs, summaryTestStartMs+100, tracev1.Status_STATUS_CODE_OK),
		&tracev1.Span{
			TraceId:           traceID,
			SpanId:            []byte("child"),
			ParentSpanId:      []byte("root"),
			Name:              "cache lookup",
			Kind:              tracev1.Span_SPAN_KIND_CLIENT,
			StartTimeUnixNano: 0, // unset
			EndTimeUnixNano:   uint64(summaryTestStartMs+60) * 1_000_000,
			Attributes:        []*commonv1.KeyValue{stringAttribute("service.name", "checkout")},
			Status:            &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK},
		},
	)

	got, err := Summarize(base, compare, SummaryOptions{})
	require.NoError(t, err)
	// base = root(100) + child(50); compare = root(100) + unset-start child(0).
	assert.Equal(t, int64(150), got.Base.SpanWorkMs)
	assert.Equal(t, int64(100), got.Compare.SpanWorkMs)
	assert.Equal(t, int64(-50), got.Stats.SpanWorkDeltaMs)
	assert.Equal(t, "decreased", got.Signals.SpanWork)
	assert.Empty(t, got.TopChanges.Improvements)
	assert.Empty(t, got.TopChanges.Regressions)
}

func TestSummarizeStructuralSurfacesAddedAndRemovedErrors(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	base := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, summaryTestStartMs, summaryTestStartMs+100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "gone", "root", "checkout", "dropped call", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+10, summaryTestStartMs+20, tracev1.Status_STATUS_CODE_ERROR),
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, summaryTestStartMs, summaryTestStartMs+100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "new", "root", "checkout", "new call", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+10, summaryTestStartMs+20, tracev1.Status_STATUS_CODE_ERROR),
	)

	got, err := Summarize(base, compare, SummaryOptions{TopN: 10})
	require.NoError(t, err)
	require.NotNil(t, got.TopChanges)

	var added, removed *ChangeSummary
	for i := range got.TopChanges.Structural {
		switch got.TopChanges.Structural[i].State {
		case "added":
			added = &got.TopChanges.Structural[i]
		case "removed":
			removed = &got.TopChanges.Structural[i]
		}
	}
	require.NotNil(t, added, "added span should appear in structural changes")
	assert.Equal(t, "new call", added.Span.Name)
	assert.Equal(t, "error", added.StatusAfter)
	require.NotNil(t, removed, "removed span should appear in structural changes")
	assert.Equal(t, "dropped call", removed.Span.Name)
	assert.Equal(t, "error", removed.StatusBefore)
	require.Len(t, got.TopChanges.Status, 2)
}

func TestSummarizeInvalidDurationWarnsAndDoesNotInflateEnvelope(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func() *tempopb.Trace {
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, summaryTestStartMs, summaryTestStartMs+100, tracev1.Status_STATUS_CODE_OK),
			&tracev1.Span{
				TraceId:           traceID,
				SpanId:            []byte("bad-child"),
				ParentSpanId:      []byte("root"),
				Name:              "bad duration",
				Kind:              tracev1.Span_SPAN_KIND_CLIENT,
				StartTimeUnixNano: 0,
				EndTimeUnixNano:   uint64(summaryTestStartMs+1000) * 1_000_000,
				Attributes:        []*commonv1.KeyValue{stringAttribute("service.name", "checkout")},
				Status:            &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK},
			},
		)
	}

	got, err := Summarize(makeTrace(), makeTrace(), SummaryOptions{})
	require.NoError(t, err)
	assert.Equal(t, int64(100), got.Base.DurationMs)
	assert.Equal(t, int64(100), got.Compare.DurationMs)
	require.Len(t, got.Warnings, 2)
	assert.Equal(t, WarningInvalidDuration, got.Warnings[0].Code)
	assert.Contains(t, got.Warnings[0].Message, "base trace")
	assert.Equal(t, WarningInvalidDuration, got.Warnings[1].Code)
	assert.Contains(t, got.Warnings[1].Message, "compare trace")
}

func TestTraceSummaryV0RankedJSONShape(t *testing.T) {
	base, compare := summaryFixtureTraces()

	got, err := Summarize(base, compare, SummaryOptions{TopN: 1})
	require.NoError(t, err)
	data, err := json.Marshal(got)
	require.NoError(t, err)

	assert.JSONEq(t, `{
  "version": "trace-summary-v0-ranked",
  "base": {
    "traceId": "74726163652d69642d30303030303031",
    "rootService": "checkout",
    "rootName": "GET /checkout",
    "spanCount": 3,
    "errorSpanCount": 0,
    "durationMs": 100,
    "spanWorkMs": 140
  },
  "compare": {
    "traceId": "74726163652d69642d30303030303031",
    "rootService": "checkout",
    "rootName": "GET /checkout",
    "spanCount": 3,
    "errorSpanCount": 1,
    "durationMs": 110,
    "spanWorkMs": 152
  },
  "signals": {
    "traceLatency": "increased",
    "spanWork": "increased",
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
    "spanWorkDeltaMs": 12,
    "spanCountDelta": 0,
    "errorSpanDelta": 1,
    "matchedSpans": 3,
    "modifiedSpans": 2,
    "addedSpans": 0,
    "removedSpans": 0,
    "fieldChanges": 1,
    "attributeChanges": 1
  },
  "topChanges": {
    "status": [
      {"span": {"path": [0, 0], "service": "checkout", "name": "cache lookup", "kind": "client"}, "state": "modified", "statusBefore": "ok", "statusAfter": "error"}
    ],
    "attributes": [
      {"span": {"path": [0], "service": "checkout", "name": "GET /checkout", "kind": "server"}, "key": "service.version", "op": "modify", "before": "v1", "after": "v2"}
    ]
  }
}`, string(data))
}

func TestTraceSummaryV0AggregateJSONShape(t *testing.T) {
	base, compare := summaryFixtureTraces()

	got, err := Summarize(base, compare, SummaryOptions{Format: SummaryFormatAggregate, TopN: 1})
	require.NoError(t, err)
	data, err := json.Marshal(got)
	require.NoError(t, err)

	assert.JSONEq(t, `{
  "version": "trace-summary-v0-aggregate",
  "base": {"traceId": "74726163652d69642d30303030303031", "rootService": "checkout", "rootName": "GET /checkout", "spanCount": 3, "errorSpanCount": 0, "durationMs": 100, "spanWorkMs": 140},
  "compare": {"traceId": "74726163652d69642d30303030303031", "rootService": "checkout", "rootName": "GET /checkout", "spanCount": 3, "errorSpanCount": 1, "durationMs": 110, "spanWorkMs": 152},
  "signals": {"traceLatency": "increased", "spanWork": "increased", "errors": "increased", "spanCount": "unchanged", "structure": "unchanged"},
  "trust": {"matchedSpanRatio": 1, "structureOverlap": 1},
  "stats": {"traceLatencyDeltaMs": 10, "spanWorkDeltaMs": 12, "spanCountDelta": 0, "errorSpanDelta": 1, "matchedSpans": 3, "modifiedSpans": 2, "addedSpans": 0, "removedSpans": 0, "fieldChanges": 1, "attributeChanges": 1}
}`, string(data))
}

func TestTraceSummaryV0GroupedJSONShape(t *testing.T) {
	base, compare := summaryFixtureTraces()

	got, err := Summarize(base, compare, SummaryOptions{Format: SummaryFormatGrouped, TopN: 1})
	require.NoError(t, err)
	data, err := json.Marshal(got)
	require.NoError(t, err)

	assert.JSONEq(t, `{
  "version": "trace-summary-v0-grouped",
  "base": {"traceId": "74726163652d69642d30303030303031", "rootService": "checkout", "rootName": "GET /checkout", "spanCount": 3, "errorSpanCount": 0, "durationMs": 100, "spanWorkMs": 140},
  "compare": {"traceId": "74726163652d69642d30303030303031", "rootService": "checkout", "rootName": "GET /checkout", "spanCount": 3, "errorSpanCount": 1, "durationMs": 110, "spanWorkMs": 152},
  "signals": {"traceLatency": "increased", "spanWork": "increased", "errors": "increased", "spanCount": "unchanged", "structure": "unchanged"},
  "trust": {"matchedSpanRatio": 1, "structureOverlap": 1},
  "stats": {"traceLatencyDeltaMs": 10, "spanWorkDeltaMs": 12, "spanCountDelta": 0, "errorSpanDelta": 1, "matchedSpans": 3, "modifiedSpans": 2, "addedSpans": 0, "removedSpans": 0, "fieldChanges": 1, "attributeChanges": 1},
  "groups": [
    {"serviceName": "checkout", "spanWorkDeltaMs": 0, "addedSpans": 0, "removedSpans": 0, "modifiedSpans": 2, "statusChanges": 1, "newErrorSpans": 1, "attributeChangedSpans": 1}
  ]
}`, string(data))
}

func TestSummarizeRootFallsBackForCycleOnlyTrace(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	// Every span's parent is present, forming a cycle with no structural root.
	// The summary must still report a deterministic root (earliest by start time,
	// then span ID) rather than leaving the root fields empty.
	makeTrace := func() *tempopb.Trace {
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "a", "b", "checkout", "span a", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+10, summaryTestStartMs+20, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "b", "a", "checkout", "span b", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs, summaryTestStartMs+30, tracev1.Status_STATUS_CODE_OK),
		)
	}

	got, err := Summarize(makeTrace(), makeTrace(), SummaryOptions{})
	require.NoError(t, err)
	assert.Equal(t, "checkout", got.Base.RootService)
	assert.Equal(t, "span b", got.Base.RootName)
}

func TestSummarizeGroupsPrioritizeStatusRegressionUnderTopN(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	// Two services change with an identical (zero) span-work delta: "aaa" only has
	// an attribute change, "zzz" has a status regression. With TopN=1 the status
	// regression must win the tie rather than being dropped for sorting after the
	// alphabetically earlier "aaa".
	aBase := spanForNormalizeTest(traceID, "a1", "root", "aaa", "a op", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+10, summaryTestStartMs+20, tracev1.Status_STATUS_CODE_OK)
	aBase.Attributes = append(aBase.Attributes, stringAttribute("version", "v1"))
	aCompare := spanForNormalizeTest(traceID, "a1", "root", "aaa", "a op", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+10, summaryTestStartMs+20, tracev1.Status_STATUS_CODE_OK)
	aCompare.Attributes = append(aCompare.Attributes, stringAttribute("version", "v2"))

	base := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, summaryTestStartMs, summaryTestStartMs+100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "z1", "root", "zzz", "z op", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+10, summaryTestStartMs+20, tracev1.Status_STATUS_CODE_OK),
		aBase,
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, summaryTestStartMs, summaryTestStartMs+100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "z1", "root", "zzz", "z op", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+10, summaryTestStartMs+20, tracev1.Status_STATUS_CODE_ERROR),
		aCompare,
	)

	got, err := Summarize(base, compare, SummaryOptions{Format: SummaryFormatGrouped, TopN: 1})
	require.NoError(t, err)
	require.Len(t, got.Groups, 1)
	assert.Equal(t, "zzz", got.Groups[0].ServiceName)
	assert.Equal(t, 1, got.Groups[0].StatusChanges)
}

func TestSummarizeGroupsPrioritizeNewErrorsOverResolvedErrorsUnderTopN(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	base := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, summaryTestStartMs, summaryTestStartMs+100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "a", "root", "aaa", "resolved", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+10, summaryTestStartMs+20, tracev1.Status_STATUS_CODE_ERROR),
		spanForNormalizeTest(traceID, "z", "root", "zzz", "new error", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+10, summaryTestStartMs+20, tracev1.Status_STATUS_CODE_OK),
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, summaryTestStartMs, summaryTestStartMs+100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "a", "root", "aaa", "resolved", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+10, summaryTestStartMs+20, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "z", "root", "zzz", "new error", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+10, summaryTestStartMs+20, tracev1.Status_STATUS_CODE_ERROR),
	)

	got, err := Summarize(base, compare, SummaryOptions{Format: SummaryFormatGrouped, TopN: 1})
	require.NoError(t, err)
	require.Len(t, got.Groups, 1)
	assert.Equal(t, "zzz", got.Groups[0].ServiceName)
	assert.Equal(t, 1, got.Groups[0].NewErrorSpans)
}

func TestSummarizeStatusChangesPrioritizeErrorsUnderTopN(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	base := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "a", "root", "checkout", "aaa recovered", tracev1.Span_SPAN_KIND_CLIENT, 0, 10, tracev1.Status_STATUS_CODE_ERROR),
		spanForNormalizeTest(traceID, "b", "root", "checkout", "bbb recovered", tracev1.Span_SPAN_KIND_CLIENT, 0, 10, tracev1.Status_STATUS_CODE_ERROR),
		spanForNormalizeTest(traceID, "z", "root", "checkout", "zzz broke", tracev1.Span_SPAN_KIND_CLIENT, 0, 10, tracev1.Status_STATUS_CODE_OK),
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "a", "root", "checkout", "aaa recovered", tracev1.Span_SPAN_KIND_CLIENT, 0, 10, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "b", "root", "checkout", "bbb recovered", tracev1.Span_SPAN_KIND_CLIENT, 0, 10, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "z", "root", "checkout", "zzz broke", tracev1.Span_SPAN_KIND_CLIENT, 0, 10, tracev1.Status_STATUS_CODE_ERROR),
	)

	// All status changes have a zero duration delta, so without severity ranking
	// the lexically-first "recovered" spans would win and the new error would be
	// truncated away.
	got, err := Summarize(base, compare, SummaryOptions{TopN: 1})
	require.NoError(t, err)
	require.NotNil(t, got.TopChanges)
	require.Len(t, got.TopChanges.Status, 1)
	assert.Equal(t, "zzz broke", got.TopChanges.Status[0].Span.Name)
	assert.Equal(t, "ok", got.TopChanges.Status[0].StatusBefore)
	assert.Equal(t, "error", got.TopChanges.Status[0].StatusAfter)
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

	got, err := Summarize(makeTrace(), makeTrace(), SummaryOptions{})
	require.NoError(t, err)
	assert.Equal(t, "checkout", got.Base.RootService)
	assert.Equal(t, "GET /checkout", got.Base.RootName)
	assert.Equal(t, "checkout", got.Compare.RootService)
	assert.Equal(t, "GET /checkout", got.Compare.RootName)
}

func doubleAttribute(key string, value float64) *commonv1.KeyValue {
	return &commonv1.KeyValue{
		Key:   key,
		Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_DoubleValue{DoubleValue: value}},
	}
}

func summaryFixtureTraces() (*tempopb.Trace, *tempopb.Trace) {
	traceID := []byte("trace-id-0000001")
	// Realistic (non-zero) starts so every span has a usable OTLP start; durations
	// are unchanged but the root now counts toward span work.
	baseRoot := spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, summaryTestStartMs, summaryTestStartMs+100, tracev1.Status_STATUS_CODE_OK)
	baseRoot.Attributes = append(baseRoot.Attributes, stringAttribute("service.version", "v1"))
	compareRoot := spanForNormalizeTest(traceID, "root2", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, summaryTestStartMs, summaryTestStartMs+110, tracev1.Status_STATUS_CODE_OK)
	compareRoot.Attributes = append(compareRoot.Attributes, stringAttribute("service.version", "v2"))

	return traceWithNamedSpans(
			baseRoot,
			spanForNormalizeTest(traceID, "cache", "root", "checkout", "cache lookup", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+10, summaryTestStartMs+20, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "db", "root", "checkout", "db query", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+20, summaryTestStartMs+50, tracev1.Status_STATUS_CODE_OK),
		), traceWithNamedSpans(
			compareRoot,
			spanForNormalizeTest(traceID, "cache2", "root2", "checkout", "cache lookup", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+10, summaryTestStartMs+20, tracev1.Status_STATUS_CODE_ERROR),
			spanForNormalizeTest(traceID, "db2", "root2", "checkout", "db query", tracev1.Span_SPAN_KIND_CLIENT, summaryTestStartMs+20, summaryTestStartMs+52, tracev1.Status_STATUS_CODE_OK),
		)
}
