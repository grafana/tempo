// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/kafkaexporter"

import (
	"context"
	"errors"
	"fmt"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka"
)

var errUnrecognizedEncoding = fmt.Errorf("unrecognized encoding")

// kafkaTracesProducer uses sarama to produce trace messages to Kafka.
type kafkaTracesProducer struct {
	cfg       Config
	producer  sarama.SyncProducer
	marshaler TracesMarshaler
	logger    *zap.Logger
}

type kafkaErrors struct {
	count int
	err   string
}

func (ke kafkaErrors) Error() string {
	return fmt.Sprintf("Failed to deliver %d messages due to %s", ke.count, ke.err)
}

func (e *kafkaTracesProducer) tracesPusher(_ context.Context, td ptrace.Traces) error {
	messages, err := e.marshaler.Marshal(td, getTopic(&e.cfg, td.ResourceSpans()))
	if err != nil {
		return consumererror.NewPermanent(err)
	}
	err = e.producer.SendMessages(messages)
	if err != nil {
		var prodErr sarama.ProducerErrors
		if errors.As(err, &prodErr) {
			if len(prodErr) > 0 {
				return kafkaErrors{len(prodErr), prodErr[0].Err.Error()}
			}
		}
		return err
	}
	return nil
}

func (e *kafkaTracesProducer) Close(context.Context) error {
	if e.producer == nil {
		return nil
	}
	return e.producer.Close()
}

func (e *kafkaTracesProducer) start(_ context.Context, _ component.Host) error {
	producer, err := newSaramaProducer(e.cfg)
	if err != nil {
		return err
	}
	e.producer = producer
	return nil
}

// kafkaMetricsProducer uses sarama to produce metrics messages to kafka
type kafkaMetricsProducer struct {
	cfg       Config
	producer  sarama.SyncProducer
	marshaler MetricsMarshaler
	logger    *zap.Logger
}

func (e *kafkaMetricsProducer) metricsDataPusher(_ context.Context, md pmetric.Metrics) error {
	messages, err := e.marshaler.Marshal(md, getTopic(&e.cfg, md.ResourceMetrics()))
	if err != nil {
		return consumererror.NewPermanent(err)
	}
	err = e.producer.SendMessages(messages)
	if err != nil {
		var prodErr sarama.ProducerErrors
		if errors.As(err, &prodErr) {
			if len(prodErr) > 0 {
				return kafkaErrors{len(prodErr), prodErr[0].Err.Error()}
			}
		}
		return err
	}
	return nil
}

func (e *kafkaMetricsProducer) Close(context.Context) error {
	if e.producer == nil {
		return nil
	}
	return e.producer.Close()
}

func (e *kafkaMetricsProducer) start(_ context.Context, _ component.Host) error {
	producer, err := newSaramaProducer(e.cfg)
	if err != nil {
		return err
	}
	e.producer = producer
	return nil
}

// kafkaLogsProducer uses sarama to produce logs messages to kafka
type kafkaLogsProducer struct {
	cfg       Config
	producer  sarama.SyncProducer
	marshaler LogsMarshaler
	logger    *zap.Logger
}

func (e *kafkaLogsProducer) logsDataPusher(_ context.Context, ld plog.Logs) error {
	messages, err := e.marshaler.Marshal(ld, getTopic(&e.cfg, ld.ResourceLogs()))
	if err != nil {
		return consumererror.NewPermanent(err)
	}
	err = e.producer.SendMessages(messages)
	if err != nil {
		var prodErr sarama.ProducerErrors
		if errors.As(err, &prodErr) {
			if len(prodErr) > 0 {
				return kafkaErrors{len(prodErr), prodErr[0].Err.Error()}
			}
		}
		return err
	}
	return nil
}

func (e *kafkaLogsProducer) Close(context.Context) error {
	if e.producer == nil {
		return nil
	}
	return e.producer.Close()
}

