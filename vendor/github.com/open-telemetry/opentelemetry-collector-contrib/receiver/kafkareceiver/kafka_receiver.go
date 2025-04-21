// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/cenkalti/backoff/v4"
	"go.opentelemetry.io/collector/client"
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

var errMemoryLimiterDataRefused = errors.New("data refused due to high memory usage")

// kafkaTracesConsumer uses sarama to consume and handle messages from kafka.
type kafkaTracesConsumer struct {
	config            *Config
	consumerGroup     sarama.ConsumerGroup
	nextConsumer      consumer.Traces
	topics            []string
	cancelConsumeLoop context.CancelFunc
	unmarshaler       ptrace.Unmarshaler
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
	config            *Config
	consumerGroup     sarama.ConsumerGroup
	nextConsumer      consumer.Metrics
	topics            []string
	cancelConsumeLoop context.CancelFunc
	unmarshaler       pmetric.Unmarshaler
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
	config            *Config
	consumerGroup     sarama.ConsumerGroup
	nextConsumer      consumer.Logs
	topics            []string
	cancelConsumeLoop context.CancelFunc
	unmarshaler       plog.Unmarshaler
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

func newTracesReceiver(config *Config, set receiver.Settings, nextConsumer consumer.Traces) (*kafkaTracesConsumer, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return &kafkaTracesConsumer{
		config:            config,
		topics:            []string{config.Traces.Topic},
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

func createKafkaClient(ctx context.Context, config *Config) (sarama.ConsumerGroup, error) {
	return kafka.NewSaramaConsumerGroup(ctx, config.ClientConfig, config.ConsumerConfig)
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

	unmarshaler, err := newTracesUnmarshaler(c.config.Traces.Encoding, c.settings, host)
	if err != nil {
		return err
	}
	c.unmarshaler = unmarshaler

	// consumerGroup may be set in tests to inject fake implementation.
	if c.consumerGroup == nil {
		if c.consumerGroup, err = createKafkaClient(ctx, c.config); err != nil {
			return err
		}
	}
	consumerGroup := &tracesConsumerGroupHandler{
		logger:            c.settings.Logger,
		encoding:          c.config.Traces.Encoding,
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

func newMetricsReceiver(config *Config, set receiver.Settings, nextConsumer consumer.Metrics) (*kafkaMetricsConsumer, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return &kafkaMetricsConsumer{
		config:            config,
		topics:            []string{config.Metrics.Topic},
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

	unmarshaler, err := newMetricsUnmarshaler(c.config.Metrics.Encoding, c.settings, host)
	if err != nil {
		return err
	}
	c.unmarshaler = unmarshaler

	// consumerGroup may be set in tests to inject fake implementation.
	if c.consumerGroup == nil {
		if c.consumerGroup, err = createKafkaClient(ctx, c.config); err != nil {
			return err
		}
	}
	metricsConsumerGroup := &metricsConsumerGroupHandler{
		logger:            c.settings.Logger,
		encoding:          c.config.Metrics.Encoding,
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

func newLogsReceiver(config *Config, set receiver.Settings, nextConsumer consumer.Logs) (*kafkaLogsConsumer, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return &kafkaLogsConsumer{
		config:            config,
		topics:            []string{config.Logs.Topic},
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

	unmarshaler, err := newLogsUnmarshaler(c.config.Logs.Encoding, c.settings, host)
	if err != nil {
		return err
	}
	c.unmarshaler = unmarshaler

	// consumerGroup may be set in tests to inject fake implementation.
	if c.consumerGroup == nil {
		if c.consumerGroup, err = createKafkaClient(ctx, c.config); err != nil {
			return err
		}
	}
	logsConsumerGroup := &logsConsumerGroupHandler{
		logger:            c.settings.Logger,
		encoding:          c.config.Logs.Encoding,
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
	encoding     string
	unmarshaler  ptrace.Unmarshaler
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
	backOffMutex      sync.Mutex
}

type metricsConsumerGroupHandler struct {
	id           component.ID
	encoding     string
	unmarshaler  pmetric.Unmarshaler
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
	backOffMutex      sync.Mutex
}

type logsConsumerGroupHandler struct {
	id           component.ID
	encoding     string
	unmarshaler  plog.Unmarshaler
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
	backOffMutex      sync.Mutex
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

			// If the Kafka exporter has propagated headers in the message,
			// create a new context with client.Info in it.
			ctx := newContextWithHeaders(session.Context(), message.Headers)
			ctx = c.obsrecv.StartTracesOp(ctx)
			attrs := attribute.NewSet(
				attribute.String(attrInstanceName, c.id.String()),
				attribute.String(attrPartition, strconv.Itoa(int(claim.Partition()))),
			)
			c.telemetryBuilder.KafkaReceiverMessages.Add(ctx, 1, metric.WithAttributeSet(attrs))
			c.telemetryBuilder.KafkaReceiverCurrentOffset.Record(ctx, message.Offset, metric.WithAttributeSet(attrs))
			c.telemetryBuilder.KafkaReceiverOffsetLag.Record(ctx, claim.HighWaterMarkOffset()-message.Offset-1, metric.WithAttributeSet(attrs))

			traces, err := c.unmarshaler.UnmarshalTraces(message.Value)
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
			err = c.nextConsumer.ConsumeTraces(ctx, traces)
			c.obsrecv.EndTracesOp(ctx, c.encoding, spanCount, err)
			if err != nil {
				if errorRequiresBackoff(err) && c.backOff != nil {
					backOffDelay := c.getNextBackoff()
					if backOffDelay != backoff.Stop {
						c.logger.Info("Backing off due to error from the next consumer.",
							zap.Error(err),
							zap.Duration("delay", backOffDelay),
							zap.String("topic", message.Topic),
							zap.Int32("partition", claim.Partition()))
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
					c.logger.Info("Stop error backoff because the configured max_elapsed_time is reached",
						zap.Duration("max_elapsed_time", c.backOff.MaxElapsedTime))
				}
				if c.messageMarking.After && c.messageMarking.OnError {
					session.MarkMessage(message, "")
				}
				return err
			}
			if c.backOff != nil {
				c.resetBackoff()
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

func (c *tracesConsumerGroupHandler) getNextBackoff() time.Duration {
	c.backOffMutex.Lock()
	defer c.backOffMutex.Unlock()
	return c.backOff.NextBackOff()
}

func (c *tracesConsumerGroupHandler) resetBackoff() {
	c.backOffMutex.Lock()
	defer c.backOffMutex.Unlock()
	c.backOff.Reset()
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

			// If the Kafka exporter has propagated headers in the message,
			// create a new context with client.Info in it.
			ctx := newContextWithHeaders(session.Context(), message.Headers)
			ctx = c.obsrecv.StartMetricsOp(ctx)
			attrs := attribute.NewSet(
				attribute.String(attrInstanceName, c.id.String()),
				attribute.String(attrPartition, strconv.Itoa(int(claim.Partition()))),
			)
			c.telemetryBuilder.KafkaReceiverMessages.Add(ctx, 1, metric.WithAttributeSet(attrs))
			c.telemetryBuilder.KafkaReceiverCurrentOffset.Record(ctx, message.Offset, metric.WithAttributeSet(attrs))
			c.telemetryBuilder.KafkaReceiverOffsetLag.Record(ctx, claim.HighWaterMarkOffset()-message.Offset-1, metric.WithAttributeSet(attrs))

			metrics, err := c.unmarshaler.UnmarshalMetrics(message.Value)
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
			err = c.nextConsumer.ConsumeMetrics(ctx, metrics)
			c.obsrecv.EndMetricsOp(ctx, c.encoding, dataPointCount, err)
			if err != nil {
				if errorRequiresBackoff(err) && c.backOff != nil {
					backOffDelay := c.getNextBackoff()
					if backOffDelay != backoff.Stop {
						c.logger.Info("Backing off due to error from the next consumer.",
							zap.Error(err),
							zap.Duration("delay", backOffDelay),
							zap.String("topic", message.Topic),
							zap.Int32("partition", claim.Partition()))
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
					c.logger.Info("Stop error backoff because the configured max_elapsed_time is reached",
						zap.Duration("max_elapsed_time", c.backOff.MaxElapsedTime))
				}
				if c.messageMarking.After && c.messageMarking.OnError {
					session.MarkMessage(message, "")
				}
				return err
			}
			if c.backOff != nil {
				c.resetBackoff()
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

func (c *metricsConsumerGroupHandler) getNextBackoff() time.Duration {
	c.backOffMutex.Lock()
	defer c.backOffMutex.Unlock()
	return c.backOff.NextBackOff()
}

func (c *metricsConsumerGroupHandler) resetBackoff() {
	c.backOffMutex.Lock()
	defer c.backOffMutex.Unlock()
	c.backOff.Reset()
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

			// If the Kafka exporter has propagated headers in the message,
			// create a new context with client.Info in it.
			ctx := newContextWithHeaders(session.Context(), message.Headers)
			ctx = c.obsrecv.StartLogsOp(ctx)
			attrs := attribute.NewSet(
				attribute.String(attrInstanceName, c.id.String()),
				attribute.String(attrPartition, strconv.Itoa(int(claim.Partition()))),
			)
			c.telemetryBuilder.KafkaReceiverMessages.Add(ctx, 1, metric.WithAttributeSet(attrs))
			c.telemetryBuilder.KafkaReceiverCurrentOffset.Record(ctx, message.Offset, metric.WithAttributeSet(attrs))
			c.telemetryBuilder.KafkaReceiverOffsetLag.Record(ctx, claim.HighWaterMarkOffset()-message.Offset-1, metric.WithAttributeSet(attrs))

			logs, err := c.unmarshaler.UnmarshalLogs(message.Value)
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
			err = c.nextConsumer.ConsumeLogs(ctx, logs)
			c.obsrecv.EndLogsOp(ctx, c.encoding, logRecordCount, err)
			if err != nil {
				if errorRequiresBackoff(err) && c.backOff != nil {
					backOffDelay := c.getNextBackoff()
					if backOffDelay != backoff.Stop {
						c.logger.Info("Backing off due to error from the next consumer.",
							zap.Error(err),
							zap.Duration("delay", backOffDelay),
							zap.String("topic", message.Topic),
							zap.Int32("partition", claim.Partition()))
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
					c.logger.Info("Stop error backoff because the configured max_elapsed_time is reached",
						zap.Duration("max_elapsed_time", c.backOff.MaxElapsedTime))
				}
				if c.messageMarking.After && c.messageMarking.OnError {
					session.MarkMessage(message, "")
				}
				return err
			}
			if c.backOff != nil {
				c.resetBackoff()
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

func (c *logsConsumerGroupHandler) getNextBackoff() time.Duration {
	c.backOffMutex.Lock()
	defer c.backOffMutex.Unlock()
	return c.backOff.NextBackOff()
}

func (c *logsConsumerGroupHandler) resetBackoff() {
	c.backOffMutex.Lock()
	defer c.backOffMutex.Unlock()
	c.backOff.Reset()
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

func errorRequiresBackoff(err error) bool {
	return err.Error() == errMemoryLimiterDataRefused.Error()
}

func newContextWithHeaders(ctx context.Context,
	headers []*sarama.RecordHeader,
) context.Context {
	if len(headers) == 0 {
		return ctx
	}
	m := make(map[string][]string, len(headers))
	for _, header := range headers {
		key := string(header.Key)
		value := string(header.Value)
		m[key] = append(m[key], value)
	}
	return client.NewContext(ctx, client.Info{Metadata: client.NewMetadata(m)})
}
