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
	configkafka.ClientConfig   `mapstructure:",squash"`
	configkafka.ConsumerConfig `mapstructure:",squash"`

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

	// Check if deprecated fields have been explicitly set
	// give them  precedence
	var zeroConfig Config
	if err := conf.Unmarshal(&zeroConfig); err != nil {
		return err
	}

	// handle deprecated topic and exclude_topic for log signal
	if zeroConfig.Logs.Topic != "" {
		if len(zeroConfig.Logs.Topics) == 0 {
			c.Logs.Topics = []string{zeroConfig.Logs.Topic}
			// copy original logs.topic to its alias
			// required for validation further down in this code
			c.Logs.topicAlias = c.Logs.Topic
			c.Logs.Topic = ""
		}
	}
	if zeroConfig.Logs.ExcludeTopic != "" {
		if len(zeroConfig.Logs.ExcludeTopics) == 0 {
			c.Logs.ExcludeTopics = []string{zeroConfig.Logs.ExcludeTopic}
			c.Logs.excludeTopicAlias = c.Logs.ExcludeTopic
			c.Logs.ExcludeTopic = ""
		}
	}

	// handle deprecated topic and exclude_topic for metric signal
	if zeroConfig.Metrics.Topic != "" {
		if len(zeroConfig.Metrics.Topics) == 0 {
			c.Metrics.Topics = []string{zeroConfig.Metrics.Topic}
			c.Metrics.topicAlias = c.Metrics.Topic
			c.Metrics.Topic = ""
		}
	}

	if zeroConfig.Metrics.ExcludeTopic != "" {
		if len(zeroConfig.Metrics.ExcludeTopics) == 0 {
			c.Metrics.ExcludeTopics = []string{zeroConfig.Metrics.ExcludeTopic}
			c.Metrics.excludeTopicAlias = c.Metrics.ExcludeTopic
			c.Metrics.ExcludeTopic = ""
		}
	}

	// handle deprecated topic and exclude_topic for trace signal
	if zeroConfig.Traces.Topic != "" {
		if len(zeroConfig.Traces.Topics) == 0 {
			c.Traces.Topics = []string{zeroConfig.Traces.Topic}
			c.Traces.topicAlias = c.Traces.Topic
			c.Traces.Topic = ""
		}
	}

	if zeroConfig.Traces.ExcludeTopic != "" {
		if len(zeroConfig.Traces.ExcludeTopics) == 0 {
			c.Traces.ExcludeTopics = []string{zeroConfig.Traces.ExcludeTopic}
			c.Traces.excludeTopicAlias = c.Traces.ExcludeTopic
			c.Traces.ExcludeTopic = ""
		}
	}

	// handle deprecated topic and exclude_topic for profile signal
	if zeroConfig.Profiles.Topic != "" {
		if len(zeroConfig.Profiles.Topics) == 0 {
			c.Profiles.Topics = []string{zeroConfig.Profiles.Topic}
			c.Profiles.topicAlias = c.Profiles.Topic
			c.Profiles.Topic = ""
		}
	}
	if zeroConfig.Profiles.ExcludeTopic != "" {
		if len(zeroConfig.Profiles.ExcludeTopics) == 0 {
			c.Profiles.ExcludeTopics = []string{zeroConfig.Profiles.ExcludeTopic}
			c.Profiles.excludeTopicAlias = c.Profiles.ExcludeTopic
			c.Profiles.ExcludeTopic = ""
		}
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
	if c.Logs.Topic != "" && len(c.Logs.Topics) != 0 {
		return errors.New("both logs.topic and logs.topics cannot be set")
	}
	if c.Metrics.Topic != "" && len(c.Metrics.Topics) != 0 {
		return errors.New("both metrics.topic and metrics.topics cannot be set")
	}
	if c.Traces.Topic != "" && len(c.Traces.Topics) != 0 {
		return errors.New("both traces.topic and traces.topics cannot be set")
	}
	if c.Profiles.Topic != "" && len(c.Profiles.Topics) != 0 {
		return errors.New("both profiles.topic and profiles.topics cannot be set")
	}

	if c.Logs.ExcludeTopic != "" && len(c.Logs.ExcludeTopics) != 0 {
		return errors.New("both logs.exclude_topic and logs.exclude_topics cannot be set")
	}
	if c.Metrics.ExcludeTopic != "" && len(c.Metrics.ExcludeTopics) != 0 {
		return errors.New("both metrics.exclude_topic and metrics.exclude_topics cannot be set")
	}
	if c.Traces.ExcludeTopic != "" && len(c.Traces.ExcludeTopics) != 0 {
		return errors.New("both traces.exclude_topic and traces.exclude_topics cannot be set")
	}
	if c.Profiles.ExcludeTopic != "" && len(c.Profiles.ExcludeTopics) != 0 {
		return errors.New("both profiles.exclude_topic and profiles.exclude_topics cannot be set")
	}

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
	// Deprecated [v0.142.0]: Use Topics
	Topic string `mapstructure:"topic"`

	// alias for Topic
	topicAlias string

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

	// Deprecated [v0.142.0]: Use ExcludeTopics
	ExcludeTopic string `mapstructure:"exclude_topic"`

	// alias for exclude_topic
	excludeTopicAlias string

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
