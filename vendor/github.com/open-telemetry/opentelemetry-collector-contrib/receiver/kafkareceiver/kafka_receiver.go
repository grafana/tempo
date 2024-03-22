// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/IBM/sarama"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka"
)

const (
	transport = "kafka"
)

var errInvalidInitialOffset = fmt.Errorf("invalid initial offset")

// kafkaTracesConsumer uses sarama to consume and handle messages from kafka.
type kafkaTracesConsumer struct {
	config            Config
	consumerGroup     sarama.ConsumerGroup
	nextConsumer      consumer.Traces
	topics            []string
	cancelConsumeLoop context.CancelFunc
	unmarshaler       TracesUnmarshaler

	settings receiver.CreateSettings

	autocommitEnabled bool
	messageMarking    MessageMarking
	headerExtraction  bool
	headers           []string
}

// kafkaMetricsConsumer uses sarama to consume and handle messages from kafka.
type kafkaMetricsConsumer struct {
	config            Config
	consumerGroup     sarama.ConsumerGroup
	nextConsumer      consumer.Metrics
	topics            []string
	cancelConsumeLoop context.CancelFunc
	unmarshaler       MetricsUnmarshaler

	settings receiver.CreateSettings

	autocommitEnabled bool
	messageMarking    MessageMarking
	headerExtraction  bool
	headers           []string
}

// kafkaLogsConsumer uses sarama to consume and handle messages from kafka.
type kafkaLogsConsumer struct {
	config            Config
	consumerGroup     sarama.ConsumerGroup
	nextConsumer      consumer.Logs
	topics            []string
	cancelConsumeLoop context.CancelFunc
	unmarshaler       LogsUnmarshaler

	settings receiver.CreateSettings

	autocommitEnabled bool
	messageMarking    MessageMarking
	headerExtraction  bool
	headers           []string
}

var _ receiver.Traces = (*kafkaTracesConsumer)(nil)
var _ receiver.Metrics = (*kafkaMetricsConsumer)(nil)
var _ receiver.Logs = (*kafkaLogsConsumer)(nil)

func newTracesReceiver(config Config, set receiver.CreateSettings, unmarshaler TracesUnmarshaler, nextConsumer consumer.Traces) (*kafkaTracesConsumer, error) {
	if unmarshaler == nil {
		return nil, errUnrecognizedEncoding
	}

	return &kafkaTracesConsumer{
		config:            config,
		topics:            []string{config.Topic},
		nextConsumer:      nextConsumer,
		unmarshaler:       unmarshaler,
		settings:          set,
		autocommitEnabled: config.AutoCommit.Enable,
		messageMarking:    config.MessageMarking,
		headerExtraction:  config.HeaderExtraction.ExtractHeaders,
		headers:           config.HeaderExtraction.Headers,
	}, nil
}

func createKafkaClient(config Config) (sarama.ConsumerGroup, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.ClientID = config.ClientID
	saramaConfig.Metadata.Full = config.Metadata.Full
	saramaConfig.Metadata.Retry.Max = config.Metadata.Retry.Max
	saramaConfig.Metadata.Retry.Backoff = config.Metadata.Retry.Backoff
	saramaConfig.Consumer.Offsets.AutoCommit.Enable = config.AutoCommit.Enable
	saramaConfig.Consumer.Offsets.AutoCommit.Interval = config.AutoCommit.Interval
	var err error
	if saramaConfig.Consumer.Offsets.Initial, err = toSaramaInitialOffset(config.InitialOffset); err != nil {
		return nil, err
	}
	if config.ResolveCanonicalBootstrapServersOnly {
		saramaConfig.Net.ResolveCanonicalBootstrapServers = true
	}
	if config.ProtocolVersion != "" {
		if saramaConfig.Version, err = sarama.ParseKafkaVersion(config.ProtocolVersion); err != nil {
			return nil, err
		}
	}
	if err := kafka.ConfigureAuthentication(config.Authentication, saramaConfig); err != nil {
		return nil, err
	}
	return sarama.NewConsumerGroup(config.Brokers, config.GroupID, saramaConfig)
}

