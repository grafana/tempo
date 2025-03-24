// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/cenkalti/backoff/v4"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver/internal/metadata"
)

const (
	transport = "kafka"
	// TODO: update the following attributes to reflect semconv
	attrInstanceName = "name"
	attrPartition    = "partition"
)

var errInvalidInitialOffset = errors.New("invalid initial offset")

var errMemoryLimiterDataRefused = errors.New("data refused due to high memory usage")

// kafkaTracesConsumer uses sarama to consume and handle messages from kafka.
type kafkaTracesConsumer struct {
	config            Config
	consumerGroup     sarama.ConsumerGroup
	nextConsumer      consumer.Traces
	topics            []string
	cancelConsumeLoop context.CancelFunc
	unmarshaler       TracesUnmarshaler
	consumeLoopWG     *sync.WaitGroup

	settings         receiver.Settings
	telemetryBuilder *metadata.TelemetryBuilder

	autocommitEnabled bool
	messageMarking    MessageMarking
	headerExtraction  bool
	headers           []string
	minFetchSize      int32
	defaultFetchSize  int32
	maxFetchSize      int32
}

// kafkaMetricsConsumer uses sarama to consume and handle messages from kafka.
type kafkaMetricsConsumer struct {
	config            Config
	consumerGroup     sarama.ConsumerGroup
	nextConsumer      consumer.Metrics
	topics            []string
	cancelConsumeLoop context.CancelFunc
	unmarshaler       MetricsUnmarshaler
	consumeLoopWG     *sync.WaitGroup

	settings         receiver.Settings
	telemetryBuilder *metadata.TelemetryBuilder

	autocommitEnabled bool
	messageMarking    MessageMarking
	headerExtraction  bool
	headers           []string
	minFetchSize      int32
	defaultFetchSize  int32
	maxFetchSize      int32
}

// kafkaLogsConsumer uses sarama to consume and handle messages from kafka.
type kafkaLogsConsumer struct {
	config            Config
	consumerGroup     sarama.ConsumerGroup
	nextConsumer      consumer.Logs
	topics            []string
	cancelConsumeLoop context.CancelFunc
	unmarshaler       LogsUnmarshaler
	consumeLoopWG     *sync.WaitGroup

	settings         receiver.Settings
	telemetryBuilder *metadata.TelemetryBuilder

	autocommitEnabled bool
	messageMarking    MessageMarking
	headerExtraction  bool
	headers           []string
	minFetchSize      int32
	defaultFetchSize  int32
	maxFetchSize      int32
}

var (
	_ receiver.Traces  = (*kafkaTracesConsumer)(nil)
	_ receiver.Metrics = (*kafkaMetricsConsumer)(nil)
	_ receiver.Logs    = (*kafkaLogsConsumer)(nil)
)

func newTracesReceiver(config Config, set receiver.Settings, nextConsumer consumer.Traces) (*kafkaTracesConsumer, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return &kafkaTracesConsumer{
		config:            config,
		topics:            []string{config.Topic},
		nextConsumer:      nextConsumer,
		consumeLoopWG:     &sync.WaitGroup{},
		settings:          set,
		autocommitEnabled: config.AutoCommit.Enable,
		messageMarking:    config.MessageMarking,
		headerExtraction:  config.HeaderExtraction.ExtractHeaders,
		headers:           config.HeaderExtraction.Headers,
		telemetryBuilder:  telemetryBuilder,
		minFetchSize:      config.MinFetchSize,
		defaultFetchSize:  config.DefaultFetchSize,
		maxFetchSize:      config.MaxFetchSize,
	}, nil
}

func createKafkaClient(ctx context.Context, config Config) (sarama.ConsumerGroup, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.ClientID = config.ClientID
	saramaConfig.Metadata.Full = config.Metadata.Full
	saramaConfig.Metadata.Retry.Max = config.Metadata.Retry.Max
	saramaConfig.Metadata.Retry.Backoff = config.Metadata.Retry.Backoff
	saramaConfig.Consumer.Offsets.AutoCommit.Enable = config.AutoCommit.Enable
	saramaConfig.Consumer.Offsets.AutoCommit.Interval = config.AutoCommit.Interval
	saramaConfig.Consumer.Group.Session.Timeout = config.SessionTimeout
	saramaConfig.Consumer.Group.Heartbeat.Interval = config.HeartbeatInterval
	saramaConfig.Consumer.Fetch.Min = config.MinFetchSize
	saramaConfig.Consumer.Fetch.Default = config.DefaultFetchSize
	saramaConfig.Consumer.Fetch.Max = config.MaxFetchSize

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
	if err := kafka.ConfigureSaramaAuthentication(ctx, config.Authentication, saramaConfig); err != nil {
		return nil, err
	}
	return sarama.NewConsumerGroup(config.Brokers, config.GroupID, saramaConfig)
}

