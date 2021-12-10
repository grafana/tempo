// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"context"
	"fmt"
	"sync"

	"github.com/Shopify/sarama"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/obsreport"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/kafkaexporter"
)

const (
	transport = "kafka"
)

var errUnrecognizedEncoding = fmt.Errorf("unrecognized encoding")

// kafkaTracesConsumer uses sarama to consume and handle messages from kafka.
type kafkaTracesConsumer struct {
	id                config.ComponentID
	consumerGroup     sarama.ConsumerGroup
	nextConsumer      consumer.Traces
	topics            []string
	cancelConsumeLoop context.CancelFunc
	unmarshaler       TracesUnmarshaler

	settings component.ReceiverCreateSettings

	autocommitEnabled bool
	messageMarking    MessageMarking
}

// kafkaMetricsConsumer uses sarama to consume and handle messages from kafka.
type kafkaMetricsConsumer struct {
	id                config.ComponentID
	consumerGroup     sarama.ConsumerGroup
	nextConsumer      consumer.Metrics
	topics            []string
	cancelConsumeLoop context.CancelFunc
	unmarshaler       MetricsUnmarshaler

	settings component.ReceiverCreateSettings

	autocommitEnabled bool
	messageMarking    MessageMarking
}

// kafkaLogsConsumer uses sarama to consume and handle messages from kafka.
type kafkaLogsConsumer struct {
	id                config.ComponentID
	consumerGroup     sarama.ConsumerGroup
	nextConsumer      consumer.Logs
	topics            []string
	cancelConsumeLoop context.CancelFunc
	unmarshaler       LogsUnmarshaler

	settings component.ReceiverCreateSettings

	autocommitEnabled bool
	messageMarking    MessageMarking
}

var _ component.Receiver = (*kafkaTracesConsumer)(nil)
var _ component.Receiver = (*kafkaMetricsConsumer)(nil)
var _ component.Receiver = (*kafkaLogsConsumer)(nil)

func newTracesReceiver(config Config, set component.ReceiverCreateSettings, unmarshalers map[string]TracesUnmarshaler, nextConsumer consumer.Traces) (*kafkaTracesConsumer, error) {
	unmarshaler := unmarshalers[config.Encoding]
	if unmarshaler == nil {
		return nil, errUnrecognizedEncoding
	}

	c := sarama.NewConfig()
	c.ClientID = config.ClientID
	c.Metadata.Full = config.Metadata.Full
	c.Metadata.Retry.Max = config.Metadata.Retry.Max
	c.Metadata.Retry.Backoff = config.Metadata.Retry.Backoff
	if config.ProtocolVersion != "" {
		version, err := sarama.ParseKafkaVersion(config.ProtocolVersion)
		if err != nil {
			return nil, err
		}
		c.Version = version
	}
	if err := kafkaexporter.ConfigureAuthentication(config.Authentication, c); err != nil {
		return nil, err
	}
	client, err := sarama.NewConsumerGroup(config.Brokers, config.GroupID, c)
	if err != nil {
		return nil, err
	}
	return &kafkaTracesConsumer{
		id:                config.ID(),
		consumerGroup:     client,
		topics:            []string{config.Topic},
		nextConsumer:      nextConsumer,
		unmarshaler:       unmarshaler,
		settings:          set,
		autocommitEnabled: config.AutoCommit.Enable,
		messageMarking:    config.MessageMarking,
	}, nil
}

func (c *kafkaTracesConsumer) Start(context.Context, component.Host) error {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelConsumeLoop = cancel
	consumerGroup := &tracesConsumerGroupHandler{
		id:           c.id,
		logger:       c.settings.Logger,
		unmarshaler:  c.unmarshaler,
		nextConsumer: c.nextConsumer,
		ready:        make(chan bool),
		obsrecv: obsreport.NewReceiver(obsreport.ReceiverSettings{
			ReceiverID:             c.id,
			Transport:              transport,
			ReceiverCreateSettings: c.settings,
		}),
		autocommitEnabled: c.autocommitEnabled,
		messageMarking:    c.messageMarking,
	}
	go c.consumeLoop(ctx, consumerGroup) // nolint:errcheck
	<-consumerGroup.ready
	return nil
}

