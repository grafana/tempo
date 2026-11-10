// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/confmap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/kafka/configkafka"
)

var _ component.Config = (*Config)(nil)

// Config defines configuration for Kafka receiver.
type Config struct {
	ClientConfig   configkafka.ClientConfig   `mapstructure:",squash"`
	ConsumerConfig configkafka.ConsumerConfig `mapstructure:",squash"`

	// Logs holds configuration about how logs should be consumed.
	Logs TopicEncodingConfig `mapstructure:"logs"`

	// Metrics holds configuration about how metrics should be consumed.
	Metrics TopicEncodingConfig `mapstructure:"metrics"`

	// Traces holds configuration about how traces should be consumed.
	Traces TopicEncodingConfig `mapstructure:"traces"`

	// Profiles holds configuration about how profiles should be consumed.
	Profiles TopicEncodingConfig `mapstructure:"profiles"`

	// MessageMarking controls the way the messages are marked as consumed.
	MessageMarking MessageMarking `mapstructure:"message_marking"`

	// PartitionProcessing controls optional independent processing of assigned
	// topic partitions.
	PartitionProcessing PartitionProcessing `mapstructure:"partition_processing"`

	// HeaderExtraction controls extraction of headers from Kafka records.
	HeaderExtraction HeaderExtraction `mapstructure:"header_extraction"`

	// ErrorBackoff controls backoff/retry behavior when the next consumer
	// returns an error.
	ErrorBackOff configretry.BackOffConfig `mapstructure:"error_backoff"`

	// Telemetry controls optional telemetry configuration.
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
}

func (c *Config) Unmarshal(conf *confmap.Conf) error {
	if err := conf.Unmarshal(c); err != nil {
		return err
	}

	// max_buffered_batches is deprecated; copy it into max_prefetch.batches.
	pp, err := conf.Sub("partition_processing")
	if err != nil {
		return err
	}
	if pp.IsSet("max_buffered_batches") {
		if pp.IsSet("max_prefetch::batches") {
			return errors.New("partition_processing.max_buffered_batches and partition_processing.max_prefetch.batches are mutually exclusive")
		}
		c.PartitionProcessing.maxBufferedBatchesSet = true
		c.PartitionProcessing.MaxPrefetch.Batches = c.PartitionProcessing.MaxBufferedBatches
	}

	// Check if deprecated fields have been explicitly set
	// give them  precedence
	var zeroConfig Config
	if err := conf.Unmarshal(&zeroConfig); err != nil {
		return err
	}

	// Set OnPermanentError default value to inherit from OnError for backward compatibility
	// Only if OnPermanentError was not explicitly set in the config
	rawConf := conf.Get("message_marking")
	if rawConf != nil {
		if messageMarkingConf, ok := rawConf.(map[string]any); ok {
			if _, hasOnPermanentError := messageMarkingConf["on_permanent_error"]; !hasOnPermanentError {
				c.MessageMarking.OnPermanentError = c.MessageMarking.OnError
			}
		}
	} else {
		// If message_marking section doesn't exist, set defaults
		c.MessageMarking.OnPermanentError = c.MessageMarking.OnError
	}

	return nil
}

