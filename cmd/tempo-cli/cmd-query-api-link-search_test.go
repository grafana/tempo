package main

import (
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidSpanIDString(t *testing.T) {
	tests := []struct {
		name     string
		spanID   string
		expected bool
	}{
		{
			name:     "valid 16 hex chars",
			spanID:   "1234567890abcdef",
			expected: true,
		},
		{
			name:     "valid uppercase hex",
			spanID:   "1234567890ABCDEF",
			expected: true,
		},
		{
			name:     "empty string",
			spanID:   "",
			expected: false,
		},
		{
			name:     "too short",
			spanID:   "1234567890abcde",
			expected: false,
		},
		{
			name:     "too long",
			spanID:   "1234567890abcdef0",
			expected: false,
		},
		{
			name:     "invalid characters",
			spanID:   "1234567890abcdeg",
			expected: false,
		},
		{
			name:     "all zeros",
			spanID:   "0000000000000000",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidSpanIDString(tt.spanID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestApplyTraceLimit(t *testing.T) {
	traces := []*tempopb.TraceSearchMetadata{
		{TraceID: "trace1"},
		{TraceID: "trace2"},
		{TraceID: "trace3"},
		{TraceID: "trace4"},
		{TraceID: "trace5"},
	}

	tests := []struct {
		name            string
		traces          []*tempopb.TraceSearchMetadata
		limit           uint32
		expectedLen     int
		expectedTrunc   bool
		expectedTraceID string // ID of first trace
	}{
		{
			name:            "limit 0 returns all",
			traces:          traces,
			limit:           0,
			expectedLen:     5,
			expectedTrunc:   false,
			expectedTraceID: "trace1",
		},
		{
			name:            "limit greater than length returns all",
			traces:          traces,
			limit:           10,
			expectedLen:     5,
			expectedTrunc:   false,
			expectedTraceID: "trace1",
		},
		{
			name:            "limit equals length returns all",
			traces:          traces,
			limit:           5,
			expectedLen:     5,
			expectedTrunc:   false,
			expectedTraceID: "trace1",
		},
		{
			name:            "limit less than length truncates",
			traces:          traces,
			limit:           3,
			expectedLen:     3,
			expectedTrunc:   true,
			expectedTraceID: "trace1",
		},
		{
			name:            "limit 1",
			traces:          traces,
			limit:           1,
			expectedLen:     1,
			expectedTrunc:   true,
			expectedTraceID: "trace1",
		},
		{
			name:          "empty traces",
			traces:        []*tempopb.TraceSearchMetadata{},
			limit:         5,
			expectedLen:   0,
			expectedTrunc: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, truncated := applyTraceLimit(tt.traces, tt.limit)
			assert.Equal(t, tt.expectedLen, len(result))
			assert.Equal(t, tt.expectedTrunc, truncated)
			if len(result) > 0 {
				assert.Equal(t, tt.expectedTraceID, result[0].TraceID)
			}
		})
	}
}

func TestExtractSpanIDsFromTraces(t *testing.T) {
	tests := []struct {
		name        string
		traces      []*tempopb.TraceSearchMetadata
		expectedIDs []string
	}{
		{
			name: "single trace with spans",
			traces: []*tempopb.TraceSearchMetadata{
				{
					TraceID: "trace1",
					SpanSets: []*tempopb.SpanSet{
						{
							Spans: []*tempopb.Span{
								{SpanID: "1234567890abcdef"},
								{SpanID: "fedcba0987654321"},
							},
						},
					},
				},
			},
			expectedIDs: []string{"1234567890abcdef", "fedcba0987654321"},
		},
		{
			name: "multiple traces deduplicated",
			traces: []*tempopb.TraceSearchMetadata{
				{
					TraceID: "trace1",
					SpanSets: []*tempopb.SpanSet{
						{
							Spans: []*tempopb.Span{
								{SpanID: "1234567890abcdef"},
							},
						},
					},
				},
				{
					TraceID: "trace2",
					SpanSets: []*tempopb.SpanSet{
						{
							Spans: []*tempopb.Span{
								{SpanID: "1234567890abcdef"}, // duplicate
								{SpanID: "fedcba0987654321"},
							},
						},
					},
				},
			},
			expectedIDs: []string{"1234567890abcdef", "fedcba0987654321"},
		},
		{
			name: "filters invalid span IDs",
			traces: []*tempopb.TraceSearchMetadata{
				{
					TraceID: "trace1",
					SpanSets: []*tempopb.SpanSet{
						{
							Spans: []*tempopb.Span{
								{SpanID: "1234567890abcdef"},
								{SpanID: "invalid"},
								{SpanID: ""},
								{SpanID: "fedcba0987654321"},
							},
						},
					},
				},
			},
			expectedIDs: []string{"1234567890abcdef", "fedcba0987654321"},
		},
		{
			name:        "empty traces",
			traces:      []*tempopb.TraceSearchMetadata{},
			expectedIDs: []string{},
		},
		{
			name: "trace with no spans",
			traces: []*tempopb.TraceSearchMetadata{
				{
					TraceID:  "trace1",
					SpanSets: []*tempopb.SpanSet{},
				},
			},
			expectedIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSpanIDsFromTraces(tt.traces)
			assert.ElementsMatch(t, tt.expectedIDs, result)
		})
	}
}

func TestDeduplicateSpansInTrace(t *testing.T) {
	tests := []struct {
		name           string
		trace          *tempopb.TraceSearchMetadata
		expectedSpanIDs []string
	}{
		{
			name: "no duplicates",
			trace: &tempopb.TraceSearchMetadata{
				TraceID: "trace1",
				SpanSets: []*tempopb.SpanSet{
					{
						Spans: []*tempopb.Span{
							{SpanID: "1234567890abcdef"},
							{SpanID: "fedcba0987654321"},
						},
					},
				},
			},
			expectedSpanIDs: []string{"1234567890abcdef", "fedcba0987654321"},
		},
		{
			name: "duplicates within same spanset",
			trace: &tempopb.TraceSearchMetadata{
				TraceID: "trace1",
				SpanSets: []*tempopb.SpanSet{
					{
						Spans: []*tempopb.Span{
							{SpanID: "1234567890abcdef"},
							{SpanID: "1234567890abcdef"},
							{SpanID: "fedcba0987654321"},
						},
					},
				},
			},
			expectedSpanIDs: []string{"1234567890abcdef", "fedcba0987654321"},
		},
		{
			name: "duplicates across spansets",
			trace: &tempopb.TraceSearchMetadata{
				TraceID: "trace1",
				SpanSets: []*tempopb.SpanSet{
					{
						Spans: []*tempopb.Span{
							{SpanID: "1234567890abcdef"},
						},
					},
					{
						Spans: []*tempopb.Span{
							{SpanID: "1234567890abcdef"},
							{SpanID: "fedcba0987654321"},
						},
					},
				},
			},
			expectedSpanIDs: []string{"1234567890abcdef", "fedcba0987654321"},
		},
		{
			name: "empty span IDs filtered out",
			trace: &tempopb.TraceSearchMetadata{
				TraceID: "trace1",
				SpanSets: []*tempopb.SpanSet{
					{
						Spans: []*tempopb.Span{
							{SpanID: ""},
							{SpanID: "1234567890abcdef"},
							{SpanID: ""},
						},
					},
				},
			},
			expectedSpanIDs: []string{"1234567890abcdef"},
		},
		{
			name: "nil spanset handled",
			trace: &tempopb.TraceSearchMetadata{
				TraceID: "trace1",
				SpanSets: []*tempopb.SpanSet{
					nil,
					{
						Spans: []*tempopb.Span{
							{SpanID: "1234567890abcdef"},
						},
					},
				},
			},
			expectedSpanIDs: []string{"1234567890abcdef"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deduplicateSpansInTrace(tt.trace)

			// Extract span IDs from result
			var resultSpanIDs []string
			for _, ss := range tt.trace.SpanSets {
				for _, span := range ss.Spans {
					resultSpanIDs = append(resultSpanIDs, span.SpanID)
				}
			}

			assert.ElementsMatch(t, tt.expectedSpanIDs, resultSpanIDs)
		})
	}
}

func TestForEachSpanInTrace(t *testing.T) {
	trace := &tempopb.TraceSearchMetadata{
		TraceID: "trace1",
		SpanSets: []*tempopb.SpanSet{
			{
				Spans: []*tempopb.Span{
					{SpanID: "span1"},
					{SpanID: "span2"},
				},
			},
			{
				Spans: []*tempopb.Span{
					{SpanID: "span3"},
				},
			},
		},
	}

	var collected []string
	forEachSpanInTrace(trace, func(span *tempopb.Span) {
		collected = append(collected, span.SpanID)
	})

	expected := []string{"span1", "span2", "span3"}
	assert.Equal(t, expected, collected)
}

func TestFilterCompleteChains(t *testing.T) {
	cmd := &queryAPILinkSearchCmd{}

	tests := []struct {
		name           string
		tracesByPhase  [][]*tempopb.TraceSearchMetadata
		spanIDsByPhase [][]string
		linkChain      []*traceql.LinkOperationInfo
		expectedTraces int
		description    string
	}{
		{
			name: "single phase",
			tracesByPhase: [][]*tempopb.TraceSearchMetadata{
				{
					{TraceID: "trace1", SpanSets: []*tempopb.SpanSet{{Spans: []*tempopb.Span{{SpanID: "span1"}}}}},
					{TraceID: "trace2", SpanSets: []*tempopb.SpanSet{{Spans: []*tempopb.Span{{SpanID: "span2"}}}}},
				},
			},
			spanIDsByPhase: [][]string{
				{"span1", "span2"},
			},
			linkChain: []*traceql.LinkOperationInfo{
				{IsLinkTo: true},
			},
			expectedTraces: 2,
			description:    "Single phase should return all traces",
		},
		{
			name: "empty traces",
			tracesByPhase: [][]*tempopb.TraceSearchMetadata{
				{},
			},
			spanIDsByPhase: [][]string{
				{},
			},
			linkChain: []*traceql.LinkOperationInfo{
				{IsLinkTo: true},
			},
			expectedTraces: 0,
			description:    "Empty traces should return empty result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cmd.filterCompleteChains(tt.tracesByPhase, tt.spanIDsByPhase, tt.linkChain)
			require.NotNil(t, result)
			assert.Equal(t, tt.expectedTraces, len(result), tt.description)
		})
	}
}

func TestParseTimeExpression(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		expr          string
		expectError   bool
		checkDuration bool
		duration      time.Duration
	}{
		{
			name:          "now",
			expr:          "now",
			expectError:   false,
			checkDuration: false,
		},
		{
			name:          "now-1h",
			expr:          "now-1h",
			expectError:   false,
			checkDuration: true,
			duration:      time.Hour,
		},
		{
			name:          "now-30m",
			expr:          "now-30m",
			expectError:   false,
			checkDuration: true,
			duration:      30 * time.Minute,
		},
		{
			name:          "now-1d",
			expr:          "now-1d",
			expectError:   false,
			checkDuration: true,
			duration:      24 * time.Hour,
		},
		{
			name:          "now-45s",
			expr:          "now-45s",
			expectError:   false,
			checkDuration: true,
			duration:      45 * time.Second,
		},
		{
			name:        "ISO8601",
			expr:        "2024-01-01T00:00:00Z",
			expectError: false,
		},
		{
			name:        "invalid format",
			expr:        "invalid",
			expectError: true,
		},
		{
			name:        "invalid relative",
			expr:        "now-abc",
			expectError: true,
		},
		{
			name:        "invalid unit",
			expr:        "now-1x",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTimeExpression(tt.expr)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.checkDuration {
				// Check that the result is approximately the expected duration ago
				expected := now.Add(-tt.duration)
				diff := expected.Sub(result).Abs()
				assert.Less(t, diff, 2*time.Second, "Time should be within 2 seconds of expected")
			}
		})
	}
}

