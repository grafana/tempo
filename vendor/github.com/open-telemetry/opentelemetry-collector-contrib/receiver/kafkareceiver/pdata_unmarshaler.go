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
)

type pdataLogsUnmarshaler struct {
	plog.Unmarshaler
	encoding string
}

func (p pdataLogsUnmarshaler) Unmarshal(buf []byte) (plog.Logs, error) {
	return p.Unmarshaler.UnmarshalLogs(buf)
}

func (p pdataLogsUnmarshaler) Encoding() string {
	return p.encoding
}

func newPdataLogsUnmarshaler(unmarshaler plog.Unmarshaler, encoding string) LogsUnmarshaler {
	return pdataLogsUnmarshaler{
		Unmarshaler: unmarshaler,
		encoding:    encoding,
	}
}

type pdataTracesUnmarshaler struct {
	ptrace.Unmarshaler
	encoding string
}

func (p pdataTracesUnmarshaler) Unmarshal(buf []byte) (ptrace.Traces, error) {
	return p.Unmarshaler.UnmarshalTraces(buf)
}

func (p pdataTracesUnmarshaler) Encoding() string {
	return p.encoding
}

func newPdataTracesUnmarshaler(unmarshaler ptrace.Unmarshaler, encoding string) TracesUnmarshaler {
	return pdataTracesUnmarshaler{
		Unmarshaler: unmarshaler,
		encoding:    encoding,
	}
}

type pdataMetricsUnmarshaler struct {
	pmetric.Unmarshaler
	encoding string
}

func (p pdataMetricsUnmarshaler) Unmarshal(buf []byte) (pmetric.Metrics, error) {
	return p.Unmarshaler.UnmarshalMetrics(buf)
}

func (p pdataMetricsUnmarshaler) Encoding() string {
	return p.encoding
}

func newPdataMetricsUnmarshaler(unmarshaler pmetric.Unmarshaler, encoding string) MetricsUnmarshaler {
	return pdataMetricsUnmarshaler{
		Unmarshaler: unmarshaler,
		encoding:    encoding,
	}
}
