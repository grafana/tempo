package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	utilpkg "github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/util"
	"go.uber.org/zap"
)

func RunValidationMode(
	ctx context.Context,
	vultureConfig vultureConfiguration,
	pushEndpoint string,
	logger *zap.Logger,
) int {
	accessPolicyToken := os.Getenv("TEMPO_ACCESS_POLICY_TOKEN")
	if accessPolicyToken == "" {
		logger.Error("TEMPO_ACCESS_POLICY_TOKEN environment variable is required in validation mode")
		return 1
	}

	// Construct the basic auth token for HTTP headers
	basicAuthToken := constructAuthToken(vultureConfig.tempoOrgID, accessPolicyToken)

	httpClient := httpclient.New(vultureConfig.tempoQueryURL, vultureConfig.tempoOrgID)
	httpClient.SetHeader("Authorization", fmt.Sprintf("Basic %s", basicAuthToken))

	// Create authenticated jaeger client for writing traces
	authJaegerClient, err := utilpkg.NewJaegerToOTLPExporterWithAuth(
		pushEndpoint,
		vultureConfig.tempoOrgID,
		basicAuthToken,
		vultureConfig.tempoPushTLS,
	)
	if err != nil {
		logger.Error("failed to create authenticated OTLP exporter", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Running in validation mode",
		zap.Int("cycles", validationCycles),
		zap.Duration("timeout", validationTimeout),
		zap.String("org_id", vultureConfig.tempoOrgID),
	)

	validationConfig := ValidationConfig{
		Cycles:                validationCycles,
		TempoOrgID:            vultureConfig.tempoOrgID,
		TempoBasicAuthToken:   basicAuthToken,
		WriteBackoffDuration:  vultureConfig.tempoWriteBackoffDuration,
		SearchBackoffDuration: vultureConfig.tempoSearchBackoffDuration,
	}

	service := NewValidationService(validationConfig, RealClock{}, logger)
	result := service.RunValidation(ctx, authJaegerClient, httpClient, httpClient)

	// Log detailed results before exiting
	logger.Info("Validation completed",
		zap.Int("total_traces", result.TotalTraces),
		zap.Int("validations_passed", result.SuccessCount),
		zap.Int("validations_failed", len(result.Failures)),
		zap.Duration("duration", result.Duration),
	)

	// Optionally log each failure for debugging
	for _, failure := range result.Failures {
		logger.Error("validation failure",
			zap.String("phase", failure.Phase),
			zap.String("traceID", failure.TraceID),
			zap.Int("cycle", failure.Cycle),
			zap.Error(failure.Error),
		)
	}

	return result.ExitCode()
}

// constructAuthToken creates base64(orgID:accessToken) if both are provided
func constructAuthToken(orgID, accessToken string) string {
	if orgID == "" || accessToken == "" {
		return ""
	}
	combined := fmt.Sprintf("%s:%s", orgID, accessToken)
	return base64.StdEncoding.EncodeToString([]byte(combined))
}

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
	TempoBasicAuthToken   string
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

	// for each cycle: write, read, search
	for i := 0; i < vs.config.Cycles; i++ {
		// Write the traces
		vs.logger.Info("Starting write/read/search cycle", zap.Int("cycle", i+1))
		traceInfo, err := vs.writeValidationTrace(ctx, writer)
		if err != nil {
			result.Failures = append(result.Failures, ValidationFailure{
				Phase:     "write",
				Error:     err,
				Timestamp: vs.clock.Now(),
			})
			result.Duration = vs.clock.Now().Sub(start)
			return result
		}
		vs.logger.Info("wrote trace", zap.String("id", traceInfo.HexID()))

		// Validate that we can retrieve them by trace ID
		readErr := vs.validateTraceRetrieval(ctx, traceInfo, querier, &result)
		if readErr != nil {
			result.Failures = append(result.Failures, ValidationFailure{
				Phase:     "read",
				Error:     readErr,
				Timestamp: vs.clock.Now(),
			})
		}

		// validate we can query them by a random attribute
		searchErr := vs.validateTraceSearch(ctx, traceInfo, searcher, &result)
		if searchErr != nil {
			result.Failures = append(result.Failures, ValidationFailure{
				Phase:     "search",
				Error:     searchErr,
				Timestamp: vs.clock.Now(),
			})
		}

		// sleep 1 sec to guarantee different timestamps on traces
		if err := vs.sleepWithContext(ctx, 1*time.Second); err != nil {
			// Context cancelled - add failure and return early
			result.Failures = append(result.Failures, ValidationFailure{
				Phase:     "timeout",
				Error:     ctx.Err(),
				Timestamp: vs.clock.Now(),
			})
			break
		}
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

func (vs *ValidationService) sleepWithContext(ctx context.Context, duration time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		return nil
	}
}

func (vs *ValidationService) writeValidationTrace(
	ctx context.Context,
	writer TraceWriter,
) (*util.TraceInfo, error) {
	timestamp := vs.clock.Now()
	info := util.NewTraceInfo(timestamp, vs.config.TempoOrgID)

	err := info.EmitBatchesWithContext(ctx, writer)
	if err != nil {
		return nil, fmt.Errorf("failed to write trace, error: %w", err) // Any write failure is critical
	}

	return info, nil
}

func (vs *ValidationService) validateTraceRetrieval(
	_ context.Context,
	trace *util.TraceInfo,
	httpClient httpclient.TempoHTTPClient,
	result *ValidationResult,
) error {

	start := trace.Timestamp().Add(-10 * time.Minute).Unix()
	end := trace.Timestamp().Add(10 * time.Minute).Unix()

	retrievedTrace, err := httpClient.QueryTraceWithRange(trace.HexID(), start, end)

	if err != nil {
		return fmt.Errorf("failed to read trace: %w", err)
	}

	if len(retrievedTrace.ResourceSpans) == 0 {
		return fmt.Errorf("retrieved trace has no spans")
	}

	vs.logger.Info("Validated trace retrieval", zap.String("traceID", trace.HexID()))
	return nil
}

// func (vs *ValidationService) validateTraceSearch(
// 	ctx context.Context,
// 	traces []traceInfo,
// 	httpClient httpclient.TempoHTTPClient,
// 	result *ValidationResult,
// ) {
// 	vs.logger.Info("Waiting for search indexing to complete...")

// 	vs.logger.Info("writtenTraces", zap.Int("count", len(traces)))
// 	vs.logger.Info("Testing search functionality - looking for each written trace...")

// 	for i, traceInfo := range traces {
// 		cycle := i + 1

// 		// Create a fresh TraceInfo to get the expected attributes
// 		info := util.NewTraceInfoWithMaxLongWrites(traceInfo.timestamp, 0, vs.config.TempoOrgID)
// 		expected, err := info.ConstructTraceFromEpoch()
// 		if err != nil {
// 			logger.Error("failed to construct expected trace for search", zap.Int("cycle", cycle), zap.Error(err))
// 			result.Failures = append(result.Failures, ValidationFailure{
// 				TraceID:   traceInfo.id,
// 				Cycle:     cycle,
// 				Phase:     "search",
// 				Error:     err,
// 				Timestamp: traceInfo.timestamp,
// 			})
// 			continue
// 		}

// 		// Get a random attribute from the expected trace (same as original vulture)
// 		attr := util.RandomAttrFromTrace(expected)
// 		if attr == nil {
// 			logger.Warn("No searchable attribute found in trace", zap.Int("cycle", cycle))
// 			continue // Skip this search, don't count as failure
// 		}

// 		searchQuery := fmt.Sprintf("{.%s=\"%s\"}", attr.Key, util.StringifyAnyValue(attr.Value))
// 		logger.Info("Searching for trace",
// 			zap.Int("cycle", cycle),
// 			zap.String("traceID", traceInfo.id),
// 			zap.String("searchQuery", searchQuery),
// 		)

// 		start := traceInfo.timestamp.Add(-30 * time.Minute).Unix()
// 		end := traceInfo.timestamp.Add(30 * time.Minute).Unix()

// 		searchResp, err := httpClient.SearchTraceQLWithRange(searchQuery, start, end)
// 		if err != nil {
// 			logger.Error("search API failed", zap.Int("cycle", cycle), zap.Error(err))
// 			result.Failures = append(result.Failures, ValidationFailure{
// 				TraceID:   traceInfo.id,
// 				Cycle:     cycle,
// 				Phase:     "search",
// 				Error:     err,
// 				Timestamp: traceInfo.timestamp,
// 			})
// 			continue
// 		}

// 		// Check if our trace ID is in the search results
// 		found := false
// 		logger.Info("found traces", zap.Int("count", len(searchResp.Traces)))
// 		for _, trace := range searchResp.Traces {
// 			logger.Info("Comparing found trace",
// 				zap.String("trace ID", trace.TraceID),
// 			)
// 			if trace.TraceID == traceInfo.id {
// 				found = true
// 				break
// 			}
// 		}

// 		if found {
// 			logger.Info("Found trace via search", zap.Int("cycle", cycle))
// 		} else {
// 			logger.Error("trace not found in search results",
// 				zap.Int("cycle", cycle),
// 				zap.String("traceID", traceInfo.id),
// 				zap.Int("foundTraces", len(searchResp.Traces)),
// 			)
// 			result.Failures = append(result.Failures, ValidationFailure{
// 				TraceID:   traceInfo.id,
// 				Cycle:     cycle,
// 				Phase:     "search",
// 				Error:     err,
// 				Timestamp: traceInfo.timestamp,
// 			})
// 		}
// 	}
// }

func (vs *ValidationService) validateTraceSearch(
	ctx context.Context,
	writtenTrace *util.TraceInfo,
	httpClient httpclient.TempoHTTPClient,
	result *ValidationResult,
) error {

	info := util.NewTraceInfo(writtenTrace.Timestamp(), vs.config.TempoOrgID)
	//hexID := info.HexID()

	// Create a fresh TraceInfo to get the expected attributes
	expected, err := info.ConstructTraceFromEpoch()

	// Get a random attribute from the expected trace
	attr := util.RandomAttrFromTrace(expected)
	if attr == nil {
		return fmt.Errorf("No searchable attribute found in trace")
	}

	searchQuery := fmt.Sprintf(`{.%s = "%s"}`, attr.Key, util.StringifyAnyValue(attr.Value))
	vs.logger.Info("Searching for trace",
		zap.String("traceID", writtenTrace.HexID()),
		zap.String("searchQuery", searchQuery),
	)

	start := writtenTrace.Timestamp().Add(-30 * time.Minute).Unix()
	end := writtenTrace.Timestamp().Add(30 * time.Minute).Unix()

	vs.logger.Info("About to execute search",
		zap.String("query", searchQuery),
		zap.Int64("start", start),
		zap.Int64("end", end),
		zap.String("orgID", vs.config.TempoOrgID),
	)

	searchResp, searchErr := httpClient.SearchTraceQLWithRange(searchQuery, start, end)
	if searchErr != nil {
		return fmt.Errorf("search API failed: %w", err)
	}

	vs.logger.Info("Search response received",
		zap.Int("totalTraces", len(searchResp.Traces)),
		zap.Any("traces", searchResp.Traces), // Log all trace IDs returned
	)

	// Check if our trace ID is in the search results
	found := false
	logger.Info("found traces", zap.Int("count", len(searchResp.Traces)))
	for _, trace := range searchResp.Traces {
		vs.logger.Info("Comparing found trace",
			zap.String("trace ID", trace.TraceID),
		)
		vs.logger.Info("written", zap.String("hexID", writtenTrace.HexID()))
		if writtenTrace.HexID() == trace.TraceID {
			found = true
			break
		}
	}

	if found {
		vs.logger.Info("Found trace via search")
	} else {
		return fmt.Errorf("Trace not found in search results")
	}
	return nil
}
