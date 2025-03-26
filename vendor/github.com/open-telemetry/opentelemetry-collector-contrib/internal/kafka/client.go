// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafka // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka"

import (
	"context"

	"github.com/IBM/sarama"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka/configkafka"
)

// NewSaramaClusterAdminClient returns a new Kafka cluster admin client with the given configuration.
func NewSaramaClusterAdminClient(ctx context.Context, config configkafka.ClientConfig) (sarama.ClusterAdmin, error) {
	saramaConfig, err := NewSaramaClientConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return sarama.NewClusterAdmin(config.Brokers, saramaConfig)
}

// TODO add NewSaramaConsumerGroup, extracted from receiver/kafkareceiver
// TODO add NewSaramaSyncProducer, extracted from exporter/kafkaexporter

// NewSaramaClientConfig returns a Sarama client config, based on the given config.
func NewSaramaClientConfig(ctx context.Context, config configkafka.ClientConfig) (*sarama.Config, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Metadata.Full = config.Metadata.Full
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
