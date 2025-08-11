package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kprom"
)

// InMemoryMessage represents a message in the in-memory queue
// Technically there is a fake cluster that listens on a port that is used for other tests.
// But that has more weight and ties us to actually using Kafka instead of using something similar at a go api level.
type InMemoryMessage struct {
	Key       []byte
	Value     []byte
	Offset    int64
	Timestamp time.Time
}

// InMemoryPartition represents a single partition with its messages and state
type InMemoryPartition struct {
	messages   []InMemoryMessage
	nextOffset int64
}

// InMemoryTopic represents a topic with multiple partitions
type InMemoryTopic struct {
	partitions map[int32]*InMemoryPartition
}

// InMemoryKafkaClient implements Client interface using in-memory queues for testing
type InMemoryKafkaClient struct {
	topics map[string]*InMemoryTopic

	// Consumer state
	consumingPartitions map[string]map[int32]kgo.Offset

	// Committed offsets per consumer group
	committedOffsets map[string]map[string]map[int32]int64 // [group][topic][partition] = offset

	closed bool
	mu     sync.RWMutex
}

// NewInMemoryKafkaClient creates a new in-memory Kafka client for testing
func NewInMemoryKafkaClient() *InMemoryKafkaClient {
	return &InMemoryKafkaClient{
		topics:              make(map[string]*InMemoryTopic),
		consumingPartitions: make(map[string]map[int32]kgo.Offset),
		committedOffsets:    make(map[string]map[string]map[int32]int64),
	}
}

// Ping always returns nil for in-memory client
func (c *InMemoryKafkaClient) Ping(_ context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}
	return nil
}

// AddConsumePartitions registers partitions for consumption
func (c *InMemoryKafkaClient) AddConsumePartitions(partitions map[string]map[int32]kgo.Offset) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for topic, partitionOffsets := range partitions {
		if c.consumingPartitions[topic] == nil {
			c.consumingPartitions[topic] = make(map[int32]kgo.Offset)
		}
		for partition, offset := range partitionOffsets {
			c.consumingPartitions[topic][partition] = offset
		}
	}
}

// RemoveConsumePartitions unregisters partitions from consumption
func (c *InMemoryKafkaClient) RemoveConsumePartitions(partitions map[string][]int32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for topic, partitionList := range partitions {
		if topicPartitions, exists := c.consumingPartitions[topic]; exists {
			for _, partition := range partitionList {
				delete(topicPartitions, partition)
			}
			if len(topicPartitions) == 0 {
				delete(c.consumingPartitions, topic)
			}
		}
	}
}

// PollFetches returns available messages from consumed partitions with proper batching
func (c *InMemoryKafkaClient) PollFetches(_ context.Context) kgo.Fetches {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return kgo.Fetches{}
	}

	topicsMap := c.buildTopicsMap()
	return c.buildFetches(topicsMap)
}

// Close marks the client as closed
func (c *InMemoryKafkaClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
}

// FetchOffsets retrieves committed offsets for a consumer group
func (c *InMemoryKafkaClient) FetchOffsets(_ context.Context, group string) (kadm.OffsetResponses, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return kadm.OffsetResponses{}, fmt.Errorf("client is closed")
	}

	responses := make(kadm.OffsetResponses)

	if groupOffsets, exists := c.committedOffsets[group]; exists {
		for topic, partitions := range groupOffsets {
			for partition, offset := range partitions {
				// Create a kadm.Offset with the committed offset
				kadmOffset := kadm.Offset{
					Topic:     topic,
					Partition: partition,
					At:        offset,
				}
				responses.Add(kadm.OffsetResponse{
					Offset: kadmOffset,
					Err:    nil,
				})
			}
		}
	}

	return responses, nil
}

// CommitOffsets commits offsets for a consumer group
func (c *InMemoryKafkaClient) CommitOffsets(_ context.Context, group string, offsets kadm.Offsets) (kadm.OffsetResponses, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return kadm.OffsetResponses{}, fmt.Errorf("client is closed")
	}

	if c.committedOffsets[group] == nil {
		c.committedOffsets[group] = make(map[string]map[int32]int64)
	}

	responses := make(kadm.OffsetResponses)

	offsets.Each(func(o kadm.Offset) {
		if c.committedOffsets[group][o.Topic] == nil {
			c.committedOffsets[group][o.Topic] = make(map[int32]int64)
		}
		c.committedOffsets[group][o.Topic][o.Partition] = o.At

		responses.Add(kadm.OffsetResponse{
			Offset: o,
			Err:    nil,
		})
	})

	return responses, nil
}

