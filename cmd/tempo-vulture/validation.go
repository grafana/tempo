package main

import (
	"context"
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

		logger.Info("Successfully wrote trace", zap.Int("cycle", i+1))
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

		logger.Info("Successfully validated trace", zap.Int("cycle", cycle), zap.Int("resourceSpans", len(retrievedTrace.ResourceSpans)))
	}

	// Phase 4: Basic search validation (optional)
	if config.tempoSearchBackoffDuration > 0 {
		logger.Info("Testing search functionality...")

		// Use the first trace's timestamp for search range
		firstTrace := writtenTraces[0]
		start := firstTrace.timestamp.Add(-10 * time.Minute).Unix()
		end := firstTrace.timestamp.Add(10 * time.Minute).Unix()

		_, err := httpClient.SearchWithRange("vulture-0=*", start, end)
		if err != nil {
			logger.Error("Search API not responding", zap.Error(err))
			failures++
		} else {
			logger.Info("Search API responding successfully")
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

	logger.Info("Validation PASSED - All traces written and retrieved successfully!")
	return 0
}
