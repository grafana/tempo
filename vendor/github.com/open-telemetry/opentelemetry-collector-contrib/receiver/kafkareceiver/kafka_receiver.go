// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"context"
	"iter"

	"github.com/cenkalti/backoff/v4"
	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/receiver/xreceiver"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver/internal/metadata"
)

const transport = "kafka"

type consumeMessageFunc func(ctx context.Context, message kafkaMessage, attrs attribute.Set) error

type newConsumeMessageFunc func(host component.Host, obsrecv *receiverhelper.ObsReport,
	telBldr *metadata.TelemetryBuilder,
) (consumeMessageFunc, error)

// messageHandler provides a generic interface for handling messages for a pdata type.
type messageHandler[T plog.Logs | pmetric.Metrics | ptrace.Traces | pprofile.Profiles] interface {
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
	// This simply calls the signal-specific receiverhelper.ObsReport.Start*Op method.
	startObsReport(ctx context.Context) context.Context

	// endObsReport ends the observation report for the unmarshaled data.
	//
	// This simply calls the signal-specific receiverhelper.ObsReport.End*Op method,
	// passing the configured encoding and number of items returned by unmarshalData.
	endObsReport(ctx context.Context, n int, err error)

	// getUnmarshalFailureCounter returns the appropriate telemetry counter for unmarshal failures
	getUnmarshalFailureCounter(telBldr *metadata.TelemetryBuilder) metric.Int64Counter
}

func newLogsReceiver(config *Config, set receiver.Settings, nextConsumer consumer.Logs) (receiver.Logs, error) {
	newConsumeMessageFunc := func(host component.Host,
		obsrecv *receiverhelper.ObsReport,
		telBldr *metadata.TelemetryBuilder,
	) (consumeMessageFunc, error) {
		unmarshaler, err := newLogsUnmarshaler(config.Logs.Encoding, set, host)
		if err != nil {
			return nil, err
		}
		return func(ctx context.Context, message kafkaMessage, attrs attribute.Set) error {
			return processMessage(ctx, message, config, set.Logger, telBldr,
				&logsHandler{
					unmarshaler: unmarshaler,
					obsrecv:     obsrecv,
					consumer:    nextConsumer,
					encoding:    config.Logs.Encoding,
				},
				attrs,
			)
		}, nil
	}
	return newReceiver(config, set, []string{config.Logs.Topic}, newConsumeMessageFunc)
}

func newMetricsReceiver(config *Config, set receiver.Settings, nextConsumer consumer.Metrics) (receiver.Metrics, error) {
	newConsumeMessageFunc := func(host component.Host,
		obsrecv *receiverhelper.ObsReport,
		telBldr *metadata.TelemetryBuilder,
	) (consumeMessageFunc, error) {
		unmarshaler, err := newMetricsUnmarshaler(config.Metrics.Encoding, set, host)
		if err != nil {
			return nil, err
		}
		return func(ctx context.Context, message kafkaMessage, attrs attribute.Set) error {
			return processMessage(ctx, message, config, set.Logger, telBldr,
				&metricsHandler{
					unmarshaler: unmarshaler,
					obsrecv:     obsrecv,
					consumer:    nextConsumer,
					encoding:    config.Metrics.Encoding,
				},
				attrs,
			)
		}, nil
	}
	return newReceiver(config, set, []string{config.Metrics.Topic}, newConsumeMessageFunc)
}

func newTracesReceiver(config *Config, set receiver.Settings, nextConsumer consumer.Traces) (receiver.Traces, error) {
	consumeFn := func(host component.Host,
		obsrecv *receiverhelper.ObsReport,
		telBldr *metadata.TelemetryBuilder,
	) (consumeMessageFunc, error) {
		unmarshaler, err := newTracesUnmarshaler(config.Traces.Encoding, set, host)
		if err != nil {
			return nil, err
		}
		return func(ctx context.Context, message kafkaMessage, attrs attribute.Set) error {
			return processMessage(ctx, message, config, set.Logger, telBldr,
				&tracesHandler{
					unmarshaler: unmarshaler,
					obsrecv:     obsrecv,
					consumer:    nextConsumer,
					encoding:    config.Traces.Encoding,
				},
				attrs,
			)
		}, nil
	}
	return newReceiver(config, set, []string{config.Traces.Topic}, consumeFn)
}

func newProfilesReceiver(config *Config, set receiver.Settings, nextConsumer xconsumer.Profiles) (xreceiver.Profiles, error) {
	consumeFn := func(host component.Host,
		obsrecv *receiverhelper.ObsReport,
		telBldr *metadata.TelemetryBuilder,
	) (consumeMessageFunc, error) {
		unmarshaler, err := newProfilesUnmarshaler(config.Profiles.Encoding, set, host)
		if err != nil {
			return nil, err
		}
		return func(ctx context.Context, message kafkaMessage, attrs attribute.Set) error {
			return processMessage(ctx, message, config, set.Logger, telBldr,
				&profilesHandler{
					unmarshaler: unmarshaler,
					obsrecv:     obsrecv,
					consumer:    nextConsumer,
					encoding:    config.Profiles.Encoding,
				},
				attrs,
			)
		}, nil
	}
	return newReceiver(config, set, []string{config.Profiles.Topic}, consumeFn)
}