// Validate checks the receiver configuration is valid.
func (c *Config) Validate() error {
	// Validate that exclude_topic is only used with regex topic patterns
	if err := validateExcludeTopic("logs", c.Logs.Topics, c.Logs.ExcludeTopics); err != nil {
		return err
	}
	if err := validateExcludeTopic("metrics", c.Metrics.Topics, c.Metrics.ExcludeTopics); err != nil {
		return err
	}
	if err := validateExcludeTopic("traces", c.Traces.Topics, c.Traces.ExcludeTopics); err != nil {
		return err
	}
	if err := validateExcludeTopic("profiles", c.Profiles.Topics, c.Profiles.ExcludeTopics); err != nil {
		return err
	}
	if c.PartitionProcessing.Independent {
		if c.PartitionProcessing.MaxPrefetch.Batches <= 0 {
			return errors.New("partition_processing.max_prefetch.batches must be greater than zero")
		}
		if c.PartitionProcessing.MaxPrefetch.Bytes < 0 {
			return errors.New("partition_processing.max_prefetch.bytes must not be negative")
		}
		if c.PartitionProcessing.MaxInFlight.Records <= 0 {
			return errors.New("partition_processing.max_in_flight.records must be greater than zero")
		}
		if c.PartitionProcessing.MaxInFlight.Bytes < 0 {
			return errors.New("partition_processing.max_in_flight.bytes must not be negative")
		}
		switch c.PartitionProcessing.MaxInFlight.Window {
		case InFlightWindowPerFetch:
		case InFlightWindowCrossFetch:
			// cross_fetch frees a records slot when Consume returns, so without a
			// byte cap finished records behind a slow one pile up across fetches.
			if c.PartitionProcessing.MaxInFlight.Records <= 1 || c.PartitionProcessing.MaxInFlight.Bytes <= 0 {
				return errors.New("partition_processing.max_in_flight.window cross_fetch requires records above 1 and bytes above 0")
			}
		default:
			return fmt.Errorf("partition_processing.max_in_flight.window must be %q or %q", InFlightWindowPerFetch, InFlightWindowCrossFetch)
		}
		if !c.ConsumerConfig.AutoCommit.Enable {
			return errors.New("partition_processing.independent requires autocommit.enable")
		}
	}
	return nil
}

// validateExcludeTopic checks that exclude_topic is only configured when topics uses regex pattern
func validateExcludeTopic(signalType string, topics, excludeTopics []string) error {
	if len(excludeTopics) == 0 {
		return nil // No exclude_topic configured, nothing to validate
	}

	// if none of the configured topic uses regex return error
	var usesRegex bool
	for _, topic := range topics {
		if strings.HasPrefix(topic, "^") {
			usesRegex = true
			break
		}
	}

	if !usesRegex {
		return fmt.Errorf(
			"%s.exclude_topics is configured but none of the configured %s.topics use regex pattern (must start with '^')",
			signalType, signalType,
		)
	}

	for _, excludeTopic := range excludeTopics {
		// Validate that exclude_topic is not empty
		if excludeTopic == "" {
			return fmt.Errorf(
				"%s.exclude_topics contains empty string, which would match all topics",
				signalType,
			)
		}
		// Validate that exclude_topic is a valid regex pattern
		if _, err := regexp.Compile(excludeTopic); err != nil {
			return fmt.Errorf(
				"%s.exclude_topic contains invalid regex pattern: %w",
				signalType, err,
			)
		}
	}

	return nil
}

// TopicEncodingConfig holds signal-specific topic and encoding configuration.
type TopicEncodingConfig struct {
	// Topics holds the name of the Kafka topics from which messages of the
	// signal type should be consumed.
	//
	// The default depends on the signal type:
	//  - "otlp_spans" for traces
	//  - "otlp_metrics" for metrics
	//  - "otlp_logs" for logs
	//  - "otlp_profiles" for profiles
	Topics []string `mapstructure:"topics"`

	// Encoding holds the expected encoding of messages for the signal type
	//
	// Defaults to "otlp_proto".
	Encoding string `mapstructure:"encoding"`

	// Optional exclude topics option, used only in regex mode.
	ExcludeTopics []string `mapstructure:"exclude_topics"`
}

type MessageMarking struct {
	// If true, the messages are marked after the pipeline execution
	After bool `mapstructure:"after"`

	// If false, only the successfully processed messages are marked. This only applies
	// to non-permanent errors. It has no impact if After is set to false.
	// Note: this can block the entire partition in case a message processing returns
	// a non-permanent error.
	OnError bool `mapstructure:"on_error"`

	// If false, only the successfully processed messages are marked. This only applies
	// to permanent errors. It has no impact if After is set to false.
	// Default value inherits from OnError for backward compatibility.
	// Note: this can block the entire partition in case a message processing returns
	// a permanent error.
	OnPermanentError bool `mapstructure:"on_permanent_error"`
}

