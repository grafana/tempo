package livestore

import (
	"context"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kprom"
)

// FranzKafkaClient wraps a kgo.Client and admin client to implement the KafkaClient interface
type FranzKafkaClient struct {
	client  *kgo.Client
	adminCl *kadm.Client
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
		client:  client,
		adminCl: adminCl,
	}, nil
}

func (c *FranzKafkaClient) Ping(ctx context.Context) error {
	_, err := c.adminCl.Metadata(ctx)
	return err
}

func (c *FranzKafkaClient) AddConsumePartitions(partitions map[string]map[int32]kgo.Offset) {
	c.client.AddConsumePartitions(partitions)
}

func (c *FranzKafkaClient) RemoveConsumePartitions(partitions map[string][]int32) {
	c.client.RemoveConsumePartitions(partitions)
}

func (c *FranzKafkaClient) PollFetches(ctx context.Context) kgo.Fetches {
	return c.client.PollFetches(ctx)
}

func (c *FranzKafkaClient) Close() {
	c.client.Close()
}

func (c *FranzKafkaClient) FetchOffsets(ctx context.Context, group string) (kadm.OffsetResponses, error) {
	return c.adminCl.FetchOffsets(ctx, group)
}

func (c *FranzKafkaClient) CommitOffsets(ctx context.Context, group string, offsets kadm.Offsets) (kadm.OffsetResponses, error) {
	return c.adminCl.CommitOffsets(ctx, group, offsets)
}

func FranzKafkaClientFunc(cfg ingest.KafkaConfig, metrics *kprom.Metrics, logger log.Logger) (KafkaClient, error) {
	return NewFranzKafkaClient(cfg, metrics, logger)
}
