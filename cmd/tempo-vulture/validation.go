package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/util"
	"go.uber.org/zap"
)

func runValidationMode(
	config vultureConfiguration,
	jaegerClient util.JaegerClient,
	httpClient httpclient.TempoHTTPClient,
	startTime time.Time,
	r *rand.Rand,
	interval time.Duration,
	logger *zap.Logger,
) int {
	_, cancel := context.WithTimeout(context.Background(), validationTimeout)
	defer cancel()

	logger.Info("Starting validation...")

	// Phase 1: Write traces and store their info for validation
	type traceInfo struct {
		id        string
		timestamp time.Time
	}

	writtenTraces := make([]traceInfo, 0, validationCycles)

	for i := 0; i < validationCycles; i++ {
		timestamp := time.Now().Add(-time.Duration(i) * time.Second)
		info := util.NewTraceInfoWithMaxLongWrites(timestamp, 0, config.tempoOrgID)

		logger.Info("Writing trace", zap.Int("cycle", i+1), zap.String("traceID", info.HexID()))

		err := info.EmitBatches(jaegerClient)
		if err != nil {
			logger.Error("Failed to write trace", zap.Int("cycle", i+1), zap.Error(err))
			return 1 // Any write failure is critical
		}

		writtenTraces = append(writtenTraces, traceInfo{
			id:        info.HexID(),
			timestamp: timestamp,
		})

		logger.Info("Wrote trace", zap.Int("cycle", i+1))
	}

	// Phase 2: Wait for traces to be available
	logger.Info("Waiting for traces to be indexed...")
	time.Sleep(config.tempoWriteBackoffDuration * 2)

	// Phase 3: Validate all written traces
	failures := 0

	for i, trace := range writtenTraces {
		cycle := i + 1
		logger.Info("Validating trace retrieval", zap.Int("cycle", cycle), zap.String("traceID", trace.id))

		start := trace.timestamp.Add(-10 * time.Minute).Unix()
		end := trace.timestamp.Add(10 * time.Minute).Unix()

		retrievedTrace, err := httpClient.QueryTraceWithRange(trace.id, start, end)
		if err != nil {
			logger.Error("Failed to read trace", zap.Int("cycle", cycle), zap.Error(err))
			failures++
			continue
		}

		if len(retrievedTrace.ResourceSpans) == 0 {
			logger.Error("Retrieved trace has no spans", zap.Int("cycle", cycle))
			failures++
			continue
		}

		logger.Info("Validated trace", zap.Int("cycle", cycle), zap.Int("resourceSpans", len(retrievedTrace.ResourceSpans)))
	}

	// Phase 4: Search validation - find each written trace
	if config.tempoSearchBackoffDuration > 0 {
		logger.Info("Waiting for search indexing to complete...")
		time.Sleep(config.tempoSearchBackoffDuration) // Wait the full search backoff (60s by default)

		logger.Info("writtenTraces", zap.Int("count", len(writtenTraces)))
		logger.Info("Testing search functionality - looking for each written trace...")
		searchFailures := 0

		for i, traceInfo := range writtenTraces {
			cycle := i + 1

			// Create a fresh TraceInfo to get the expected attributes (like original code does)
			info := util.NewTraceInfoWithMaxLongWrites(traceInfo.timestamp, 0, config.tempoOrgID)
			expected, err := info.ConstructTraceFromEpoch()
			if err != nil {
				logger.Error("Failed to construct expected trace for search", zap.Int("cycle", cycle), zap.Error(err))
				searchFailures++
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
				searchFailures++
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
				searchFailures++
			}
		}

		if searchFailures > 0 {
			logger.Error("Search validation failed", zap.Int("search_failures", searchFailures))
			failures += searchFailures
		} else {
			logger.Info("Search validation PASSED - all traces found via search")
		}
	}

	// Phase 5: Evaluate results
	logger.Info("Validation summary",
		zap.Int("total_traces", validationCycles),
		zap.Int("failures", failures),
	)

	if failures > 0 {
		logger.Error("Validation FAILED", zap.Int("failed_operations", failures))
		return 1
	}

	logger.Info("Validation PASSED - All traces written and retrieved")
	return 0
}