// PartitionProcessing controls optional independent processing of assigned
// Kafka partitions.
type PartitionProcessing struct {
	// Independent enables independent partition workers. Records stay ordered
	// within a partition unless MaxInFlight.Records is above 1.
	Independent bool `mapstructure:"independent"`

	// MaxBufferedBatches bounds the number of fetched batches waiting for each
	// partition worker.
	//
	// Deprecated [v0.164.0]: use partition_processing.max_prefetch.batches instead. Will be removed in v0.166.0.
	MaxBufferedBatches int `mapstructure:"max_buffered_batches"`

	// maxBufferedBatchesSet records whether max_buffered_batches appeared in the
	// config, so the deprecation warning fires even when its value is 0.
	maxBufferedBatchesSet bool

	// MaxPrefetch limits the fetched batches waiting for each independent
	// partition worker.
	MaxPrefetch MaxPrefetchConfig `mapstructure:"max_prefetch"`

	// MaxInFlight limits concurrent unmarshal-plus-Consume calls for each
	// independent partition worker.
	MaxInFlight MaxInFlightConfig `mapstructure:"max_in_flight"`
}

// MaxPrefetchConfig limits the fetched batches waiting for one partition worker.
type MaxPrefetchConfig struct {
	// Batches is how many fetched batches may wait for each partition worker.
	// Default 1. Must be greater than zero when independent processing is enabled.
	Batches int `mapstructure:"batches"`

	// Bytes caps the bytes of the waiting batches, counted the way
	// max_in_flight.bytes counts records. Fetching pauses once the waiting batches
	// reach the cap, so a batch that crosses it is still admitted and the overshoot
	// is at most one batch. 0 means no cap. It has no effect when Batches is 1.
	Bytes int `mapstructure:"bytes"`

	_ struct{} // avoids unkeyed_literal_initialization.
}

// MaxInFlightConfig limits concurrent unmarshal-plus-Consume calls for one
// partition worker.
type MaxInFlightConfig struct {
	// Records is how many calls may run at once. Default 1.
	Records int `mapstructure:"records"`

	// Bytes caps the bytes of the records in flight, from the start of their call
	// until the mark passes them. A record counts its key, value, and headers
	// after decompression, plus a fixed overhead of about 250 bytes per record
	// and 40 per header, so empty records count too. 0 means no cap. A record above the cap still runs,
	// alone. It has no effect when Records is 1.
	Bytes int `mapstructure:"bytes"`

	// Window is per_fetch (default) or cross_fetch. per_fetch waits for every
	// call of a fetched batch before it starts the next batch. cross_fetch keeps
	// starting records from later batches while both limits have room. Use it
	// when the next consumer holds Consume open until it has a full batch, such
	// as an exporter with wait_for_result and batch.min_size, so it sees more
	// than one fetch at a time. More records sit unmarked behind a slow one, so
	// more can replay after a crash or rebalance. cross_fetch requires Records
	// above 1 and Bytes above 0.
	Window string `mapstructure:"window"`

	_ struct{} // avoids unkeyed_literal_initialization.
}

const (
	InFlightWindowPerFetch   = "per_fetch"
	InFlightWindowCrossFetch = "cross_fetch"
)

type HeaderExtraction struct {
	ExtractHeaders bool     `mapstructure:"extract_headers"`
	Headers        []string `mapstructure:"headers"`
}

type TelemetryConfig struct {
	Metrics MetricsConfig `mapstructure:"metrics"`
	_       struct{}      // avoids unkeyed_literal_initialization
}

// MetricsConfig provides config for optional receiver metrics.
type MetricsConfig struct {
	// KafkaReceiverRecordsDelay controls whether the metric kafka_receiver_records_delay
	// that measures the time in seconds between producing and receiving a batch of records
	// will be reported or not. This metric is not reported by default because
	// it may slow down high-volume consuming.
	KafkaReceiverRecordsDelay MetricConfig `mapstructure:"kafka_receiver_records_delay"`
	_                         struct{}     // avoids unkeyed_literal_initialization
}

// MetricConfig provides common config for a particular metric.
type MetricConfig struct {
	Enabled bool     `mapstructure:"enabled"`
	_       struct{} // avoids unkeyed_literal_initialization
}
