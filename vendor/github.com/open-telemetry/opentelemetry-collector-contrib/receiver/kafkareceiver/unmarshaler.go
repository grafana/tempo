// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"

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

// defaultTracesUnmarshalers returns map of supported encodings with TracesUnmarshaler.
func defaultTracesUnmarshalers() map[string]TracesUnmarshaler {
	otlpPb := newPdataTracesUnmarshaler(ptrace.NewProtoUnmarshaler(), defaultEncoding)
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
	otlpPb := newPdataMetricsUnmarshaler(pmetric.NewProtoUnmarshaler(), defaultEncoding)
	return map[string]MetricsUnmarshaler{
		otlpPb.Encoding(): otlpPb,
	}
}

func defaultLogsUnmarshalers() map[string]LogsUnmarshaler {
	otlpPb := newPdataLogsUnmarshaler(plog.NewProtoUnmarshaler(), defaultEncoding)
	return map[string]LogsUnmarshaler{
		otlpPb.Encoding(): otlpPb,
	}
}
