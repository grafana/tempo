package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/util"
	"go.uber.org/zap"
)

type traceInfo struct {
	id        string
	timestamp time.Time
}

type Clock interface {
	Now() time.Time
	Sleep(duration time.Duration)
}

type RealClock struct{}

func (RealClock) Now() time.Time        { return time.Now() }
func (RealClock) Sleep(d time.Duration) { time.Sleep(d) }

type ValidationConfig struct {
	Cycles                int
	Timeout               time.Duration
	TempoOrgID            string
	WriteBackoffDuration  time.Duration
	SearchBackoffDuration time.Duration
}

type ValidationResult struct {
	TotalTraces  int
	SuccessCount int
	Failures     []ValidationFailure
	Duration     time.Duration
}

type ValidationFailure struct {
	TraceID   string
	Cycle     int
	Phase     string
	Error     error
	Timestamp time.Time
}

func (vr ValidationResult) ExitCode() int {
	if len(vr.Failures) > 0 {
		return 1
	}
	return 0
}

type ValidationService struct {
	config ValidationConfig
	clock  Clock
	logger *zap.Logger
}

func NewValidationService(config ValidationConfig, clock Clock, logger *zap.Logger) *ValidationService {
	return &ValidationService{
		config: config,
		clock:  clock,
		logger: logger,
	}
}

func (vs *ValidationService) RunValidation(
	ctx context.Context,
	writer TraceWriter,
	querier TraceQuerier,
	searcher TraceSearcher,
) ValidationResult {
	start := vs.clock.Now()

	// This ValidationResult will be passed to each validation method
	result := ValidationResult{
		TotalTraces: vs.config.Cycles,
		Failures:    make([]ValidationFailure, 0),
	}

	// Write the traces
	traces, err := vs.writeValidationTraces(ctx, writer)
	if err != nil {
		result.Failures = append(result.Failures, ValidationFailure{
			Phase:     "write",
			Error:     err,
			Timestamp: vs.clock.Now(),
		})
		result.Duration = vs.clock.Now().Sub(start)
		return result
	}

	// Wait for indexing
	vs.clock.Sleep(vs.config.WriteBackoffDuration * 2)

	// Validate that we can retrieve them by trace ID
	vs.validateTraceRetrieval(ctx, traces, querier, &result)

	// If search is enabled, validate we can query for the traces
	if vs.config.SearchBackoffDuration > 0 {
		vs.clock.Sleep(vs.config.SearchBackoffDuration)
		vs.validateTraceSearch(ctx, traces, searcher, &result)
	}

	result.Duration = vs.clock.Now().Sub(start)
	result.SuccessCount = (result.TotalTraces * 2) - len(result.Failures) // 2 validations per trace
	return result
}

// Aliases for interfaces
type (
	TraceWriter   = util.JaegerClient
	TraceQuerier  = httpclient.TempoHTTPClient
	TraceSearcher = httpclient.TempoHTTPClient
)

func (vs *ValidationService) writeValidationTraces(
	ctx context.Context,
	writer TraceWriter,
) ([]traceInfo, error) {
	traces := make([]traceInfo, 0, vs.config.Cycles)

	for i := 0; i < vs.config.Cycles; i++ {
		timestamp := vs.clock.Now().Add(-time.Duration(i) * time.Second)
		info := util.NewTraceInfoWithMaxLongWrites(timestamp, 0, vs.config.TempoOrgID)

		vs.logger.Info("Writing trace", zap.Int("cycle", i+1), zap.String("traceID", info.HexID()))

		err := info.EmitBatches(writer)
		if err != nil {
			vs.logger.Error("Failed to write trace", zap.Int("cycle", i+1), zap.Error(err))
			return traces, err // Any write failure is critical
		}

		traces = append(traces, traceInfo{
			id:        info.HexID(),
			timestamp: timestamp,
		})

		vs.logger.Info("Wrote trace", zap.Int("cycle", i+1))
	}

	return traces, nil
}

