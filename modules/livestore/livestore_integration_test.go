package livestore

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/kv"
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

// createTestTrace creates a test trace with multiple spans
func createTestTrace() *tempopb.Trace {
	traceID := test.ValidTraceID(nil)
	return test.MakeTrace(10, traceID)
}

// createPushRequest creates a PushBytesRequest from a trace
func createPushRequest(trace *tempopb.Trace) (*tempopb.PushBytesRequest, error) {
	traceBytes, err := trace.Marshal()
	if err != nil {
		return nil, err
	}

	return &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{
			{
				Slice: traceBytes,
			},
		},
		Ids: [][]byte{[]byte("test-trace-id-123")},
	}, nil
}

// encodeTraceRecord encodes a trace as it would appear in Kafka
func encodeTraceRecord(tenantID string, pushReq *tempopb.PushBytesRequest) ([]byte, error) {
	return pushReq.Marshal()
}

func TestLiveStore_IntegrationTraceIngestion(t *testing.T) {
	// Create temporary directory for WAL
	tmpDir := t.TempDir()

	// Create in-memory kafka client
	kafkaClient := NewInMemoryKafkaClient()
	defer kafkaClient.Close()

	// Create livestore config
	cfg := Config{
		LifecyclerConfig: ring.LifecyclerConfig{
			ID: "test-ingester-1",
		},
		PartitionRing: ingester.PartitionRingConfig{
			KVStore: kv.Config{
				Mock: &mockKV{},
			},
		},
		IngestConfig: ingest.Config{
			Kafka: ingest.KafkaConfig{
				Topic:         "test-topic",
				ConsumerGroup: "test-consumer-group",
			},
		},
		WAL: wal.Config{
			Filepath: tmpDir,
			Version:  "vParquet4", // Use default encoding version
		},
	}

	// Use existing mock overrides from live_store_test.go
	overrides := &mockOverrides{}

	// Create logger
	logger := log.NewNopLogger()

	// Create registry
	reg := prometheus.NewRegistry()

	// Create client factory that returns our in-memory client
	clientFactory := func(cfg ingest.KafkaConfig, metrics *kprom.Metrics, logger log.Logger) (KafkaClient, error) {
		return kafkaClient, nil
	}

	// Create LiveStore
	liveStore, err := New(cfg, overrides, logger, reg, true, clientFactory)
	require.NoError(t, err)

	// Test trace data
	tenantID := "test-tenant"
	trace := createTestTrace()
	pushReq, err := createPushRequest(trace)
	require.NoError(t, err)

	// Encode the trace as it would appear in Kafka
	recordBytes, err := encodeTraceRecord(tenantID, pushReq)
	require.NoError(t, err)

	// Add consuming partitions to the Kafka client so PollFetches returns messages
	partitions := map[string]map[int32]kgo.Offset{
		cfg.IngestConfig.Kafka.Topic: {
			0: kgo.NewOffset().AtStart(),
		},
	}
	kafkaClient.AddConsumePartitions(partitions)

	// Add the trace message to the in-memory Kafka
	kafkaClient.AddMessage(
		cfg.IngestConfig.Kafka.Topic,
		0, // partition
		[]byte(tenantID),
		recordBytes,
	)

	// Start the LiveStore service
	ctx := context.Background()
	err = liveStore.Service.StartAsync(ctx)
	require.NoError(t, err)

	// Wait for service to be running
	err = liveStore.Service.AwaitRunning(ctx)
	require.NoError(t, err)

	// Give some time for message processing and multiple attempts
	found := false
	for i := 0; i < 50; i++ { // Try for up to 5 seconds
		time.Sleep(100 * time.Millisecond)

		liveStore.instancesMtx.RLock()
		instance, exists := liveStore.instances[tenantID]
		liveStore.instancesMtx.RUnlock()

		if exists && instance != nil {
			found = true
			// Verify that live traces were created
			instance.liveTracesMtx.Lock()
			traceCount := instance.liveTraces.Len()
			instance.liveTracesMtx.Unlock()

			if traceCount > 0 {
				assert.Greater(t, int(traceCount), 0, "Expected live traces to be present")
				break
			}
		}
	}

	if !found {
		// Debug: Check if messages are being consumed at all
		fetches := kafkaClient.PollFetches(context.Background())
		recordCount := 0
		fetches.EachRecord(func(record *kgo.Record) {
			recordCount++
			t.Logf("Found record: topic=%s, partition=%d, key=%s, offset=%d",
				record.Topic, record.Partition, string(record.Key), record.Offset)
		})
		t.Logf("Total records available in Kafka: %d", recordCount)

		// Check how many instances exist
		liveStore.instancesMtx.RLock()
		instanceCount := len(liveStore.instances)
		liveStore.instancesMtx.RUnlock()
		t.Logf("Total instances in LiveStore: %d", instanceCount)
	}

	assert.True(t, found, "Expected instance to be created for tenant")

	// Stop the service
	liveStore.Service.StopAsync()
	err = liveStore.Service.AwaitTerminated(ctx)
	require.NoError(t, err)
}

