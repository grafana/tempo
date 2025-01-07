// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/kafkaexporter"

import (
	"github.com/IBM/sarama"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/traceutil"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/batchpersignal"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil"
)

type pdataLogsMarshaler struct {
	marshaler              plog.Marshaler
	encoding               string
	partitionedByResources bool
}

func (p pdataLogsMarshaler) Marshal(ld plog.Logs, topic string) ([]*sarama.ProducerMessage, error) {
	var msgs []*sarama.ProducerMessage
	if p.partitionedByResources {
		logs := ld.ResourceLogs()

		for i := 0; i < logs.Len(); i++ {
			resourceMetrics := logs.At(i)
			hash := pdatautil.MapHash(resourceMetrics.Resource().Attributes())

			newLogs := plog.NewLogs()
			resourceMetrics.CopyTo(newLogs.ResourceLogs().AppendEmpty())

			bts, err := p.marshaler.MarshalLogs(newLogs)
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, &sarama.ProducerMessage{
				Topic: topic,
				Value: sarama.ByteEncoder(bts),
				Key:   sarama.ByteEncoder(hash[:]),
			})
		}
	} else {
		bts, err := p.marshaler.MarshalLogs(ld)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, &sarama.ProducerMessage{
			Topic: topic,
			Value: sarama.ByteEncoder(bts),
		})
	}
	return msgs, nil
}

func (p pdataLogsMarshaler) Encoding() string {
	return p.encoding
}

func newPdataLogsMarshaler(marshaler plog.Marshaler, encoding string, partitionedByResources bool) LogsMarshaler {
	return pdataLogsMarshaler{
		marshaler:              marshaler,
		encoding:               encoding,
		partitionedByResources: partitionedByResources,
	}
}

type pdataMetricsMarshaler struct {
	marshaler              pmetric.Marshaler
	encoding               string
	partitionedByResources bool
}

func (p pdataMetricsMarshaler) Marshal(ld pmetric.Metrics, topic string) ([]*sarama.ProducerMessage, error) {
	var msgs []*sarama.ProducerMessage
	if p.partitionedByResources {
		metrics := ld.ResourceMetrics()

		for i := 0; i < metrics.Len(); i++ {
			resourceMetrics := metrics.At(i)
			hash := pdatautil.MapHash(resourceMetrics.Resource().Attributes())

			newMetrics := pmetric.NewMetrics()
			resourceMetrics.CopyTo(newMetrics.ResourceMetrics().AppendEmpty())

			bts, err := p.marshaler.MarshalMetrics(newMetrics)
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, &sarama.ProducerMessage{
				Topic: topic,
				Value: sarama.ByteEncoder(bts),
				Key:   sarama.ByteEncoder(hash[:]),
			})
		}
	} else {
		bts, err := p.marshaler.MarshalMetrics(ld)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, &sarama.ProducerMessage{
			Topic: topic,
			Value: sarama.ByteEncoder(bts),
		})
	}

	return msgs, nil
}

func (p pdataMetricsMarshaler) Encoding() string {
	return p.encoding
}

func newPdataMetricsMarshaler(marshaler pmetric.Marshaler, encoding string, partitionedByResources bool) MetricsMarshaler {
	return &pdataMetricsMarshaler{
		marshaler:              marshaler,
		encoding:               encoding,
		partitionedByResources: partitionedByResources,
	}
}

type pdataTracesMarshaler struct {
	marshaler            ptrace.Marshaler
	encoding             string
	partitionedByTraceID bool
}

func (p *pdataTracesMarshaler) Marshal(td ptrace.Traces, topic string) ([]*sarama.ProducerMessage, error) {
	var msgs []*sarama.ProducerMessage
	if p.partitionedByTraceID {
		for _, trace := range batchpersignal.SplitTraces(td) {
			bts, err := p.marshaler.MarshalTraces(trace)
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, &sarama.ProducerMessage{
				Topic: topic,
				Value: sarama.ByteEncoder(bts),
				Key:   sarama.ByteEncoder(traceutil.TraceIDToHexOrEmptyString(trace.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).TraceID())),
			})
		}
	} else {
		bts, err := p.marshaler.MarshalTraces(td)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, &sarama.ProducerMessage{
			Topic: topic,
			Value: sarama.ByteEncoder(bts),
		})
	}

	return msgs, nil
}

func (p *pdataTracesMarshaler) Encoding() string {
	return p.encoding
}

func newPdataTracesMarshaler(marshaler ptrace.Marshaler, encoding string, partitionedByTraceID bool) TracesMarshaler {
	return &pdataTracesMarshaler{
		marshaler:            marshaler,
		encoding:             encoding,
		partitionedByTraceID: partitionedByTraceID,
	}
}