func (c *kafkaTracesConsumer) consumeLoop(ctx context.Context, handler sarama.ConsumerGroupHandler) error {
	for {
		// `Consume` should be called inside an infinite loop, when a
		// server-side rebalance happens, the consumer session will need to be
		// recreated to get the new claims
		if err := c.consumerGroup.Consume(ctx, c.topics, handler); err != nil {
			c.settings.Logger.Error("Error from consumer", zap.Error(err))
		}
		// check if context was cancelled, signaling that the consumer should stop
		if ctx.Err() != nil {
			c.settings.Logger.Info("Consumer stopped", zap.Error(ctx.Err()))
			return ctx.Err()
		}
	}
}

func (c *kafkaTracesConsumer) Shutdown(context.Context) error {
	c.cancelConsumeLoop()
	return c.consumerGroup.Close()
}

func newMetricsReceiver(config Config, set component.ReceiverCreateSettings, unmarshalers map[string]MetricsUnmarshaler, nextConsumer consumer.Metrics) (*kafkaMetricsConsumer, error) {
	unmarshaler := unmarshalers[config.Encoding]
	if unmarshaler == nil {
		return nil, errUnrecognizedEncoding
	}

	c := sarama.NewConfig()
	c.ClientID = config.ClientID
	c.Metadata.Full = config.Metadata.Full
	c.Metadata.Retry.Max = config.Metadata.Retry.Max
	c.Metadata.Retry.Backoff = config.Metadata.Retry.Backoff
	c.Consumer.Offsets.AutoCommit.Enable = config.AutoCommit.Enable
	c.Consumer.Offsets.AutoCommit.Interval = config.AutoCommit.Interval

	if config.ProtocolVersion != "" {
		version, err := sarama.ParseKafkaVersion(config.ProtocolVersion)
		if err != nil {
			return nil, err
		}
		c.Version = version
	}
	if err := kafkaexporter.ConfigureAuthentication(config.Authentication, c); err != nil {
		return nil, err
	}
	client, err := sarama.NewConsumerGroup(config.Brokers, config.GroupID, c)
	if err != nil {
		return nil, err
	}
	return &kafkaMetricsConsumer{
		id:                config.ID(),
		consumerGroup:     client,
		topics:            []string{config.Topic},
		nextConsumer:      nextConsumer,
		unmarshaler:       unmarshaler,
		settings:          set,
		autocommitEnabled: config.AutoCommit.Enable,
		messageMarking:    config.MessageMarking,
	}, nil
}

func (c *kafkaMetricsConsumer) Start(context.Context, component.Host) error {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelConsumeLoop = cancel
	metricsConsumerGroup := &metricsConsumerGroupHandler{
		id:           c.id,
		logger:       c.settings.Logger,
		unmarshaler:  c.unmarshaler,
		nextConsumer: c.nextConsumer,
		ready:        make(chan bool),
		obsrecv: obsreport.NewReceiver(obsreport.ReceiverSettings{
			ReceiverID:             c.id,
			Transport:              transport,
			ReceiverCreateSettings: c.settings,
		}),
		autocommitEnabled: c.autocommitEnabled,
		messageMarking:    c.messageMarking,
	}
	go c.consumeLoop(ctx, metricsConsumerGroup)
	<-metricsConsumerGroup.ready
	return nil
}

