package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/modules/frontend"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func TestTraceLookupAPI(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOneLocal, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	// Set up Jaeger client
	jaegerClient, err := util.NewJaegerToOTLPExporter(tempo.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	// Create and export traces
	traceIDs := []string{}

	for i := 0; i < 3; i++ {
		batch := util.MakeThriftBatchWithSpanCount(2)
		err = jaegerClient.EmitBatch(context.Background(), batch)
		require.NoError(t, err)
		
		// Extract trace ID from the batch
		traceID := fmt.Sprintf("%016x%016x", uint64(batch.Spans[0].TraceIdHigh), uint64(batch.Spans[0].TraceIdLow))
		traceIDs = append(traceIDs, traceID)
	}

	// Add some non-existent trace IDs for testing
	nonExistentTraceIDs := []string{
		"1234567890abcdef1234567890abcdef",
		"fedcba0987654321fedcba0987654321",
	}
	allTraceIDs := append(traceIDs, nonExistentTraceIDs...)

	// Wait for traces to be ingested
	time.Sleep(time.Second * 2)

	t.Run("TraceLookupViaFrontend", func(t *testing.T) {
		testTraceLookupEndpoint(t, tempo, "/api/trace-lookup", allTraceIDs, traceIDs, nonExistentTraceIDs)
	})

	t.Run("TraceLookupViaQuerier", func(t *testing.T) {
		testTraceLookupEndpoint(t, tempo, "/querier/api/trace-lookup", allTraceIDs, traceIDs, nonExistentTraceIDs)
	})

	// Wait for blocks to be flushed to backend storage
	time.Sleep(blockFlushTimeout)

	t.Run("TraceLookupFromStorage", func(t *testing.T) {
		testTraceLookupEndpoint(t, tempo, "/api/trace-lookup", allTraceIDs, traceIDs, nonExistentTraceIDs)
	})
}

func testTraceLookupEndpoint(t *testing.T, tempo *e2e.HTTPService, endpoint string, allTraceIDs, existingTraceIDs, nonExistentTraceIDs []string) {
	// Test the trace lookup endpoint
	reqBody := frontend.TraceLookupRequest{
		IDs: allTraceIDs,
	}

	jsonBytes, err := json.Marshal(reqBody)
	require.NoError(t, err)

	url := fmt.Sprintf("http://%s%s", tempo.Endpoint(tempoPort), endpoint)
	resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBytes))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 status code")

	var lookupResp tempopb.TraceLookupResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&lookupResp))

	// Verify all trace IDs are present in the response
	require.Len(t, lookupResp.TraceIDs, len(allTraceIDs), "Response should contain all requested trace IDs")

	// Verify existing traces are marked as found
	for _, traceID := range existingTraceIDs {
		require.Contains(t, lookupResp.TraceIDs, traceID, "Response should contain trace ID %s", traceID)
	}

	// Verify non-existent traces are marked as not found
	for _, traceID := range nonExistentTraceIDs {
		require.NotContains(t, lookupResp.TraceIDs, traceID, "Response should not contain trace ID %s", traceID)
	}

	// Verify metrics are present
	require.NotNil(t, lookupResp.Metrics, "Response should contain metrics")
	require.Greater(t, lookupResp.Metrics.CompletedJobs, uint32(0), "Should have completed at least one job")
}

func TestTraceLookupAPIErrors(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOneLocal, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	url := fmt.Sprintf("http://%s/api/trace-lookup", tempo.Endpoint(tempoPort))

	t.Run("InvalidJSON", func(t *testing.T) {
		resp, err := http.Post(url, "application/json", bytes.NewReader([]byte("{invalid json")))
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("EmptyTraceIDs", func(t *testing.T) {
		reqBody := frontend.TraceLookupRequest{IDs: []string{}}
		jsonBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBytes))
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("InvalidTraceID", func(t *testing.T) {
		reqBody := frontend.TraceLookupRequest{IDs: []string{"invalid-trace-id"}}
		jsonBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBytes))
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("WrongHTTPMethod", func(t *testing.T) {
		resp, err := http.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func TestTraceLookupAPIPerformance(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOneLocal, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	// Test with a large number of trace IDs
	largeTraceIDList := make([]string, 100)
	for i := 0; i < 100; i++ {
		largeTraceIDList[i] = fmt.Sprintf("%032d", i) // 32-character hex string
	}

	reqBody := frontend.TraceLookupRequest{
		IDs: largeTraceIDList,
	}

	jsonBytes, err := json.Marshal(reqBody)
	require.NoError(t, err)

	url := fmt.Sprintf("http://%s/api/trace-lookup", tempo.Endpoint(tempoPort))
	
	start := time.Now()
	resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBytes))
	duration := time.Since(start)
	
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var lookupResp tempopb.TraceLookupResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&lookupResp))

	// Verify all trace IDs are in the response
	require.Len(t, lookupResp.TraceIDs, 100)

	// Performance check - should complete within reasonable time
	require.Less(t, duration, 10*time.Second, "TraceLookup should complete within 10 seconds for 100 trace IDs")

	t.Logf("TraceLookup for 100 trace IDs completed in %v", duration)
}

func TestTraceLookupAPIContentTypes(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOneLocal, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	reqBody := frontend.TraceLookupRequest{
		IDs: []string{"1234567890abcdef1234567890abcdef"},
	}
	jsonBytes, err := json.Marshal(reqBody)
	require.NoError(t, err)

	url := fmt.Sprintf("http://%s/api/trace-lookup", tempo.Endpoint(tempoPort))

	t.Run("JSONContentType", func(t *testing.T) {
		resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBytes))
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	})
}