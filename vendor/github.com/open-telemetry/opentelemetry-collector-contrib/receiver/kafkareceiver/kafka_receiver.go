// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"context"
	"errors"
	"iter"
	"strconv"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/cenkalti/backoff/v4"
	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
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
	attrTopic        = "topic"
	attrPartition    = "partition"
)

var errMemoryLimiterDataRefused = errors.New("data refused due to high memory usage")

type consumeMessageFunc func(ctx context.Context, message *sarama.ConsumerMessage) error

// messageHandler provides a generic interface for handling messages for a pdata type.
type messageHandler[T plog.Logs | pmetric.Metrics | ptrace.Traces] interface {
	// unmarshalData unmarshals the message payload into a pdata type (plog.Logs, etc.)
	// and returns the number of items (log records, metric data points, spans) within it.
	unmarshalData(data []byte) (T, int, error)

	// consumeData passes the unmarshaled data to the next consumer for the signal type.
	// This simply calls the signal-specific Consume* method.
	consumeData(ctx context.Context, data T) error

	// getResources returns the resources associated with the unmarshaled data.
	// This is used for header extraction for adding resource attributes.
	getResources(T) iter.Seq[pcommon.Resource]

	// startObsReport starts an observation report for the unmarshaled data.
	//
	// This simply calls the the signal-specific receiverhelper.ObsReport.Start*Op method.
	startObsReport(ctx context.Context) context.Context

	// endObsReport ends the observation report for the unmarshaled data.
	//
	// This simply calls the signal-specific receiverherlper.ObsReport.End*Op method,
	// passing the configured encoding and number of items returned by unmarshalData.
	endObsReport(ctx context.Context, n int, err error)
}

func newLogsReceiver(config *Config, set receiver.Settings, nextConsumer consumer.Logs) (receiver.Logs, error) {
	newConsumeMessageFunc := func(c *consumerGroupHandler, host component.Host) (consumeMessageFunc, error) {
		unmarshaler, err := newLogsUnmarshaler(config.Logs.Encoding, set, host)
		if err != nil {
			return nil, err
		}
		return newMessageHandlerConsumeFunc(
			config, set.Logger,
			c.telemetryBuilder.KafkaReceiverUnmarshalFailedLogRecords,
			[]metric.AddOption{
				metric.WithAttributeSet(attribute.NewSet(
					attribute.String(attrInstanceName, c.id.String()),
					attribute.String(attrTopic, config.Logs.Topic),
				)),
			},
			&logsHandler{
				unmarshaler: unmarshaler,
				obsrecv:     c.obsrecv,
				consumer:    nextConsumer,
				encoding:    config.Logs.Encoding,
			},
		), nil
	}
	return newKafkaConsumer(config, set, []string{config.Logs.Topic}, newConsumeMessageFunc)
}

func newMetricsReceiver(config *Config, set receiver.Settings, nextConsumer consumer.Metrics) (receiver.Metrics, error) {
	newConsumeMessageFunc := func(c *consumerGroupHandler, host component.Host) (consumeMessageFunc, error) {
		unmarshaler, err := newMetricsUnmarshaler(config.Metrics.Encoding, set, host)
		if err != nil {
			return nil, err
		}
		return newMessageHandlerConsumeFunc(
			config, set.Logger,
			c.telemetryBuilder.KafkaReceiverUnmarshalFailedMetricPoints,
			[]metric.AddOption{
				metric.WithAttributeSet(attribute.NewSet(
					attribute.String(attrInstanceName, c.id.String()),
					attribute.String(attrTopic, config.Metrics.Topic),
				)),
			},
			&metricsHandler{
				unmarshaler: unmarshaler,
				obsrecv:     c.obsrecv,
				consumer:    nextConsumer,
				encoding:    config.Metrics.Encoding,
			},
		), nil
	}
	return newKafkaConsumer(config, set, []string{config.Metrics.Topic}, newConsumeMessageFunc)
}

