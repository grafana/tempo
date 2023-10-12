// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"time"

	"go.opentelemetry.io/collector/component"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/kafkaexporter"
)

type AutoCommit struct {
	// Whether or not to auto-commit updated offsets back to the broker.
	// (default enabled).
	Enable bool `mapstructure:"enable"`
	// How frequently to commit updated offsets. Ineffective unless
	// auto-commit is enabled (default 1s)
	Interval time.Duration `mapstructure:"interval"`
}

type MessageMarking struct {
	// If true, the messages are marked after the pipeline execution
	After bool `mapstructure:"after"`

	// If false, only the successfully processed messages are marked, it has no impact if
	// After is set to false.
	// Note: this can block the entire partition in case a message processing returns
	// a permanent error.
	OnError bool `mapstructure:"on_error"`
}

// Config defines configuration for Kafka receiver.
type Config struct {
	// The list of kafka brokers (default localhost:9092)
	Brokers []string `mapstructure:"brokers"`
	// Kafka protocol version
	ProtocolVersion string `mapstructure:"protocol_version"`
	// The name of the kafka topic to consume from (default "otlp_spans")
	Topic string `mapstructure:"topic"`
	// Encoding of the messages (default "otlp_proto")
	Encoding string `mapstructure:"encoding"`
	// The consumer group that receiver will be consuming messages from (default "otel-collector")
	GroupID string `mapstructure:"group_id"`
	// The consumer client ID that receiver will use (default "otel-collector")
	ClientID string `mapstructure:"client_id"`
	// The initial offset to use if no offset was previously committed.
	// Must be `latest` or `earliest` (default "latest").
	InitialOffset string `mapstructure:"initial_offset"`

	// Metadata is the namespace for metadata management properties used by the
	// Client, and shared by the Producer/Consumer.
	Metadata kafkaexporter.Metadata `mapstructure:"metadata"`

	Authentication kafkaexporter.Authentication `mapstructure:"auth"`

	// Controls the auto-commit functionality
	AutoCommit AutoCommit `mapstructure:"autocommit"`

	// Controls the way the messages are marked as consumed
	MessageMarking MessageMarking `mapstructure:"message_marking"`
}

const (
	offsetLatest   string = "latest"
	offsetEarliest string = "earliest"
)

var _ component.Config = (*Config)(nil)

// Validate checks the receiver configuration is valid
func (cfg *Config) Validate() error {
	return nil
}
