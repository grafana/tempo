package livestore

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/kv/consul"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kprom"
)

func TestLiveStore_TraceProcessingToBlocks(t *testing.T) {
	// Setup
	topic := "test-topic"
	tenantID := "batch-tenant"
	traceCount := 5

	liveStore, kafkaClient := setupLiveStoreForTest(t, "test-ingester-2", topic, "test-consumer-group")
	defer kafkaClient.Close()

	// Write test data
	expectedTraceIDs := writeTestTraces(t, kafkaClient.(*InMemoryKafkaClient), topic, tenantID, traceCount)

	// Start the service
	ctx := context.Background()
	err := liveStore.StartAsync(ctx)
	require.NoError(t, err)

	err = liveStore.AwaitRunning(ctx)
	require.NoError(t, err)

	// Wait for processing to complete
	assert.Eventually(t, func() bool {
		liveStore.instancesMtx.RLock()
		instance, exists := liveStore.instances[tenantID]
		liveStore.instancesMtx.RUnlock()

		if exists && instance != nil {
			// Check if we have traces in live traces
			instance.liveTracesMtx.Lock()
			liveTraceCount := instance.liveTraces.Len()
			instance.liveTracesMtx.Unlock()

			t.Logf("Found instance with %d live traces (expected: %d)", liveTraceCount, traceCount)
			return int(liveTraceCount) >= traceCount
		}
		t.Logf("No instance found yet")
		return false
	}, 5*time.Second, 100*time.Millisecond, "Expected all traces to be processed into live traces")

	// Get final instance for further testing
	liveStore.instancesMtx.RLock()
	instance, exists := liveStore.instances[tenantID]
	liveStore.instancesMtx.RUnlock()

	require.True(t, exists, "Instance should exist")
	require.NotNil(t, instance, "Instance should not be nil")

	// Verify live traces content before cutting to blocks
	instance.liveTracesMtx.Lock()
	traceMap := make(map[string]bool)
	expectedTraceIDsMap := make(map[string]bool)

	// Convert expected trace IDs slice to map for verification
	for _, traceID := range expectedTraceIDs {
		expectedTraceIDsMap[traceID] = true
	}

	// Verify live traces contain the expected trace IDs
	for _, liveTrace := range instance.liveTraces.Traces {
		traceIDStr := string(liveTrace.ID)
		traceMap[traceIDStr] = true
		assert.True(t, expectedTraceIDsMap[traceIDStr], "Unexpected trace ID found: %s", traceIDStr)

		// Verify each live trace has batches
		assert.Greater(t, len(liveTrace.Batches), 0, "Live trace should have batches")

		// Verify batch content
		for _, batch := range liveTrace.Batches {
			assert.NotNil(t, batch, "Batch should not be nil")
			assert.Greater(t, len(batch.ScopeSpans), 0, "Batch should have scope spans")
		}
	}
	instance.liveTracesMtx.Unlock()

	// Verify we found all expected traces
	assert.Equal(t, len(expectedTraceIDsMap), len(traceMap), "Should have found all expected trace IDs")
	t.Logf("Verified %d live traces with correct content", len(traceMap))

	// Force cut traces to blocks (simulate idle timeout)
	err = instance.cutIdleTraces(true) // immediate = true
	require.NoError(t, err)

	// Check that head block has data
	instance.blocksMtx.Lock()
	headBlock := instance.headBlock
	instance.blocksMtx.Unlock()

	assert.True(t, headBlock != nil, "Expected head block to be present after cutting traces")

	// Verify block contains the expected trace data using iterator
	blockTraceIDs := make(map[string]bool)
	iter, err := headBlock.Iterator()
	require.NoError(t, err, "Should be able to create block iterator")

	for {
		traceID, trace, err := iter.Next(ctx)
		if err != nil {
			break
		}
		if traceID == nil {
			break
		}

		blockTraceIDs[string(traceID)] = true
		assert.NotNil(t, trace, "Trace from block should not be nil")

		// Verify trace structure
		assert.Greater(t, len(trace.ResourceSpans), 0, "Trace should have resource spans")
		t.Logf("Found trace in block: %s", string(traceID))
	}

	// Verify all expected traces are in the block
	for _, expectedID := range expectedTraceIDs {
		assert.True(t, blockTraceIDs[expectedID], "Expected trace ID %s not found in block", expectedID)
	}

	assert.Equal(t, len(expectedTraceIDs), len(blockTraceIDs), "Block should contain all expected traces")
	t.Logf("Verified %d traces correctly written to WAL block", len(blockTraceIDs))

	// Clean up
	liveStore.StopAsync()
	err = liveStore.AwaitTerminated(ctx)
	require.NoError(t, err)
}