func (e *kafkaLogsProducer) start(_ context.Context, _ component.Host) error {
	producer, err := newSaramaProducer(e.cfg)
	if err != nil {
		return err
	}
	e.producer = producer
	return nil
}

func newSaramaProducer(config Config) (sarama.SyncProducer, error) {
	c := sarama.NewConfig()

	c.ClientID = config.ClientID

	// These setting are required by the sarama.SyncProducer implementation.
	c.Producer.Return.Successes = true
	c.Producer.Return.Errors = true
	c.Producer.RequiredAcks = config.Producer.RequiredAcks
	// Because sarama does not accept a Context for every message, set the Timeout here.
	c.Producer.Timeout = config.Timeout
	c.Metadata.Full = config.Metadata.Full
	c.Metadata.Retry.Max = config.Metadata.Retry.Max
	c.Metadata.Retry.Backoff = config.Metadata.Retry.Backoff
	c.Producer.MaxMessageBytes = config.Producer.MaxMessageBytes
	c.Producer.Flush.MaxMessages = config.Producer.FlushMaxMessages

	if config.ResolveCanonicalBootstrapServersOnly {
		c.Net.ResolveCanonicalBootstrapServers = true
	}

	if config.ProtocolVersion != "" {
		version, err := sarama.ParseKafkaVersion(config.ProtocolVersion)
		if err != nil {
			return nil, err
		}
		c.Version = version
	}

	if err := kafka.ConfigureAuthentication(config.Authentication, c); err != nil {
		return nil, err
	}

	compression, err := saramaProducerCompressionCodec(config.Producer.Compression)
	if err != nil {
		return nil, err
	}
	c.Producer.Compression = compression

	producer, err := sarama.NewSyncProducer(config.Brokers, c)
	if err != nil {
		return nil, err
	}
	return producer, nil
}

func newMetricsExporter(config Config, set exporter.CreateSettings, marshalers map[string]MetricsMarshaler) (*kafkaMetricsProducer, error) {
	marshaler := marshalers[config.Encoding]
	if marshaler == nil {
		return nil, errUnrecognizedEncoding
	}
	if config.PartitionMetricsByResourceAttributes {
		if keyableMarshaler, ok := marshaler.(KeyableMetricsMarshaler); ok {
			keyableMarshaler.Key()
		}
	}

	return &kafkaMetricsProducer{
		cfg:       config,
		marshaler: marshaler,
		logger:    set.Logger,
	}, nil

}

// newTracesExporter creates Kafka exporter.
func newTracesExporter(config Config, set exporter.CreateSettings, marshalers map[string]TracesMarshaler) (*kafkaTracesProducer, error) {
	marshaler := marshalers[config.Encoding]
	if marshaler == nil {
		return nil, errUnrecognizedEncoding
	}
	if config.PartitionTracesByID {
		if keyableMarshaler, ok := marshaler.(KeyableTracesMarshaler); ok {
			keyableMarshaler.Key()
		}
	}

	return &kafkaTracesProducer{
		cfg:       config,
		marshaler: marshaler,
		logger:    set.Logger,
	}, nil
}

func newLogsExporter(config Config, set exporter.CreateSettings, marshalers map[string]LogsMarshaler) (*kafkaLogsProducer, error) {
	marshaler := marshalers[config.Encoding]
	if marshaler == nil {
		return nil, errUnrecognizedEncoding
	}

	return &kafkaLogsProducer{
		cfg:       config,
		marshaler: marshaler,
		logger:    set.Logger,
	}, nil

}

type resourceSlice[T any] interface {
	Len() int
	At(int) T
}

type resource interface {
	Resource() pcommon.Resource
}

func getTopic[T resource](cfg *Config, resources resourceSlice[T]) string {
	if cfg.TopicFromAttribute == "" {
		return cfg.Topic
	}
	for i := 0; i < resources.Len(); i++ {
		rv, ok := resources.At(i).Resource().Attributes().Get(cfg.TopicFromAttribute)
		if ok && rv.Str() != "" {
			return rv.Str()
		}
	}
	return cfg.Topic
}
