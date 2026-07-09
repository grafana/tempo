package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/model/tracediff"
	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExperimentalTraceDiffWritesTracePatch(t *testing.T) {
	dir := t.TempDir()
	traceA := filepath.Join(dir, "trace-a.json")
	traceB := filepath.Join(dir, "trace-b.json")
	out := filepath.Join(dir, "diff.json")
	require.NoError(t, os.WriteFile(traceA, []byte(`{}`), 0o600))
	require.NoError(t, os.WriteFile(traceB, []byte(`{}`), 0o600))

	cmd := experimentalTraceDiffCmd{
		TraceA: traceA,
		TraceB: traceB,
		Format: string(tracediff.FormatTracePatchV0),
		Out:    out,
	}
	require.NoError(t, cmd.Run(nil))

	bytes, err := os.ReadFile(out)
	require.NoError(t, err)

	var result tracediff.Result
	require.NoError(t, json.Unmarshal(bytes, &result))
	require.Equal(t, tracediff.VersionTracePatchV0, result.Version)
	require.Empty(t, result.Modified)
	require.Empty(t, result.Added)
	require.Empty(t, result.Removed)
}

func TestExperimentalTraceDiffPatchEncodesNonFiniteAttributes(t *testing.T) {
	dir := t.TempDir()
	traceA := filepath.Join(dir, "trace-a.json")
	traceB := filepath.Join(dir, "trace-b.json")
	out := filepath.Join(dir, "diff.json")

	base := experimentalTraceDiffTrace(1, 100_000_000)
	base.ResourceSpans[0].ScopeSpans[0].Spans[0].Attributes = append(base.ResourceSpans[0].ScopeSpans[0].Spans[0].Attributes, experimentalTraceDiffDoubleKV("score", 1.5))
	compare := experimentalTraceDiffTrace(1, 100_000_000)
	compare.ResourceSpans[0].ScopeSpans[0].Spans[0].Attributes = append(compare.ResourceSpans[0].ScopeSpans[0].Spans[0].Attributes, experimentalTraceDiffDoubleKV("score", math.NaN()))
	writeExperimentalTraceDiffTraceFile(t, traceA, base)
	writeExperimentalTraceDiffTraceFile(t, traceB, compare)

	cmd := experimentalTraceDiffCmd{
		TraceA: traceA,
		TraceB: traceB,
		Format: string(tracediff.FormatTracePatchV0),
		Out:    out,
	}
	require.NoError(t, cmd.Run(nil))

	bytes, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Contains(t, string(bytes), `"before":1.5`)
	assert.Contains(t, string(bytes), `"after":"NaN"`)
}

func TestExperimentalTraceDiffWritesTraceSummary(t *testing.T) {
	dir := t.TempDir()
	traceAFile := filepath.Join(dir, "trace-a.json")
	traceBFile := filepath.Join(dir, "trace-b.json")
	outFile := filepath.Join(dir, "summary.json")

	writeExperimentalTraceDiffTraceFile(t, traceAFile, experimentalTraceDiffTrace(1, 100_000_000))
	writeExperimentalTraceDiffTraceFile(t, traceBFile, experimentalTraceDiffTrace(2, 150_000_000))

	cmd := experimentalTraceDiffCmd{
		TraceA: traceAFile,
		TraceB: traceBFile,
		Format: tracediff.VersionTraceSummaryV0Native,
		Out:    outFile,
	}
	require.NoError(t, cmd.Run(nil))

	bytes, err := os.ReadFile(outFile)
	require.NoError(t, err)

	var result tracediff.SummaryResult
	require.NoError(t, json.Unmarshal(bytes, &result))
	require.Equal(t, tracediff.VersionTraceSummaryV0Native, result.Version)
	require.Equal(t, "increased", result.Signals.TraceLatency)
	require.Equal(t, int64(50), result.Stats.TraceLatencyDeltaMs)
	require.Equal(t, "increased", result.Signals.SumSpanDuration)
	require.Equal(t, "checkout", result.Base.RootService)
	require.Len(t, result.Services, 1)
}

