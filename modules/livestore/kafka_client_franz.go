package livestore

import (
	"context"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kprom"
)

// FranzKafkaClient wraps a kgo.Client to implement the KafkaClient interface
type FranzKafkaClient struct {
	client          *kgo.Client
	adminCl         *kadm.Client
	logger          log.Logger
	cfg             ingest.KafkaConfig
	consumingTopics map[string]bool // Track which topics we're consuming
}

// NewFranzKafkaClient creates a new Franz/kgo-based Kafka client
func NewFranzKafkaClient(cfg ingest.KafkaConfig, metrics *kprom.Metrics, logger log.Logger) (KafkaClient, error) {
	// Create client with topic consumption configured
	opts := []kgo.Opt{
		kgo.ConsumerGroup(cfg.ConsumerGroup),
		kgo.ConsumeTopics(cfg.Topic),
	}

	client, err := ingest.NewReaderClient(cfg, metrics, logger, opts...)
	if err != nil {
		return nil, err
	}

	adminCl := kadm.NewClient(client)

	return &FranzKafkaClient{
		client:          client,
		adminCl:         adminCl,
		logger:          logger,
		cfg:             cfg,
		consumingTopics: make(map[string]bool),
	}, nil
}

// Ping checks if the Kafka cluster is reachable
func (c *FranzKafkaClient) Ping(ctx context.Context) error {
	// Use metadata request to check connectivity
	_, err := c.adminCl.Metadata(ctx)
	return err
}

// AddConsumePartitions adds partitions to consume from
// For the Franz client, topics are configured at client creation time
func (c *FranzKafkaClient) AddConsumePartitions(partitions map[string]map[int32]kgo.Offset) {
	c.client.AddConsumePartitions(partitions)
}

// RemoveConsumePartitions removes partitions from consumption
func (c *FranzKafkaClient) RemoveConsumePartitions(partitions map[string][]int32) {
	c.client.RemoveConsumePartitions(partitions)
}

// PollFetches polls for new messages
func (c *FranzKafkaClient) PollFetches(ctx context.Context) kgo.Fetches {
	return c.client.PollFetches(ctx)
}

// Close closes the Kafka client
func (c *FranzKafkaClient) Close() {
	if c.client != nil {
		c.client.Close()
	}
}

// FetchOffsets retrieves committed offsets for a consumer group
func (c *FranzKafkaClient) FetchOffsets(ctx context.Context, group string) (kadm.OffsetResponses, error) {
	// Create context with timeout for admin operations
	adminCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return c.adminCl.FetchOffsets(adminCtx, group)
}

// CommitOffsets commits offsets for a consumer group
func (c *FranzKafkaClient) CommitOffsets(ctx context.Context, group string, offsets kadm.Offsets) (kadm.OffsetResponses, error) {
	// Create context with timeout for admin operations
	adminCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := c.adminCl.CommitOffsets(adminCtx, group, offsets)
	if err != nil {
		level.Error(c.logger).Log("msg", "failed to commit offsets", "group", group, "err", err)
		return kadm.OffsetResponses{}, err
	}

	return resp, nil
}

// FranzKafkaClientFunc creates a Franz-based Kafka client factory
func FranzKafkaClientFunc(cfg ingest.KafkaConfig, metrics *kprom.Metrics, logger log.Logger) (KafkaClient, error) {
	return NewFranzKafkaClient(cfg, metrics, logger)
}