func (c *kafkaTracesConsumer) Start(_ context.Context, host component.Host) error {
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
	// extensions take precedence over internal encodings
	if unmarshaler, errExt := loadEncodingExtension[ptrace.Unmarshaler](
		host,
		c.config.Encoding,
	); errExt == nil {
		c.unmarshaler = &tracesEncodingUnmarshaler{
			unmarshaler: *unmarshaler,
			encoding:    c.config.Encoding,
		}
	}
	if unmarshaler, ok := defaultTracesUnmarshalers()[c.config.Encoding]; c.unmarshaler == nil && ok {
		c.unmarshaler = unmarshaler
	}
	if c.unmarshaler == nil {
		return errUnrecognizedEncoding
	}
	// consumerGroup may be set in tests to inject fake implementation.
	if c.consumerGroup == nil {
		if c.consumerGroup, err = createKafkaClient(ctx, c.config); err != nil {
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
		telemetryBuilder:  c.telemetryBuilder,
		backOff:           newExponentialBackOff(c.config.ErrorBackOff),
	}
	if c.headerExtraction {
		consumerGroup.headerExtractor = &headerExtractor{
			logger:  c.settings.Logger,
			headers: c.headers,
		}
	}
	c.consumeLoopWG.Add(1)
	go c.consumeLoop(ctx, consumerGroup)
	<-consumerGroup.ready
	return nil
}

func (c *kafkaTracesConsumer) consumeLoop(ctx context.Context, handler sarama.ConsumerGroupHandler) {
	defer c.consumeLoopWG.Done()
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
			return
		}
	}
}

func (c *kafkaTracesConsumer) Shutdown(context.Context) error {
	if c.cancelConsumeLoop == nil {
		return nil
	}
	c.cancelConsumeLoop()
	c.consumeLoopWG.Wait()
	if c.consumerGroup == nil {
		return nil
	}
	return c.consumerGroup.Close()
}

func newMetricsReceiver(config Config, set receiver.Settings, nextConsumer consumer.Metrics) (*kafkaMetricsConsumer, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return &kafkaMetricsConsumer{
		config:            config,
		topics:            []string{config.Topic},
		nextConsumer:      nextConsumer,
		consumeLoopWG:     &sync.WaitGroup{},
		settings:          set,
		autocommitEnabled: config.AutoCommit.Enable,
		messageMarking:    config.MessageMarking,
		headerExtraction:  config.HeaderExtraction.ExtractHeaders,
		headers:           config.HeaderExtraction.Headers,
		telemetryBuilder:  telemetryBuilder,
		minFetchSize:      config.MinFetchSize,
		defaultFetchSize:  config.DefaultFetchSize,
		maxFetchSize:      config.MaxFetchSize,
	}, nil
}

func (c *kafkaMetricsConsumer) Start(_ context.Context, host component.Host) error {
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
	// extensions take precedence over internal encodings
	if unmarshaler, errExt := loadEncodingExtension[pmetric.Unmarshaler](
		host,
		c.config.Encoding,
	); errExt == nil {
		c.unmarshaler = &metricsEncodingUnmarshaler{
			unmarshaler: *unmarshaler,
			encoding:    c.config.Encoding,
		}
	}
	if unmarshaler, ok := defaultMetricsUnmarshalers()[c.config.Encoding]; c.unmarshaler == nil && ok {
		c.unmarshaler = unmarshaler
	}
	if c.unmarshaler == nil {
		return errUnrecognizedEncoding
	}
	// consumerGroup may be set in tests to inject fake implementation.
	if c.consumerGroup == nil {
		if c.consumerGroup, err = createKafkaClient(ctx, c.config); err != nil {
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
		telemetryBuilder:  c.telemetryBuilder,
		backOff:           newExponentialBackOff(c.config.ErrorBackOff),
	}
	if c.headerExtraction {
		metricsConsumerGroup.headerExtractor = &headerExtractor{
			logger:  c.settings.Logger,
			headers: c.headers,
		}
	}
	c.consumeLoopWG.Add(1)
	go c.consumeLoop(ctx, metricsConsumerGroup)
	<-metricsConsumerGroup.ready
	return nil
}

func (c *kafkaMetricsConsumer) consumeLoop(ctx context.Context, handler sarama.ConsumerGroupHandler) {
	defer c.consumeLoopWG.Done()
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
			return
		}
	}
}