func newTracesReceiver(config *Config, set receiver.Settings, nextConsumer consumer.Traces) (receiver.Traces, error) {
	newConsumeMessageFunc := func(c *consumerGroupHandler, host component.Host) (consumeMessageFunc, error) {
		unmarshaler, err := newTracesUnmarshaler(config.Traces.Encoding, set, host)
		if err != nil {
			return nil, err
		}
		return newMessageHandlerConsumeFunc(
			config, set.Logger,
			c.telemetryBuilder.KafkaReceiverUnmarshalFailedSpans,
			[]metric.AddOption{
				metric.WithAttributeSet(attribute.NewSet(
					attribute.String(attrInstanceName, c.id.String()),
					attribute.String(attrTopic, config.Traces.Topic),
				)),
			},
			&tracesHandler{
				unmarshaler: unmarshaler,
				obsrecv:     c.obsrecv,
				consumer:    nextConsumer,
				encoding:    config.Traces.Encoding,
			},
		), nil
	}
	return newKafkaConsumer(config, set, []string{config.Traces.Topic}, newConsumeMessageFunc)
}

func newMessageHandlerConsumeFunc[T plog.Logs | pmetric.Metrics | ptrace.Traces](
	config *Config,
	logger *zap.Logger,
	unmarshalFailedCounter metric.Int64Counter,
	metricAddOpts []metric.AddOption,
	h messageHandler[T],
) consumeMessageFunc {
	return func(ctx context.Context, message *sarama.ConsumerMessage) (err error) {
		ctx = h.startObsReport(ctx)
		var data T
		var n int
		defer func() {
			h.endObsReport(ctx, n, err)
		}()

		data, n, err = h.unmarshalData(message.Value)
		if err != nil {
			logger.Error("failed to unmarshal message", zap.Error(err))
			metricAddOpts = append(metricAddOpts, metric.WithAttributes(attribute.String(attrPartition, strconv.Itoa(int(message.Partition)))))
			unmarshalFailedCounter.Add(ctx, 1, metricAddOpts...)
			return err
		}

		if config.HeaderExtraction.ExtractHeaders {
			for key, value := range getMessageHeaderResourceAttributes(
				message, config.HeaderExtraction.Headers,
			) {
				for resource := range h.getResources(data) {
					resource.Attributes().PutStr(key, value)
				}
			}
		}

		return h.consumeData(ctx, data)
	}
}

type logsHandler struct {
	unmarshaler plog.Unmarshaler
	obsrecv     *receiverhelper.ObsReport
	consumer    consumer.Logs
	encoding    string
}

func (h *logsHandler) unmarshalData(data []byte) (plog.Logs, int, error) {
	logs, err := h.unmarshaler.UnmarshalLogs(data)
	if err != nil {
		return plog.Logs{}, 0, err
	}
	return logs, logs.LogRecordCount(), nil
}

func (h *logsHandler) consumeData(ctx context.Context, data plog.Logs) error {
	return h.consumer.ConsumeLogs(ctx, data)
}

func (h *logsHandler) startObsReport(ctx context.Context) context.Context {
	return h.obsrecv.StartLogsOp(ctx)
}

func (h *logsHandler) endObsReport(ctx context.Context, n int, err error) {
	h.obsrecv.EndLogsOp(ctx, h.encoding, n, err)
}

func (h *logsHandler) getResources(data plog.Logs) iter.Seq[pcommon.Resource] {
	return func(yield func(pcommon.Resource) bool) {
		for _, rm := range data.ResourceLogs().All() {
			if !yield(rm.Resource()) {
				return
			}
		}
	}
}

type metricsHandler struct {
	unmarshaler pmetric.Unmarshaler
	obsrecv     *receiverhelper.ObsReport
	consumer    consumer.Metrics
	encoding    string
}

