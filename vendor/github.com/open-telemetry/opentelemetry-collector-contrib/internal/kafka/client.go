// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafka // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka"

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/collector/config/configcompression"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/kafka/configkafka"
)

var saramaCompressionCodecs = map[string]sarama.CompressionCodec{
	"none":   sarama.CompressionNone,
	"gzip":   sarama.CompressionGZIP,
	"snappy": sarama.CompressionSnappy,
	"lz4":    sarama.CompressionLZ4,
	"zstd":   sarama.CompressionZSTD,
}

func convertToSaramaCompressionLevel(p configcompression.Level) int {
	if p == configcompression.DefaultCompressionLevel {
		return sarama.CompressionLevelDefault
	}
	return int(p)
}

var saramaInitialOffsets = map[string]int64{
	configkafka.EarliestOffset: sarama.OffsetOldest,
	configkafka.LatestOffset:   sarama.OffsetNewest,
}

// NewSaramaClient returns a new Kafka client with the given configuration.
func NewSaramaClient(ctx context.Context, config configkafka.ClientConfig) (sarama.Client, error) {
	saramaConfig, err := newSaramaClientConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return sarama.NewClient(config.Brokers, saramaConfig)
}

// NewSaramaClusterAdminClient returns a new Kafka cluster admin client with the given configuration.
func NewSaramaClusterAdminClient(ctx context.Context, config configkafka.ClientConfig) (sarama.ClusterAdmin, error) {
	saramaConfig, err := newSaramaClientConfig(ctx, config)
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
	saramaConfig, err := newSaramaClientConfig(ctx, clientConfig)
	if err != nil {
		return nil, err
	}
	saramaConfig.Consumer.Group.Session.Timeout = consumerConfig.SessionTimeout
	saramaConfig.Consumer.Group.Heartbeat.Interval = consumerConfig.HeartbeatInterval
	saramaConfig.Consumer.Fetch.Min = consumerConfig.MinFetchSize
	saramaConfig.Consumer.Fetch.Default = consumerConfig.DefaultFetchSize
	saramaConfig.Consumer.Fetch.Max = consumerConfig.MaxFetchSize
	saramaConfig.Consumer.MaxWaitTime = consumerConfig.MaxFetchWait
	saramaConfig.Consumer.Offsets.AutoCommit.Enable = consumerConfig.AutoCommit.Enable
	saramaConfig.Consumer.Offsets.AutoCommit.Interval = consumerConfig.AutoCommit.Interval
	saramaConfig.Consumer.Offsets.Initial = saramaInitialOffsets[consumerConfig.InitialOffset]
	// Set the rebalance strategy
	rebalanceStrategy := rebalanceStrategy(consumerConfig.GroupRebalanceStrategy)
	if rebalanceStrategy != nil {
		saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{rebalanceStrategy}
	}
	if len(consumerConfig.GroupInstanceID) > 0 {
		saramaConfig.Consumer.Group.InstanceId = consumerConfig.GroupInstanceID
	}
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
	saramaConfig, err := newSaramaClientConfig(ctx, clientConfig)
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
	saramaConfig.Producer.CompressionLevel = convertToSaramaCompressionLevel(producerConfig.CompressionParams.Level)
	return sarama.NewSyncProducer(clientConfig.Brokers, saramaConfig)
}

// newSaramaClientConfig returns a Sarama client config, based on the given config.
func newSaramaClientConfig(ctx context.Context, config configkafka.ClientConfig) (*sarama.Config, error) {
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

	tlsConfig := config.TLS
	if tlsConfig == nil {
		tlsConfig = config.Authentication.TLS
	}
	if tlsConfig != nil {
		if tlsConfig, err := tlsConfig.LoadTLSConfig(ctx); err != nil {
			return nil, err
		} else if tlsConfig != nil {
			saramaConfig.Net.TLS.Config = tlsConfig
			saramaConfig.Net.TLS.Enable = true
		}
	} else if config.Authentication.SASL != nil && config.Authentication.SASL.Mechanism == "AWS_MSK_IAM_OAUTHBEARER" {
		saramaConfig.Net.TLS.Config = &tls.Config{}
		saramaConfig.Net.SASL.Enable = true
	}
	configureSaramaAuthentication(ctx, config.Authentication, saramaConfig)
	return saramaConfig, nil
}

func rebalanceStrategy(strategy string) sarama.BalanceStrategy {
	switch strategy {
	case sarama.RangeBalanceStrategyName:
		return sarama.NewBalanceStrategyRange()
	case sarama.StickyBalanceStrategyName:
		return sarama.NewBalanceStrategySticky()
	case sarama.RoundRobinBalanceStrategyName:
		return sarama.NewBalanceStrategyRoundRobin()
	default:
		return nil
	}
}
