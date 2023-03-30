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

// Parts of this package are based on the code from the datadog-agent,
// https://github.com/DataDog/datadog-agent/blob/main/pkg/metrics/sketch_series.go

// Package sketches is a copy of part from github.com/DataDog/datadog-agent/pkg/metrics.
// TODO(mx-psi): import pkg/metrics from datadog-agent directly
package sketches // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/metrics/sketches"

import (
	"github.com/DataDog/agent-payload/v5/gogen"
	"github.com/DataDog/opentelemetry-mapping-go/pkg/quantile"
)

const (
	SketchSeriesEndpoint string = "/api/beta/sketches"
)

// A SketchSeries is a timeseries of quantile sketches.
type SketchSeries struct {
	Name     string        `json:"metric"`
	Tags     []string      `json:"tags"`
	Host     string        `json:"host"`
	Interval int64         `json:"interval"`
	Points   []SketchPoint `json:"points"`
}

// A SketchPoint represents a quantile sketch at a specific time
type SketchPoint struct {
	Sketch *quantile.Sketch `json:"sketch"`
	Ts     int64            `json:"ts"`
}

// A SketchSeriesList implements marshaler.Marshaler
type SketchSeriesList []SketchSeries

// Marshal encodes this series list.
func (sl SketchSeriesList) Marshal() ([]byte, error) {
	pb := &gogen.SketchPayload{
		Sketches: make([]gogen.SketchPayload_Sketch, 0, len(sl)),
	}

	for _, ss := range sl {
		dsl := make([]gogen.SketchPayload_Sketch_Dogsketch, 0, len(ss.Points))

		for _, p := range ss.Points {
			b := p.Sketch.Basic
			k, n := p.Sketch.Cols()
			dsl = append(dsl, gogen.SketchPayload_Sketch_Dogsketch{
				Ts:  p.Ts,
				Cnt: b.Cnt,
				Min: b.Min,
				Max: b.Max,
				Avg: b.Avg,
				Sum: b.Sum,
				K:   k,
				N:   n,
			})
		}

		pb.Sketches = append(pb.Sketches, gogen.SketchPayload_Sketch{
			Metric:      ss.Name,
			Host:        ss.Host,
			Tags:        ss.Tags,
			Dogsketches: dsl,
		})
	}
	return pb.Marshal()
}
