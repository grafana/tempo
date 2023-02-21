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

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"go.opentelemetry.io/collector/component"
)

// newMetricSeries creates a new Datadog metric series given a name, a Unix nanoseconds timestamp
// a value and a slice of tags
func newMetricSeries(name string, ts uint64, value float64, tags []string) datadogV2.MetricSeries {
	// Transform UnixNano timestamp into Unix timestamp
	// 1 second = 1e9 ns
	timestamp := int64(ts / 1e9)

	metric := datadogV2.MetricSeries{
		Metric: name,
		Points: []datadogV2.MetricPoint{
			{
				Timestamp: datadog.PtrInt64(timestamp),
				Value:     datadog.PtrFloat64(value),
			},
		},
		Tags: tags,
	}
	return metric
}

// NewMetric creates a new DatadogV2 metric given a name, a type, a Unix nanoseconds timestamp
// a value and a slice of tags
func NewMetric(name string, dt datadogV2.MetricIntakeType, ts uint64, value float64, tags []string) datadogV2.MetricSeries {
	metric := newMetricSeries(name, ts, value, tags)
	metric.SetType(dt)
	return metric
}

// NewGauge creates a new DatadogV2 Gauge metric given a name, a Unix nanoseconds timestamp
// a value and a slice of tags
func NewGauge(name string, ts uint64, value float64, tags []string) datadogV2.MetricSeries {
	return NewMetric(name, datadogV2.METRICINTAKETYPE_GAUGE, ts, value, tags)
}

// NewCount creates a new DatadogV2 count metric given a name, a Unix nanoseconds timestamp
// a value and a slice of tags
func NewCount(name string, ts uint64, value float64, tags []string) datadogV2.MetricSeries {
	return NewMetric(name, datadogV2.METRICINTAKETYPE_COUNT, ts, value, tags)
}

// DefaultMetrics creates built-in metrics to report that an exporter is running
func DefaultMetrics(exporterType string, hostname string, timestamp uint64, buildInfo component.BuildInfo) []datadogV2.MetricSeries {
	var tags []string
	if buildInfo.Version != "" {
		tags = append(tags, "version:"+buildInfo.Version)
	}
	if buildInfo.Command != "" {
		tags = append(tags, "command:"+buildInfo.Command)
	}
	metrics := []datadogV2.MetricSeries{
		NewGauge(fmt.Sprintf("otel.datadog_exporter.%s.running", exporterType), timestamp, 1.0, tags),
	}
	for i := range metrics {
		metrics[i].SetResources([]datadogV2.MetricResource{
			{
				Name: datadog.PtrString(hostname),
				Type: datadog.PtrString("host"),
			},
		})
	}
	return metrics
}