// createConsulInMemoryClient creates a consul in-memory client for testing
func createConsulInMemoryClient() kv.Client {
	client, _ := consul.NewInMemoryClient(
		ring.GetPartitionRingCodec(),
		log.NewNopLogger(),
		nil,
	)
	return client
}

// createTestTrace creates a test trace with multiple spans
func createTestTrace() *tempopb.Trace {
	traceID := test.ValidTraceID(nil)
	return test.MakeTrace(10, traceID)
}

// encodeTraceRecord encodes a trace as it would appear in Kafka
func encodeTraceRecord(_ string, pushReq *tempopb.PushBytesRequest) ([]byte, error) {
	return pushReq.Marshal()
}

// setupLiveStoreForTest creates and configures a LiveStore with in-memory Kafka for testing
func setupLiveStoreForTest(t *testing.T, ingesterID, topic, consumerGroup string) (*LiveStore, KafkaClient) {
	// Create temporary directory for WAL
	tmpDir := t.TempDir()

	// Create in-memory kafka client
	kafkaClient := NewInMemoryKafkaClient()

	// Create livestore config
	cfg := Config{
		LifecyclerConfig: ring.LifecyclerConfig{
			ID: ingesterID,
		},
		PartitionRing: ingester.PartitionRingConfig{
			KVStore: kv.Config{
				Mock: createConsulInMemoryClient(),
			},
		},
		IngestConfig: ingest.Config{
			Kafka: ingest.KafkaConfig{
				Topic:         topic,
				ConsumerGroup: consumerGroup,
			},
		},
		WAL: wal.Config{
			Filepath: tmpDir,
			Version:  "vParquet4", // Use default encoding version
		},
	}

	overrides := &mockOverrides{}
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()

	clientFactory := func(_ ingest.KafkaConfig, _ *kprom.Metrics, _ log.Logger) (KafkaClient, error) {
		return kafkaClient, nil
	}

	liveStore, err := New(cfg, overrides, logger, reg, true, clientFactory)
	require.NoError(t, err)

	// Add consuming partitions
	partitions := map[string]map[int32]kgo.Offset{
		topic: {
			0: kgo.NewOffset().AtStart(),
		},
	}
	kafkaClient.AddConsumePartitions(partitions)

	return liveStore, kafkaClient
}

// writeTestTraces creates and writes test trace data to Kafka
func writeTestTraces(t *testing.T, kafkaClient *InMemoryKafkaClient, topic, tenantID string, traceCount int) []string {
	expectedTraceIDs := make([]string, traceCount)

	// Add multiple trace messages
	for i := range traceCount {
		trace := createTestTrace()
		// Make each trace unique
		trace.ResourceSpans[0].ScopeSpans[0].Spans[0].TraceId = []byte(fmt.Sprintf("trace-%d", i))

		// Create unique trace ID for the PushBytesRequest
		traceBytes, err := trace.Marshal()
		require.NoError(t, err)

		traceID := fmt.Sprintf("test-trace-id-%d", i)
		expectedTraceIDs[i] = traceID

		pushReq := &tempopb.PushBytesRequest{
			Traces: []tempopb.PreallocBytes{
				{
					Slice: traceBytes,
				},
			},
			Ids: [][]byte{[]byte(traceID)},
		}

		recordBytes, err := encodeTraceRecord(tenantID, pushReq)
		require.NoError(t, err)

		kafkaClient.AddMessage(
			topic,
			0,
			[]byte(tenantID),
			recordBytes,
		)
		t.Logf("Added message %d to Kafka", i)
	}

	return expectedTraceIDs
}
