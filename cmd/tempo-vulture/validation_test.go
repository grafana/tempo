package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestWriteValidationTraces(t *testing.T) {
	tests := []struct {
		name           string
		config         ValidationConfig
		expectError    bool
		emitBatchError error
	}{
		{
			name: "successful writes",
			config: ValidationConfig{
				Cycles:     2,
				TempoOrgID: "test-org",
			},
			expectError: false,
		},
		{
			name: "emit batch failure",
			config: ValidationConfig{
				Cycles:     1,
				TempoOrgID: "test-org",
			},
			expectError:    true,
			emitBatchError: errors.New("network error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use existing MockReporter instead of custom MockJaegerClient
			mockReporter := &MockReporter{
				err: tt.emitBatchError, // Set error if test expects failure
			}

			mockClock := &MockClock{
				now: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			}

			vs := &ValidationService{
				config: tt.config,
				clock:  mockClock,
				logger: zap.NewNop(),
			}

			// Call the function
			traces, err := vs.writeValidationTraces(context.Background(), mockReporter)

			// Assertions
			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, traces)
			} else {
				assert.NoError(t, err)
				assert.Len(t, traces, tt.config.Cycles)

				// Verify trace info is populated correctly
				for i, trace := range traces {
					assert.NotEmpty(t, trace.id, "trace ID should be set")
					expectedTime := mockClock.now.Add(-time.Duration(i) * time.Second)
					assert.Equal(t, expectedTime, trace.timestamp)
				}

				// Verify batches were actually emitted
				emittedBatches := mockReporter.GetEmittedBatches()
				assert.NotEmpty(t, emittedBatches, "should have emitted some batches")
			}
		})
	}
}

func TestValidateTraceRetrieval(t *testing.T) {
	tests := []struct {
		name             string
		traces           []traceInfo
		queryError       error
		traceResponse    *tempopb.Trace
		expectedFailures int
	}{
		{
			name: "successful retrieval",
			traces: []traceInfo{
				{id: "trace-1", timestamp: time.Now()},
				{id: "trace-2", timestamp: time.Now()},
			},
			traceResponse: &tempopb.Trace{
				ResourceSpans: []*v1.ResourceSpans{{}}, // Non-empty
			},
			expectedFailures: 0,
		},
		{
			name: "query error",
			traces: []traceInfo{
				{id: "trace-1", timestamp: time.Now()},
			},
			queryError:       errors.New("query failed"),
			expectedFailures: 1,
		},
		{
			name: "empty trace spans",
			traces: []traceInfo{
				{id: "trace-1", timestamp: time.Now()},
			},
			traceResponse: &tempopb.Trace{
				ResourceSpans: []*v1.ResourceSpans{}, // Empty!
			},
			expectedFailures: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use existing MockHTTPClient
			mockHTTP := &MockHTTPClient{
				err:       tt.queryError,
				traceResp: tt.traceResponse,
			}

			vs := &ValidationService{
				logger: zap.NewNop(),
			}

			result := &ValidationResult{
				TotalTraces: len(tt.traces),
				Failures:    []ValidationFailure{},
			}

			// Call the function
			vs.validateTraceRetrieval(context.Background(), tt.traces, mockHTTP, result)

			// Assertions
			assert.Len(t, result.Failures, tt.expectedFailures)

			// Verify HTTP client was called the right number of times
			if tt.expectedFailures == 0 {
				assert.Equal(t, len(tt.traces), mockHTTP.GetRequestsCount())
			}
		})
	}
}

func TestValidateTraceSearch(t *testing.T) {
	tests := []struct {
		name             string
		traces           []traceInfo
		searchError      error
		searchResponse   []*tempopb.TraceSearchMetadata
		expectedFailures int
	}{
		{
			name: "successful search",
			traces: []traceInfo{
				{id: "trace-1", timestamp: time.Now()},
			},
			searchResponse: []*tempopb.TraceSearchMetadata{
				{TraceID: "trace-1"}, // Found our trace!
			},
			expectedFailures: 0,
		},
		{
			name: "trace not found in search",
			traces: []traceInfo{
				{id: "trace-1", timestamp: time.Now()},
			},
			searchResponse: []*tempopb.TraceSearchMetadata{
				{TraceID: "different-trace"}, // Our trace not in results
			},
			expectedFailures: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := &MockHTTPClient{
				err:            tt.searchError,
				searchResponse: tt.searchResponse,
			}

			vs := &ValidationService{
				config: ValidationConfig{TempoOrgID: "test-org"},
				logger: zap.NewNop(),
			}

			result := &ValidationResult{
				TotalTraces: len(tt.traces),
				Failures:    []ValidationFailure{},
			}

			// Call the function
			vs.validateTraceSearch(context.Background(), tt.traces, mockHTTP, result)

			// Assertions
			assert.Len(t, result.Failures, tt.expectedFailures)

			// Verify search was called
			assert.Equal(t, len(tt.traces), mockHTTP.GetSearchesCount())
		})
	}
}
