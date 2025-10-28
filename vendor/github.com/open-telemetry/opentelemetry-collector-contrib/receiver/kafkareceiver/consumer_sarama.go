// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"context"
	"encoding/binary"
	"errors"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/cenkalti/backoff/v4"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver/internal/metadata"
)

func newSaramaConsumer(
	config *Config,
	set receiver.Settings,
	topics []string,
	newConsumeFn newConsumeMessageFunc,
) (*saramaConsumer, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return &saramaConsumer{
		config:           config,
		topics:           topics,
		newConsumeFn:     newConsumeFn,
		settings:         set,
		telemetryBuilder: telemetryBuilder,
	}, nil
}

// saramaConsumer consumes messages from a set of Kafka topics,
// decodes telemetry data using a given unmarshaler, and passes
// them to a consumer.
type saramaConsumer struct {
	config           *Config
	topics           []string
	settings         receiver.Settings
	telemetryBuilder *metadata.TelemetryBuilder
	newConsumeFn     newConsumeMessageFunc

	mu                sync.Mutex
	started           bool
	shutdown          bool
	closing           chan struct{}
	consumeLoopClosed chan struct{}
}

func (c *saramaConsumer) Start(_ context.Context, host component.Host) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.shutdown {
		return errors.New("kafka consumer already shut down")
	}
	if c.started {
		return errors.New("kafka consumer already started")
	}

	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             c.settings.ID,
		Transport:              transport,
		ReceiverCreateSettings: c.settings,
	})
	if err != nil {
		return err
	}

	handler := &consumerGroupHandler{
		host:              host,
		logger:            c.settings.Logger,
		obsrecv:           obsrecv,
		autocommitEnabled: c.config.AutoCommit.Enable,
		messageMarking:    c.config.MessageMarking,
		telemetryBuilder:  c.telemetryBuilder,
		backOff:           newExponentialBackOff(c.config.ErrorBackOff),
	}
	consumeMessage, err := c.newConsumeFn(host, obsrecv, c.telemetryBuilder)
	if err != nil {
		return err
	}
	handler.consumeMessage = consumeMessage

	c.consumeLoopClosed = make(chan struct{})
	c.started = true
	c.closing = make(chan struct{})
	go c.consumeLoop(handler, host)
	return nil
}

func (c *saramaConsumer) consumeLoop(handler sarama.ConsumerGroupHandler, host component.Host) {
	defer close(c.consumeLoopClosed)
	defer componentstatus.ReportStatus(host, componentstatus.NewEvent(componentstatus.StatusStopped))
	componentstatus.ReportStatus(host, componentstatus.NewEvent(componentstatus.StatusStarting))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-ctx.Done():
		case <-c.closing:
			componentstatus.ReportStatus(host, componentstatus.NewEvent(componentstatus.StatusStopping))
			cancel()
		}
	}()

	// kafka.NewSaramaConsumerGroup (actually sarama.NewConsumerGroup)
	// may perform synchronous operations that can fail due to transient
	// errors, so we retry until it succeeds or the context is canceled.
	var consumerGroup sarama.ConsumerGroup
	err := backoff.Retry(func() (err error) {
		consumerGroup, err = kafka.NewSaramaConsumerGroup(ctx, c.config.ClientConfig, c.config.ConsumerConfig)
		if err != nil {
			if ctx.Err() == nil {
				// We only report an error if the context is not canceled.
				// If the context is canceled it means the receiver is
				// shutting down, which will lead to reporting StatusStopped
				// when consumeLoop exits.
				c.settings.Logger.Error("Error creating consumer group", zap.Error(err))
				componentstatus.ReportStatus(host, componentstatus.NewRecoverableErrorEvent(err))
			}
			return err
		}
		return nil
	}, backoff.WithContext(
		// Use a zero max elapsed time to retry indefinitely until the context is canceled.
		backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(0)),
		ctx,
	))
	if err != nil {
		return
	}
	defer func() {
		if err := consumerGroup.Close(); err != nil {
			c.settings.Logger.Error("Error closing consumer group", zap.Error(err))
		}
	}()
	c.settings.Logger.Debug("Created consumer group")

	for {
		// `Consume` should be called inside an infinite loop, when a
		// server-side rebalance happens, the consumer session will need to be
		// recreated to get the new claims
		if err := consumerGroup.Consume(ctx, c.topics, handler); err != nil {
			if errors.Is(err, context.Canceled) {
				// Shutting down
				return
			}
			if errors.Is(err, sarama.ErrClosedConsumerGroup) {
				// Consumer group stopped unexpectedly.
				c.settings.Logger.Warn("Consumer stopped", zap.Error(ctx.Err()))
				return
			}
			c.settings.Logger.Error("Error from consumer", zap.Error(err))
			componentstatus.ReportStatus(host, componentstatus.NewRecoverableErrorEvent(err))
		}
	}
}

func (c *saramaConsumer) Shutdown(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.started || c.shutdown {
		return nil
	}
	c.shutdown = true
	close(c.closing)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.consumeLoopClosed:
	}
	return nil
}

