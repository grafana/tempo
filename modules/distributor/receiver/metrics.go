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

// The code in this file is taken from multiple files in the Opentelemetry Collector Contrib project:
// go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics

package receiver

import (
	"fmt"

	"contrib.go.opencensus.io/exporter/prometheus"
	prom_client "github.com/prometheus/client_golang/prometheus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

const (
	// ReceiverKey used to identify receivers in metrics and traces.
	ReceiverKey = "receiver"
	// TransportKey used to identify the transport used to received the data.
	TransportKey = "transport"

	// AcceptedSpansKey used to identify spans accepted by the Collector.
	AcceptedSpansKey = "accepted_spans"
	// RefusedSpansKey used to identify spans refused (ie.: not ingested) by the Collector.
	RefusedSpansKey = "refused_spans"

	ReceiverPrefix = ReceiverKey + "/"
)

var (
	TagKeyReceiver, _  = tag.NewKey(ReceiverKey)
	TagKeyTransport, _ = tag.NewKey(TransportKey)

	ReceiverAcceptedSpans = stats.Int64(
		ReceiverPrefix+AcceptedSpansKey,
		"Number of spans successfully pushed into the pipeline.",
		stats.UnitDimensionless)
	ReceiverRefusedSpans = stats.Int64(
		ReceiverPrefix+RefusedSpansKey,
		"Number of spans that could not be pushed into the pipeline.",
		stats.UnitDimensionless)
)

func traceReceiverViews() []*view.View {
	var views []*view.View
	// Receiver traceReceiverViews.
	measures := []*stats.Int64Measure{
		ReceiverAcceptedSpans,
		ReceiverRefusedSpans,
	}
	tagKeys := []tag.Key{
		TagKeyReceiver, TagKeyTransport,
	}
	views = append(views, genViews(measures, tagKeys, view.Sum())...)
	return views
}

func genViews(
	measures []*stats.Int64Measure,
	tagKeys []tag.Key,
	aggregation *view.Aggregation,
) []*view.View {
	views := make([]*view.View, 0, len(measures))
	for _, measure := range measures {
		views = append(views, &view.View{
			Name:        measure.Name(),
			Description: measure.Description(),
			TagKeys:     tagKeys,
			Measure:     measure,
			Aggregation: aggregation,
		})
	}
	return views
}

func newMetricViews() ([]*view.View, error) {
	views := traceReceiverViews()

	err := view.Register(views...)
	if err != nil {
		return nil, fmt.Errorf("failed to register traceReceiverViews: %w", err)
	}

	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace:  "tempo",
		Registerer: prom_client.DefaultRegisterer,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	view.RegisterExporter(pe)

	return views, nil
}