func (c *kafkaMetricsConsumer) Shutdown(context.Context) error {
	if c.cancelConsumeLoop == nil {
		return nil
	}
	c.cancelConsumeLoop()
	c.consumeLoopWG.Wait()
	if c.consumerGroup == nil {
		return nil
	}
	return c.consumerGroup.Close()
}

func newLogsReceiver(config Config, set receiver.Settings, nextConsumer consumer.Logs) (*kafkaLogsConsumer, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return &kafkaLogsConsumer{
		config:            config,
		topics:            []string{config.Topic},
		nextConsumer:      nextConsumer,
		consumeLoopWG:     &sync.WaitGroup{},
		settings:          set,
		autocommitEnabled: config.AutoCommit.Enable,
		messageMarking:    config.MessageMarking,
		headerExtraction:  config.HeaderExtraction.ExtractHeaders,
		headers:           config.HeaderExtraction.Headers,
		telemetryBuilder:  telemetryBuilder,
		minFetchSize:      config.MinFetchSize,
		defaultFetchSize:  config.DefaultFetchSize,
		maxFetchSize:      config.MaxFetchSize,
	}, nil
}

func (c *kafkaLogsConsumer) Start(_ context.Context, host component.Host) error {
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
	// extensions take precedence over internal encodings
	if unmarshaler, errExt := loadEncodingExtension[plog.Unmarshaler](
		host,
		c.config.Encoding,
	); errExt == nil {
		c.unmarshaler = &logsEncodingUnmarshaler{
			unmarshaler: *unmarshaler,
			encoding:    c.config.Encoding,
		}
	}
	if unmarshaler, errInt := getLogsUnmarshaler(
		c.config.Encoding,
		defaultLogsUnmarshalers(c.settings.BuildInfo.Version, c.settings.Logger),
	); c.unmarshaler == nil && errInt == nil {
		c.unmarshaler = unmarshaler
	}
	if c.unmarshaler == nil {
		return errUnrecognizedEncoding
	}
	// consumerGroup may be set in tests to inject fake implementation.
	if c.consumerGroup == nil {
		if c.consumerGroup, err = createKafkaClient(ctx, c.config); err != nil {
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
		telemetryBuilder:  c.telemetryBuilder,
		backOff:           newExponentialBackOff(c.config.ErrorBackOff),
	}
	if c.headerExtraction {
		logsConsumerGroup.headerExtractor = &headerExtractor{
			logger:  c.settings.Logger,
			headers: c.headers,
		}
	}
	c.consumeLoopWG.Add(1)
	go c.consumeLoop(ctx, logsConsumerGroup)
	<-logsConsumerGroup.ready
	return nil
}

func (c *kafkaLogsConsumer) consumeLoop(ctx context.Context, handler sarama.ConsumerGroupHandler) {
	defer c.consumeLoopWG.Done()
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
			return
		}
	}
}

func (c *kafkaLogsConsumer) Shutdown(context.Context) error {
	if c.cancelConsumeLoop == nil {
		return nil
	}
	c.cancelConsumeLoop()
	c.consumeLoopWG.Wait()
	if c.consumerGroup == nil {
		return nil
	}
	return c.consumerGroup.Close()
}

type tracesConsumerGroupHandler struct {
	id           component.ID
	unmarshaler  TracesUnmarshaler
	nextConsumer consumer.Traces
	ready        chan bool
	readyCloser  sync.Once

	logger *zap.Logger

	obsrecv          *receiverhelper.ObsReport
	telemetryBuilder *metadata.TelemetryBuilder

	autocommitEnabled bool
	messageMarking    MessageMarking
	headerExtractor   HeaderExtractor
	backOff           *backoff.ExponentialBackOff
}

type metricsConsumerGroupHandler struct {
	id           component.ID
	unmarshaler  MetricsUnmarshaler
	nextConsumer consumer.Metrics
	ready        chan bool
	readyCloser  sync.Once

	logger *zap.Logger

	obsrecv          *receiverhelper.ObsReport
	telemetryBuilder *metadata.TelemetryBuilder

	autocommitEnabled bool
	messageMarking    MessageMarking
	headerExtractor   HeaderExtractor
	backOff           *backoff.ExponentialBackOff
}