func (h *metricsHandler) unmarshalData(data []byte) (pmetric.Metrics, int, error) {
	metrics, err := h.unmarshaler.UnmarshalMetrics(data)
	if err != nil {
		return pmetric.Metrics{}, 0, err
	}
	return metrics, metrics.DataPointCount(), nil
}

func (h *metricsHandler) consumeData(ctx context.Context, data pmetric.Metrics) error {
	return h.consumer.ConsumeMetrics(ctx, data)
}

func (h *metricsHandler) startObsReport(ctx context.Context) context.Context {
	return h.obsrecv.StartMetricsOp(ctx)
}

func (h *metricsHandler) endObsReport(ctx context.Context, n int, err error) {
	h.obsrecv.EndMetricsOp(ctx, h.encoding, n, err)
}

func (h *metricsHandler) getResources(data pmetric.Metrics) iter.Seq[pcommon.Resource] {
	return func(yield func(pcommon.Resource) bool) {
		for _, rm := range data.ResourceMetrics().All() {
			if !yield(rm.Resource()) {
				return
			}
		}
	}
}

type tracesHandler struct {
	unmarshaler ptrace.Unmarshaler
	obsrecv     *receiverhelper.ObsReport
	consumer    consumer.Traces
	encoding    string
}

func (h *tracesHandler) unmarshalData(data []byte) (ptrace.Traces, int, error) {
	traces, err := h.unmarshaler.UnmarshalTraces(data)
	if err != nil {
		return ptrace.Traces{}, 0, err
	}
	return traces, traces.SpanCount(), nil
}

func (h *tracesHandler) consumeData(ctx context.Context, data ptrace.Traces) error {
	return h.consumer.ConsumeTraces(ctx, data)
}

func (h *tracesHandler) startObsReport(ctx context.Context) context.Context {
	return h.obsrecv.StartTracesOp(ctx)
}

func (h *tracesHandler) endObsReport(ctx context.Context, n int, err error) {
	h.obsrecv.EndTracesOp(ctx, h.encoding, n, err)
}

func (h *tracesHandler) getResources(data ptrace.Traces) iter.Seq[pcommon.Resource] {
	return func(yield func(pcommon.Resource) bool) {
		for _, rm := range data.ResourceSpans().All() {
			if !yield(rm.Resource()) {
				return
			}
		}
	}
}

func newKafkaConsumer(
	config *Config,
	set receiver.Settings,
	topics []string,
	newConsumeMessageFunc func(*consumerGroupHandler, component.Host) (consumeMessageFunc, error),
) (*kafkaConsumer, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return &kafkaConsumer{
		config:                config,
		topics:                topics,
		newConsumeMessageFunc: newConsumeMessageFunc,
		settings:              set,
		telemetryBuilder:      telemetryBuilder,
	}, nil
}

// kafkaConsumer consumes messages from a set of Kafka topics,
// decodes telemetry data using a given unmarshaler, and passes
// them to a consumer.
type kafkaConsumer struct {
	config                *Config
	topics                []string
	settings              receiver.Settings
	telemetryBuilder      *metadata.TelemetryBuilder
	newConsumeMessageFunc func(*consumerGroupHandler, component.Host) (consumeMessageFunc, error)

	mu                sync.Mutex
	started           bool
	shutdown          bool
	consumeLoopClosed chan struct{}
	consumerGroup     sarama.ConsumerGroup
}

func (c *kafkaConsumer) Start(_ context.Context, host component.Host) error {
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

	consumerGroup, err := kafka.NewSaramaConsumerGroup(
		context.Background(),
		c.config.ClientConfig,
		c.config.ConsumerConfig,
	)
	if err != nil {
		return err
	}
	c.consumerGroup = consumerGroup

	handler := &consumerGroupHandler{
		id:                c.settings.ID,
		logger:            c.settings.Logger,
		ready:             make(chan bool),
		obsrecv:           obsrecv,
		autocommitEnabled: c.config.AutoCommit.Enable,
		messageMarking:    c.config.MessageMarking,
		telemetryBuilder:  c.telemetryBuilder,
		backOff:           newExponentialBackOff(c.config.ErrorBackOff),
	}
	consumeMessage, err := c.newConsumeMessageFunc(handler, host)
	if err != nil {
		return err
	}
	handler.consumeMessage = consumeMessage

	c.consumeLoopClosed = make(chan struct{})
	c.started = true
	go c.consumeLoop(handler)
	return nil
}

