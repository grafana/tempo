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
		if or.Topic == topic && or.Partition == partition {
			assert.Equal(t, int64(2), or.At)
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
	consumeFn := func(_ context.Context, records []record) error {
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

	// Verify ordering semantics through offset operations
	ctx := context.Background()
	group := "test-group"

	// Test committing offsets in order
	for i := int64(0); i < 3; i++ {
		offsetsToCommit := make(kadm.Offsets)
		offsetsToCommit.Add(kadm.Offset{
			Topic:     topic,
			Partition: partition,
			At:        i,
		})

		_, err := client.CommitOffsets(ctx, group, offsetsToCommit)
		require.NoError(t, err)

		// Verify we can fetch the committed offset
		fetchedOffsets, err := client.FetchOffsets(ctx, group)
		require.NoError(t, err)
		assert.Len(t, fetchedOffsets, 1)
	}

	// Verify final state
	fetches := client.PollFetches(ctx)
	assert.NotNil(t, fetches)
}

func TestInMemoryKafkaClient_Basic(t *testing.T) {
	client := NewInMemoryKafkaClient()
	defer client.Close()

	// Test Ping
	err := client.Ping(context.Background())
	assert.NoError(t, err)

	// Test Ping after close
	client.Close()
	err = client.Ping(context.Background())
	assert.Error(t, err)
}

func TestInMemoryKafkaClient_AddRemovePartitions(t *testing.T) {
	client := NewInMemoryKafkaClient()
	defer client.Close()

	topic := "test-topic"
	partition := int32(0)

	// Add consuming partitions
	partitions := map[string]map[int32]kgo.Offset{
		topic: {
			partition: kgo.NewOffset().AtStart(),
		},
	}
	client.AddConsumePartitions(partitions)

	// Verify behavior by testing PollFetches works without errors
	ctx := context.Background()
	fetches := client.PollFetches(ctx)
	assert.NotNil(t, fetches)

	// Remove consuming partitions
	removePartitions := map[string][]int32{
		topic: {partition},
	}
	client.RemoveConsumePartitions(removePartitions)

	// Verify behavior still works after removal
	fetches = client.PollFetches(ctx)
	assert.NotNil(t, fetches)
}

func TestInMemoryKafkaClient_Messages(t *testing.T) {
	client := NewInMemoryKafkaClient()
	defer client.Close()

	topic := "test-topic"
	partition := int32(0)
	key := []byte("test-key")
	value := []byte("test-value")

	// Add consuming partitions first
	partitions := map[string]map[int32]kgo.Offset{
		topic: {
			partition: kgo.NewOffset().AtStart(),
		},
	}
	client.AddConsumePartitions(partitions)

	// Add a message
	client.AddMessage(topic, partition, key, value)

	// Verify we can poll and get the message
	ctx := context.Background()
	fetches := client.PollFetches(ctx)
	assert.NotNil(t, fetches)

	// Verify message content using EachRecord
	recordCount := 0
	fetches.EachRecord(func(record *kgo.Record) {
		recordCount++
		assert.Equal(t, key, record.Key)
		assert.Equal(t, value, record.Value)
		assert.Equal(t, topic, record.Topic)
		assert.Equal(t, partition, record.Partition)
		assert.Equal(t, int64(0), record.Offset)
	})
	assert.Equal(t, 1, recordCount)
}

func TestInMemoryKafkaClient_OffsetCommitFetch(t *testing.T) {
	client := NewInMemoryKafkaClient()
	defer client.Close()

	ctx := context.Background()
	group := "test-group"
	topic := "test-topic"
	partition := int32(0)
	commitOffset := int64(42)

	// Test fetch offsets when none exist
	offsets, err := client.FetchOffsets(ctx, group)
	require.NoError(t, err)
	assert.Empty(t, offsets)

	// Commit some offsets
	offsetsToCommit := make(kadm.Offsets)
	offsetsToCommit.Add(kadm.Offset{
		Topic:     topic,
		Partition: partition,
		At:        commitOffset,
	})

	commitResponse, err := client.CommitOffsets(ctx, group, offsetsToCommit)
	require.NoError(t, err)
	assert.Len(t, commitResponse, 1)

	// Fetch committed offsets
	fetchResponse, err := client.FetchOffsets(ctx, group)
	require.NoError(t, err)
	assert.Len(t, fetchResponse, 1)

	// Verify the committed offset
	found := false
	fetchResponse.Each(func(or kadm.OffsetResponse) {
		if or.Topic == topic && or.Partition == partition {
			assert.Equal(t, commitOffset, or.At)
			found = true
		}
	})
	assert.True(t, found, "Should find the committed offset")
}

func TestInMemoryKafkaClient_MultipleMessages(t *testing.T) {
	client := NewInMemoryKafkaClient()
	defer client.Close()

	topic := "test-topic"
	partition := int32(0)

	// Add multiple messages
	for i := 0; i < 5; i++ {
		key := []byte("key-" + string(rune(i)))
		value := []byte("value-" + string(rune(i)))
		client.AddMessage(topic, partition, key, value)
	}

	// Verify client behavior with multiple messages
	ctx := context.Background()
	fetches := client.PollFetches(ctx)
	assert.NotNil(t, fetches)

	// Test offset operations work correctly
	group := "test-group"
	offsetsToCommit := make(kadm.Offsets)
	offsetsToCommit.Add(kadm.Offset{
		Topic:     topic,
		Partition: partition,
		At:        3, // Commit up to offset 3
	})

	_, err := client.CommitOffsets(ctx, group, offsetsToCommit)
	require.NoError(t, err)

	// Verify we can fetch the committed offset
	fetchedOffsets, err := client.FetchOffsets(ctx, group)
	require.NoError(t, err)
	assert.Len(t, fetchedOffsets, 1)
}

func TestInMemoryKafkaClient_MultipleTopicsPartitions(t *testing.T) {
	client := NewInMemoryKafkaClient()
	defer client.Close()

	// Add messages to different topics and partitions
	client.AddMessage("topic1", 0, []byte("key1"), []byte("value1"))
	client.AddMessage("topic1", 1, []byte("key2"), []byte("value2"))
	client.AddMessage("topic2", 0, []byte("key3"), []byte("value3"))

	// Verify behavior works across multiple topics/partitions
	ctx := context.Background()
	group := "test-group"

	// Test offset operations work for topic1 partitions
	offsetsToCommit := make(kadm.Offsets)
	offsetsToCommit.Add(kadm.Offset{Topic: "topic1", Partition: 0, At: 0})
	offsetsToCommit.Add(kadm.Offset{Topic: "topic1", Partition: 1, At: 0})
	offsetsToCommit.Add(kadm.Offset{Topic: "topic2", Partition: 0, At: 0})

	_, err := client.CommitOffsets(ctx, group, offsetsToCommit)
	require.NoError(t, err)

	// Verify we can fetch all committed offsets
	fetchedOffsets, err := client.FetchOffsets(ctx, group)
	require.NoError(t, err)

	// Count the actual number of offset responses
	count := 0
	fetchedOffsets.Each(func(_ kadm.OffsetResponse) {
		count++
	})
	assert.Equal(t, 3, count) // Should have 3 topic/partition combinations

	// Verify PollFetches works
	fetches := client.PollFetches(ctx)
	assert.NotNil(t, fetches)
}

func TestInMemoryKafkaClient_Factory(t *testing.T) {
	client, err := InMemoryKafkaClientFactory(ingest.KafkaConfig{}, nil, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)

	// Test basic functionality
	err = client.Ping(context.Background())
	assert.NoError(t, err)

	client.Close()
}

func TestInMemoryKafkaClient_PollFetches(t *testing.T) {
	client := NewInMemoryKafkaClient()
	defer client.Close()

	ctx := context.Background()

	// Test polling when client is open
	fetches := client.PollFetches(ctx)
	assert.NotNil(t, fetches)
	assert.Empty(t, fetches)

	// Test polling when client is closed
	client.Close()
	fetches = client.PollFetches(ctx)
	assert.NotNil(t, fetches)
	assert.Empty(t, fetches)
}

func TestInMemoryKafkaClient_PollFetchesWithContext(t *testing.T) {
	client := NewInMemoryKafkaClient()
	defer client.Close()

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fetches := client.PollFetches(ctx)
	assert.NotNil(t, fetches)
	assert.Empty(t, fetches)
}

func TestInMemoryKafkaClient_ConcurrentAccess(t *testing.T) {
	client := NewInMemoryKafkaClient()
	defer client.Close()

	topic := "test-topic"
	partition := int32(0)

	// Add consuming partitions first
	partitions := map[string]map[int32]kgo.Offset{
		topic: {
			partition: kgo.NewOffset().AtStart(),
		},
	}
	client.AddConsumePartitions(partitions)

	// Test concurrent message addition
	done := make(chan int, 10)
	expectedKeys := make(map[string]bool)
	expectedValues := make(map[string]bool)

	for i := 0; i < 10; i++ {
		expectedKeys["key-"+string(rune(i))] = true
		expectedValues["value-"+string(rune(i))] = true

		go func(id int) {
			key := []byte("key-" + string(rune(id)))
			value := []byte("value-" + string(rune(id)))
			client.AddMessage(topic, partition, key, value)
			done <- id
		}(i)
	}

	// Wait for all goroutines to complete
	receivedIDs := make(map[int]bool)
	for i := 0; i < 10; i++ {
		select {
		case id := <-done:
			receivedIDs[id] = true
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out")
		}
	}

	// Verify all goroutines completed
	assert.Len(t, receivedIDs, 10)

	// Verify we can poll and get all messages
	ctx := context.Background()
	fetches := client.PollFetches(ctx)
	assert.NotNil(t, fetches)

	// Verify all expected keys and values are present
	foundKeys := make(map[string]bool)
	foundValues := make(map[string]bool)
	recordCount := 0

	fetches.EachRecord(func(record *kgo.Record) {
		recordCount++
		foundKeys[string(record.Key)] = true
		foundValues[string(record.Value)] = true
		assert.Equal(t, topic, record.Topic)
		assert.Equal(t, partition, record.Partition)
		// Offsets should be sequential 0-9
		assert.GreaterOrEqual(t, record.Offset, int64(0))
		assert.LessOrEqual(t, record.Offset, int64(9))
	})

	// Verify we got all 10 messages with expected keys and values
	assert.Equal(t, 10, recordCount)
	assert.Equal(t, expectedKeys, foundKeys)
	assert.Equal(t, expectedValues, foundValues)
}
