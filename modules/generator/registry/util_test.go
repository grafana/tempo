package registry

import (
	"testing"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

func TestGetSeriesLabels(t *testing.T) {
	for _, tc := range []struct {
		name           string
		lbls           labels.Labels
		externalLabels map[string]string
	}{
		{
			name: "without external labels",
			lbls: labels.FromStrings("service", "api", "span_name", "GET /api"),
		},
		{
			name: "with external labels",
			lbls: labels.FromStrings("service", "api"),
			externalLabels: map[string]string{
				"cluster": "prod",
			},
		},
		{
			name: "with existing metric name",
			lbls: labels.FromStrings(model.MetricNameLabel, "old_metric", "service", "api"),
		},
		{
			name: "with reserved label before metric name",
			lbls: labels.FromStrings("__foo__", "bar", "service", "api"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, getSeriesLabelsWithBuilder("test_metric", tc.lbls, tc.externalLabels), getSeriesLabels("test_metric", tc.lbls, tc.externalLabels))
		})
	}
}

func getSeriesLabelsWithBuilder(metricName string, lbls labels.Labels, externalLabels map[string]string) labels.Labels {
	var builder labels.Builder
	builder.Reset(lbls)
	for name, value := range externalLabels {
		builder.Set(name, value)
	}
	builder.Set(model.MetricNameLabel, metricName)
	return builder.Labels()
}
