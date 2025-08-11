package livestore

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/ingest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

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

	// Verify partitions were added
	client.mu.RLock()
	assert.Contains(t, client.consumingPartitions, topic)
	assert.Contains(t, client.consumingPartitions[topic], partition)
	client.mu.RUnlock()

	// Remove consuming partitions
	removePartitions := map[string][]int32{
		topic: {partition},
	}
	client.RemoveConsumePartitions(removePartitions)

	// Verify partitions were removed
	client.mu.RLock()
	assert.NotContains(t, client.consumingPartitions, topic)
	client.mu.RUnlock()
}

func TestInMemoryKafkaClient_Messages(t *testing.T) {
	client := NewInMemoryKafkaClient()
	defer client.Close()

	topic := "test-topic"
	partition := int32(0)
	key := []byte("test-key")
	value := []byte("test-value")

	// Add a message
	client.AddMessage(topic, partition, key, value)

	// Verify message was added
	client.mu.RLock()
	assert.Contains(t, client.topics, topic)
	topicData := client.topics[topic]
	client.mu.RUnlock()

	topicData.mu.RLock()
	assert.Contains(t, topicData.partitions, partition)
	partitionData := topicData.partitions[partition]
	topicData.mu.RUnlock()

	partitionData.mu.RLock()
	assert.Len(t, partitionData.messages, 1)
	msg := partitionData.messages[0]
	assert.Equal(t, key, msg.Key)
	assert.Equal(t, value, msg.Value)
	assert.Equal(t, int64(0), msg.Offset)
	partitionData.mu.RUnlock()
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
			assert.Equal(t, commitOffset, or.Offset.At)
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

	// Verify all messages were added with correct offsets
	client.mu.RLock()
	topicData := client.topics[topic]
	client.mu.RUnlock()

	topicData.mu.RLock()
	partitionData := topicData.partitions[partition]
	topicData.mu.RUnlock()

	partitionData.mu.RLock()
	assert.Len(t, partitionData.messages, 5)
	for i, msg := range partitionData.messages {
		assert.Equal(t, int64(i), msg.Offset)
		assert.Equal(t, []byte("key-"+string(rune(i))), msg.Key)
		assert.Equal(t, []byte("value-"+string(rune(i))), msg.Value)
	}
	assert.Equal(t, int64(5), partitionData.nextOffset)
	partitionData.mu.RUnlock()
}

func TestInMemoryKafkaClient_MultipleTopicsPartitions(t *testing.T) {
	client := NewInMemoryKafkaClient()
	defer client.Close()

	// Add messages to different topics and partitions
	client.AddMessage("topic1", 0, []byte("key1"), []byte("value1"))
	client.AddMessage("topic1", 1, []byte("key2"), []byte("value2"))
	client.AddMessage("topic2", 0, []byte("key3"), []byte("value3"))

	// Verify structure
	client.mu.RLock()
	assert.Len(t, client.topics, 2)
	assert.Contains(t, client.topics, "topic1")
	assert.Contains(t, client.topics, "topic2")
	client.mu.RUnlock()

	// Check topic1 partitions
	topic1 := client.topics["topic1"]
	topic1.mu.RLock()
	assert.Len(t, topic1.partitions, 2)
	assert.Contains(t, topic1.partitions, int32(0))
	assert.Contains(t, topic1.partitions, int32(1))
	topic1.mu.RUnlock()

	// Check topic2 partitions
	topic2 := client.topics["topic2"]
	topic2.mu.RLock()
	assert.Len(t, topic2.partitions, 1)
	assert.Contains(t, topic2.partitions, int32(0))
	topic2.mu.RUnlock()
}

func TestInMemoryKafkaClient_Factory(t *testing.T) {
	client, err := InMemoryKafkaClientFactory(ingest.KafkaConfig{}, nil, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)

	// Verify it implements the interface
	var _ Client = client

	// Test basic functionality
	err = client.Ping(context.Background())
	assert.NoError(t, err)

	client.Close()
}

func TestInMemoryKafkaClient_ConcurrentAccess(t *testing.T) {
	client := NewInMemoryKafkaClient()
	defer client.Close()

	topic := "test-topic"
	partition := int32(0)

	// Test concurrent message addition
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := []byte("key-" + string(rune(id)))
			value := []byte("value-" + string(rune(id)))
			client.AddMessage(topic, partition, key, value)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out")
		}
	}

	// Verify all messages were added
	client.mu.RLock()
	topicData := client.topics[topic]
	client.mu.RUnlock()

	topicData.mu.RLock()
	partitionData := topicData.partitions[partition]
	topicData.mu.RUnlock()

	partitionData.mu.RLock()
	assert.Len(t, partitionData.messages, 10)
	partitionData.mu.RUnlock()
}
