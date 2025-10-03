package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	utilpkg "github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"go.uber.org/zap"
)

const TempoAccessPolicyToken = "TEMPO_ACCESS_POLICY_TOKEN"

func RunValidationMode(
	ctx context.Context,
	vultureConfig vultureConfiguration,
	pushEndpoint string,
	logger *zap.Logger,
) int {
	accessPolicyToken := os.Getenv(TempoAccessPolicyToken)
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
	id          string
	timestamp   time.Time
	actualTrace *tempopb.Trace
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

	result := ValidationResult{
		TotalTraces: vs.config.Cycles,
		Failures:    make([]ValidationFailure, 0),
	}

	traces := []traceInfo{}

	// for each cycle: do the reads and writes right away, but wait a bit for the
	// traces to be available to search
	for i := 0; i < vs.config.Cycles; i++ {
		// Write the traces
		trace, actual, err := vs.writeValidationTrace(ctx, writer)
		if err != nil {
			result.Failures = append(result.Failures, ValidationFailure{
				Phase:     "write",
				Error:     err,
				Timestamp: vs.clock.Now(),
			})
			result.Duration = vs.clock.Now().Sub(start)
			result.SuccessCount = (result.TotalTraces) - len(result.Failures)
			return result
		}
		vs.logger.Info("Wrote trace", zap.String("id", trace.HexID()))

		traces = append(traces, traceInfo{
			id:          trace.HexID(),
			timestamp:   trace.Timestamp(),
			actualTrace: actual,
		})

		// Validate that we can retrieve them by trace ID
		readErr := vs.validateTraceRetrieval(ctx, trace, querier)
		if readErr != nil {
			result.Failures = append(result.Failures, ValidationFailure{
				Phase:     "read",
				Error:     readErr,
				Timestamp: vs.clock.Now(),
			})
		}

		// sleep 1 sec to guarantee different timestamps on traces
		if err := vs.sleepWithContext(ctx, 1*time.Second); err != nil {
			// Context cancelled - add failure and return early
			result.Failures = append(result.Failures, ValidationFailure{
				Phase:     "timeout",
				Error:     err,
				Timestamp: vs.clock.Now(),
			})
			break
		}
	}

	if vs.config.SearchBackoffDuration > 0 {
		// give them time to be ready for search
		if err := vs.sleepWithContext(ctx, 60*time.Second); err != nil {
			result.Failures = append(result.Failures, ValidationFailure{
				Phase:     "timeout",
				Error:     err,
				Timestamp: vs.clock.Now(),
			})
		}

		for _, trace := range traces {
			traceInfo := util.NewTraceInfo(trace.timestamp, vs.config.TempoOrgID)
			// validate we can query them by a random attribute
			searchErr := vs.validateTraceSearch(ctx, traceInfo, trace.actualTrace, searcher)
			if searchErr != nil {
				result.Failures = append(result.Failures, ValidationFailure{
					Phase:     "search",
					Error:     searchErr,
					Timestamp: vs.clock.Now(),
				})
			}
		}
	}

	result.Duration = vs.clock.Now().Sub(start)
	result.SuccessCount = (result.TotalTraces) - len(result.Failures) // update this to 2 validations per trace when we enable search
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
	default:
		vs.clock.Sleep(duration)
		return nil
	}
}

func (vs *ValidationService) writeValidationTrace(
	_ context.Context,
	writer TraceWriter,
) (*util.TraceInfo, *tempopb.Trace, error) {
	traceInfo := util.NewTraceInfoWithMaxLongWrites(
		vs.clock.Now(),
		0,
		vs.config.TempoOrgID,
	)

	// Construct the trace structure BEFORE writing
	traceStructure, err := traceInfo.ConstructTraceFromEpoch()
	if err != nil {
		return nil, nil, err
	}

	writeErr := traceInfo.EmitAllBatches(writer)
	if writeErr != nil {
		return nil, nil, fmt.Errorf("failed to write trace, error: %w", err) // Any write failure is critical
	}

	return traceInfo, traceStructure, nil
}