func (c *kafkaMetricsConsumer) consumeLoop(ctx context.Context, handler sarama.ConsumerGroupHandler) error {
	for {
		// `Consume` should be called inside an infinite loop, when a
		// server-side rebalance happens, the consumer session will need to be
		// recreated to get the new claims
		if err := c.consumerGroup.Consume(ctx, c.topics, handler); err != nil {
			c.settings.Logger.Error("Error from consumer", zap.Error(err))
		}
		// check if context was cancelled, signaling that the consumer should stop
		if ctx.Err() != nil {
			c.settings.Logger.Info("Consumer stopped", zap.Error(ctx.Err()))
			return ctx.Err()
		}
	}
}
func (c *kafkaMetricsConsumer) Shutdown(context.Context) error {
	c.cancelConsumeLoop()
	return c.consumerGroup.Close()
}

func newLogsReceiver(config Config, set component.ReceiverCreateSettings, unmarshalers map[string]LogsUnmarshaler, nextConsumer consumer.Logs) (*kafkaLogsConsumer, error) {
	unmarshaler := unmarshalers[config.Encoding]
	if unmarshaler == nil {
		return nil, errUnrecognizedEncoding
	}

	c := sarama.NewConfig()
	c.ClientID = config.ClientID
	c.Metadata.Full = config.Metadata.Full
	c.Metadata.Retry.Max = config.Metadata.Retry.Max
	c.Metadata.Retry.Backoff = config.Metadata.Retry.Backoff
	if config.ProtocolVersion != "" {
		version, err := sarama.ParseKafkaVersion(config.ProtocolVersion)
		if err != nil {
			return nil, err
		}
		c.Version = version
	}
	if err := kafkaexporter.ConfigureAuthentication(config.Authentication, c); err != nil {
		return nil, err
	}
	client, err := sarama.NewConsumerGroup(config.Brokers, config.GroupID, c)
	if err != nil {
		return nil, err
	}
	return &kafkaLogsConsumer{
		id:                config.ID(),
		consumerGroup:     client,
		topics:            []string{config.Topic},
		nextConsumer:      nextConsumer,
		unmarshaler:       unmarshaler,
		settings:          set,
		autocommitEnabled: config.AutoCommit.Enable,
		messageMarking:    config.MessageMarking,
	}, nil
}

func (c *kafkaLogsConsumer) Start(context.Context, component.Host) error {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelConsumeLoop = cancel
	logsConsumerGroup := &logsConsumerGroupHandler{
		id:           c.id,
		logger:       c.settings.Logger,
		unmarshaler:  c.unmarshaler,
		nextConsumer: c.nextConsumer,
		ready:        make(chan bool),
		obsrecv: obsreport.NewReceiver(obsreport.ReceiverSettings{
			ReceiverID:             c.id,
			Transport:              transport,
			ReceiverCreateSettings: c.settings,
		}),
		autocommitEnabled: c.autocommitEnabled,
		messageMarking:    c.messageMarking,
	}
	go c.consumeLoop(ctx, logsConsumerGroup)
	<-logsConsumerGroup.ready
	return nil
}

func (c *kafkaLogsConsumer) consumeLoop(ctx context.Context, handler sarama.ConsumerGroupHandler) error {
	for {
		// `Consume` should be called inside an infinite loop, when a
		// server-side rebalance happens, the consumer session will need to be
		// recreated to get the new claims
		if err := c.consumerGroup.Consume(ctx, c.topics, handler); err != nil {
			c.settings.Logger.Error("Error from consumer", zap.Error(err))
		}
		// check if context was cancelled, signaling that the consumer should stop
		if ctx.Err() != nil {
			c.settings.Logger.Info("Consumer stopped", zap.Error(ctx.Err()))
			return ctx.Err()
		}
	}
}

func (c *kafkaLogsConsumer) Shutdown(context.Context) error {
	c.cancelConsumeLoop()
	return c.consumerGroup.Close()
}

type tracesConsumerGroupHandler struct {
	id           config.ComponentID
	unmarshaler  TracesUnmarshaler
	nextConsumer consumer.Traces
	ready        chan bool
	readyCloser  sync.Once

	logger *zap.Logger

	obsrecv *obsreport.Receiver

	autocommitEnabled bool
	messageMarking    MessageMarking
}

