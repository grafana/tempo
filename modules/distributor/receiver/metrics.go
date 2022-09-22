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
