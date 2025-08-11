package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

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