func (vs *ValidationService) validateTraceRetrieval(
	ctx context.Context,
	traces []traceInfo,
	httpClient httpclient.TempoHTTPClient,
	result *ValidationResult,
) {
	for i, trace := range traces {
		cycle := i + 1
		vs.logger.Info("Validating trace retrieval", zap.Int("cycle", cycle), zap.String("traceID", trace.id))

		start := trace.timestamp.Add(-10 * time.Minute).Unix()
		end := trace.timestamp.Add(10 * time.Minute).Unix()

		retrievedTrace, err := httpClient.QueryTraceWithRange(trace.id, start, end)
		if err != nil {
			vs.logger.Error("Failed to read trace", zap.Int("cycle", cycle), zap.Error(err))
			// failures++
			result.Failures = append(result.Failures, ValidationFailure{
				TraceID:   trace.id,
				Cycle:     cycle,
				Phase:     "retrieval",
				Error:     err,
				Timestamp: trace.timestamp,
			})
			continue
		}

		if len(retrievedTrace.ResourceSpans) == 0 {
			vs.logger.Error("Retrieved trace has no spans", zap.Int("cycle", cycle))
			result.Failures = append(result.Failures, ValidationFailure{
				TraceID:   trace.id,
				Cycle:     cycle,
				Phase:     "retrieval",
				Error:     errors.New("retrieved trace has no spans"),
				Timestamp: trace.timestamp,
			})
			continue
		}
		result.SuccessCount++

		vs.logger.Info("Validated trace", zap.Int("cycle", cycle), zap.Int("resourceSpans", len(retrievedTrace.ResourceSpans)))
	}
}

func (vs *ValidationService) validateTraceSearch(
	ctx context.Context,
	traces []traceInfo,
	httpClient httpclient.TempoHTTPClient,
	result *ValidationResult,
) {
	vs.logger.Info("Waiting for search indexing to complete...")

	vs.logger.Info("writtenTraces", zap.Int("count", len(traces)))
	vs.logger.Info("Testing search functionality - looking for each written trace...")

	for i, traceInfo := range traces {
		cycle := i + 1

		// Create a fresh TraceInfo to get the expected attributes
		info := util.NewTraceInfoWithMaxLongWrites(traceInfo.timestamp, 0, vs.config.TempoOrgID)
		expected, err := info.ConstructTraceFromEpoch()
		if err != nil {
			logger.Error("Failed to construct expected trace for search", zap.Int("cycle", cycle), zap.Error(err))
			result.Failures = append(result.Failures, ValidationFailure{
				TraceID:   traceInfo.id,
				Cycle:     cycle,
				Phase:     "search",
				Error:     err,
				Timestamp: traceInfo.timestamp,
			})
			continue
		}

		// Get a random attribute from the expected trace (same as original vulture)
		attr := util.RandomAttrFromTrace(expected)
		if attr == nil {
			logger.Warn("No searchable attribute found in trace", zap.Int("cycle", cycle))
			continue // Skip this search, don't count as failure
		}

		searchQuery := fmt.Sprintf("%s=%s", attr.Key, util.StringifyAnyValue(attr.Value))
		logger.Info("Searching for trace",
			zap.Int("cycle", cycle),
			zap.String("traceID", traceInfo.id),
			zap.String("searchQuery", searchQuery),
		)

		start := traceInfo.timestamp.Add(-30 * time.Minute).Unix()
		end := traceInfo.timestamp.Add(30 * time.Minute).Unix()

		searchResp, err := httpClient.SearchWithRange(searchQuery, start, end)
		if err != nil {
			logger.Error("Search API failed", zap.Int("cycle", cycle), zap.Error(err))
			result.Failures = append(result.Failures, ValidationFailure{
				TraceID:   traceInfo.id,
				Cycle:     cycle,
				Phase:     "search",
				Error:     err,
				Timestamp: traceInfo.timestamp,
			})
			continue
		}

		// Check if our trace ID is in the search results
		found := false
		logger.Info("found traces", zap.Int("count", len(searchResp.Traces)))
		for _, trace := range searchResp.Traces {
			logger.Info("Comparing found trace",
				zap.String("trace ID", trace.TraceID),
			)
			if trace.TraceID == traceInfo.id {
				found = true
				break
			}
		}

		if found {
			logger.Info("Found trace via search", zap.Int("cycle", cycle))
		} else {
			logger.Error("Trace not found in search results",
				zap.Int("cycle", cycle),
				zap.String("traceID", traceInfo.id),
				zap.Int("foundTraces", len(searchResp.Traces)),
			)
			result.Failures = append(result.Failures, ValidationFailure{
				TraceID:   traceInfo.id,
				Cycle:     cycle,
				Phase:     "search",
				Error:     err,
				Timestamp: traceInfo.timestamp,
			})
		}
	}
}