func (c *kafkaTracesConsumer) Start(_ context.Context, _ component.Host) error {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelConsumeLoop = cancel
	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             c.settings.ID,
		Transport:              transport,
		ReceiverCreateSettings: c.settings,
	})
	if err != nil {
		return err
	}
	// consumerGroup may be set in tests to inject fake implementation.
	if c.consumerGroup == nil {
		if c.consumerGroup, err = createKafkaClient(c.config); err != nil {
			return err
		}
	}
	consumerGroup := &tracesConsumerGroupHandler{
		logger:            c.settings.Logger,
		unmarshaler:       c.unmarshaler,
		nextConsumer:      c.nextConsumer,
		ready:             make(chan bool),
		obsrecv:           obsrecv,
		autocommitEnabled: c.autocommitEnabled,
		messageMarking:    c.messageMarking,
		headerExtractor:   &nopHeaderExtractor{},
	}
	if c.headerExtraction {
		consumerGroup.headerExtractor = &headerExtractor{
			logger:  c.settings.Logger,
			headers: c.headers,
		}
	}
	go func() {
		if err := c.consumeLoop(ctx, consumerGroup); err != nil {
			c.settings.ReportStatus(component.NewFatalErrorEvent(err))
		}
	}()
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
	if c.cancelConsumeLoop == nil {
		return nil
	}
	c.cancelConsumeLoop()
	return c.consumerGroup.Close()
}

func newMetricsReceiver(config Config, set receiver.CreateSettings, unmarshaler MetricsUnmarshaler, nextConsumer consumer.Metrics) (*kafkaMetricsConsumer, error) {
	if unmarshaler == nil {
		return nil, errUnrecognizedEncoding
	}

	return &kafkaMetricsConsumer{
		config:            config,
		topics:            []string{config.Topic},
		nextConsumer:      nextConsumer,
		unmarshaler:       unmarshaler,
		settings:          set,
		autocommitEnabled: config.AutoCommit.Enable,
		messageMarking:    config.MessageMarking,
		headerExtraction:  config.HeaderExtraction.ExtractHeaders,
		headers:           config.HeaderExtraction.Headers,
	}, nil
}

func (c *kafkaMetricsConsumer) Start(_ context.Context, _ component.Host) error {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelConsumeLoop = cancel
	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             c.settings.ID,
		Transport:              transport,
		ReceiverCreateSettings: c.settings,
	})
	if err != nil {
		return err
	}
	// consumerGroup may be set in tests to inject fake implementation.
	if c.consumerGroup == nil {
		if c.consumerGroup, err = createKafkaClient(c.config); err != nil {
			return err
		}
	}
	metricsConsumerGroup := &metricsConsumerGroupHandler{
		logger:            c.settings.Logger,
		unmarshaler:       c.unmarshaler,
		nextConsumer:      c.nextConsumer,
		ready:             make(chan bool),
		obsrecv:           obsrecv,
		autocommitEnabled: c.autocommitEnabled,
		messageMarking:    c.messageMarking,
		headerExtractor:   &nopHeaderExtractor{},
	}
	if c.headerExtraction {
		metricsConsumerGroup.headerExtractor = &headerExtractor{
			logger:  c.settings.Logger,
			headers: c.headers,
		}
	}
	go func() {
		if err := c.consumeLoop(ctx, metricsConsumerGroup); err != nil {
			c.settings.ReportStatus(component.NewFatalErrorEvent(err))
		}
	}()
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
	if c.cancelConsumeLoop == nil {
		return nil
	}
	c.cancelConsumeLoop()
	return c.consumerGroup.Close()
}

func newLogsReceiver(config Config, set receiver.CreateSettings, unmarshaler LogsUnmarshaler, nextConsumer consumer.Logs) (*kafkaLogsConsumer, error) {
	if unmarshaler == nil {
		return nil, errUnrecognizedEncoding
	}

	return &kafkaLogsConsumer{
		config:            config,
		topics:            []string{config.Topic},
		nextConsumer:      nextConsumer,
		unmarshaler:       unmarshaler,
		settings:          set,
		autocommitEnabled: config.AutoCommit.Enable,
		messageMarking:    config.MessageMarking,
		headerExtraction:  config.HeaderExtraction.ExtractHeaders,
		headers:           config.HeaderExtraction.Headers,
	}, nil
}

func (c *kafkaLogsConsumer) Start(_ context.Context, _ component.Host) error {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelConsumeLoop = cancel
	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             c.settings.ID,
		Transport:              transport,
		ReceiverCreateSettings: c.settings,
	})
	if err != nil {
		return err
	}
	// consumerGroup may be set in tests to inject fake implementation.
	if c.consumerGroup == nil {
		if c.consumerGroup, err = createKafkaClient(c.config); err != nil {
			return err
		}
	}
	logsConsumerGroup := &logsConsumerGroupHandler{
		logger:            c.settings.Logger,
		unmarshaler:       c.unmarshaler,
		nextConsumer:      c.nextConsumer,
		ready:             make(chan bool),
		obsrecv:           obsrecv,
		autocommitEnabled: c.autocommitEnabled,
		messageMarking:    c.messageMarking,
		headerExtractor:   &nopHeaderExtractor{},
	}
	if c.headerExtraction {
		logsConsumerGroup.headerExtractor = &headerExtractor{
			logger:  c.settings.Logger,
			headers: c.headers,
		}
	}
	go func() {
		if err := c.consumeLoop(ctx, logsConsumerGroup); err != nil {
			c.settings.ReportStatus(component.NewFatalErrorEvent(err))
		}
	}()
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
	if c.cancelConsumeLoop == nil {
		return nil
	}
	c.cancelConsumeLoop()
	return c.consumerGroup.Close()
}

