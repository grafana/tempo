package livestore

import (
	"context"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/util/kafka"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kprom"
)

// KafkaClient wraps a kgo.Client and admin client to implement the KafkaClient interface
type KafkaClient struct {
	client  *kgo.Client
	adminCl *kadm.Client
}

// NewKafkaClient creates a new Franz/kgo-based Kafka client
func NewKafkaClient(cfg ingest.KafkaConfig, metrics *kprom.Metrics, logger log.Logger) (kafka.KafkaClient, error) {

	client, err := ingest.NewReaderClient(cfg, metrics, logger)
	if err != nil {
		return nil, err
	}

	adminCl := kadm.NewClient(client)

	return &KafkaClient{
		client:  client,
		adminCl: adminCl,
	}, nil
}

func (c *KafkaClient) Ping(ctx context.Context) error {
	_, err := c.adminCl.Metadata(ctx)
	return err
}

func (c *KafkaClient) AddConsumePartitions(partitions map[string]map[int32]kgo.Offset) {
	c.client.AddConsumePartitions(partitions)
}

func (c *KafkaClient) RemoveConsumePartitions(partitions map[string][]int32) {
	c.client.RemoveConsumePartitions(partitions)
}

func (c *KafkaClient) PollFetches(ctx context.Context) kgo.Fetches {
	return c.client.PollFetches(ctx)
}

func (c *KafkaClient) Close() {
	c.client.Close()
	c.adminCl.Close()
}

func (c *KafkaClient) FetchOffsets(ctx context.Context, group string) (kadm.OffsetResponses, error) {
	return c.adminCl.FetchOffsets(ctx, group)
}

func (c *KafkaClient) CommitOffsets(ctx context.Context, group string, offsets kadm.Offsets) (kadm.OffsetResponses, error) {
	return c.adminCl.CommitOffsets(ctx, group, offsets)
}

func FranzKafkaClientFunc(cfg ingest.KafkaConfig, metrics *kprom.Metrics, logger log.Logger) (kafka.KafkaClient, error) {
	return NewKafkaClient(cfg, metrics, logger)
}
