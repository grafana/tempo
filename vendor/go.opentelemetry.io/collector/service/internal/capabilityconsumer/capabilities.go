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

package capabilityconsumer // import "go.opentelemetry.io/collector/service/internal/capabilityconsumer"

import (
	"go.opentelemetry.io/collector/consumer"
)

func NewLogs(logs consumer.Logs, cap consumer.Capabilities) consumer.Logs {
	if logs.Capabilities() == cap {
		return logs
	}
	return capLogs{Logs: logs, cap: cap}
}

type capLogs struct {
	consumer.Logs
	cap consumer.Capabilities
}

func (mts capLogs) Capabilities() consumer.Capabilities {
	return mts.cap
}

func NewMetrics(metrics consumer.Metrics, cap consumer.Capabilities) consumer.Metrics {
	if metrics.Capabilities() == cap {
		return metrics
	}
	return capMetrics{Metrics: metrics, cap: cap}
}

type capMetrics struct {
	consumer.Metrics
	cap consumer.Capabilities
}

func (mts capMetrics) Capabilities() consumer.Capabilities {
	return mts.cap
}

func NewTraces(traces consumer.Traces, cap consumer.Capabilities) consumer.Traces {
	if traces.Capabilities() == cap {
		return traces
	}
	return capTraces{Traces: traces, cap: cap}
}

type capTraces struct {
	consumer.Traces
	cap consumer.Capabilities
}

func (mts capTraces) Capabilities() consumer.Capabilities {
	return mts.cap
}
