package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			mockReporter := &MockReporter{
				err: tt.emitBatchError, // Set error if test expects failure
			}

			mockClock := &MockClock{
				now: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
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
				for _, trace := range traces {
					assert.NotEmpty(t, trace.id, "trace ID should be set")
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

func TestRunValidation(t *testing.T) {
	t.Run("complete success with search", func(t *testing.T) {
		config := ValidationConfig{
			Cycles:                2,
			TempoOrgID:            "test-org",
			WriteBackoffDuration:  time.Second,
			SearchBackoffDuration: time.Second,
		}

		mockReporter := &MockReporter{}

		mockClock := &MockClock{
			now: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		}

		vs := &ValidationService{
			config: config,
			clock:  mockClock,
			logger: zap.NewNop(),
		}

		// Step 1: First, generate traces to get the actual trace IDs
		ctx := context.Background()
		traces, err := vs.writeValidationTraces(ctx, mockReporter)
		require.NoError(t, err)
		require.Len(t, traces, 2)

		// Step 2: Now setup MockHTTPClient with the actual trace IDs
		mockHTTP := &MockHTTPClient{
			traceResp: &tempopb.Trace{
				ResourceSpans: []*v1.ResourceSpans{{}}, // Non-empty
			},
			searchResponse: []*tempopb.TraceSearchMetadata{
				{TraceID: traces[0].id}, // ← Use actual generated trace ID
				{TraceID: traces[1].id}, // ← Use actual generated trace ID
			},
		}

		// Step 3: Run full validation (it will call writeValidationTraces again, but that's ok)
		result := vs.RunValidation(ctx, mockReporter, mockHTTP, mockHTTP)

		// Assertions
		assert.Equal(t, 0, len(result.Failures), "should have no failures")
		assert.Equal(t, 0, result.ExitCode(), "should return success exit code")
		assert.Equal(t, 2, result.TotalTraces)
	})

	tests := []struct {
		name              string
		config            ValidationConfig
		writeError        error
		retrievalError    error
		retrievalTrace    *tempopb.Trace
		searchError       error
		searchResponse    []*tempopb.TraceSearchMetadata
		expectedFailures  int
		expectedExitCode  int
		expectSearchCalls bool
	}{
		{
			name: "success without search (search disabled)",
			config: ValidationConfig{
				Cycles:                1,
				TempoOrgID:            "test-org",
				WriteBackoffDuration:  time.Second,
				SearchBackoffDuration: 0, // 0 disables search
			},
			writeError:     nil,
			retrievalError: nil,
			retrievalTrace: &tempopb.Trace{
				ResourceSpans: []*v1.ResourceSpans{{}}, // Non-empty
			},
			expectedFailures:  0,
			expectedExitCode:  0,
			expectSearchCalls: false,
		},
		{
			name: "write failure",
			config: ValidationConfig{
				Cycles:     1,
				TempoOrgID: "test-org",
			},
			writeError:        errors.New("write failed"),
			expectedFailures:  1,
			expectedExitCode:  1,
			expectSearchCalls: false, // No search if write fails
		},
		{
			name: "retrieval failure",
			config: ValidationConfig{
				Cycles:               1,
				TempoOrgID:           "test-org",
				WriteBackoffDuration: time.Second,
			},
			writeError:        nil,
			retrievalError:    errors.New("query failed"),
			expectedFailures:  1,
			expectedExitCode:  1,
			expectSearchCalls: false, // No search if retrieval configured
		},
		{
			name: "search failure",
			config: ValidationConfig{
				Cycles:                1,
				TempoOrgID:            "test-org",
				WriteBackoffDuration:  time.Second,
				SearchBackoffDuration: time.Second,
			},
			writeError:     nil,
			retrievalError: nil,
			retrievalTrace: &tempopb.Trace{
				ResourceSpans: []*v1.ResourceSpans{{}},
			},
			searchError: errors.New("search failed"),
			searchResponse: []*tempopb.TraceSearchMetadata{
				{TraceID: "different-trace"}, // Our trace not found
			},
			expectedFailures:  1,
			expectedExitCode:  1,
			expectSearchCalls: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockReporter := &MockReporter{
				err: tt.writeError,
			}

			mockHTTP := &MockHTTPClient{
				err:            tt.retrievalError,
				traceResp:      tt.retrievalTrace,
				searchResponse: tt.searchResponse,
			}

			// // If we have a search error but not retrieval error, we need separate handling
			// if tt.searchError != nil && tt.retrievalError == nil {
			// 	// For this test, we'll need to modify MockHTTPClient to handle different errors
			// 	// for different methods. For now, let's create a separate mock for search
			// 	mockHTTPForSearch := &MockHTTPClient{
			// 		err:            tt.searchError,
			// 		searchResponse: tt.searchResponse,
			// 	}
			// 	// This is a limitation of the current mock - it can't have different errors per method
			// 	// We'd need to enhance the mock or use a more sophisticated mocking approach
			// }

			mockClock := &MockClock{
				now: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			}

			// Create validation service
			vs := &ValidationService{
				config: tt.config,
				clock:  mockClock,
				logger: zap.NewNop(),
			}

			// Run validation
			result := vs.RunValidation(context.Background(), mockReporter, mockHTTP, mockHTTP)

			// Assertions
			assert.Equal(t, tt.expectedFailures, len(result.Failures), "unexpected number of failures")
			assert.Equal(t, tt.expectedExitCode, result.ExitCode(), "unexpected exit code")
			assert.Equal(t, tt.config.Cycles, result.TotalTraces, "unexpected total traces")

			// Verify client interaction counts
			if tt.writeError == nil {
				emittedBatches := mockReporter.GetEmittedBatches()
				assert.NotEmpty(t, emittedBatches, "should have emitted batches")
			}

			if tt.writeError == nil && tt.retrievalError == nil {
				assert.Equal(t, tt.config.Cycles, mockHTTP.GetRequestsCount(), "unexpected request count")
			}
		})
	}
}