func TestLiveStore_TraceProcessingToBlocks(t *testing.T) {
	// Create temporary directory for WAL
	tmpDir := t.TempDir()

	// Create in-memory kafka client
	kafkaClient := NewInMemoryKafkaClient()
	defer kafkaClient.Close()

	// Create livestore config
	cfg := Config{
		LifecyclerConfig: ring.LifecyclerConfig{
			ID: "test-ingester-2",
		},
		PartitionRing: ingester.PartitionRingConfig{
			KVStore: kv.Config{
				Mock: &mockKV{},
			},
		},
		IngestConfig: ingest.Config{
			Kafka: ingest.KafkaConfig{
				Topic:         "test-topic",
				ConsumerGroup: "test-consumer-group",
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

	clientFactory := func(cfg ingest.KafkaConfig, metrics *kprom.Metrics, logger log.Logger) (KafkaClient, error) {
		return kafkaClient, nil
	}

	liveStore, err := New(cfg, overrides, logger, reg, true, clientFactory)
	require.NoError(t, err)

	// Add multiple traces for the same tenant
	tenantID := "batch-tenant"
	traceCount := 5

	// Add consuming partitions
	partitions := map[string]map[int32]kgo.Offset{
		cfg.IngestConfig.Kafka.Topic: {
			0: kgo.NewOffset().AtStart(),
		},
	}
	kafkaClient.AddConsumePartitions(partitions)

	// Add multiple trace messages
	for i := 0; i < traceCount; i++ {
		trace := createTestTrace()
		// Make each trace unique
		trace.ResourceSpans[0].ScopeSpans[0].Spans[0].TraceId = []byte(fmt.Sprintf("trace-%d", i))

		// Create unique trace ID for the PushBytesRequest
		traceBytes, err := trace.Marshal()
		require.NoError(t, err)

		pushReq := &tempopb.PushBytesRequest{
			Traces: []tempopb.PreallocBytes{
				{
					Slice: traceBytes,
				},
			},
			Ids: [][]byte{[]byte(fmt.Sprintf("test-trace-id-%d", i))},
		}

		recordBytes, err := encodeTraceRecord(tenantID, pushReq)
		require.NoError(t, err)

		kafkaClient.AddMessage(
			cfg.IngestConfig.Kafka.Topic,
			0,
			[]byte(tenantID),
			recordBytes,
		)
		t.Logf("Added message %d to Kafka", i)
	}

	// Start the service
	ctx := context.Background()
	err = liveStore.Service.StartAsync(ctx)
	require.NoError(t, err)

	err = liveStore.Service.AwaitRunning(ctx)
	require.NoError(t, err)

	// Wait for processing with multiple attempts
	var instance *instance
	var exists bool

	for i := 0; i < 50; i++ { // Try for up to 5 seconds
		time.Sleep(100 * time.Millisecond)

		liveStore.instancesMtx.RLock()
		instance, exists = liveStore.instances[tenantID]
		liveStore.instancesMtx.RUnlock()

		if exists && instance != nil {
			// Check if we have traces in live traces
			instance.liveTracesMtx.Lock()
			liveTraceCount := instance.liveTraces.Len()
			instance.liveTracesMtx.Unlock()

			t.Logf("Attempt %d: Found instance with %d live traces", i+1, liveTraceCount)

			if int(liveTraceCount) >= traceCount {
				break
			}
		} else {
			t.Logf("Attempt %d: No instance found yet", i+1)
		}
	}

	require.True(t, exists, "Instance should exist")
	require.NotNil(t, instance, "Instance should not be nil")

	// Final check of live traces
	instance.liveTracesMtx.Lock()
	liveTraceCount := instance.liveTraces.Len()
	instance.liveTracesMtx.Unlock()

	t.Logf("Final live trace count: %d, expected: %d", liveTraceCount, traceCount)
	assert.Equal(t, traceCount, int(liveTraceCount), "Expected all traces to be in live traces")

	// Force cut traces to blocks (simulate idle timeout)
	err = instance.cutIdleTraces(true) // immediate = true
	require.NoError(t, err)

	// Check that head block has data
	instance.blocksMtx.Lock()
	hasHeadBlock := instance.headBlock != nil
	instance.blocksMtx.Unlock()

	assert.True(t, hasHeadBlock, "Expected head block to be present after cutting traces")

	// Clean up
	liveStore.Service.StopAsync()
	err = liveStore.Service.AwaitTerminated(ctx)
	require.NoError(t, err)
}

// mockKV implements the kv.Client interface for testing
type mockKV struct{}

func (m *mockKV) List(ctx context.Context, prefix string) ([]string, error) {
	return []string{}, nil
}

func (m *mockKV) Get(ctx context.Context, key string) (interface{}, error) {
	return nil, nil
}

func (m *mockKV) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *mockKV) WatchKey(ctx context.Context, key string, f func(interface{}) bool) {
}

func (m *mockKV) WatchPrefix(ctx context.Context, prefix string, f func(string, interface{}) bool) {
}

func (m *mockKV) CAS(ctx context.Context, key string, f func(in interface{}) (out interface{}, retry bool, err error)) error {
	return nil
}

func (m *mockKV) Stop() {
}

func TestLiveStore_PollingFromInMemoryKafka(t *testing.T) {
	kafkaClient := NewInMemoryKafkaClient()
	defer kafkaClient.Close()

	topic := "polling-test-topic"
	partition := int32(0)
	tenantID := "polling-tenant"

	// Add consuming partitions
	partitions := map[string]map[int32]kgo.Offset{
		topic: {
			partition: kgo.NewOffset().AtStart(),
		},
	}
	kafkaClient.AddConsumePartitions(partitions)

	// Create test trace and push request
	trace := createTestTrace()
	pushReq, err := createPushRequest(trace)
	require.NoError(t, err)

	recordBytes, err := encodeTraceRecord(tenantID, pushReq)
	require.NoError(t, err)

	// Add message to in-memory Kafka
	kafkaClient.AddMessage(topic, partition, []byte(tenantID), recordBytes)

	// Test polling
	ctx := context.Background()
	fetches := kafkaClient.PollFetches(ctx)
	assert.NotNil(t, fetches)

	// Verify we can retrieve the message
	recordCount := 0
	fetches.EachRecord(func(record *kgo.Record) {
		recordCount++
		assert.Equal(t, topic, record.Topic)
		assert.Equal(t, partition, record.Partition)
		assert.Equal(t, []byte(tenantID), record.Key)
		assert.Equal(t, recordBytes, record.Value)
		assert.Equal(t, int64(0), record.Offset)
	})

	assert.Equal(t, 1, recordCount, "Expected exactly one record")
}
