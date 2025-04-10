// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafka // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka"

import (
	"context"
	"time"

	"github.com/IBM/sarama"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka/configkafka"
)

var saramaCompressionCodecs = map[string]sarama.CompressionCodec{
	"none":   sarama.CompressionNone,
	"gzip":   sarama.CompressionGZIP,
	"snappy": sarama.CompressionSnappy,
	"lz4":    sarama.CompressionLZ4,
	"zstd":   sarama.CompressionZSTD,
}

var saramaInitialOffsets = map[string]int64{
	configkafka.EarliestOffset: sarama.OffsetOldest,
	configkafka.LatestOffset:   sarama.OffsetNewest,
}

// NewSaramaClient returns a new Kafka client with the given configuration.
func NewSaramaClient(ctx context.Context, config configkafka.ClientConfig) (sarama.Client, error) {
	saramaConfig, err := NewSaramaClientConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return sarama.NewClient(config.Brokers, saramaConfig)
}

// NewSaramaClusterAdminClient returns a new Kafka cluster admin client with the given configuration.
func NewSaramaClusterAdminClient(ctx context.Context, config configkafka.ClientConfig) (sarama.ClusterAdmin, error) {
	saramaConfig, err := NewSaramaClientConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return sarama.NewClusterAdmin(config.Brokers, saramaConfig)
}

// NewSaramaConsumerGroup returns a new Kafka consumer group with the given configuration.
func NewSaramaConsumerGroup(
	ctx context.Context,
	clientConfig configkafka.ClientConfig,
	consumerConfig configkafka.ConsumerConfig,
) (sarama.ConsumerGroup, error) {
	saramaConfig, err := NewSaramaClientConfig(ctx, clientConfig)
	if err != nil {
		return nil, err
	}
	saramaConfig.Consumer.Group.Session.Timeout = consumerConfig.SessionTimeout
	saramaConfig.Consumer.Group.Heartbeat.Interval = consumerConfig.HeartbeatInterval
	saramaConfig.Consumer.Fetch.Min = consumerConfig.MinFetchSize
	saramaConfig.Consumer.Fetch.Default = consumerConfig.DefaultFetchSize
	saramaConfig.Consumer.Fetch.Max = consumerConfig.MaxFetchSize
	saramaConfig.Consumer.Offsets.AutoCommit.Enable = consumerConfig.AutoCommit.Enable
	saramaConfig.Consumer.Offsets.AutoCommit.Interval = consumerConfig.AutoCommit.Interval
	saramaConfig.Consumer.Offsets.Initial = saramaInitialOffsets[consumerConfig.InitialOffset]
	return sarama.NewConsumerGroup(clientConfig.Brokers, consumerConfig.GroupID, saramaConfig)
}

// NewSaramaSyncProducer returns a new synchronous Kafka producer with the given configuration.
//
// NewSaramaSyncProducer takes a timeout for produce operations, which is the maximum time to
// wait for required_acks. This is required since SyncProducer methods cannot be cancelled with
// a context.Context.
func NewSaramaSyncProducer(
	ctx context.Context,
	clientConfig configkafka.ClientConfig,
	producerConfig configkafka.ProducerConfig,
	producerTimeout time.Duration,
) (sarama.SyncProducer, error) {
	saramaConfig, err := NewSaramaClientConfig(ctx, clientConfig)
	if err != nil {
		return nil, err
	}
	saramaConfig.Producer.Return.Successes = true // required for SyncProducer
	saramaConfig.Producer.Return.Errors = true    // required for SyncProducer
	saramaConfig.Producer.MaxMessageBytes = producerConfig.MaxMessageBytes
	saramaConfig.Producer.Flush.MaxMessages = producerConfig.FlushMaxMessages
	saramaConfig.Producer.RequiredAcks = sarama.RequiredAcks(producerConfig.RequiredAcks)
	saramaConfig.Producer.Timeout = producerTimeout
	saramaConfig.Producer.Compression = saramaCompressionCodecs[producerConfig.Compression]
	return sarama.NewSyncProducer(clientConfig.Brokers, saramaConfig)
}

// NewSaramaClientConfig returns a Sarama client config, based on the given config.
func NewSaramaClientConfig(ctx context.Context, config configkafka.ClientConfig) (*sarama.Config, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Metadata.Full = config.Metadata.Full
	saramaConfig.Metadata.RefreshFrequency = config.Metadata.RefreshInterval
	saramaConfig.Metadata.Retry.Max = config.Metadata.Retry.Max
	saramaConfig.Metadata.Retry.Backoff = config.Metadata.Retry.Backoff
	if config.ResolveCanonicalBootstrapServersOnly {
		saramaConfig.Net.ResolveCanonicalBootstrapServers = true
	}
	if config.ProtocolVersion != "" {
		var err error
		if saramaConfig.Version, err = sarama.ParseKafkaVersion(config.ProtocolVersion); err != nil {
			return nil, err
		}
	}
	if err := ConfigureSaramaAuthentication(ctx, config.Authentication, saramaConfig); err != nil {
		return nil, err
	}
	return saramaConfig, nil
}
