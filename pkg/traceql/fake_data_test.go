package traceql

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAccuracy = 100 * time.Millisecond

func TestPerformanceTestingHints_Search(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		engine := NewEngine()

		req := &tempopb.SearchRequest{
			Query: `{} with (debug_return_in=100s)`,
			Start: uint32(time.Now().Add(-1 * time.Hour).Unix()),
			End:   uint32(time.Now().Unix()),
		}

		start := time.Now()
		resp, err := engine.ExecuteSearch(context.Background(), req, nil, true)
		elapsed := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotEmpty(t, resp.Traces)
		assert.InDelta(t, elapsed, 100*time.Second, float64(testAccuracy))
	})
}

func TestPerformanceTestingHints_SearchWithStdDev(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		engine := NewEngine()

		req := &tempopb.SearchRequest{
			Query: `{} with (debug_return_in=100s, debug_std_dev=10s)`,
			Start: uint32(time.Now().Add(-1 * time.Hour).Unix()),
			End:   uint32(time.Now().Unix()),
		}

		start := time.Now()
		resp, err := engine.ExecuteSearch(context.Background(), req, nil, true)
		elapsed := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotEmpty(t, resp.Traces)
		assert.InDelta(t, elapsed, 100*time.Second, float64(10*time.Second+testAccuracy))
	})
}

func TestGenerateFakeSearchResponse(t *testing.T) {
	resp := GenerateFakeSearchResponse()

	require.NotNil(t, resp)
	require.NotEmpty(t, resp.Traces)
	require.NotNil(t, resp.Metrics)

	// Verify trace IDs and spansets are generated
	for _, trace := range resp.Traces {
		require.NotEmpty(t, trace.TraceID)
		require.NotEmpty(t, trace.RootServiceName)

		// Verify spansets are present
		require.NotEmpty(t, trace.SpanSets, "SpanSets should not be empty")

		// Verify each spanset has spans
		for _, spanSet := range trace.SpanSets {
			require.NotEmpty(t, spanSet.Spans, "SpanSet should have spans")
			require.Greater(t, spanSet.Matched, uint32(0), "Matched count should be > 0")

			// Verify spans have required fields
			for _, span := range spanSet.Spans {
				require.NotEmpty(t, span.SpanID, "Span should have ID")
				require.NotEmpty(t, span.Name, "Span should have name")
				require.NotEmpty(t, span.Attributes, "Span should have attributes")
			}
		}
	}
}

func TestSimulateLatency(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		SimulateLatency(100*time.Second, 0)
		elapsed := time.Since(start)

		assert.InDelta(t, elapsed, 100*time.Second, float64(testAccuracy))
	})

	synctest.Test(t, func(t *testing.T) {
		// Test with std dev
		start := time.Now()
		SimulateLatency(100*time.Second, 10*time.Second)
		elapsed := time.Since(start)

		// With std dev, should be around 100ms but can vary
		assert.InDelta(t, elapsed, 100*time.Second, float64(10*time.Second+testAccuracy))
	})
}
