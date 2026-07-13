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

	assert.Equal(t, []string{"checkout"}, got.ChangedServices)
	assert.Equal(t, []ServiceRollup{
		{Name: "checkout", SpanWorkDeltaMs: 12, Modified: 2, NewErrors: 1},
	}, got.Services)
	assert.Len(t, got.Patterns, 2)
	assert.Nil(t, got.PatternsTruncated)
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
  "changedServices": [
    "checkout"
  ],
  "services": [
    {
      "name": "checkout",
      "spanWorkDeltaMs": 12,
      "modified": 2,
      "added": 0,
      "removed": 0,
      "newErrors": 1,
      "resolvedErrors": 0
    }
  ],
  "patterns": [
    {
      "state": "modified",
      "span": {
        "service": "checkout",
        "name": "GET /checkout",
        "kind": "server"
      },
      "count": 1,
      "sampleSpans": [
        {
          "path": [
            0
          ],
          "service": "checkout",
          "name": "GET /checkout",
          "kind": "server"
        }
      ],
      "changes": [
        {
          "op": "modify",
          "target": {
            "type": "attribute",
            "scope": "span",
            "key": "service.version"
          },
          "before": "v1",
          "after": "v2"
        }
      ]
    },
    {
      "state": "modified",
      "span": {
        "service": "checkout",
        "name": "cache lookup",
        "kind": "client"
      },
      "count": 1,
      "sampleSpans": [
        {
          "path": [
            0,
            0
          ],
          "service": "checkout",
          "name": "cache lookup",
          "kind": "client"
        }
      ],
      "changes": [
        {
          "op": "modify",
          "target": {
            "type": "field",
            "name": "status"
          },
          "before": "ok",
          "after": "error"
        }
      ]
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
	assert.Empty(t, got.Patterns)
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

func TestSummarizeNativeEncodesNonFiniteAttributes(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	baseRoot := spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK)
	baseRoot.Attributes = append(baseRoot.Attributes, doubleAttribute("score", 1.5))
	compareRoot := spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK)
	compareRoot.Attributes = append(compareRoot.Attributes, doubleAttribute("score", math.NaN()))

	got, err := Summarize(traceWithNamedSpans(baseRoot), traceWithNamedSpans(compareRoot))
	require.NoError(t, err)
	require.Len(t, got.Patterns, 1)
	require.Len(t, got.Patterns[0].Changes, 1)
	change := got.Patterns[0].Changes[0]
	assert.Equal(t, "score", change.Target.Key)
	assert.Equal(t, 1.5, change.Before)
	assert.Equal(t, "NaN", change.After)

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

	got, err := Summarize(makeTrace(), makeTrace())
	require.NoError(t, err)
	assert.Equal(t, "unchanged", got.Signals.TraceLatency)
	assert.Equal(t, "unchanged", got.Signals.SpanWork)
	assert.Equal(t, "unchanged", got.Signals.Errors)
	assert.Equal(t, 0, got.Stats.AttributeChanges)
	assert.Empty(t, got.Services)
	assert.Empty(t, got.Patterns)
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
	assert.Equal(t, int64(100), got.Base.SpanWorkMs)
	assert.Equal(t, int64(100), got.Compare.SpanWorkMs)
	assert.Zero(t, got.Stats.SpanWorkDeltaMs)
	assert.Equal(t, "unchanged", got.Signals.TraceLatency)
	assert.Equal(t, "unchanged", got.Signals.SpanWork)
}

func TestSummarizeUnsetSpanStartDoesNotInflateSpanWork(t *testing.T) {
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
	assert.Equal(t, int64(150), got.Base.SpanWorkMs)
	assert.Equal(t, int64(100), got.Compare.SpanWorkMs)
	assert.Equal(t, int64(-50), got.Stats.SpanWorkDeltaMs)
	assert.Equal(t, "decreased", got.Signals.SpanWork)
	// The duration change with an invalid side renders null and must not
	// fabricate delta stats.
	require.Len(t, got.Patterns, 1)
	require.Len(t, got.Patterns[0].Changes, 1)
	assert.Equal(t, "duration_ms", got.Patterns[0].Changes[0].Target.Name)
	assert.Equal(t, int64(50), got.Patterns[0].Changes[0].Before)
	assert.Nil(t, got.Patterns[0].Changes[0].After)
	assert.Nil(t, got.Patterns[0].DurationDeltaMs)
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
		Name:            "checkout",
		SpanWorkDeltaMs: 0,
		Added:           1,
		Removed:         1,
		NewErrors:       1,
		ResolvedErrors:  1,
	}, got.Services[0])

	require.Len(t, got.Patterns, 2)
	added := got.Patterns[0]
	assert.Equal(t, "added", added.State)
	assert.Equal(t, "new call", added.Span.Name)
	assert.Empty(t, added.Changes)
	removed := got.Patterns[1]
	assert.Equal(t, "removed", removed.State)
	assert.Equal(t, "dropped call", removed.Span.Name)
	assert.Empty(t, removed.Changes)
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
		TraceLatency: "unchanged",
		SpanWork:     "unchanged",
		Errors:       "unchanged",
		SpanCount:    "unchanged",
		Structure:    "unchanged",
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

