package livestore

import (
	"context"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kprom"
)

// Client is an interface for Kafka client operations used by the live store
type Client interface {
	Ping(ctx context.Context) error
	AddConsumePartitions(partitions map[string]map[int32]kgo.Offset)
	RemoveConsumePartitions(partitions map[string][]int32)
	PollFetches(ctx context.Context) kgo.Fetches
	Close()
	// Admin operations
	FetchOffsets(ctx context.Context, group string) (kadm.OffsetResponses, error)
	CommitOffsets(ctx context.Context, group string, offsets kadm.Offsets) (kadm.OffsetResponses, error)
}

// ClientFactory is a function that creates a KafkaClient
type ClientFactory func(cfg ingest.KafkaConfig, metrics *kprom.Metrics, logger log.Logger) (Client, error)

// defaultKafkaClient wraps a kgo.Client to implement the KafkaClient interface
type defaultKafkaClient struct {
	client *kgo.Client
	adm    *kadm.Client
}

// Ping delegates to the underlying client
func (c *defaultKafkaClient) Ping(ctx context.Context) error {
	return c.client.Ping(ctx)
}

// AddConsumePartitions delegates to the underlying client
func (c *defaultKafkaClient) AddConsumePartitions(partitions map[string]map[int32]kgo.Offset) {
	c.client.AddConsumePartitions(partitions)
}

// RemoveConsumePartitions delegates to the underlying client
func (c *defaultKafkaClient) RemoveConsumePartitions(partitions map[string][]int32) {
	c.client.RemoveConsumePartitions(partitions)
}

// PollFetches delegates to the underlying client
func (c *defaultKafkaClient) PollFetches(ctx context.Context) kgo.Fetches {
	return c.client.PollFetches(ctx)
}

// Close delegates to the underlying client
func (c *defaultKafkaClient) Close() {
	c.client.Close()
}

// FetchOffsets delegates to the admin client
func (c *defaultKafkaClient) FetchOffsets(ctx context.Context, group string) (kadm.OffsetResponses, error) {
	return c.adm.FetchOffsets(ctx, group)
}

// CommitOffsets delegates to the admin client
func (c *defaultKafkaClient) CommitOffsets(ctx context.Context, group string, offsets kadm.Offsets) (kadm.OffsetResponses, error) {
	return c.adm.CommitOffsets(ctx, group, offsets)
}

// Client returns the underlying kgo.Client
func (c *defaultKafkaClient) Client() *kgo.Client {
	return c.client
}

// DefaultKafkaClientFactory creates a default Kafka client using ingest.NewReaderClient
func DefaultKafkaClientFactory(cfg ingest.KafkaConfig, metrics *kprom.Metrics, logger log.Logger) (Client, error) {
	client, err := ingest.NewReaderClient(cfg, metrics, logger)
	if err != nil {
		return nil, err
	}
	adm := kadm.NewClient(client)
	return &defaultKafkaClient{client: client, adm: adm}, nil
}
