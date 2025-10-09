package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestWriteValidationTrace(t *testing.T) {
	tests := []struct {
		name           string
		config         ValidationConfig
		expectError    bool
		emitBatchError error
	}{
		{
			name: "successful writes",
			config: ValidationConfig{
				TempoOrgID: "test-org",
			},
			expectError: false,
		},
		{
			name: "emit batch failure",
			config: ValidationConfig{
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
			traceInfo, actualTrace, err := vs.writeValidationTrace(context.Background(), mockReporter)

			// Assertions
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, traceInfo, "trace info should be nil")
				assert.Nil(t, actualTrace, "actual trace should be nil")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, traceInfo, "traceInfo should not be nil")
				assert.NotNil(t, actualTrace, "actualTrace should not be nil")

				// verify traceInfo is populated correctly
				assert.NotEmpty(t, traceInfo.HexID(), "traceInfo.HexID should not be empty")
				assert.NotEmpty(t, actualTrace.ResourceSpans, "actualTrace.ResourceSpans should not be empty")

				// Verify batches were actually emitted
				emittedBatches := mockReporter.GetEmittedBatches()
				assert.NotEmpty(t, emittedBatches, "should have emitted some batches")
			}
		})
	}
}

func TestValidateTraceRetrieval(t *testing.T) {
	tests := []struct {
		name          string
		queryError    error
		traceResponse *tempopb.Trace
		expectError   bool
	}{
		{
			name: "successful retrieval",
			// We'll create matching trace ID dynamically
			traceResponse: &tempopb.Trace{
				ResourceSpans: []*v1.ResourceSpans{
					{
						ScopeSpans: []*v1.ScopeSpans{
							{
								Spans: []*v1.Span{
									{
										// TraceId will be set dynamically to match
										Name: "test-span",
									},
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name:        "query error",
			queryError:  errors.New("query failed"),
			expectError: true,
		},
		{
			name: "empty trace spans",
			traceResponse: &tempopb.Trace{
				ResourceSpans: []*v1.ResourceSpans{}, // Empty!
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a TraceInfo
			mockClock := &MockClock{
				now: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			}

			vs := &ValidationService{
				config: ValidationConfig{TempoOrgID: "test-org"},
				clock:  mockClock,
				logger: zap.NewNop(),
			}

			traceInfo := util.NewTraceInfo(mockClock.Now(), "test-org")

			// Set matching trace ID in the mock response
			if tt.traceResponse != nil && len(tt.traceResponse.ResourceSpans) > 0 {
				traceIDBytes, _ := traceInfo.TraceID()
				if len(tt.traceResponse.ResourceSpans[0].ScopeSpans) > 0 &&
					len(tt.traceResponse.ResourceSpans[0].ScopeSpans[0].Spans) > 0 {
					tt.traceResponse.ResourceSpans[0].ScopeSpans[0].Spans[0].TraceId = traceIDBytes
				}
			}

			mockHTTP := &MockHTTPClient{
				err:       tt.queryError,
				traceResp: tt.traceResponse,
			}

			// Call the function
			err := vs.validateTraceRetrieval(context.Background(), traceInfo, mockHTTP)

			// Assertions
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, 1, mockHTTP.GetRequestsCount())
			}
		})
	}
}

// func TestValidateTraceSearch(t *testing.T) {
// 	tests := []struct {
// 		name             string
// 		traces           []traceInfo
// 		searchError      error
// 		searchResponse   []*tempopb.TraceSearchMetadata
// 		expectedFailures int
// 	}{
// 		{
// 			name: "successful search",
// 			traces: []traceInfo{
// 				{id: "trace-1", timestamp: time.Now()},
// 			},
// 			searchResponse: []*tempopb.TraceSearchMetadata{
// 				{TraceID: "trace-1"}, // Found our trace!
// 			},
// 			expectedFailures: 0,
// 		},
// 		{
// 			name: "trace not found in search",
// 			traces: []traceInfo{
// 				{id: "trace-1", timestamp: time.Now()},
// 			},
// 			searchResponse: []*tempopb.TraceSearchMetadata{
// 				{TraceID: "different-trace"}, // Our trace not in results
// 			},
// 			expectedFailures: 1,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mockHTTP := &MockHTTPClient{
// 				err:            tt.searchError,
// 				searchResponse: tt.searchResponse,
// 			}

// 			vs := &ValidationService{
// 				config: ValidationConfig{TempoOrgID: "test-org"},
// 				logger: zap.NewNop(),
// 			}

// 			result := &ValidationResult{
// 				TotalTraces: len(tt.traces),
// 				Failures:    []ValidationFailure{},
// 			}

// 			// Call the function
// 			vs.validateTraceSearch(context.Background(), tt.traces, mockHTTP, result)

// 			// Assertions
// 			assert.Len(t, result.Failures, tt.expectedFailures)

// 			// Verify search was called
// 			assert.Equal(t, len(tt.traces), mockHTTP.GetSearchesCount())
// 		})
// 	}
// }

func TestRunValidation(t *testing.T) {
	tests := []struct {
		name                string
		config              ValidationConfig
		writeError          error
		retrievalError      error
		retrievalTrace      *tempopb.Trace
		expectedFailures    int
		expectedExitCode    int
		expectedTotalTraces int
		expectEarlyReturn   bool // For write failures
	}{
		{
			name: "complete success - no search",
			config: ValidationConfig{
				Cycles:                2,
				TempoOrgID:            "test-org",
				WriteBackoffDuration:  time.Millisecond, // Fast for testing
				SearchBackoffDuration: 0,                // Disable search
			},
			retrievalTrace: &tempopb.Trace{
				ResourceSpans: []*v1.ResourceSpans{
					{
						ScopeSpans: []*v1.ScopeSpans{
							{
								Spans: []*v1.Span{
									{Name: "test-span"},
								},
							},
						},
					},
				},
			},
			expectedFailures:    0,
			expectedExitCode:    0,
			expectedTotalTraces: 2,
		},
		{
			name: "write failure - early return",
			config: ValidationConfig{
				Cycles:                3,
				TempoOrgID:            "test-org",
				WriteBackoffDuration:  time.Millisecond,
				SearchBackoffDuration: 0,
			},
			writeError:          errors.New("write failed"),
			expectedFailures:    1,
			expectedExitCode:    1,
			expectedTotalTraces: 3,
			expectEarlyReturn:   true,
		},
		{
			name: "retrieval failure - continues processing",
			config: ValidationConfig{
				Cycles:                2,
				TempoOrgID:            "test-org",
				WriteBackoffDuration:  time.Millisecond,
				SearchBackoffDuration: 0,
			},
			retrievalError:      errors.New("query failed"),
			expectedFailures:    2, // One failure per cycle
			expectedExitCode:    1,
			expectedTotalTraces: 2,
		},
		{
			name: "empty trace spans - retrieval failure",
			config: ValidationConfig{
				Cycles:                1,
				TempoOrgID:            "test-org",
				WriteBackoffDuration:  time.Millisecond,
				SearchBackoffDuration: 0,
			},
			retrievalTrace: &tempopb.Trace{
				ResourceSpans: []*v1.ResourceSpans{}, // Empty spans
			},
			expectedFailures:    1,
			expectedExitCode:    1,
			expectedTotalTraces: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockReporter := &MockReporter{
				err: tt.writeError,
			}

			mockClock := &MockClock{
				now: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			}

			// Create validation service
			vs := &ValidationService{
				config: tt.config,
				clock:  mockClock,
				logger: zap.NewNop(),
			}

			// Setup HTTP client mock - need to handle trace ID matching
			mockHTTP := &MockHTTPClient{
				err:       tt.retrievalError,
				traceResp: tt.retrievalTrace,
			}

			// If we have a successful trace response, we need to set matching trace IDs
			if tt.retrievalTrace != nil && len(tt.retrievalTrace.ResourceSpans) > 0 {
				// We'll need to create a TraceInfo to get the expected trace ID format
				sampleTraceInfo := util.NewTraceInfo(mockClock.Now(), tt.config.TempoOrgID)
				traceIDBytes, _ := sampleTraceInfo.TraceID()

				// Set the trace ID in all spans of the mock response
				for _, resourceSpan := range tt.retrievalTrace.ResourceSpans {
					for _, scopeSpan := range resourceSpan.ScopeSpans {
						for _, span := range scopeSpan.Spans {
							span.TraceId = traceIDBytes
						}
					}
				}
			}

			// Run validation
			result := vs.RunValidation(context.Background(), mockReporter, mockHTTP, mockHTTP)

			// Assertions
			assert.Equal(t, tt.expectedFailures, len(result.Failures), "unexpected number of failures")
			assert.Equal(t, tt.expectedExitCode, result.ExitCode(), "unexpected exit code")
			assert.Equal(t, tt.expectedTotalTraces, result.TotalTraces, "unexpected total traces")

			// Check success count calculation
			expectedSuccessCount := (result.TotalTraces * 1) - len(result.Failures) // 1 validation per trace (no search)
			if expectedSuccessCount < 0 {
				expectedSuccessCount = 0
			}
			assert.Equal(t, expectedSuccessCount, result.SuccessCount, "unexpected success count")

			// Verify write attempts
			if tt.expectEarlyReturn {
				// Should only try to write once before failing
				assert.LessOrEqual(t, len(mockReporter.GetEmittedBatches()), 1, "should stop writing after first failure")
			} else if tt.writeError == nil {
				// Should have attempted all writes
				assert.NotEmpty(t, mockReporter.GetEmittedBatches(), "should have emitted batches")
			}

			// Verify retrieval attempts
			if !tt.expectEarlyReturn && tt.writeError == nil {
				expectedRetrievals := tt.config.Cycles
				assert.Equal(t, expectedRetrievals, mockHTTP.GetRequestsCount(), "unexpected number of retrieval attempts")
			}

			// Verify failure details
			for _, failure := range result.Failures {
				assert.NotEmpty(t, failure.Phase, "failure phase should be set")
				assert.NotNil(t, failure.Error, "failure error should be set")
				assert.NotZero(t, failure.Timestamp, "failure timestamp should be set")
			}
		})
	}
}
