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

package otlp // import "go.opentelemetry.io/collector/model/otlp"

import (
	"go.opentelemetry.io/collector/model/internal"
	otlplogs "go.opentelemetry.io/collector/model/internal/data/protogen/logs/v1"
	otlpmetrics "go.opentelemetry.io/collector/model/internal/data/protogen/metrics/v1"
	otlptrace "go.opentelemetry.io/collector/model/internal/data/protogen/trace/v1"
	"go.opentelemetry.io/collector/model/pdata"
)

type pbUnmarshaler struct{}

// NewProtobufTracesUnmarshaler returns a model.TracesUnmarshaler. Unmarshals from OTLP binary protobuf bytes.
func NewProtobufTracesUnmarshaler() pdata.TracesUnmarshaler {
	return newPbUnmarshaler()
}

// NewProtobufMetricsUnmarshaler returns a model.MetricsUnmarshaler. Unmarshals from OTLP binary protobuf bytes.
func NewProtobufMetricsUnmarshaler() pdata.MetricsUnmarshaler {
	return newPbUnmarshaler()
}

// NewProtobufLogsUnmarshaler returns a model.LogsUnmarshaler. Unmarshals from OTLP binary protobuf bytes.
func NewProtobufLogsUnmarshaler() pdata.LogsUnmarshaler {
	return newPbUnmarshaler()
}

func newPbUnmarshaler() *pbUnmarshaler {
	return &pbUnmarshaler{}
}

func (d *pbUnmarshaler) UnmarshalLogs(buf []byte) (pdata.Logs, error) {
	ld := &otlplogs.LogsData{}
	err := ld.Unmarshal(buf)
	return pdata.LogsFromInternalRep(internal.LogsFromOtlp(ld)), err
}

func (d *pbUnmarshaler) UnmarshalMetrics(buf []byte) (pdata.Metrics, error) {
	md := &otlpmetrics.MetricsData{}
	err := md.Unmarshal(buf)
	return pdata.MetricsFromInternalRep(internal.MetricsFromOtlp(md)), err
}

func (d *pbUnmarshaler) UnmarshalTraces(buf []byte) (pdata.Traces, error) {
	td := &otlptrace.TracesData{}
	err := td.Unmarshal(buf)
	return pdata.TracesFromInternalRep(internal.TracesFromOtlp(td)), err
}