type consumerGroupHandler struct {
	host           component.Host
	consumeMessage consumeMessageFunc
	logger         *zap.Logger

	obsrecv          *receiverhelper.ObsReport
	telemetryBuilder *metadata.TelemetryBuilder

	autocommitEnabled bool
	messageMarking    MessageMarking
	backOff           *backoff.ExponentialBackOff
	backOffMutex      sync.Mutex
}

func (c *consumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	c.logger.Debug("Consumer group session established")
	componentstatus.ReportStatus(c.host, componentstatus.NewEvent(componentstatus.StatusOK))
	c.telemetryBuilder.KafkaReceiverPartitionStart.Add(session.Context(), 1)
	return nil
}

func (c *consumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	c.logger.Debug("Consumer group session stopped")
	c.telemetryBuilder.KafkaReceiverPartitionClose.Add(session.Context(), 1)
	return nil
}

func (c *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	c.logger.Debug(
		"Consuming Kafka topic-partition",
		zap.String("topic", claim.Topic()),
		zap.Int32("partition", claim.Partition()),
		zap.Int64("initial_offset", claim.InitialOffset()),
	)
	if !c.autocommitEnabled {
		defer session.Commit()
	}
	for {
		select {
		case <-session.Context().Done():
			// Should return when the session's context is canceled.
			//
			// If we do not, will raise `ErrRebalanceInProgress` or `read tcp <ip>:<port>: i/o timeout`
			// when rebalancing. See: https://github.com/IBM/sarama/issues/1192
			return nil
		case message, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			if err := c.handleMessage(session, claim, message); err != nil {
				return err
			}
		}
	}
}

func (c *consumerGroupHandler) handleMessage(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
	message *sarama.ConsumerMessage,
) error {
	if !c.messageMarking.After {
		session.MarkMessage(message, "")
	}

	attrs := attribute.NewSet(
		attribute.String("topic", message.Topic),
		attribute.Int64("partition", int64(claim.Partition())),
	)
	c.telemetryBuilder.KafkaReceiverCurrentOffset.Record(
		context.Background(),
		message.Offset,
		metric.WithAttributeSet(attrs),
	)
	c.telemetryBuilder.KafkaReceiverOffsetLag.Record(
		context.Background(),
		claim.HighWaterMarkOffset()-message.Offset-1,
		metric.WithAttributeSet(attrs),
	)
	// KafkaReceiverMessages is deprecated in favor of KafkaReceiverRecords.
	c.telemetryBuilder.KafkaReceiverMessages.Add(
		context.Background(),
		1,
		metric.WithAttributeSet(attrs),
		metric.WithAttributes(attribute.String("outcome", "success")),
	)
	c.telemetryBuilder.KafkaReceiverRecords.Add(
		context.Background(),
		1,
		metric.WithAttributeSet(attrs),
		metric.WithAttributes(attribute.String("outcome", "success")),
	)
	c.telemetryBuilder.KafkaReceiverBytesUncompressed.Add(
		context.Background(),
		byteSize(message),
		metric.WithAttributeSet(attrs),
		metric.WithAttributes(attribute.String("outcome", "success")),
	)
	msg := wrapSaramaMsg(message)
	if err := c.consumeMessage(session.Context(), msg, attrs); err != nil {
		if c.backOff != nil && !consumererror.IsPermanent(err) {
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
			c.logger.Warn(
				"Stop error backoff because the configured max_elapsed_time is reached",
				zap.Duration("max_elapsed_time", c.backOff.MaxElapsedTime),
			)
		}

		isPermanent := consumererror.IsPermanent(err)
		shouldMark := (!isPermanent && c.messageMarking.OnError) || (isPermanent && c.messageMarking.OnPermanentError)

		if c.messageMarking.After && !shouldMark {
			// Only return an error if messages are marked after successful processing
			// and the error type is not configured to be marked.
			return err
		}
		// We're either marking messages as consumed ahead of time (disregarding outcome),
		// or after processing but including errors. Either way we should not return an error,
		// as that will restart the consumer unnecessarily.
		c.logger.Error("failed to consume message, skipping due to message_marking config",
			zap.Error(err),
			zap.String("topic", message.Topic),
			zap.Int32("partition", claim.Partition()),
			zap.Int64("offset", message.Offset),
		)
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
	return nil
}

func (c *consumerGroupHandler) getNextBackoff() time.Duration {
	c.backOffMutex.Lock()
	defer c.backOffMutex.Unlock()
	return c.backOff.NextBackOff()
}

func (c *consumerGroupHandler) resetBackoff() {
	c.backOffMutex.Lock()
	defer c.backOffMutex.Unlock()
	c.backOff.Reset()
}

// byteSize calculates kafka message size according to
// https://pkg.go.dev/github.com/Shopify/sarama#ProducerMessage.ByteSize
func byteSize(m *sarama.ConsumerMessage) int64 {
	size := 5*binary.MaxVarintLen32 + binary.MaxVarintLen64 + 1
	for _, h := range m.Headers {
		size += len(h.Key) + len(h.Value) + 2*binary.MaxVarintLen32
	}
	if m.Key != nil {
		size += len(m.Key)
	}
	if m.Value != nil {
		size += len(m.Value)
	}
	return int64(size)
}