type tracesConsumerGroupHandler struct {
	id           component.ID
	unmarshaler  TracesUnmarshaler
	nextConsumer consumer.Traces
	ready        chan bool
	readyCloser  sync.Once

	logger *zap.Logger

	obsrecv *receiverhelper.ObsReport

	autocommitEnabled bool
	messageMarking    MessageMarking
	headerExtractor   HeaderExtractor
}

type metricsConsumerGroupHandler struct {
	id           component.ID
	unmarshaler  MetricsUnmarshaler
	nextConsumer consumer.Metrics
	ready        chan bool
	readyCloser  sync.Once

	logger *zap.Logger

	obsrecv *receiverhelper.ObsReport

	autocommitEnabled bool
	messageMarking    MessageMarking
	headerExtractor   HeaderExtractor
}

type logsConsumerGroupHandler struct {
	id           component.ID
	unmarshaler  LogsUnmarshaler
	nextConsumer consumer.Logs
	ready        chan bool
	readyCloser  sync.Once

	logger *zap.Logger

	obsrecv *receiverhelper.ObsReport

	autocommitEnabled bool
	messageMarking    MessageMarking
	headerExtractor   HeaderExtractor
}

var _ sarama.ConsumerGroupHandler = (*tracesConsumerGroupHandler)(nil)
var _ sarama.ConsumerGroupHandler = (*metricsConsumerGroupHandler)(nil)
var _ sarama.ConsumerGroupHandler = (*logsConsumerGroupHandler)(nil)

func (c *tracesConsumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	c.readyCloser.Do(func() {
		close(c.ready)
	})
	statsTags := []tag.Mutator{tag.Upsert(tagInstanceName, c.id.Name())}
	_ = stats.RecordWithTags(session.Context(), statsTags, statPartitionStart.M(1))
	return nil
}

func (c *tracesConsumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	statsTags := []tag.Mutator{tag.Upsert(tagInstanceName, c.id.Name())}
	_ = stats.RecordWithTags(session.Context(), statsTags, statPartitionClose.M(1))
	return nil
}

func (c *tracesConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	c.logger.Info("Starting consumer group", zap.Int32("partition", claim.Partition()))
	if !c.autocommitEnabled {
		defer session.Commit()
	}
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			c.logger.Debug("Kafka message claimed",
				zap.String("value", string(message.Value)),
				zap.Time("timestamp", message.Timestamp),
				zap.String("topic", message.Topic))
			if !c.messageMarking.After {
				session.MarkMessage(message, "")
			}

			ctx := c.obsrecv.StartTracesOp(session.Context())
			statsTags := []tag.Mutator{
				tag.Upsert(tagInstanceName, c.id.String()),
				tag.Upsert(tagPartition, strconv.Itoa(int(claim.Partition()))),
			}
			_ = stats.RecordWithTags(ctx, statsTags,
				statMessageCount.M(1),
				statMessageOffset.M(message.Offset),
				statMessageOffsetLag.M(claim.HighWaterMarkOffset()-message.Offset-1))

			traces, err := c.unmarshaler.Unmarshal(message.Value)
			if err != nil {
				c.logger.Error("failed to unmarshal message", zap.Error(err))
				_ = stats.RecordWithTags(
					ctx,
					[]tag.Mutator{tag.Upsert(tagInstanceName, c.id.String())},
					statUnmarshalFailedSpans.M(1))
				if c.messageMarking.After && c.messageMarking.OnError {
					session.MarkMessage(message, "")
				}
				return err
			}

			c.headerExtractor.extractHeadersTraces(traces, message)
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

		// Should return when `session.Context()` is done.
		// If not, will raise `ErrRebalanceInProgress` or `read tcp <ip>:<port>: i/o timeout` when kafka rebalance. see:
		// https://github.com/IBM/sarama/issues/1192
		case <-session.Context().Done():
			return nil
		}
	}
}

func (c *metricsConsumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	c.readyCloser.Do(func() {
		close(c.ready)
	})
	statsTags := []tag.Mutator{tag.Upsert(tagInstanceName, c.id.Name())}
	_ = stats.RecordWithTags(session.Context(), statsTags, statPartitionStart.M(1))
	return nil
}

func (c *metricsConsumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	statsTags := []tag.Mutator{tag.Upsert(tagInstanceName, c.id.Name())}
	_ = stats.RecordWithTags(session.Context(), statsTags, statPartitionClose.M(1))
	return nil
}

