// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/zipkinv1"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/zipkinv2"
)

// TracesUnmarshaler deserializes the message body.
type TracesUnmarshaler interface {
	// Unmarshal deserializes the message body into traces.
	Unmarshal([]byte) (ptrace.Traces, error)
	// Encoding of the serialized messages.
	Encoding() string
}

// MetricsUnmarshaler deserializes the message body
type MetricsUnmarshaler interface {
	// Unmarshal deserializes the message body into traces
	Unmarshal([]byte) (pmetric.Metrics, error)
	// Encoding of the serialized messages
	Encoding() string
}

// LogsUnmarshaler deserializes the message body.
type LogsUnmarshaler interface {
	// Unmarshal deserializes the message body into traces.
	Unmarshal([]byte) (plog.Logs, error)
	// Encoding of the serialized messages.
	Encoding() string
}

type LogsUnmarshalerWithEnc interface {
	LogsUnmarshaler
	// WithEnc sets the character encoding (UTF-8, GBK, etc.) of the unmarshaler.
	WithEnc(string) (LogsUnmarshalerWithEnc, error)
}

// defaultTracesUnmarshalers returns map of supported encodings with TracesUnmarshaler.
func defaultTracesUnmarshalers() map[string]TracesUnmarshaler {
	otlpPb := newPdataTracesUnmarshaler(&ptrace.ProtoUnmarshaler{}, defaultEncoding)
	otlpJSON := newPdataTracesUnmarshaler(&ptrace.JSONUnmarshaler{}, "otlp_json")
	jaegerProto := jaegerProtoSpanUnmarshaler{}
	jaegerJSON := jaegerJSONSpanUnmarshaler{}
	zipkinProto := newPdataTracesUnmarshaler(zipkinv2.NewProtobufTracesUnmarshaler(false, false), "zipkin_proto")
	zipkinJSON := newPdataTracesUnmarshaler(zipkinv2.NewJSONTracesUnmarshaler(false), "zipkin_json")
	zipkinThrift := newPdataTracesUnmarshaler(zipkinv1.NewThriftTracesUnmarshaler(), "zipkin_thrift")
	return map[string]TracesUnmarshaler{
		otlpPb.Encoding():       otlpPb,
		otlpJSON.Encoding():     otlpJSON,
		jaegerProto.Encoding():  jaegerProto,
		jaegerJSON.Encoding():   jaegerJSON,
		zipkinProto.Encoding():  zipkinProto,
		zipkinJSON.Encoding():   zipkinJSON,
		zipkinThrift.Encoding(): zipkinThrift,
	}
}

func defaultMetricsUnmarshalers() map[string]MetricsUnmarshaler {
	otlpPb := newPdataMetricsUnmarshaler(&pmetric.ProtoUnmarshaler{}, defaultEncoding)
	otlpJSON := newPdataMetricsUnmarshaler(&pmetric.JSONUnmarshaler{}, "otlp_json")
	return map[string]MetricsUnmarshaler{
		otlpPb.Encoding():   otlpPb,
		otlpJSON.Encoding(): otlpJSON,
	}
}

func defaultLogsUnmarshalers(version string, logger *zap.Logger) map[string]LogsUnmarshaler {
	azureResourceLogs := newAzureResourceLogsUnmarshaler(version, logger)
	otlpPb := newPdataLogsUnmarshaler(&plog.ProtoUnmarshaler{}, defaultEncoding)
	otlpJSON := newPdataLogsUnmarshaler(&plog.JSONUnmarshaler{}, "otlp_json")
	raw := newRawLogsUnmarshaler()
	text := newTextLogsUnmarshaler()
	json := newJSONLogsUnmarshaler()
	return map[string]LogsUnmarshaler{
		azureResourceLogs.Encoding(): azureResourceLogs,
		otlpPb.Encoding():            otlpPb,
		otlpJSON.Encoding():          otlpJSON,
		raw.Encoding():               raw,
		text.Encoding():              text,
		json.Encoding():              json,
	}
}

// tracesEncodingUnmarshaler is a wrapper around ptrace.Unmarshaler that implements TracesUnmarshaler.
type tracesEncodingUnmarshaler struct {
	unmarshaler ptrace.Unmarshaler
	encoding    string
}

func (t *tracesEncodingUnmarshaler) Unmarshal(data []byte) (ptrace.Traces, error) {
	return t.unmarshaler.UnmarshalTraces(data)
}

func (t *tracesEncodingUnmarshaler) Encoding() string {
	return t.encoding
}

// metricsEncodingUnmarshaler is a wrapper around pmetric.Unmarshaler that implements MetricsUnmarshaler.
type metricsEncodingUnmarshaler struct {
	unmarshaler pmetric.Unmarshaler
	encoding    string
}

func (m *metricsEncodingUnmarshaler) Unmarshal(data []byte) (pmetric.Metrics, error) {
	return m.unmarshaler.UnmarshalMetrics(data)
}

func (m *metricsEncodingUnmarshaler) Encoding() string {
	return m.encoding
}

// logsEncodingUnmarshaler is a wrapper around plog.Unmarshaler that implements LogsUnmarshaler.
type logsEncodingUnmarshaler struct {
	unmarshaler plog.Unmarshaler
	encoding    string
}

func (l *logsEncodingUnmarshaler) Unmarshal(data []byte) (plog.Logs, error) {
	return l.unmarshaler.UnmarshalLogs(data)
}

func (l *logsEncodingUnmarshaler) Encoding() string {
	return l.encoding
}