func TestExperimentalTraceDiffWritesComposed(t *testing.T) {
	dir := t.TempDir()
	traceAFile := filepath.Join(dir, "trace-a.json")
	traceBFile := filepath.Join(dir, "trace-b.json")
	outFile := filepath.Join(dir, "composed.json")

	writeExperimentalTraceDiffTraceFile(t, traceAFile, experimentalTraceDiffTrace(1, 100_000_000))
	writeExperimentalTraceDiffResponseFile(t, traceBFile, &tempopb.TraceByIDResponse{
		Trace:   experimentalTraceDiffTrace(2, 150_000_000),
		Status:  tempopb.PartialStatus_PARTIAL,
		Message: "deadline exceeded",
	})

	cmd := experimentalTraceDiffCmd{
		TraceA: traceAFile,
		TraceB: traceBFile,
		Format: tracediff.VersionTraceSummaryV0Composed,
		Out:    outFile,
	}
	require.NoError(t, cmd.Run(nil))

	bytes, err := os.ReadFile(outFile)
	require.NoError(t, err)

	var result tracediff.ComposedResult
	require.NoError(t, json.Unmarshal(bytes, &result))
	require.Equal(t, tracediff.VersionTraceSummaryV0Composed, result.Version)
	require.NotNil(t, result.Summary)
	require.Equal(t, tracediff.VersionTraceSummaryV0Native, result.Summary.Version)
	require.Nil(t, result.PatchOmitted)

	var patch tracediff.Result
	require.NoError(t, json.Unmarshal(result.Patch, &patch))
	require.Equal(t, tracediff.VersionTracePatchV0, patch.Version)
	require.Len(t, patch.Warnings, 1)
	assert.Equal(t, tracediff.WarningPartialTrace, patch.Warnings[0].Code)

	require.Len(t, result.Summary.Warnings, 1)
	assert.Equal(t, tracediff.WarningPartialTrace, result.Summary.Warnings[0].Code)
	assert.Contains(t, result.Summary.Warnings[0].Message, "trace-b")
}

func TestExperimentalTraceDiffSummaryWarnsOnPartialTraceFile(t *testing.T) {
	dir := t.TempDir()
	traceAFile := filepath.Join(dir, "trace-a.json")
	traceBFile := filepath.Join(dir, "trace-b.json")
	outFile := filepath.Join(dir, "summary.json")

	writeExperimentalTraceDiffTraceFile(t, traceAFile, experimentalTraceDiffTrace(1, 100_000_000))
	writeExperimentalTraceDiffResponseFile(t, traceBFile, &tempopb.TraceByIDResponse{
		Trace:   experimentalTraceDiffTrace(2, 150_000_000),
		Status:  tempopb.PartialStatus_PARTIAL,
		Message: "deadline exceeded",
	})

	cmd := experimentalTraceDiffCmd{
		TraceA: traceAFile,
		TraceB: traceBFile,
		Format: tracediff.VersionTraceSummaryV0Native,
		Out:    outFile,
	}
	require.NoError(t, cmd.Run(nil))

	bytes, err := os.ReadFile(outFile)
	require.NoError(t, err)

	var result tracediff.SummaryResult
	require.NoError(t, json.Unmarshal(bytes, &result))
	require.Equal(t, tracediff.VersionTraceSummaryV0Native, result.Version)
	require.Len(t, result.Warnings, 1)
	assert.Equal(t, tracediff.WarningPartialTrace, result.Warnings[0].Code)
	assert.Contains(t, result.Warnings[0].Message, "trace-b")
	assert.Contains(t, result.Warnings[0].Message, "deadline exceeded")
}

func writeExperimentalTraceDiffTraceFile(t *testing.T, path string, trace *tempopb.Trace) {
	t.Helper()
	data, err := tempopb.MarshalToJSONV1(trace)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))
}

func writeExperimentalTraceDiffResponseFile(t *testing.T, path string, resp *tempopb.TraceByIDResponse) {
	t.Helper()
	data, err := (&jsonpb.Marshaler{}).MarshalToString(resp)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
}

func experimentalTraceDiffTrace(traceID byte, durationNanos uint64) *tempopb.Trace {
	const startNanos = uint64(1_700_000_000_000_000_000)
	return &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			{
				Resource: &resourcev1.Resource{
					Attributes: []*commonv1.KeyValue{experimentalTraceDiffStringKV("service.name", "checkout")},
				},
				ScopeSpans: []*tracev1.ScopeSpans{
					{Spans: []*tracev1.Span{
						{
							TraceId:           experimentalTraceDiffTraceID(traceID),
							SpanId:            []byte{traceID},
							Name:              "GET /checkout",
							Kind:              tracev1.Span_SPAN_KIND_SERVER,
							StartTimeUnixNano: startNanos,
							EndTimeUnixNano:   startNanos + durationNanos,
						},
					}},
				},
			},
		},
	}
}

func experimentalTraceDiffTraceID(id byte) []byte {
	traceID := make([]byte, 16)
	traceID[15] = id
	return traceID
}

func experimentalTraceDiffStringKV(key, value string) *commonv1.KeyValue {
	return &commonv1.KeyValue{
		Key: key,
		Value: &commonv1.AnyValue{
			Value: &commonv1.AnyValue_StringValue{StringValue: value},
		},
	}
}

func experimentalTraceDiffDoubleKV(key string, value float64) *commonv1.KeyValue {
	return &commonv1.KeyValue{
		Key: key,
		Value: &commonv1.AnyValue{
			Value: &commonv1.AnyValue_DoubleValue{DoubleValue: value},
		},
	}
}