type metricsConsumerGroupHandler struct {
	id           config.ComponentID
	unmarshaler  MetricsUnmarshaler
	nextConsumer consumer.Metrics
	ready        chan bool
	readyCloser  sync.Once

	logger *zap.Logger

	obsrecv *obsreport.Receiver

	autocommitEnabled bool
	messageMarking    MessageMarking
}

type logsConsumerGroupHandler struct {
	id           config.ComponentID
	unmarshaler  LogsUnmarshaler
	nextConsumer consumer.Logs
	ready        chan bool
	readyCloser  sync.Once

	logger *zap.Logger

	obsrecv *obsreport.Receiver

	autocommitEnabled bool
	messageMarking    MessageMarking
}

var _ sarama.ConsumerGroupHandler = (*tracesConsumerGroupHandler)(nil)
var _ sarama.ConsumerGroupHandler = (*metricsConsumerGroupHandler)(nil)
var _ sarama.ConsumerGroupHandler = (*logsConsumerGroupHandler)(nil)

func (c *tracesConsumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	c.readyCloser.Do(func() {
		close(c.ready)
	})
	statsTags := []tag.Mutator{tag.Insert(tagInstanceName, c.id.Name())}
	_ = stats.RecordWithTags(session.Context(), statsTags, statPartitionStart.M(1))
	return nil
}

func (c *tracesConsumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	statsTags := []tag.Mutator{tag.Insert(tagInstanceName, c.id.Name())}
	_ = stats.RecordWithTags(session.Context(), statsTags, statPartitionClose.M(1))
	return nil
}

func (c *tracesConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	c.logger.Info("Starting consumer group", zap.Int32("partition", claim.Partition()))
	if !c.autocommitEnabled {
		defer session.Commit()
	}
	for message := range claim.Messages() {
		c.logger.Debug("Kafka message claimed",
			zap.String("value", string(message.Value)),
			zap.Time("timestamp", message.Timestamp),
			zap.String("topic", message.Topic))
		if !c.messageMarking.After {
			session.MarkMessage(message, "")
		}

		ctx := c.obsrecv.StartTracesOp(session.Context())
		statsTags := []tag.Mutator{tag.Insert(tagInstanceName, c.id.String())}
		_ = stats.RecordWithTags(ctx, statsTags,
			statMessageCount.M(1),
			statMessageOffset.M(message.Offset),
			statMessageOffsetLag.M(claim.HighWaterMarkOffset()-message.Offset-1))

		traces, err := c.unmarshaler.Unmarshal(message.Value)
		if err != nil {
			c.logger.Error("failed to unmarshal message", zap.Error(err))
			if c.messageMarking.After && c.messageMarking.OnError {
				session.MarkMessage(message, "")
			}
			return err
		}

		spanCount := traces.SpanCount()
		err = c.nextConsumer.ConsumeTraces(session.Context(), traces)
		c.obsrecv.EndTracesOp(ctx, c.unmarshaler.Encoding(), spanCount, err)
		if err != nil {
			if c.messageMarking.After && c.messageMarking.OnError {
				session.MarkMessage(message, "")
			}
			return err
		}
		if c.messageMarking.After {
			session.MarkMessage(message, "")
		}
		if !c.autocommitEnabled {
			session.Commit()
		}
	}
	return nil
}

func (c *metricsConsumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	c.readyCloser.Do(func() {
		close(c.ready)
	})
	statsTags := []tag.Mutator{tag.Insert(tagInstanceName, c.id.Name())}
	_ = stats.RecordWithTags(session.Context(), statsTags, statPartitionStart.M(1))
	return nil
}

func (c *metricsConsumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	statsTags := []tag.Mutator{tag.Insert(tagInstanceName, c.id.Name())}
	_ = stats.RecordWithTags(session.Context(), statsTags, statPartitionClose.M(1))
	return nil
}

