// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"context"
	"iter"
	"strconv"

	"github.com/cenkalti/backoff/v4"
	"github.com/twmb/franz-go/pkg/kgo"
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

type consumeMessageFunc func(ctx context.Context, record *kgo.Record, attrs attribute.Set) error

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

		headerAttrKeys := buildHeaderAttrKeys(config)
		return func(ctx context.Context, record *kgo.Record, attrs attribute.Set) error {
			return processMessage(ctx, record, config, set.Logger, telBldr,
				&logsHandler{
					unmarshaler: unmarshaler,
					obsrecv:     obsrecv,
					consumer:    nextConsumer,
					encoding:    config.Logs.Encoding,
				},
				attrs,
				headerAttrKeys,
			)
		}, nil
	}
	return newReceiver(config, set, config.Logs.Topics, config.Logs.ExcludeTopics, newConsumeMessageFunc)
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

		headerAttrKeys := buildHeaderAttrKeys(config)
		return func(ctx context.Context, record *kgo.Record, attrs attribute.Set) error {
			return processMessage(ctx, record, config, set.Logger, telBldr,
				&metricsHandler{
					unmarshaler: unmarshaler,
					obsrecv:     obsrecv,
					consumer:    nextConsumer,
					encoding:    config.Metrics.Encoding,
				},
				attrs,
				headerAttrKeys,
			)
		}, nil
	}
	return newReceiver(config, set, config.Metrics.Topics, config.Metrics.ExcludeTopics, newConsumeMessageFunc)
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

		headerAttrKeys := buildHeaderAttrKeys(config)
		return func(ctx context.Context, record *kgo.Record, attrs attribute.Set) error {
			return processMessage(ctx, record, config, set.Logger, telBldr,
				&tracesHandler{
					unmarshaler: unmarshaler,
					obsrecv:     obsrecv,
					consumer:    nextConsumer,
					encoding:    config.Traces.Encoding,
				},
				attrs,
				headerAttrKeys,
			)
		}, nil
	}
	return newReceiver(config, set, config.Traces.Topics, config.Traces.ExcludeTopics, consumeFn)
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

		headerAttrKeys := buildHeaderAttrKeys(config)
		return func(ctx context.Context, record *kgo.Record, attrs attribute.Set) error {
			return processMessage(ctx, record, config, set.Logger, telBldr,
				&profilesHandler{
					unmarshaler: unmarshaler,
					obsrecv:     obsrecv,
					consumer:    nextConsumer,
					encoding:    config.Profiles.Encoding,
				},
				attrs,
				headerAttrKeys,
			)
		}, nil
	}
	return newReceiver(config, set, config.Profiles.Topics, config.Profiles.ExcludeTopics, consumeFn)
}

func newReceiver(
	config *Config,
	set receiver.Settings,
	topics []string,
	excludeTopics []string,
	consumeFn func(host component.Host,
		obsrecv *receiverhelper.ObsReport,
		telBldr *metadata.TelemetryBuilder,
	) (consumeMessageFunc, error),
) (component.Component, error) {
	return newFranzKafkaConsumer(config, set, topics, excludeTopics, consumeFn)
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
	return h.obsrecv.StartProfilesOp(ctx)
}

func (h *profilesHandler) endObsReport(ctx context.Context, n int, err error) {
	h.obsrecv.EndProfilesOp(ctx, h.encoding, n, err)
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

// processMessage is a generic function that processes a Kafka record (*kgo.Record) using a messageHandler
func processMessage[T plog.Logs | pmetric.Metrics | ptrace.Traces | pprofile.Profiles](
	ctx context.Context,
	record *kgo.Record,
	config *Config,
	logger *zap.Logger,
	telBldr *metadata.TelemetryBuilder,
	handler messageHandler[T],
	attrs attribute.Set,
	headerAttrKeys map[string]string,
) error {
	if logger.Core().Enabled(zap.DebugLevel) {
		logger.Debug("kafka message received",
			zap.String("value", string(record.Value)),
			zap.Time("timestamp", record.Timestamp),
			zap.String("topic", record.Topic),
			zap.Int32("partition", record.Partition),
			zap.Int64("offset", record.Offset),
		)
	}

	ctx = contextWithMetadata(ctx, record)

	obsCtx := handler.startObsReport(ctx)
	data, n, err := handler.unmarshalData(record.Value)
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
			record.Headers, headerAttrKeys,
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

func getMessageHeaderResourceAttributes(headers []kgo.RecordHeader, headerKeys map[string]string) iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for rawKey, attrKey := range headerKeys {
			for _, h := range headers {
				if h.Key == rawKey {
					if !yield(attrKey, string(h.Value)) {
						return
					}
					break
				}
			}
		}
	}
}

// buildHeaderAttrKeys pre-computes the mapping from raw header names to their
// "kafka.header." prefixed attribute keys. Returns nil when header extraction
// is disabled.
func buildHeaderAttrKeys(config *Config) map[string]string {
	if !config.HeaderExtraction.ExtractHeaders {
		return nil
	}
	m := make(map[string]string, len(config.HeaderExtraction.Headers))
	for _, h := range config.HeaderExtraction.Headers {
		m[h] = "kafka.header." + h
	}
	return m
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

func contextWithMetadata(ctx context.Context, record *kgo.Record) context.Context {
	m := map[string][]string{
		"kafka.topic":     {record.Topic},
		"kafka.partition": {strconv.FormatInt(int64(record.Partition), 10)},
		"kafka.offset":    {strconv.FormatInt(record.Offset, 10)},
	}
	for _, h := range record.Headers {
		m[h.Key] = append(m[h.Key], string(h.Value))
	}
	return client.NewContext(ctx, client.Info{Metadata: client.NewMetadata(m)})
}
