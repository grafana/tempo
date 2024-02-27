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
	jaegerProto := jaegerProtoSpanUnmarshaler{}
	jaegerJSON := jaegerJSONSpanUnmarshaler{}
	zipkinProto := newPdataTracesUnmarshaler(zipkinv2.NewProtobufTracesUnmarshaler(false, false), "zipkin_proto")
	zipkinJSON := newPdataTracesUnmarshaler(zipkinv2.NewJSONTracesUnmarshaler(false), "zipkin_json")
	zipkinThrift := newPdataTracesUnmarshaler(zipkinv1.NewThriftTracesUnmarshaler(), "zipkin_thrift")
	return map[string]TracesUnmarshaler{
		otlpPb.Encoding():       otlpPb,
		jaegerProto.Encoding():  jaegerProto,
		jaegerJSON.Encoding():   jaegerJSON,
		zipkinProto.Encoding():  zipkinProto,
		zipkinJSON.Encoding():   zipkinJSON,
		zipkinThrift.Encoding(): zipkinThrift,
	}
}

func defaultMetricsUnmarshalers() map[string]MetricsUnmarshaler {
	otlpPb := newPdataMetricsUnmarshaler(&pmetric.ProtoUnmarshaler{}, defaultEncoding)
	return map[string]MetricsUnmarshaler{
		otlpPb.Encoding(): otlpPb,
	}
}

func defaultLogsUnmarshalers(version string, logger *zap.Logger) map[string]LogsUnmarshaler {
	azureResourceLogs := newAzureResourceLogsUnmarshaler(version, logger)
	otlpPb := newPdataLogsUnmarshaler(&plog.ProtoUnmarshaler{}, defaultEncoding)
	raw := newRawLogsUnmarshaler()
	text := newTextLogsUnmarshaler()
	json := newJSONLogsUnmarshaler()
	return map[string]LogsUnmarshaler{
		azureResourceLogs.Encoding(): azureResourceLogs,
		otlpPb.Encoding():            otlpPb,
		raw.Encoding():               raw,
		text.Encoding():              text,
		json.Encoding():              json,
	}
}
