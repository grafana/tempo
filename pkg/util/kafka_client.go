package util

import (
	"context"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kprom"
)

// KafkaClient is an interface for Kafka client operations used by the live store
type KafkaClient interface {
	Ping(ctx context.Context) error
	AddConsumePartitions(partitions map[string]map[int32]kgo.Offset)
	RemoveConsumePartitions(partitions map[string][]int32)
	PollFetches(ctx context.Context) kgo.Fetches
	Close()
	FetchOffsets(ctx context.Context, group string) (kadm.OffsetResponses, error)
	CommitOffsets(ctx context.Context, group string, offsets kadm.Offsets) (kadm.OffsetResponses, error)
}

// KafkaClientFunc is a function that creates a KafkaClient
type KafkaClientFunc func(cfg ingest.KafkaConfig, metrics *kprom.Metrics, logger log.Logger) (KafkaClient, error)
