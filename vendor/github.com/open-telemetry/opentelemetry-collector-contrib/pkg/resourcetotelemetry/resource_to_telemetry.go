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

package resourcetotelemetry // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/resourcetotelemetry"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/model/pdata"
)

// Settings defines configuration for converting resource attributes to telemetry attributes.
// When used, it must be embedded in the exporter configuration:
// type Config struct {
//   // ...
//   resourcetotelemetry.Settings `mapstructure:"resource_to_telemetry_conversion"`
// }
type Settings struct {
	// Enabled indicates whether to convert resource attributes to telemetry attributes. Default is `false`.
	Enabled bool `mapstructure:"enabled"`
}

type wrapperMetricsExporter struct {
	component.MetricsExporter
}

func (wme *wrapperMetricsExporter) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	return wme.MetricsExporter.ConsumeMetrics(ctx, convertToMetricsAttributes(md))
}

func (wme *wrapperMetricsExporter) Capabilities() consumer.Capabilities {
	// Always return false since this wrapper clones the data.
	return consumer.Capabilities{MutatesData: false}
}

// WrapMetricsExporter wraps a given component.MetricsExporter and based on the given settings
// converts incoming resource attributes to metrics attributes.
func WrapMetricsExporter(set Settings, exporter component.MetricsExporter) component.MetricsExporter {
	if !set.Enabled {
		return exporter
	}
	return &wrapperMetricsExporter{MetricsExporter: exporter}
}

func convertToMetricsAttributes(md pdata.Metrics) pdata.Metrics {
	cloneMd := md.Clone()
	rms := cloneMd.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		resource := rms.At(i).Resource()

		ilms := rms.At(i).InstrumentationLibraryMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)
			metricSlice := ilm.Metrics()
			for k := 0; k < metricSlice.Len(); k++ {
				metric := metricSlice.At(k)
				addAttributesToMetric(&metric, resource.Attributes())
			}
		}
	}
	return cloneMd
}

// addAttributesToMetric adds additional labels to the given metric
func addAttributesToMetric(metric *pdata.Metric, labelMap pdata.AttributeMap) {
	switch metric.DataType() {
	case pdata.MetricDataTypeGauge:
		addAttributesToNumberDataPoints(metric.Gauge().DataPoints(), labelMap)
	case pdata.MetricDataTypeSum:
		addAttributesToNumberDataPoints(metric.Sum().DataPoints(), labelMap)
	case pdata.MetricDataTypeHistogram:
		addAttributesToHistogramDataPoints(metric.Histogram().DataPoints(), labelMap)
	}
}

func addAttributesToNumberDataPoints(ps pdata.NumberDataPointSlice, newAttributeMap pdata.AttributeMap) {
	for i := 0; i < ps.Len(); i++ {
		joinAttributeMaps(newAttributeMap, ps.At(i).Attributes())
	}
}

func addAttributesToHistogramDataPoints(ps pdata.HistogramDataPointSlice, newAttributeMap pdata.AttributeMap) {
	for i := 0; i < ps.Len(); i++ {
		joinAttributeMaps(newAttributeMap, ps.At(i).Attributes())
	}
}

func joinAttributeMaps(from, to pdata.AttributeMap) {
	from.Range(func(k string, v pdata.AttributeValue) bool {
		to.Upsert(k, v)
		return true
	})
}