func (c *metricsConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	c.logger.Info("Starting consumer group", zap.Int32("partition", claim.Partition()))
	if !c.autocommitEnabled {
		defer session.Commit()
	}

	for message := range claim.Messages() {
		c.logger.Debug("Kafka message claimed",
			zap.String("value", string(message.Value)),
			zap.Time("timestamp", message.Timestamp),
			zap.String("topic", message.Topic))
		if !c.messageMarking.After {
			session.MarkMessage(message, "")
		}

		ctx := c.obsrecv.StartMetricsOp(session.Context())
		statsTags := []tag.Mutator{tag.Insert(tagInstanceName, c.id.String())}
		_ = stats.RecordWithTags(ctx, statsTags,
			statMessageCount.M(1),
			statMessageOffset.M(message.Offset),
			statMessageOffsetLag.M(claim.HighWaterMarkOffset()-message.Offset-1))

		metrics, err := c.unmarshaler.Unmarshal(message.Value)
		if err != nil {
			c.logger.Error("failed to unmarshal message", zap.Error(err))
			if c.messageMarking.After && c.messageMarking.OnError {
				session.MarkMessage(message, "")
			}
			return err
		}

		dataPointCount := metrics.DataPointCount()
		err = c.nextConsumer.ConsumeMetrics(session.Context(), metrics)
		c.obsrecv.EndMetricsOp(ctx, c.unmarshaler.Encoding(), dataPointCount, err)
		if err != nil {
			if c.messageMarking.After && c.messageMarking.OnError {
				session.MarkMessage(message, "")
			}
			return err
		}
		if c.messageMarking.After {
			session.MarkMessage(message, "")
		}
		if !c.autocommitEnabled {
			session.Commit()
		}
	}
	return nil
}

func (c *logsConsumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	c.readyCloser.Do(func() {
		close(c.ready)
	})
	_ = stats.RecordWithTags(
		session.Context(),
		[]tag.Mutator{tag.Insert(tagInstanceName, c.id.String())},
		statPartitionStart.M(1))
	return nil
}

func (c *logsConsumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	_ = stats.RecordWithTags(
		session.Context(),
		[]tag.Mutator{tag.Insert(tagInstanceName, c.id.String())},
		statPartitionClose.M(1))
	return nil
}

func (c *logsConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	c.logger.Info("Starting consumer group", zap.Int32("partition", claim.Partition()))
	if !c.autocommitEnabled {
		defer session.Commit()
	}
	for message := range claim.Messages() {
		c.logger.Debug("Kafka message claimed",
			zap.String("value", string(message.Value)),
			zap.Time("timestamp", message.Timestamp),
			zap.String("topic", message.Topic))
		if !c.messageMarking.After {
			session.MarkMessage(message, "")
		}

		ctx := c.obsrecv.StartTracesOp(session.Context())
		_ = stats.RecordWithTags(
			ctx,
			[]tag.Mutator{tag.Insert(tagInstanceName, c.id.String())},
			statMessageCount.M(1),
			statMessageOffset.M(message.Offset),
			statMessageOffsetLag.M(claim.HighWaterMarkOffset()-message.Offset-1))

		logs, err := c.unmarshaler.Unmarshal(message.Value)
		if err != nil {
			c.logger.Error("failed to unmarshal message", zap.Error(err))
			if c.messageMarking.After && c.messageMarking.OnError {
				session.MarkMessage(message, "")
			}
			return err
		}

		err = c.nextConsumer.ConsumeLogs(session.Context(), logs)
		// TODO
		c.obsrecv.EndTracesOp(ctx, c.unmarshaler.Encoding(), logs.LogRecordCount(), err)
		if err != nil {
			if c.messageMarking.After && c.messageMarking.OnError {
				session.MarkMessage(message, "")
			}
			return err
		}
		if c.messageMarking.After {
			session.MarkMessage(message, "")
		}
		if !c.autocommitEnabled {
			session.Commit()
		}
	}
	return nil
}
