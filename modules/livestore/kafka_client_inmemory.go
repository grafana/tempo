package livestore

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
type InMemoryMessage struct {
	Key       []byte
	Value     []byte
	Offset    int64
	Timestamp time.Time
}

// InMemoryPartition represents a single partition with its messages and state
type InMemoryPartition struct {
	messages []InMemoryMessage
	nextOffset int64
	mu sync.RWMutex
}

// InMemoryTopic represents a topic with multiple partitions
type InMemoryTopic struct {
	partitions map[int32]*InMemoryPartition
	mu sync.RWMutex
}

// InMemoryKafkaClient implements KafkaClient interface using in-memory queues for testing
type InMemoryKafkaClient struct {
	topics map[string]*InMemoryTopic
	
	// Consumer state
	consumingPartitions map[string]map[int32]kgo.Offset
	
	// Committed offsets per consumer group
	committedOffsets map[string]map[string]map[int32]int64 // [group][topic][partition] = offset
	
	closed bool
	mu sync.RWMutex
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
func (c *InMemoryKafkaClient) Ping(ctx context.Context) error {
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

// PollFetches returns available messages from consumed partitions
// Note: This is a simplified mock implementation that returns empty fetches
// In a real test scenario, you would implement a more sophisticated mock
func (c *InMemoryKafkaClient) PollFetches(ctx context.Context) kgo.Fetches {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.closed {
		return kgo.Fetches{}
	}
	
	// For now, return empty fetches. In a complete implementation,
	// you would build proper kgo.Fetches with the available messages
	// This requires deeper integration with kgo's internal structures
	return kgo.Fetches{}
}

// Close marks the client as closed
func (c *InMemoryKafkaClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
}

// FetchOffsets retrieves committed offsets for a consumer group
func (c *InMemoryKafkaClient) FetchOffsets(ctx context.Context, group string) (kadm.OffsetResponses, error) {
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
func (c *InMemoryKafkaClient) CommitOffsets(ctx context.Context, group string, offsets kadm.Offsets) (kadm.OffsetResponses, error) {
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

// Client returns nil as there's no underlying kgo.Client for in-memory implementation
func (c *InMemoryKafkaClient) Client() *kgo.Client {
	return nil
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
	topicData.mu.Lock()
	defer topicData.mu.Unlock()
	
	if topicData.partitions[partition] == nil {
		topicData.partitions[partition] = &InMemoryPartition{
			messages: make([]InMemoryMessage, 0),
			nextOffset: 0,
		}
	}
	
	partitionData := topicData.partitions[partition]
	partitionData.mu.Lock()
	defer partitionData.mu.Unlock()
	
	msg := InMemoryMessage{
		Key:       key,
		Value:     value,
		Offset:    partitionData.nextOffset,
		Timestamp: time.Now(),
	}
	
	partitionData.messages = append(partitionData.messages, msg)
	partitionData.nextOffset++
}

// InMemoryKafkaClientFactory creates an in-memory Kafka client factory for testing
func InMemoryKafkaClientFactory(cfg ingest.KafkaConfig, metrics *kprom.Metrics, logger log.Logger) (KafkaClient, error) {
	return NewInMemoryKafkaClient(), nil
}