package combiner

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/tempopb"
)

func TestCombineSearchResults(t *testing.T) {
	logger := log.NewNopLogger()
	c := New(10*1024*1024, logger)

	tests := []struct {
		name            string
		results         []QueryResult
		wantTracesCount int
		wantResponded   int
		wantFailed      int
	}{
		{
			name:            "empty results",
			results:         []QueryResult{},
			wantTracesCount: 0,
		},
		{
			name: "single instance with traces",
			results: []QueryResult{
				{
					Instance: "inst1",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"traces":[{"traceID":"abc123","rootServiceName":"svc1","startTimeUnixNano":"1000000000","durationMs":100}],"metrics":{"inspectedTraces":"10","inspectedBytes":"1000"}}`),
				},
			},
			wantTracesCount: 1,
			wantResponded:   1,
		},
		{
			name: "multiple instances with duplicate traces",
			results: []QueryResult{
				{
					Instance: "inst1",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"traces":[{"traceID":"abc123","rootServiceName":"svc1","startTimeUnixNano":"1000000000","durationMs":100}],"metrics":{"inspectedTraces":"10"}}`),
				},
				{
					Instance: "inst2",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"traces":[{"traceID":"abc123","rootServiceName":"svc1","startTimeUnixNano":"1000000000","durationMs":100}],"metrics":{"inspectedTraces":"20"}}`),
				},
			},
			wantTracesCount: 1, // Deduplicated
			wantResponded:   2,
		},
		{
			name: "multiple instances with different traces",
			results: []QueryResult{
				{
					Instance: "inst1",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"traces":[{"traceID":"abc123","rootServiceName":"svc1","startTimeUnixNano":"1000000000","durationMs":100}],"metrics":{}}`),
				},
				{
					Instance: "inst2",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"traces":[{"traceID":"def456","rootServiceName":"svc2","startTimeUnixNano":"2000000000","durationMs":200}],"metrics":{}}`),
				},
			},
			wantTracesCount: 2,
			wantResponded:   2,
		},
		{
			name: "one failed instance",
			results: []QueryResult{
				{
					Instance: "inst1",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"traces":[{"traceID":"abc123","startTimeUnixNano":"1000000000","durationMs":100}],"metrics":{}}`),
				},
				{
					Instance: "inst2",
					Error:    fmt.Errorf("connection refused"),
				},
			},
			wantTracesCount: 1,
			wantResponded:   1,
			wantFailed:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, metadata, err := c.CombineSearchResults(tt.results)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(resp.Traces) != tt.wantTracesCount {
				t.Errorf("traces count = %d, want %d", len(resp.Traces), tt.wantTracesCount)
			}
			if metadata.InstancesResponded != tt.wantResponded {
				t.Errorf("InstancesResponded = %d, want %d", metadata.InstancesResponded, tt.wantResponded)
			}
			if metadata.InstancesFailed != tt.wantFailed {
				t.Errorf("InstancesFailed = %d, want %d", metadata.InstancesFailed, tt.wantFailed)
			}
		})
	}
}

func TestCombineSearchResultMetadata(t *testing.T) {
	tests := []struct {
		name     string
		existing *tempopb.TraceSearchMetadata
		incoming *tempopb.TraceSearchMetadata
		want     *tempopb.TraceSearchMetadata
	}{
		{
			name: "fills in missing root service name",
			existing: &tempopb.TraceSearchMetadata{
				TraceID:           "abc123",
				RootServiceName:   "",
				StartTimeUnixNano: 1000,
			},
			incoming: &tempopb.TraceSearchMetadata{
				TraceID:           "abc123",
				RootServiceName:   "my-service",
				StartTimeUnixNano: 1000,
			},
			want: &tempopb.TraceSearchMetadata{
				TraceID:           "abc123",
				RootServiceName:   "my-service",
				StartTimeUnixNano: 1000,
			},
		},
		{
			name: "uses earliest start time",
			existing: &tempopb.TraceSearchMetadata{
				TraceID:           "abc123",
				StartTimeUnixNano: 2000,
			},
			incoming: &tempopb.TraceSearchMetadata{
				TraceID:           "abc123",
				StartTimeUnixNano: 1000,
			},
			want: &tempopb.TraceSearchMetadata{
				TraceID:           "abc123",
				StartTimeUnixNano: 1000,
			},
		},
		{
			name: "uses longest duration",
			existing: &tempopb.TraceSearchMetadata{
				TraceID:    "abc123",
				DurationMs: 100,
			},
			incoming: &tempopb.TraceSearchMetadata{
				TraceID:    "abc123",
				DurationMs: 200,
			},
			want: &tempopb.TraceSearchMetadata{
				TraceID:    "abc123",
				DurationMs: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			combineSearchResultMetadata(tt.existing, tt.incoming)

			if tt.existing.RootServiceName != tt.want.RootServiceName {
				t.Errorf("RootServiceName = %s, want %s", tt.existing.RootServiceName, tt.want.RootServiceName)
			}
			if tt.existing.StartTimeUnixNano != tt.want.StartTimeUnixNano {
				t.Errorf("StartTimeUnixNano = %d, want %d", tt.existing.StartTimeUnixNano, tt.want.StartTimeUnixNano)
			}
			if tt.existing.DurationMs != tt.want.DurationMs {
				t.Errorf("DurationMs = %d, want %d", tt.existing.DurationMs, tt.want.DurationMs)
			}
		})
	}
}

func TestSortTracesByStartTime(t *testing.T) {
	traces := []*tempopb.TraceSearchMetadata{
		{TraceID: "a", StartTimeUnixNano: 1000},
		{TraceID: "b", StartTimeUnixNano: 3000},
		{TraceID: "c", StartTimeUnixNano: 2000},
	}

	sortTracesByStartTime(traces)

	// Should be sorted descending (most recent first)
	if traces[0].TraceID != "b" {
		t.Errorf("first trace should be 'b', got %s", traces[0].TraceID)
	}
	if traces[1].TraceID != "c" {
		t.Errorf("second trace should be 'c', got %s", traces[1].TraceID)
	}
	if traces[2].TraceID != "a" {
		t.Errorf("third trace should be 'a', got %s", traces[2].TraceID)
	}
}