// AddMessage adds a message to a topic/partition for testing purposes
func (c *InMemoryKafkaClient) AddMessage(topic string, partition int32, key, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.topics[topic] == nil {
		c.topics[topic] = &InMemoryTopic{
			partitions: make(map[int32]*InMemoryPartition),
		}
	}

	topicData := c.topics[topic]
	if topicData.partitions[partition] == nil {
		topicData.partitions[partition] = &InMemoryPartition{
			messages:   make([]InMemoryMessage, 0),
			nextOffset: 0,
		}
	}

	partitionData := topicData.partitions[partition]
	msg := InMemoryMessage{
		Key:       key,
		Value:     value,
		Offset:    partitionData.nextOffset,
		Timestamp: time.Now(),
	}

	partitionData.messages = append(partitionData.messages, msg)
	partitionData.nextOffset++
}

// buildTopicsMap creates a map of topics to their fetch partitions
func (c *InMemoryKafkaClient) buildTopicsMap() map[string][]kgo.FetchPartition {
	topicsMap := make(map[string][]kgo.FetchPartition)

	for topicName, partitionOffsets := range c.consumingPartitions {
		topicData, exists := c.topics[topicName]
		if !exists {
			continue
		}

		for partitionID, startOffset := range partitionOffsets {
			fetchPartition, updatedOffset := c.createFetchPartition(topicName, partitionID, startOffset, topicData)
			if fetchPartition != nil {
				topicsMap[topicName] = append(topicsMap[topicName], *fetchPartition)
				// Update the consuming offset for this partition
				c.consumingPartitions[topicName][partitionID] = updatedOffset
			}
		}
	}

	return topicsMap
}

// createFetchPartition creates a fetch partition for the given topic/partition
func (c *InMemoryKafkaClient) createFetchPartition(topicName string, partitionID int32, startOffset kgo.Offset, topicData *InMemoryTopic) (*kgo.FetchPartition, kgo.Offset) {
	partitionData, exists := topicData.partitions[partitionID]
	if !exists {
		return nil, kgo.Offset{}
	}

	records, maxOffset := c.createRecordsFromMessages(topicName, partitionID, startOffset, partitionData.messages)
	if len(records) == 0 {
		return nil, kgo.Offset{}
	}

	fetchPartition := &kgo.FetchPartition{
		Partition: partitionID,
		Records:   records,
	}

	// Return updated offset for this partition
	updatedOffset := kgo.NewOffset().At(maxOffset + 1)
	return fetchPartition, updatedOffset
}

// createRecordsFromMessages extracts records from stored messages based on offset
func (c *InMemoryKafkaClient) createRecordsFromMessages(topicName string, partitionID int32, startOffset kgo.Offset, messages []InMemoryMessage) ([]*kgo.Record, int64) {
	startFrom := c.getStartOffset(startOffset)

	var records []*kgo.Record
	var maxOffset int64 = -1
	batchSize := 0
	// Reasonable batch size, the franz library internally will queue and batch records during a Fetch so this simulates that.
	const maxBatchSize = 10

	for _, msg := range messages {
		if msg.Offset >= startFrom && batchSize < maxBatchSize {
			record := &kgo.Record{
				Key:       msg.Key,
				Value:     msg.Value,
				Topic:     topicName,
				Partition: partitionID,
				Offset:    msg.Offset,
				Timestamp: msg.Timestamp,
			}
			records = append(records, record)
			if msg.Offset > maxOffset {
				maxOffset = msg.Offset
			}
			batchSize++
		}
	}

	return records, maxOffset
}

// getStartOffset extracts the actual offset value from kgo.Offset
func (c *InMemoryKafkaClient) getStartOffset(startOffset kgo.Offset) int64 {
	epochOffset := startOffset.EpochOffset()
	startFrom := epochOffset.Offset

	// If offset is negative (special cases like AtStart), start from 0
	if startFrom < 0 {
		startFrom = 0
	}

	return startFrom
}

// buildFetches constructs the final kgo.Fetches from the topics map
func (c *InMemoryKafkaClient) buildFetches(topicsMap map[string][]kgo.FetchPartition) kgo.Fetches {
	if len(topicsMap) == 0 {
		return kgo.Fetches{}
	}

	var fetchTopics []kgo.FetchTopic
	for topicName, partitions := range topicsMap {
		fetchTopic := kgo.FetchTopic{
			Topic:      topicName,
			Partitions: partitions,
		}
		fetchTopics = append(fetchTopics, fetchTopic)
	}

	fetch := kgo.Fetch{
		Topics: fetchTopics,
	}

	return kgo.Fetches{fetch}
}

// InMemoryKafkaClientFactory creates an in-memory Kafka client factory for testing
func InMemoryKafkaClientFactory(_ ingest.KafkaConfig, _ *kprom.Metrics, _ log.Logger) (Client, error) {
	return NewInMemoryKafkaClient(), nil
}
