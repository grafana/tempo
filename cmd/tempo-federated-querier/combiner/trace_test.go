package combiner

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/tempopb"
)

func TestCombineTraceResults(t *testing.T) {
	logger := log.NewNopLogger()
	c := New(10*1024*1024, logger)

	tests := []struct {
		name           string
		results        []QueryResult
		wantTrace      bool
		wantSpanCount  int
		wantWithTrace  int
		wantNotFound   int
		wantFailed     int
	}{
		{
			name:          "empty results",
			results:       []QueryResult{},
			wantTrace:     false,
			wantSpanCount: 0,
		},
		{
			name: "all 404s",
			results: []QueryResult{
				{Instance: "inst1", Response: &http.Response{StatusCode: http.StatusNotFound}},
				{Instance: "inst2", Response: &http.Response{StatusCode: http.StatusNotFound}},
			},
			wantTrace:    false,
			wantNotFound: 2,
		},
		{
			name: "one error",
			results: []QueryResult{
				{Instance: "inst1", Error: fmt.Errorf("connection refused")},
			},
			wantTrace:  false,
			wantFailed: 1,
		},
		{
			name: "valid trace from one instance",
			results: []QueryResult{
				{
					Instance: "inst1",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"batches":[{"resource":{"attributes":[]},"scopeSpans":[{"scope":{},"spans":[{"traceId":"AAAAAAAAAAAAAAAAAAAAAA==","spanId":"AAAAAAAAAAE=","name":"test-span","startTimeUnixNano":"1000000000","endTimeUnixNano":"2000000000"}]}]}]}`),
				},
			},
			wantTrace:     true,
			wantSpanCount: 1,
			wantWithTrace: 1,
		},
		{
			name: "mixed results - one valid, one 404, one error",
			results: []QueryResult{
				{
					Instance: "inst1",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"batches":[{"resource":{"attributes":[]},"scopeSpans":[{"scope":{},"spans":[{"traceId":"AAAAAAAAAAAAAAAAAAAAAA==","spanId":"AAAAAAAAAAE=","name":"test-span","startTimeUnixNano":"1000000000","endTimeUnixNano":"2000000000"}]}]}]}`),
				},
				{Instance: "inst2", Response: &http.Response{StatusCode: http.StatusNotFound}},
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
		results       []QueryResult
		wantTrace     bool
		wantWithTrace int
	}{
		{
			name: "valid v2 response",
			results: []QueryResult{
				{
					Instance: "inst1",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"trace":{"batches":[{"resource":{"attributes":[]},"scopeSpans":[{"scope":{},"spans":[{"traceId":"AAAAAAAAAAAAAAAAAAAAAA==","spanId":"AAAAAAAAAAE=","name":"test-span","startTimeUnixNano":"1000000000","endTimeUnixNano":"2000000000"}]}]}]},"metrics":{}}`),
				},
			},
			wantTrace:     true,
			wantWithTrace: 1,
		},
		{
			name: "empty trace in v2 response",
			results: []QueryResult{
				{
					Instance: "inst1",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"trace":null,"metrics":{}}`),
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