func (c *metricsConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	c.logger.Info("Starting consumer group", zap.Int32("partition", claim.Partition()))
	if !c.autocommitEnabled {
		defer session.Commit()
	}
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			c.logger.Debug("Kafka message claimed",
				zap.String("value", string(message.Value)),
				zap.Time("timestamp", message.Timestamp),
				zap.String("topic", message.Topic))
			if !c.messageMarking.After {
				session.MarkMessage(message, "")
			}

			ctx := c.obsrecv.StartMetricsOp(session.Context())
			statsTags := []tag.Mutator{
				tag.Upsert(tagInstanceName, c.id.String()),
				tag.Upsert(tagPartition, strconv.Itoa(int(claim.Partition()))),
			}
			_ = stats.RecordWithTags(ctx, statsTags,
				statMessageCount.M(1),
				statMessageOffset.M(message.Offset),
				statMessageOffsetLag.M(claim.HighWaterMarkOffset()-message.Offset-1))

			metrics, err := c.unmarshaler.Unmarshal(message.Value)
			if err != nil {
				c.logger.Error("failed to unmarshal message", zap.Error(err))
				_ = stats.RecordWithTags(
					ctx,
					[]tag.Mutator{tag.Upsert(tagInstanceName, c.id.String())},
					statUnmarshalFailedMetricPoints.M(1))
				if c.messageMarking.After && c.messageMarking.OnError {
					session.MarkMessage(message, "")
				}
				return err
			}
			c.headerExtractor.extractHeadersMetrics(metrics, message)

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

		// Should return when `session.Context()` is done.
		// If not, will raise `ErrRebalanceInProgress` or `read tcp <ip>:<port>: i/o timeout` when kafka rebalance. see:
		// https://github.com/IBM/sarama/issues/1192
		case <-session.Context().Done():
			return nil
		}
	}
}

func (c *logsConsumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	c.readyCloser.Do(func() {
		close(c.ready)
	})
	_ = stats.RecordWithTags(
		session.Context(),
		[]tag.Mutator{tag.Upsert(tagInstanceName, c.id.String())},
		statPartitionStart.M(1))
	return nil
}

func (c *logsConsumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	_ = stats.RecordWithTags(
		session.Context(),
		[]tag.Mutator{tag.Upsert(tagInstanceName, c.id.String())},
		statPartitionClose.M(1))
	return nil
}

func (c *logsConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	c.logger.Info("Starting consumer group", zap.Int32("partition", claim.Partition()))
	if !c.autocommitEnabled {
		defer session.Commit()
	}
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			c.logger.Debug("Kafka message claimed",
				zap.String("value", string(message.Value)),
				zap.Time("timestamp", message.Timestamp),
				zap.String("topic", message.Topic))
			if !c.messageMarking.After {
				session.MarkMessage(message, "")
			}

			ctx := c.obsrecv.StartLogsOp(session.Context())
			statsTags := []tag.Mutator{
				tag.Upsert(tagInstanceName, c.id.String()),
				tag.Upsert(tagPartition, strconv.Itoa(int(claim.Partition()))),
			}
			_ = stats.RecordWithTags(
				ctx,
				statsTags,
				statMessageCount.M(1),
				statMessageOffset.M(message.Offset),
				statMessageOffsetLag.M(claim.HighWaterMarkOffset()-message.Offset-1))

			logs, err := c.unmarshaler.Unmarshal(message.Value)
			if err != nil {
				c.logger.Error("failed to unmarshal message", zap.Error(err))
				_ = stats.RecordWithTags(
					ctx,
					[]tag.Mutator{tag.Upsert(tagInstanceName, c.id.String())},
					statUnmarshalFailedLogRecords.M(1))
				if c.messageMarking.After && c.messageMarking.OnError {
					session.MarkMessage(message, "")
				}
				return err
			}
			c.headerExtractor.extractHeadersLogs(logs, message)
			logRecordCount := logs.LogRecordCount()
			err = c.nextConsumer.ConsumeLogs(session.Context(), logs)
			c.obsrecv.EndLogsOp(ctx, c.unmarshaler.Encoding(), logRecordCount, err)
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

		// Should return when `session.Context()` is done.
		// If not, will raise `ErrRebalanceInProgress` or `read tcp <ip>:<port>: i/o timeout` when kafka rebalance. see:
		// https://github.com/IBM/sarama/issues/1192
		case <-session.Context().Done():
			return nil
		}
	}
}

func toSaramaInitialOffset(initialOffset string) (int64, error) {
	switch initialOffset {
	case offsetEarliest:
		return sarama.OffsetOldest, nil
	case offsetLatest:
		fallthrough
	case "":
		return sarama.OffsetNewest, nil
	default:
		return 0, errInvalidInitialOffset
	}
}