type logsConsumerGroupHandler struct {
	id           component.ID
	unmarshaler  LogsUnmarshaler
	nextConsumer consumer.Logs
	ready        chan bool
	readyCloser  sync.Once

	logger *zap.Logger

	obsrecv          *receiverhelper.ObsReport
	telemetryBuilder *metadata.TelemetryBuilder

	autocommitEnabled bool
	messageMarking    MessageMarking
	headerExtractor   HeaderExtractor
	backOff           *backoff.ExponentialBackOff
}

var (
	_ sarama.ConsumerGroupHandler = (*tracesConsumerGroupHandler)(nil)
	_ sarama.ConsumerGroupHandler = (*metricsConsumerGroupHandler)(nil)
	_ sarama.ConsumerGroupHandler = (*logsConsumerGroupHandler)(nil)
)

func (c *tracesConsumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	c.readyCloser.Do(func() {
		close(c.ready)
	})
	c.telemetryBuilder.KafkaReceiverPartitionStart.Add(session.Context(), 1, metric.WithAttributes(attribute.String(attrInstanceName, c.id.Name())))
	return nil
}

func (c *tracesConsumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	c.telemetryBuilder.KafkaReceiverPartitionClose.Add(session.Context(), 1, metric.WithAttributes(attribute.String(attrInstanceName, c.id.Name())))
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
			attrs := attribute.NewSet(
				attribute.String(attrInstanceName, c.id.String()),
				attribute.String(attrPartition, strconv.Itoa(int(claim.Partition()))),
			)
			c.telemetryBuilder.KafkaReceiverMessages.Add(ctx, 1, metric.WithAttributeSet(attrs))
			c.telemetryBuilder.KafkaReceiverCurrentOffset.Record(ctx, message.Offset, metric.WithAttributeSet(attrs))
			c.telemetryBuilder.KafkaReceiverOffsetLag.Record(ctx, claim.HighWaterMarkOffset()-message.Offset-1, metric.WithAttributeSet(attrs))

			traces, err := c.unmarshaler.Unmarshal(message.Value)
			if err != nil {
				c.logger.Error("failed to unmarshal message", zap.Error(err))
				c.telemetryBuilder.KafkaReceiverUnmarshalFailedSpans.Add(session.Context(), 1, metric.WithAttributes(attribute.String(attrInstanceName, c.id.String())))
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
				if errorRequiresBackoff(err) && c.backOff != nil {
					backOffDelay := c.backOff.NextBackOff()
					if backOffDelay != backoff.Stop {
						select {
						case <-session.Context().Done():
							return nil
						case <-time.After(backOffDelay):
							if !c.messageMarking.After {
								// Unmark the message so it can be retried
								session.ResetOffset(claim.Topic(), claim.Partition(), message.Offset, "")
							}
							return err
						}
					}
				}
				if c.messageMarking.After && c.messageMarking.OnError {
					session.MarkMessage(message, "")
				}
				return err
			}
			if c.backOff != nil {
				c.backOff.Reset()
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
	c.telemetryBuilder.KafkaReceiverPartitionStart.Add(session.Context(), 1, metric.WithAttributes(attribute.String(attrInstanceName, c.id.Name())))
	return nil
}

func (c *metricsConsumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	c.telemetryBuilder.KafkaReceiverPartitionClose.Add(session.Context(), 1, metric.WithAttributes(attribute.String(attrInstanceName, c.id.Name())))
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
			attrs := attribute.NewSet(
				attribute.String(attrInstanceName, c.id.String()),
				attribute.String(attrPartition, strconv.Itoa(int(claim.Partition()))),
			)
			c.telemetryBuilder.KafkaReceiverMessages.Add(ctx, 1, metric.WithAttributeSet(attrs))
			c.telemetryBuilder.KafkaReceiverCurrentOffset.Record(ctx, message.Offset, metric.WithAttributeSet(attrs))
			c.telemetryBuilder.KafkaReceiverOffsetLag.Record(ctx, claim.HighWaterMarkOffset()-message.Offset-1, metric.WithAttributeSet(attrs))

			metrics, err := c.unmarshaler.Unmarshal(message.Value)
			if err != nil {
				c.logger.Error("failed to unmarshal message", zap.Error(err))
				c.telemetryBuilder.KafkaReceiverUnmarshalFailedMetricPoints.Add(session.Context(), 1, metric.WithAttributes(attribute.String(attrInstanceName, c.id.String())))
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
				if errorRequiresBackoff(err) && c.backOff != nil {
					backOffDelay := c.backOff.NextBackOff()
					if backOffDelay != backoff.Stop {
						select {
						case <-session.Context().Done():
							return nil
						case <-time.After(backOffDelay):
							if !c.messageMarking.After {
								// Unmark the message so it can be retried
								session.ResetOffset(claim.Topic(), claim.Partition(), message.Offset, "")
							}
							return err
						}
					}
				}
				if c.messageMarking.After && c.messageMarking.OnError {
					session.MarkMessage(message, "")
				}
				return err
			}
			if c.backOff != nil {
				c.backOff.Reset()
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
	c.telemetryBuilder.KafkaReceiverPartitionStart.Add(session.Context(), 1, metric.WithAttributes(attribute.String(attrInstanceName, c.id.String())))
	return nil
}

func (c *logsConsumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	c.telemetryBuilder.KafkaReceiverPartitionClose.Add(session.Context(), 1, metric.WithAttributes(attribute.String(attrInstanceName, c.id.String())))
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
			attrs := attribute.NewSet(
				attribute.String(attrInstanceName, c.id.String()),
				attribute.String(attrPartition, strconv.Itoa(int(claim.Partition()))),
			)
			c.telemetryBuilder.KafkaReceiverMessages.Add(ctx, 1, metric.WithAttributeSet(attrs))
			c.telemetryBuilder.KafkaReceiverCurrentOffset.Record(ctx, message.Offset, metric.WithAttributeSet(attrs))
			c.telemetryBuilder.KafkaReceiverOffsetLag.Record(ctx, claim.HighWaterMarkOffset()-message.Offset-1, metric.WithAttributeSet(attrs))

			logs, err := c.unmarshaler.Unmarshal(message.Value)
			if err != nil {
				c.logger.Error("failed to unmarshal message", zap.Error(err))
				c.telemetryBuilder.KafkaReceiverUnmarshalFailedLogRecords.Add(ctx, 1, metric.WithAttributes(attribute.String(attrInstanceName, c.id.String())))
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
				if errorRequiresBackoff(err) && c.backOff != nil {
					backOffDelay := c.backOff.NextBackOff()
					if backOffDelay != backoff.Stop {
						select {
						case <-session.Context().Done():
							return nil
						case <-time.After(backOffDelay):
							if !c.messageMarking.After {
								// Unmark the message so it can be retried
								session.ResetOffset(claim.Topic(), claim.Partition(), message.Offset, "")
							}
							return err
						}
					}
				}
				if c.messageMarking.After && c.messageMarking.OnError {
					session.MarkMessage(message, "")
				}
				return err
			}
			if c.backOff != nil {
				c.backOff.Reset()
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

func newExponentialBackOff(config configretry.BackOffConfig) *backoff.ExponentialBackOff {
	if !config.Enabled {
		return nil
	}
	backOff := backoff.NewExponentialBackOff()
	backOff.InitialInterval = config.InitialInterval
	backOff.RandomizationFactor = config.RandomizationFactor
	backOff.Multiplier = config.Multiplier
	backOff.MaxInterval = config.MaxInterval
	backOff.MaxElapsedTime = config.MaxElapsedTime
	backOff.Reset()
	return backOff
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

// loadEncodingExtension tries to load an available extension for the given encoding.
func loadEncodingExtension[T any](host component.Host, encoding string) (*T, error) {
	extensionID, err := encodingToComponentID(encoding)
	if err != nil {
		return nil, err
	}
	encodingExtension, ok := host.GetExtensions()[*extensionID]
	if !ok {
		return nil, fmt.Errorf("unknown encoding extension %q", encoding)
	}
	unmarshaler, ok := encodingExtension.(T)
	if !ok {
		return nil, fmt.Errorf("extension %q is not an unmarshaler", encoding)
	}
	return &unmarshaler, nil
}

// encodingToComponentID converts an encoding string to a component ID using the given encoding as type.
func encodingToComponentID(encoding string) (*component.ID, error) {
	componentType, err := component.NewType(encoding)
	if err != nil {
		return nil, fmt.Errorf("invalid component type: %w", err)
	}
	id := component.NewID(componentType)
	return &id, nil
}

func errorRequiresBackoff(err error) bool {
	return err.Error() == errMemoryLimiterDataRefused.Error()
}