func (c *kafkaConsumer) consumeLoop(handler sarama.ConsumerGroupHandler) {
	defer close(c.consumeLoopClosed)

	ctx := context.Background()
	for {
		// `Consume` should be called inside an infinite loop, when a
		// server-side rebalance happens, the consumer session will need to be
		// recreated to get the new claims
		if err := c.consumerGroup.Consume(ctx, c.topics, handler); err != nil {
			if errors.Is(err, sarama.ErrClosedConsumerGroup) {
				c.settings.Logger.Info("Consumer stopped", zap.Error(ctx.Err()))
				return
			}
			c.settings.Logger.Error("Error from consumer", zap.Error(err))
		}
	}
}

func (c *kafkaConsumer) Shutdown(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.shutdown {
		return nil
	}
	c.shutdown = true
	if !c.started {
		return nil
	}

	if err := c.consumerGroup.Close(); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.consumeLoopClosed:
	}
	return nil
}

type consumerGroupHandler struct {
	id             component.ID
	consumeMessage consumeMessageFunc
	ready          chan bool
	readyCloser    sync.Once
	logger         *zap.Logger

	obsrecv          *receiverhelper.ObsReport
	telemetryBuilder *metadata.TelemetryBuilder

	autocommitEnabled bool
	messageMarking    MessageMarking
	backOff           *backoff.ExponentialBackOff
	backOffMutex      sync.Mutex
}

func (c *consumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	c.readyCloser.Do(func() {
		close(c.ready)
	})
	c.telemetryBuilder.KafkaReceiverPartitionStart.Add(
		session.Context(), 1, metric.WithAttributes(attribute.String(attrInstanceName, c.id.Name())),
	)
	return nil
}

func (c *consumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	c.telemetryBuilder.KafkaReceiverPartitionClose.Add(
		session.Context(), 1, metric.WithAttributes(attribute.String(attrInstanceName, c.id.Name())),
	)
	return nil
}

func (c *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	c.logger.Info("Starting consumer group", zap.Int32("partition", claim.Partition()))
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
	attrs := attribute.NewSet(
		attribute.String(attrInstanceName, c.id.String()),
		attribute.String(attrTopic, message.Topic),
		attribute.String(attrPartition, strconv.Itoa(int(claim.Partition()))),
	)
	c.telemetryBuilder.KafkaReceiverMessages.Add(ctx, 1, metric.WithAttributeSet(attrs))
	c.telemetryBuilder.KafkaReceiverCurrentOffset.Record(ctx, message.Offset, metric.WithAttributeSet(attrs))
	c.telemetryBuilder.KafkaReceiverOffsetLag.Record(ctx, claim.HighWaterMarkOffset()-message.Offset-1, metric.WithAttributeSet(attrs))

	if err := c.consumeMessage(ctx, message); err != nil {
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

// getMessageHeaderResourceAttributes returns key-value pairs to add
// to the resource attributes of decoded data. This is used by the
// "header extraction" feature of the receiver.
func getMessageHeaderResourceAttributes(message *sarama.ConsumerMessage, headers []string) iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for _, header := range headers {
			value, ok := getHeaderValue(message.Headers, header)
			if !ok {
				continue
			}
			if !yield("kafka.header."+header, value) {
				return
			}
		}
	}
}

func getHeaderValue(headers []*sarama.RecordHeader, header string) (string, bool) {
	for _, kafkaHeader := range headers {
		headerKey := string(kafkaHeader.Key)
		if headerKey == header {
			return string(kafkaHeader.Value), true
		}
	}
	return "", false
}
