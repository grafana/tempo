package blockbuilder

import (
	"context"
	"fmt"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kprom"
)

type Consumer struct{}

type PartitionsFn func(context.Context, *kgo.Client, map[string][]int32)

// NewConsumer creates a new Kafka consumer for the block-builder
func NewConsumer(kafkaCfg ingest.KafkaConfig, metrics *kprom.Metrics, logger log.Logger, onRevoked, onAssigned PartitionsFn, opts ...kgo.Opt) (*kgo.Client, error) {
	opts = append(opts,
		kgo.DisableAutoCommit(),
		kgo.OnPartitionsRevoked(onRevoked),
		kgo.OnPartitionsAssigned(onAssigned),

		// Block-builder group consumes
		kgo.ConsumerGroup(kafkaCfg.ConsumerGroup),
	)

	client, err := ingest.NewReaderClient(kafkaCfg, metrics, logger, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka client: %w", err)
	}
	return client, nil
}
