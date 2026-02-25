// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafka // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka"

import (
	"context"
	"crypto/tls"

	"github.com/IBM/sarama"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/kafka/configkafka"
)

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

// newSaramaClientConfig returns a Sarama client config, based on the given config.
func newSaramaClientConfig(ctx context.Context, config configkafka.ClientConfig) (*sarama.Config, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.ClientID = config.ClientID
	if config.RackID != "" {
		saramaConfig.RackID = config.RackID
	}
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
		saramaConfig.Net.TLS.Enable = true
	}
	configureSaramaAuthentication(ctx, config.Authentication, saramaConfig)
	return saramaConfig, nil
}