func (vs *ValidationService) validateTraceRetrieval(
	ctx context.Context,
	trace *util.TraceInfo,
	httpClient httpclient.TempoHTTPClient,
) error {
	start := trace.Timestamp().Add(-10 * time.Minute).Unix()
	end := trace.Timestamp().Add(10 * time.Minute).Unix()

	retrievedTrace, err := httpClient.QueryTraceWithRange(ctx, trace.HexID(), start, end)
	if err != nil {
		return fmt.Errorf("failed to read trace: %w", err)
	}

	if len(retrievedTrace.ResourceSpans) == 0 {
		return fmt.Errorf("retrieved trace has no spans")
	}

	retrievedTraceID := extractTraceID(retrievedTrace)

	equal, err := util.EqualHexStringTraceIDs(trace.HexID(), retrievedTraceID)
	if err != nil {
		return fmt.Errorf("error comparing trace IDs: %w", err)
	}

	if !equal {
		return fmt.Errorf("trace IDs do not match")
	}

	vs.logger.Info("Retrieved trace", zap.String("traceID", retrievedTraceID))

	return nil
}

func (vs *ValidationService) validateTraceSearch(
	_ context.Context,
	writtenTrace *util.TraceInfo,
	actualTrace *tempopb.Trace,
	httpClient httpclient.TempoHTTPClient,
) error {
	vs.logTraceAttributes(actualTrace, writtenTrace.HexID())

	// Get a random attribute from the expected trace
	attr := util.RandomAttrFromTrace(actualTrace)
	if attr == nil {
		return fmt.Errorf("no searchable attribute found in trace")
	}

	searchQuery := fmt.Sprintf(`{.%s = "%s"}`, attr.Key, util.StringifyAnyValue(attr.Value))
	vs.logger.Info("Searching for trace",
		zap.String("traceID", writtenTrace.HexID()),
		zap.String("searchQuery", searchQuery),
	)

	start := writtenTrace.Timestamp().Add(-30 * time.Minute).Unix()
	end := writtenTrace.Timestamp().Add(30 * time.Minute).Unix()

	searchResp, searchErr := httpClient.SearchTraceQLWithRange(searchQuery, start, end)
	if searchErr != nil {
		return fmt.Errorf("search API failed: %w", searchErr)
	}

	vs.logger.Info("Search response received",
		zap.Int("traces_found", len(searchResp.Traces)),
	)

	// Check if our trace ID is in the search results
	found := false
	for _, trace := range searchResp.Traces {
		vs.logger.Info("written trace", zap.String("trace_id", writtenTrace.HexID()))
		vs.logger.Info("found trace", zap.String("trace_id", trace.TraceID))

		equal, err := util.EqualHexStringTraceIDs(writtenTrace.HexID(), trace.TraceID)
		if err != nil {
			return fmt.Errorf("error comparing trace IDs: %w", err)
		}
		if !equal {
			return fmt.Errorf("trace IDs do not match")
		}

		if equal {
			found = true
			break
		}
	}

	if found {
		vs.logger.Info("Found trace via search")
	} else {
		return fmt.Errorf("trace not found in search results")
	}
	return nil
}

func extractTraceID(trace *tempopb.Trace) string {
	if len(trace.ResourceSpans) == 0 {
		return ""
	}

	for _, resourceSpan := range trace.ResourceSpans {
		for _, scopeSpan := range resourceSpan.ScopeSpans {
			for _, span := range scopeSpan.Spans {
				if len(span.TraceId) > 0 {
					return fmt.Sprintf("%x", span.TraceId)
				}
			}
		}
	}
	return ""
}

func (vs *ValidationService) logTraceAttributes(trace *tempopb.Trace, traceID string) {
	vs.logger.Debug("=== TRACE ATTRIBUTES DEBUG ===", zap.String("traceID", traceID))

	for i, resourceSpan := range trace.ResourceSpans {
		vs.logger.Info("Resource attributes", zap.Int("resourceSpan", i))
		if resourceSpan.Resource != nil {
			for _, attr := range resourceSpan.Resource.Attributes {
				vs.logger.Info("  Resource attr",
					zap.String("key", attr.Key),
					zap.String("value", util.StringifyAnyValue(attr.Value)),
				)
			}
		}

		for j, scopeSpan := range resourceSpan.ScopeSpans {
			vs.logger.Info("Scope spans", zap.Int("scopeSpan", j))
			for k, span := range scopeSpan.Spans {
				vs.logger.Info("  Span",
					zap.Int("spanIndex", k),
					zap.String("name", span.Name),
				)
				for _, attr := range span.Attributes {
					vs.logger.Info("    Span attr",
						zap.String("key", attr.Key),
						zap.String("value", util.StringifyAnyValue(attr.Value)),
					)
				}
			}
		}
	}
	vs.logger.Info("=== END TRACE DEBUG ===")
}
