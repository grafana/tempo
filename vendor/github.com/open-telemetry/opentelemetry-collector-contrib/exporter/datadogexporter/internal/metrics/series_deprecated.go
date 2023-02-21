// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/metrics"

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	zorkian "gopkg.in/zorkian/go-datadog-api.v2"
)

type MetricType string

const (
	// Gauge is the Datadog Gauge metric type
	Gauge MetricType = "gauge"
	// Count is the Datadog Count metric type
	Count MetricType = "count"
)

// newZorkianMetric creates a new Zorkian Datadog metric given a name, a Unix nanoseconds timestamp
// a value and a slice of tags
func newZorkianMetric(name string, ts uint64, value float64, tags []string) zorkian.Metric {
	// Transform UnixNano timestamp into Unix timestamp
	// 1 second = 1e9 ns
	timestamp := float64(ts / 1e9)

	metric := zorkian.Metric{
		Points: []zorkian.DataPoint{[2]*float64{&timestamp, &value}},
		Tags:   tags,
	}
	metric.SetMetric(name)
	return metric
}

// NewZorkianMetric creates a new Zorkian Datadog metric given a name, a type, a Unix nanoseconds timestamp
// a value and a slice of tags
func NewZorkianMetric(name string, dt MetricType, ts uint64, value float64, tags []string) zorkian.Metric {
	metric := newZorkianMetric(name, ts, value, tags)
	metric.SetType(string(dt))
	return metric
}

// NewZorkianGauge creates a new Datadog Gauge metric given a name, a Unix nanoseconds timestamp
// a value and a slice of tags
func NewZorkianGauge(name string, ts uint64, value float64, tags []string) zorkian.Metric {
	return NewZorkianMetric(name, Gauge, ts, value, tags)
}

// NewZorkianCount creates a new Datadog count metric given a name, a Unix nanoseconds timestamp
// a value and a slice of tags
func NewZorkianCount(name string, ts uint64, value float64, tags []string) zorkian.Metric {
	return NewZorkianMetric(name, Count, ts, value, tags)
}

// DefaultZorkianMetrics creates built-in metrics to report that an exporter is running
func DefaultZorkianMetrics(exporterType string, hostname string, timestamp uint64, buildInfo component.BuildInfo) []zorkian.Metric {
	var tags []string

	if buildInfo.Version != "" {
		tags = append(tags, "version:"+buildInfo.Version)
	}

	if buildInfo.Command != "" {
		tags = append(tags, "command:"+buildInfo.Command)
	}

	metrics := []zorkian.Metric{
		NewZorkianGauge(fmt.Sprintf("otel.datadog_exporter.%s.running", exporterType), timestamp, 1.0, tags),
	}

	for i := range metrics {
		metrics[i].SetHost(hostname)
	}

	return metrics
}
