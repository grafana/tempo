package combiner

import (
	"fmt"
	"testing"

	"github.com/go-kit/log"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/pkg/tempopb"
)

func TestCombineTraceResults(t *testing.T) {
	logger := log.NewNopLogger()
	c := New(10*1024*1024, logger)

	tests := []struct {
		name          string
		results       []TraceResult
		wantTrace     bool
		wantSpanCount int
		wantWithTrace int
		wantNotFound  int
		wantFailed    int
	}{
		{
			name:          "empty results",
			results:       []TraceResult{},
			wantTrace:     false,
			wantSpanCount: 0,
		},
		{
			name: "all 404s",
			results: []TraceResult{
				{Instance: "inst1", NotFound: true},
				{Instance: "inst2", NotFound: true},
			},
			wantTrace:    false,
			wantNotFound: 2,
		},
		{
			name: "one error",
			results: []TraceResult{
				{Instance: "inst1", Error: fmt.Errorf("connection refused")},
			},
			wantTrace:  false,
			wantFailed: 1,
		},
		{
			name: "valid trace from one instance",
			results: []TraceResult{
				{
					Instance: "inst1",
					Response: &tempopb.Trace{
						ResourceSpans: []*v1.ResourceSpans{
							{
								Resource: &v1_resource.Resource{},
								ScopeSpans: []*v1.ScopeSpans{
									{
										Scope: &v1_common.InstrumentationScope{},
										Spans: []*v1.Span{
											{TraceId: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, SpanId: []byte{0, 0, 0, 0, 0, 0, 0, 1}, Name: "test-span", StartTimeUnixNano: 1000000000, EndTimeUnixNano: 2000000000},
										},
									},
								},
							},
						},
					},
				},
			},
			wantTrace:     true,
			wantSpanCount: 1,
			wantWithTrace: 1,
		},
		{
			name: "mixed results - one valid, one 404, one error",
			results: []TraceResult{
				{
					Instance: "inst1",
					Response: &tempopb.Trace{
						ResourceSpans: []*v1.ResourceSpans{
							{
								Resource: &v1_resource.Resource{},
								ScopeSpans: []*v1.ScopeSpans{
									{
										Scope: &v1_common.InstrumentationScope{},
										Spans: []*v1.Span{
											{TraceId: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, SpanId: []byte{0, 0, 0, 0, 0, 0, 0, 1}, Name: "test-span", StartTimeUnixNano: 1000000000, EndTimeUnixNano: 2000000000},
										},
									},
								},
							},
						},
					},
				},
				{Instance: "inst2", NotFound: true},
				{Instance: "inst3", Error: fmt.Errorf("timeout")},
			},
			wantTrace:     true,
			wantSpanCount: 1,
			wantWithTrace: 1,
			wantNotFound:  1,
			wantFailed:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trace, metadata, err := c.CombineTraceResults(tt.results)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantTrace {
				if trace == nil {
					t.Error("expected trace, got nil")
				}
			} else {
				if trace != nil && len(trace.ResourceSpans) > 0 {
					t.Error("expected no trace, got one")
				}
			}

			if metadata.InstancesWithTrace != tt.wantWithTrace {
				t.Errorf("InstancesWithTrace = %d, want %d", metadata.InstancesWithTrace, tt.wantWithTrace)
			}
			if metadata.InstancesNotFound != tt.wantNotFound {
				t.Errorf("InstancesNotFound = %d, want %d", metadata.InstancesNotFound, tt.wantNotFound)
			}
			if metadata.InstancesFailed != tt.wantFailed {
				t.Errorf("InstancesFailed = %d, want %d", metadata.InstancesFailed, tt.wantFailed)
			}
		})
	}
}

func TestCombineTraceResultsV2(t *testing.T) {
	logger := log.NewNopLogger()
	c := New(10*1024*1024, logger)

	tests := []struct {
		name          string
		results       []TraceByIDResult
		wantTrace     bool
		wantWithTrace int
	}{
		{
			name: "valid v2 response",
			results: []TraceByIDResult{
				{
					Instance: "inst1",
					Response: &tempopb.TraceByIDResponse{
						Trace: &tempopb.Trace{
							ResourceSpans: []*v1.ResourceSpans{
								{
									Resource: &v1_resource.Resource{},
									ScopeSpans: []*v1.ScopeSpans{
										{
											Scope: &v1_common.InstrumentationScope{},
											Spans: []*v1.Span{
												{TraceId: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, SpanId: []byte{0, 0, 0, 0, 0, 0, 0, 1}, Name: "test-span", StartTimeUnixNano: 1000000000, EndTimeUnixNano: 2000000000},
											},
										},
									},
								},
							},
						},
						Metrics: &tempopb.TraceByIDMetrics{},
					},
				},
			},
			wantTrace:     true,
			wantWithTrace: 1,
		},
		{
			name: "empty trace in v2 response",
			results: []TraceByIDResult{
				{
					Instance: "inst1",
					Response: &tempopb.TraceByIDResponse{
						Trace:   nil,
						Metrics: &tempopb.TraceByIDMetrics{},
					},
				},
			},
			wantTrace:     false,
			wantWithTrace: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trace, metadata, err := c.CombineTraceResultsV2(tt.results)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantTrace {
				if trace == nil {
					t.Error("expected trace, got nil")
				}
			} else {
				if trace != nil && len(trace.ResourceSpans) > 0 {
					t.Error("expected no trace, got one")
				}
			}

			if metadata.InstancesWithTrace != tt.wantWithTrace {
				t.Errorf("InstancesWithTrace = %d, want %d", metadata.InstancesWithTrace, tt.wantWithTrace)
			}
		})
	}
}

// Helper to create a mock trace search metadata
func mockTraceSearchMetadata(traceID string, startTime uint64, duration uint32) *tempopb.TraceSearchMetadata {
	return &tempopb.TraceSearchMetadata{
		TraceID:           traceID,
		StartTimeUnixNano: startTime,
		DurationMs:        duration,
	}
}
