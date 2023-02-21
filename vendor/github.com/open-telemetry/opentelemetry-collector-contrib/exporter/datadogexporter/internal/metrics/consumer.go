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
	"context"

	"github.com/DataDog/datadog-agent/pkg/otlp/model/translator"
	"github.com/DataDog/datadog-agent/pkg/quantile"
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"go.opentelemetry.io/collector/component"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/metrics/sketches"
)

var _ translator.Consumer = (*Consumer)(nil)
var _ translator.HostConsumer = (*Consumer)(nil)
var _ translator.TagsConsumer = (*Consumer)(nil)
var _ translator.APMStatsConsumer = (*Consumer)(nil)

// Consumer implements translator.Consumer. It records consumed metrics, sketches and
// APM stats payloads. It provides them to the caller using the All method.
type Consumer struct {
	ms        []datadogV2.MetricSeries
	sl        sketches.SketchSeriesList
	as        []pb.ClientStatsPayload
	seenHosts map[string]struct{}
	seenTags  map[string]struct{}
}

// NewConsumer creates a new Datadog consumer. It implements translator.Consumer.
func NewConsumer() *Consumer {
	return &Consumer{
		seenHosts: make(map[string]struct{}),
		seenTags:  make(map[string]struct{}),
	}
}

// toDataType maps translator datatypes to DatadogV2's datatypes.
func (c *Consumer) toDataType(dt translator.MetricDataType) (out datadogV2.MetricIntakeType) {
	out = datadogV2.METRICINTAKETYPE_UNSPECIFIED

	switch dt {
	case translator.Count:
		out = datadogV2.METRICINTAKETYPE_COUNT
	case translator.Gauge:
		out = datadogV2.METRICINTAKETYPE_GAUGE
	}

	return
}

// runningMetrics gets the running metrics for the exporter.
func (c *Consumer) runningMetrics(timestamp uint64, buildInfo component.BuildInfo) (series []datadogV2.MetricSeries) {
	for host := range c.seenHosts {
		// Report the host as running
		runningMetric := DefaultMetrics("metrics", host, timestamp, buildInfo)
		series = append(series, runningMetric...)
	}

	for tag := range c.seenTags {
		runningMetrics := DefaultMetrics("metrics", "", timestamp, buildInfo)
		for i := range runningMetrics {
			runningMetrics[i].Tags = append(runningMetrics[i].Tags, tag)
		}
		series = append(series, runningMetrics...)
	}

	return
}

// All gets all metrics (consumed metrics and running metrics).
func (c *Consumer) All(timestamp uint64, buildInfo component.BuildInfo, tags []string) ([]datadogV2.MetricSeries, sketches.SketchSeriesList, []pb.ClientStatsPayload) {
	series := c.ms
	series = append(series, c.runningMetrics(timestamp, buildInfo)...)
	if len(tags) == 0 {
		return series, c.sl, c.as
	}
	for i := range series {
		series[i].Tags = append(series[i].Tags, tags...)
	}
	for i := range c.sl {
		c.sl[i].Tags = append(c.sl[i].Tags, tags...)
	}
	for i := range c.as {
		c.as[i].Tags = append(c.as[i].Tags, tags...)
	}
	return series, c.sl, c.as
}

// ConsumeAPMStats implements translator.APMStatsConsumer.
func (c *Consumer) ConsumeAPMStats(s pb.ClientStatsPayload) {
	c.as = append(c.as, s)
}

// ConsumeTimeSeries implements the translator.Consumer interface.
func (c *Consumer) ConsumeTimeSeries(
	_ context.Context,
	dims *translator.Dimensions,
	typ translator.MetricDataType,
	timestamp uint64,
	value float64,
) {
	dt := c.toDataType(typ)
	met := NewMetric(dims.Name(), dt, timestamp, value, dims.Tags())
	met.SetResources([]datadogV2.MetricResource{
		{
			Name: datadog.PtrString(dims.Host()),
			Type: datadog.PtrString("host"),
		},
	})
	c.ms = append(c.ms, met)
}

// ConsumeSketch implements the translator.Consumer interface.
func (c *Consumer) ConsumeSketch(
	_ context.Context,
	dims *translator.Dimensions,
	timestamp uint64,
	sketch *quantile.Sketch,
) {
	c.sl = append(c.sl, sketches.SketchSeries{
		Name:     dims.Name(),
		Tags:     dims.Tags(),
		Host:     dims.Host(),
		Interval: 1,
		Points: []sketches.SketchPoint{{
			Ts:     int64(timestamp / 1e9),
			Sketch: sketch,
		}},
	})
}

// ConsumeHost implements the translator.HostConsumer interface.
func (c *Consumer) ConsumeHost(host string) {
	c.seenHosts[host] = struct{}{}
}

// ConsumeTag implements the translator.TagsConsumer interface.
func (c *Consumer) ConsumeTag(tag string) {
	c.seenTags[tag] = struct{}{}
}