func TestSummarizeEmptySectionsEncodeAsEmptyArrays(t *testing.T) {
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
	assert.Contains(t, string(data), `"patterns":[]`)
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
	assert.Empty(t, got.Patterns)
	assert.Equal(t, []ServiceRollup{
		{Name: "backend", SpanWorkDeltaMs: 13},
		{Name: "database", SpanWorkDeltaMs: 13},
		{Name: "frontend", SpanWorkDeltaMs: 13},
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
			assert.Empty(t, got.Patterns)
			assert.Zero(t, got.Stats.SpanWorkDeltaMs)
			assert.Equal(t, "unchanged", got.Signals.SpanWork)
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
	assert.Equal(t, int64(15), got.Base.SpanWorkMs)
	assert.Equal(t, int64(16), got.Compare.SpanWorkMs)
	assert.Equal(t, int64(1), got.Stats.SpanWorkDeltaMs)
	assert.Equal(t, "increased", got.Signals.SpanWork)
	require.Len(t, got.Services, 1)
	assert.Equal(t, int64(1), got.Services[0].SpanWorkDeltaMs)
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
	assert.Empty(t, got.Patterns)
	assert.Contains(t, got.Warnings, Warning{
		Code:    WarningAmbiguousSpanMatch,
		Message: "1 duplicate logical span group(s) may match ambiguously; matching minimizes changes, but instance-level transitions may not identify the same physical operation",
	})
}

func TestSummarizeUsesCanonicalAmbiguousPatterns(t *testing.T) {
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
	require.Len(t, got.Patterns, 1)
	require.Len(t, got.Patterns[0].Changes, 1)
	assert.Equal(t, "b", got.Patterns[0].Changes[0].Before)
	assert.Equal(t, "c", got.Patterns[0].Changes[0].After)
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

func TestSummarizePatternsCompressRepeatedChanges(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func(childDurationMs func(i uint64) uint64) *tempopb.Trace {
		spans := []*tracev1.Span{
			spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 300, tracev1.Status_STATUS_CODE_OK),
		}
		for i := uint64(0); i < 30; i++ {
			start := 10 + i
			spans = append(spans, spanForNormalizeTest(traceID, fmt.Sprintf("child-%02d", i), "root", "backend", "repeated op", tracev1.Span_SPAN_KIND_CLIENT, start, start+childDurationMs(i), tracev1.Status_STATUS_CODE_OK))
		}
		return traceWithNamedSpans(spans...)
	}

	base := makeTrace(func(uint64) uint64 { return 100 })
	// Every span slows in the same direction but by different amounts (+50..+79ms,
	// all above tolerance), so direction-based grouping must still compress them.
	compare := makeTrace(func(i uint64) uint64 { return 150 + i })

	got, err := Summarize(base, compare)
	require.NoError(t, err)
	require.Len(t, got.Patterns, 1)
	pattern := got.Patterns[0]
	assert.Equal(t, "modified", pattern.State)
	assert.Equal(t, PatternSpan{Service: "backend", Name: "repeated op", Kind: "client"}, pattern.Span)
	assert.Equal(t, 30, pattern.Count)
	assert.Len(t, pattern.SampleSpans, patternSampleSpans)
	require.NotNil(t, pattern.DurationDeltaMs)
	assert.Equal(t, DurationDeltaStats{Min: 50, Max: 79, Total: 1935}, *pattern.DurationDeltaMs)
	require.Len(t, pattern.Changes, 1)
	assert.Equal(t, "duration_ms", pattern.Changes[0].Target.Name)
	assert.Equal(t, int64(100), pattern.Changes[0].Before)
	assert.Equal(t, int64(150), pattern.Changes[0].After)
	assert.Nil(t, got.PatternsTruncated)
}

func TestSummarizePatternsCapWithDisclosure(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func(childDurationMs uint64) *tempopb.Trace {
		spans := []*tracev1.Span{
			spanForNormalizeTest(traceID, "root", "", "checkout", "GET /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, 300, tracev1.Status_STATUS_CODE_OK),
		}
		for i := uint64(0); i < 25; i++ {
			start := 10 + i
			spans = append(spans, spanForNormalizeTest(traceID, fmt.Sprintf("span-%02d", i), "root", "backend", fmt.Sprintf("op-%02d", i), tracev1.Span_SPAN_KIND_CLIENT, start, start+childDurationMs, tracev1.Status_STATUS_CODE_OK))
		}
		return traceWithNamedSpans(spans...)
	}

	// 25 distinct span names, each slowed above tolerance: 25 distinct patterns.
	got, err := Summarize(makeTrace(100), makeTrace(150))
	require.NoError(t, err)
	require.Len(t, got.Patterns, patternCap)
	assert.Equal(t, "op-00", got.Patterns[0].Span.Name)
	assert.Equal(t, "op-19", got.Patterns[patternCap-1].Span.Name)
	require.NotNil(t, got.PatternsTruncated)
	assert.Equal(t, PatternsTruncated{Patterns: 5, Spans: 5}, *got.PatternsTruncated)
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

func TestSummarizePatternDurationTotalSumsBeforeTruncating(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	makeTrace := func(childDurationNanos uint64) *tempopb.Trace {
		children := []*tracev1.Span{
			spanForNormalizeTest(traceID, "a", "root", "backend", "op", tracev1.Span_SPAN_KIND_CLIENT, 10, 10, tracev1.Status_STATUS_CODE_OK),
			spanForNormalizeTest(traceID, "b", "root", "backend", "op", tracev1.Span_SPAN_KIND_CLIENT, 20, 20, tracev1.Status_STATUS_CODE_OK),
		}
		for _, child := range children {
			child.EndTimeUnixNano = child.StartTimeUnixNano + childDurationNanos
		}
		return traceWithNamedSpans(
			spanForNormalizeTest(traceID, "root", "", "frontend", "GET /", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
			children[0],
			children[1],
		)
	}

	got, err := Summarize(makeTrace(0), makeTrace(1_500_000))
	require.NoError(t, err)
	require.Len(t, got.Patterns, 1)
	require.NotNil(t, got.Patterns[0].DurationDeltaMs)
	assert.Equal(t, DurationDeltaStats{Min: 1, Max: 1, Total: 3}, *got.Patterns[0].DurationDeltaMs)
}

func TestSummarizeGroupsInvalidDurationTransitions(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	invalid := func(span *tracev1.Span) *tracev1.Span {
		span.StartTimeUnixNano = 0
		return span
	}
	base := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root", "", "frontend", "GET /", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "parent-a", "root", "a", "parent a", tracev1.Span_SPAN_KIND_INTERNAL, 10, 40, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "child-a", "parent-a", "db", "query", tracev1.Span_SPAN_KIND_CLIENT, 15, 25, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "parent-b", "root", "b", "parent b", tracev1.Span_SPAN_KIND_INTERNAL, 50, 80, tracev1.Status_STATUS_CODE_OK),
		invalid(spanForNormalizeTest(traceID, "child-b", "parent-b", "db", "query", tracev1.Span_SPAN_KIND_CLIENT, 55, 65, tracev1.Status_STATUS_CODE_OK)),
	)
	compare := traceWithNamedSpans(
		spanForNormalizeTest(traceID, "root-2", "", "frontend", "GET /", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "parent-a-2", "root-2", "a", "parent a", tracev1.Span_SPAN_KIND_INTERNAL, 10, 40, tracev1.Status_STATUS_CODE_OK),
		invalid(spanForNormalizeTest(traceID, "child-a-2", "parent-a-2", "db", "query", tracev1.Span_SPAN_KIND_CLIENT, 15, 25, tracev1.Status_STATUS_CODE_OK)),
		spanForNormalizeTest(traceID, "parent-b-2", "root-2", "b", "parent b", tracev1.Span_SPAN_KIND_INTERNAL, 50, 80, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "child-b-2", "parent-b-2", "db", "query", tracev1.Span_SPAN_KIND_CLIENT, 55, 65, tracev1.Status_STATUS_CODE_OK),
	)

	got, err := Summarize(base, compare)
	require.NoError(t, err)
	require.Len(t, got.Patterns, 1)
	assert.Equal(t, 2, got.Patterns[0].Count)
	require.Len(t, got.Patterns[0].Changes, 1)
	assert.Nil(t, got.Patterns[0].Changes[0].After)
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
	assert.Contains(t, string(data), `"service":""`)
	assert.Contains(t, string(data), `"service":"\u003cunknown\u003e"`)
}

func TestSummarizeAddedPatternsFollowReferenceCompression(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	root := func() *tracev1.Span {
		return spanForNormalizeTest(traceID, "root", "", "frontend", "GET /", tracev1.Span_SPAN_KIND_SERVER, 0, 100, tracev1.Status_STATUS_CODE_OK)
	}
	base := traceWithNamedSpans(root())
	compare := traceWithNamedSpans(
		root(),
		spanForNormalizeTest(traceID, "ok", "root", "backend", "call", tracev1.Span_SPAN_KIND_CLIENT, 10, 20, tracev1.Status_STATUS_CODE_OK),
		spanForNormalizeTest(traceID, "error", "root", "backend", "call", tracev1.Span_SPAN_KIND_CLIENT, 30, 40, tracev1.Status_STATUS_CODE_ERROR),
	)

	got, err := Summarize(base, compare)
	require.NoError(t, err)
	require.Len(t, got.Patterns, 1)
	assert.Equal(t, 2, got.Patterns[0].Count)
	assert.Empty(t, got.Patterns[0].Changes)
	assert.Nil(t, got.Patterns[0].DurationDeltaMs)
}

func TestPatternKeySeparatesNestedCollectionShapes(t *testing.T) {
	target := Target{Type: TargetAttribute, Scope: ScopeSpan, Key: "value"}
	a := patternInput{state: patternStateModified, ref: SpanRef{Name: "op"}, changes: []Change{{
		Op: OperationModify, Target: target, Before: []any{[]any{}, "x"}, After: nil,
	}}}
	b := patternInput{state: patternStateModified, ref: SpanRef{Name: "op"}, changes: []Change{{
		Op: OperationModify, Target: target, Before: []any{[]any{"x"}}, After: nil,
	}}}

	assert.NotEqual(t, patternKey(a), patternKey(b))
}

func TestBuildPatternsTieOrderingIsDeterministic(t *testing.T) {
	patch := &Result{}
	for i := 0; i < patternCap+5; i++ {
		patch.Modified = append(patch.Modified, ModifiedSpan{
			Span: SpanRef{Service: "backend", Name: "op", Kind: fmt.Sprintf("kind-%02d", i)},
			Changes: []Change{{
				Op:     OperationModify,
				Target: Target{Type: TargetAttribute, Scope: ScopeSpan, Key: "value"},
				Before: int64(i),
				After:  int64(i + 1),
			}},
		})
	}

	var expected []byte
	for i := 0; i < 20; i++ {
		patterns, _ := buildPatterns(patch)
		data, err := json.Marshal(patterns)
		require.NoError(t, err)
		if i == 0 {
			expected = data
			continue
		}
		assert.Equal(t, string(expected), string(data))
	}
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

func TestBuildServiceSectionsRetainsLargestWorkChange(t *testing.T) {
	const serviceCount = 50
	baseWork := make(map[string]int64, serviceCount+1)
	compareWork := make(map[string]int64, serviceCount+1)
	patch := &Result{}
	for i := 0; i < serviceCount; i++ {
		service := fmt.Sprintf("error-%02d", i)
		baseWork[service] = 10_000_000
		compareWork[service] = 10_000_000
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
	baseWork["regressor"] = 10_000_000
	compareWork["regressor"] = 1_010_000_000

	_, services := buildServiceSections(patch, baseWork, compareWork, nil)
	require.Len(t, services, serviceCount+1)
	assert.Equal(t, "regressor", services[0].Name)
}

func TestBuildServiceSectionsDoesNotLetJitterDisplaceErrors(t *testing.T) {
	const serviceCount = 50
	baseWork := make(map[string]int64, serviceCount*2)
	compareWork := make(map[string]int64, serviceCount*2)
	patch := &Result{}
	for i := 0; i < serviceCount; i++ {
		service := fmt.Sprintf("error-%02d", i)
		baseWork[service] = 10_000_000
		compareWork[service] = 10_000_000
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
		baseWork[service] = 10_000_000
		compareWork[service] = 10_000_001
	}

	_, services := buildServiceSections(patch, baseWork, compareWork, nil)
	require.Len(t, services, serviceCount)
	for _, service := range services {
		assert.Equal(t, 1, service.NewErrors)
	}
}

func TestBuildPatternsRetainsFullExemplarData(t *testing.T) {
	changes := make([]Change, 25)
	for i := range changes {
		changes[i] = Change{
			Op:     OperationModify,
			Target: Target{Type: TargetAttribute, Scope: ScopeSpan, Key: fmt.Sprintf("attribute-%02d", i)},
			Before: []any{"before", int64(i)},
			After:  []any{"after", int64(i)},
		}
	}
	path := make([]int, 70)
	patch := &Result{Modified: []ModifiedSpan{{
		Span:    SpanRef{Path: path, Service: "service", Name: "operation", Kind: "client"},
		Changes: changes,
	}}}

	patterns, _ := buildPatterns(patch)
	require.Len(t, patterns, 1)
	assert.Equal(t, changes, patterns[0].Changes)
	require.Len(t, patterns[0].SampleSpans, 1)
	assert.Equal(t, path, patterns[0].SampleSpans[0].Path)
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
