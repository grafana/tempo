package livestore

import (
	"context"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestInMemoryKafkaClient_Integration(t *testing.T) {
	// Create an in-memory Kafka client
	client := NewInMemoryKafkaClient()
	defer client.Close()

	// Test basic functionality
	ctx := context.Background()
	
	// Test ping
	err := client.Ping(ctx)
	require.NoError(t, err)

	// Add some test messages
	topic := "test-topic"
	partition := int32(0)
	
	client.AddMessage(topic, partition, []byte("tenant1"), []byte("trace-data-1"))
	client.AddMessage(topic, partition, []byte("tenant2"), []byte("trace-data-2"))
	client.AddMessage(topic, partition, []byte("tenant1"), []byte("trace-data-3"))

	// Test partition management
	partitions := map[string]map[int32]kgo.Offset{
		topic: {
			partition: kgo.NewOffset().AtStart(),
		},
	}
	client.AddConsumePartitions(partitions)

	// Test offset commit/fetch
	group := "test-consumer-group"
	
	// Initially no offsets should be committed
	offsets, err := client.FetchOffsets(ctx, group)
	require.NoError(t, err)
	assert.Empty(t, offsets)

	// Commit an offset
	offsetsToCommit := make(kadm.Offsets)
	offsetsToCommit.Add(kadm.Offset{
		Topic:     topic,
		Partition: partition,
		At:        2, // Committed up to offset 2
	})

	_, err = client.CommitOffsets(ctx, group, offsetsToCommit)
	require.NoError(t, err)

	// Verify offset was committed
	fetchedOffsets, err := client.FetchOffsets(ctx, group)
	require.NoError(t, err)
	assert.Len(t, fetchedOffsets, 1)

	// Check the committed offset
	found := false
	fetchedOffsets.Each(func(or kadm.OffsetResponse) {
		if or.Offset.Topic == topic && or.Offset.Partition == partition {
			assert.Equal(t, int64(2), or.Offset.At)
			assert.NoError(t, or.Err)
			found = true
		}
	})
	assert.True(t, found)

	// Clean up
	removePartitions := map[string][]int32{
		topic: {partition},
	}
	client.RemoveConsumePartitions(removePartitions)
}

func TestPartitionReader_WithInMemoryClient(t *testing.T) {
	// Create an in-memory Kafka client
	client := NewInMemoryKafkaClient()
	defer client.Close()

	// Add some test data
	topic := "test-topic"
	partition := int32(0)
	client.AddMessage(topic, partition, []byte("tenant1"), []byte("trace-data-1"))
	client.AddMessage(topic, partition, []byte("tenant2"), []byte("trace-data-2"))

	// Create a consume function that tracks consumed records
	var consumedRecords []record
	consumeFn := func(ctx context.Context, records []record) error {
		consumedRecords = append(consumedRecords, records...)
		return nil
	}

	// Create Kafka config
	cfg := ingest.KafkaConfig{
		Topic:         topic,
		ConsumerGroup: "test-group",
	}

	// Create partition reader with in-memory client
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	
	reader, err := NewPartitionReaderForPusher(client, partition, cfg, consumeFn, logger, reg)
	require.NoError(t, err)
	require.NotNil(t, reader)

	// The partition reader should be able to work with the in-memory client
	// Note: In a real test, you would start the reader service and verify message consumption
	// For now, we just verify the reader was created successfully with our interface
	assert.Equal(t, partition, reader.partitionID)
	assert.Equal(t, topic, reader.topic)
	assert.Equal(t, cfg.ConsumerGroup, reader.consumerGroup)
}

func TestInMemoryKafkaClientFactory_Integration(t *testing.T) {
	// Test the factory function
	cfg := ingest.KafkaConfig{
		Topic:         "test-topic",
		ConsumerGroup: "test-group",
	}

	client, err := InMemoryKafkaClientFactory(cfg, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, client)

	// Verify it implements the KafkaClient interface
	var _ KafkaClient = client

	// Test that it works like a regular client
	ctx := context.Background()
	err = client.Ping(ctx)
	assert.NoError(t, err)

	// Test admin operations
	group := "test-group"
	offsets, err := client.FetchOffsets(ctx, group)
	require.NoError(t, err)
	assert.Empty(t, offsets) // Should be empty initially

	client.Close()
}

func TestInMemoryKafkaClient_MessageOrdering(t *testing.T) {
	client := NewInMemoryKafkaClient()
	defer client.Close()

	topic := "test-topic"
	partition := int32(0)

	// Add messages in order
	messages := []struct {
		key   string
		value string
	}{
		{"key1", "value1"},
		{"key2", "value2"},
		{"key3", "value3"},
	}

	for _, msg := range messages {
		client.AddMessage(topic, partition, []byte(msg.key), []byte(msg.value))
	}

	// Verify messages are stored in order with correct offsets
	client.mu.RLock()
	topicData := client.topics[topic]
	client.mu.RUnlock()

	topicData.mu.RLock()
	partitionData := topicData.partitions[partition]
	topicData.mu.RUnlock()

	partitionData.mu.RLock()
	assert.Len(t, partitionData.messages, 3)
	
	for i, expectedMsg := range messages {
		actualMsg := partitionData.messages[i]
		assert.Equal(t, []byte(expectedMsg.key), actualMsg.Key)
		assert.Equal(t, []byte(expectedMsg.value), actualMsg.Value)
		assert.Equal(t, int64(i), actualMsg.Offset)
		assert.WithinDuration(t, time.Now(), actualMsg.Timestamp, time.Second)
	}
	
	assert.Equal(t, int64(3), partitionData.nextOffset)
	partitionData.mu.RUnlock()
}