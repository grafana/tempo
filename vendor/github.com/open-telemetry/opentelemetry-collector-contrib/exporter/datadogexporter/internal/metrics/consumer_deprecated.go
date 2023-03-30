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

	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/opentelemetry-mapping-go/pkg/otlp/metrics"
	"github.com/DataDog/opentelemetry-mapping-go/pkg/quantile"
	"go.opentelemetry.io/collector/component"
	zorkian "gopkg.in/zorkian/go-datadog-api.v2"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/metrics/sketches"
)

var _ metrics.Consumer = (*ZorkianConsumer)(nil)
var _ metrics.HostConsumer = (*ZorkianConsumer)(nil)
var _ metrics.TagsConsumer = (*ZorkianConsumer)(nil)
var _ metrics.APMStatsConsumer = (*ZorkianConsumer)(nil)

// ZorkianConsumer implements metrics.Consumer. It records consumed metrics, sketches and
// APM stats payloads. It provides them to the caller using the All method.
type ZorkianConsumer struct {
	ms        []zorkian.Metric
	sl        sketches.SketchSeriesList
	as        []pb.ClientStatsPayload
	seenHosts map[string]struct{}
	seenTags  map[string]struct{}
}

// NewZorkianConsumer creates a new ZorkianConsumer. It implements metrics.Consumer.
func NewZorkianConsumer() *ZorkianConsumer {
	return &ZorkianConsumer{
		seenHosts: make(map[string]struct{}),
		seenTags:  make(map[string]struct{}),
	}
}

// toDataType maps translator datatypes to Zorkian's datatypes.
func (c *ZorkianConsumer) toDataType(dt metrics.DataType) (out MetricType) {
	out = MetricType("unknown")

	switch dt {
	case metrics.Count:
		out = Count
	case metrics.Gauge:
		out = Gauge
	}

	return
}

// runningMetrics gets the running metrics for the exporter.
func (c *ZorkianConsumer) runningMetrics(timestamp uint64, buildInfo component.BuildInfo) (series []zorkian.Metric) {
	for host := range c.seenHosts {
		// Report the host as running
		runningMetric := DefaultZorkianMetrics("metrics", host, timestamp, buildInfo)
		series = append(series, runningMetric...)
	}

	for tag := range c.seenTags {
		runningMetrics := DefaultZorkianMetrics("metrics", "", timestamp, buildInfo)
		for i := range runningMetrics {
			runningMetrics[i].Tags = append(runningMetrics[i].Tags, tag)
		}
		series = append(series, runningMetrics...)
	}

	return
}

// All gets all metrics (consumed metrics and running metrics).
func (c *ZorkianConsumer) All(timestamp uint64, buildInfo component.BuildInfo, tags []string) ([]zorkian.Metric, sketches.SketchSeriesList, []pb.ClientStatsPayload) {
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

// ConsumeAPMStats implements metrics.APMStatsConsumer.
func (c *ZorkianConsumer) ConsumeAPMStats(s pb.ClientStatsPayload) {
	c.as = append(c.as, s)
}

// ConsumeTimeSeries implements the metrics.Consumer interface.
func (c *ZorkianConsumer) ConsumeTimeSeries(
	_ context.Context,
	dims *metrics.Dimensions,
	typ metrics.DataType,
	timestamp uint64,
	value float64,
) {
	dt := c.toDataType(typ)
	met := NewZorkianMetric(dims.Name(), dt, timestamp, value, dims.Tags())
	met.SetHost(dims.Host())
	c.ms = append(c.ms, met)
}

// ConsumeSketch implements the metrics.Consumer interface.
func (c *ZorkianConsumer) ConsumeSketch(
	_ context.Context,
	dims *metrics.Dimensions,
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

// ConsumeHost implements the metrics.HostConsumer interface.
func (c *ZorkianConsumer) ConsumeHost(host string) {
	c.seenHosts[host] = struct{}{}
}

// ConsumeTag implements the metrics.TagsConsumer interface.
func (c *ZorkianConsumer) ConsumeTag(tag string) {
	c.seenTags[tag] = struct{}{}
}