func newReceiver(
	config *Config,
	set receiver.Settings,
	topics []string,
	consumeFn func(host component.Host,
		obsrecv *receiverhelper.ObsReport,
		telBldr *metadata.TelemetryBuilder,
	) (consumeMessageFunc, error),
) (component.Component, error) {
	if franzGoConsumerFeatureGate.IsEnabled() {
		return newFranzKafkaConsumer(config, set, topics, consumeFn)
	}
	return newSaramaConsumer(config, set, topics, consumeFn)
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

func (*logsHandler) getResources(data plog.Logs) iter.Seq[pcommon.Resource] {
	return func(yield func(pcommon.Resource) bool) {
		for _, rm := range data.ResourceLogs().All() {
			if !yield(rm.Resource()) {
				return
			}
		}
	}
}

func (*logsHandler) getUnmarshalFailureCounter(telBldr *metadata.TelemetryBuilder) metric.Int64Counter {
	return telBldr.KafkaReceiverUnmarshalFailedLogRecords
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

func (*metricsHandler) getResources(data pmetric.Metrics) iter.Seq[pcommon.Resource] {
	return func(yield func(pcommon.Resource) bool) {
		for _, rm := range data.ResourceMetrics().All() {
			if !yield(rm.Resource()) {
				return
			}
		}
	}
}

func (*metricsHandler) getUnmarshalFailureCounter(telBldr *metadata.TelemetryBuilder) metric.Int64Counter {
	return telBldr.KafkaReceiverUnmarshalFailedMetricPoints
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

func (*tracesHandler) getResources(data ptrace.Traces) iter.Seq[pcommon.Resource] {
	return func(yield func(pcommon.Resource) bool) {
		for _, rm := range data.ResourceSpans().All() {
			if !yield(rm.Resource()) {
				return
			}
		}
	}
}

func (*tracesHandler) getUnmarshalFailureCounter(telBldr *metadata.TelemetryBuilder) metric.Int64Counter {
	return telBldr.KafkaReceiverUnmarshalFailedSpans
}

type profilesHandler struct {
	unmarshaler pprofile.Unmarshaler
	obsrecv     *receiverhelper.ObsReport
	consumer    xconsumer.Profiles
	encoding    string
}

func (h *profilesHandler) unmarshalData(data []byte) (pprofile.Profiles, int, error) {
	profiles, err := h.unmarshaler.UnmarshalProfiles(data)
	if err != nil {
		return pprofile.Profiles{}, 0, err
	}
	return profiles, profiles.SampleCount(), nil
}

func (h *profilesHandler) consumeData(ctx context.Context, data pprofile.Profiles) error {
	return h.consumer.ConsumeProfiles(ctx, data)
}

func (h *profilesHandler) startObsReport(ctx context.Context) context.Context {
	return h.obsrecv.StartTracesOp(ctx)
}

func (h *profilesHandler) endObsReport(ctx context.Context, n int, err error) {
	h.obsrecv.EndTracesOp(ctx, h.encoding, n, err)
}

func (*profilesHandler) getResources(data pprofile.Profiles) iter.Seq[pcommon.Resource] {
	return func(yield func(pcommon.Resource) bool) {
		for _, rm := range data.ResourceProfiles().All() {
			if !yield(rm.Resource()) {
				return
			}
		}
	}
}

func (*profilesHandler) getUnmarshalFailureCounter(telBldr *metadata.TelemetryBuilder) metric.Int64Counter {
	return telBldr.KafkaReceiverUnmarshalFailedProfiles
}

// processMessage is a generic function that processes any KafkaMessage using a messageHandler
func processMessage[T plog.Logs | pmetric.Metrics | ptrace.Traces | pprofile.Profiles](
	ctx context.Context,
	message kafkaMessage,
	config *Config,
	logger *zap.Logger,
	telBldr *metadata.TelemetryBuilder,
	handler messageHandler[T],
	attrs attribute.Set,
) error {
	if logger.Core().Enabled(zap.DebugLevel) {
		logger.Debug("kafka message received",
			zap.String("value", string(message.value())),
			zap.Time("timestamp", message.timestamp()),
			zap.String("topic", message.topic()),
			zap.Int32("partition", message.partition()),
			zap.Int64("offset", message.offset()),
		)
	}

	ctx = contextWithHeaders(ctx, message.headers())

	obsCtx := handler.startObsReport(ctx)
	data, n, err := handler.unmarshalData(message.value())
	if err != nil {
		handler.getUnmarshalFailureCounter(telBldr).Add(ctx, 1, metric.WithAttributeSet(attrs))
		logger.Error("failed to unmarshal message", zap.Error(err))
		handler.endObsReport(obsCtx, n, err)
		// Return permanent error for unmarshalling failures
		return consumererror.NewPermanent(err)
	}

	// Add resource attributes from headers if configured
	if config.HeaderExtraction.ExtractHeaders {
		for key, value := range getMessageHeaderResourceAttributes(
			message.headers(), config.HeaderExtraction.Headers,
		) {
			for resource := range handler.getResources(data) {
				resource.Attributes().PutStr(key, value)
			}
		}
	}

	err = handler.consumeData(ctx, data)
	handler.endObsReport(obsCtx, n, err)
	return err
}

func getMessageHeaderResourceAttributes(h messageHeaders, resHeaders []string) iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for _, resHeader := range resHeaders {
			value, ok := h.get(resHeader)
			if !ok {
				continue
			}
			if !yield("kafka.header."+resHeader, value) {
				return
			}
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

func contextWithHeaders(ctx context.Context, headers messageHeaders) context.Context {
	m := make(map[string][]string)
	for header := range headers.all() {
		key := header.key
		value := string(header.value)
		m[key] = append(m[key], value)
	}
	if len(m) == 0 {
		return ctx
	}
	return client.NewContext(ctx, client.Info{Metadata: client.NewMetadata(m)})
}
