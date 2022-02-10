package processor

import (
	"fmt"
	"math"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	prometheus_model "github.com/prometheus/client_model/go"
	"github.com/prometheus/prometheus/model/exemplar"
	prometheus_labels "github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
)

// Registry is a prometheus.Registerer that can gather metrics and push them directly into a
// Prometheus storage.Appender.
//
// Currently, only counters and histograms are supported. Descriptions are ignored.
type Registry struct {
	prometheus.Registerer

	gatherer prometheus.Gatherer

	now func() time.Time
}

func NewRegistry(externalLabels map[string]string) Registry {
	registry := prometheus.NewRegistry()

	registerer := prometheus.WrapRegistererWith(externalLabels, registry)

	return Registry{
		Registerer: registerer,
		gatherer:   registry,
		now:        time.Now,
	}
}

func (r *Registry) Gather(appender storage.Appender) error {
	metricFamilies, err := r.gatherer.Gather()
	if err != nil {
		return err
	}

	timestamp := r.now().UnixMilli()

	for _, metricFamily := range metricFamilies {

		switch metricFamily.GetType() {
		case prometheus_model.MetricType_COUNTER:
			for _, metric := range metricFamily.GetMetric() {
				labels := labelPairsToLabels(metric.Label)
				labels = withLabel(labels, "__name__", metricFamily.GetName())

				_, err := appender.Append(0, labels, timestamp, metric.GetCounter().GetValue())
				if err != nil {
					return err
				}
			}

		case prometheus_model.MetricType_HISTOGRAM:
			for _, metric := range metricFamily.GetMetric() {
				labels := labelPairsToLabels(metric.Label)

				histogram := metric.GetHistogram()

				// _count
				countLabels := withLabel(labels, "__name__", fmt.Sprintf("%s_count", metricFamily.GetName()))
				_, err := appender.Append(0, countLabels, timestamp, float64(histogram.GetSampleCount()))
				if err != nil {
					return err
				}

				// _sum
				sumLabels := withLabel(labels, "__name__", fmt.Sprintf("%s_sum", metricFamily.GetName()))
				_, err = appender.Append(0, sumLabels, timestamp, histogram.GetSampleSum())
				if err != nil {
					return err
				}

				addedInfBucket := false

				// _bucket
				bucketLabels := withLabel(labels, "__name__", fmt.Sprintf("%s_bucket", metricFamily.GetName()))
				for _, bucket := range histogram.GetBucket() {

					if bucket.GetUpperBound() == math.Inf(1) {
						addedInfBucket = true
					}

					// TODO make a complete copy of this slice, not sure why this is needed
					//  it works without in registry_test.go but fails in servicegraphs_test.go 🙃
					//  in the servicegraphs test the le labels gets overwritten with the latest value 🤷
					bucketLabelsCopy := make(prometheus_labels.Labels, len(bucketLabels))
					copy(bucketLabelsCopy, bucketLabels)

					bucketWithLeLabels := withLabel(bucketLabelsCopy, "le", fmt.Sprintf("%g", bucket.GetUpperBound()))
					_, err = appender.Append(0, bucketWithLeLabels, timestamp, float64(bucket.GetCumulativeCount()))
					if err != nil {
						return err
					}

					e := bucket.GetExemplar()
					if e != nil {
						_, err = appender.AppendExemplar(0, bucketWithLeLabels, exemplar.Exemplar{
							Labels: labelPairsToLabels(e.GetLabel()),
							Value:  e.GetValue(),
							Ts:     e.GetTimestamp().AsTime().UnixMilli(),
							HasTs:  e.GetTimestamp() != nil,
						})
						if err != nil {
							return err
						}
					}
				}

				if !addedInfBucket {
					// _bucket, le="+Inf"
					bucketInfLabels := withLabel(bucketLabels, "le", "+Inf")
					_, err = appender.Append(0, bucketInfLabels, timestamp, float64(histogram.GetSampleCount()))
					if err != nil {
						return err
					}
				}
			}

		default:
			return fmt.Errorf("metric type %s is not supported by Registry", metricFamily.GetType())
		}
	}

	return nil
}

func labelPairsToLabels(labelPairs []*prometheus_model.LabelPair) prometheus_labels.Labels {
	labels := make(prometheus_labels.Labels, len(labelPairs))

	for i, labelPair := range labelPairs {
		labels[i] = prometheus_labels.Label{
			Name:  labelPair.GetName(),
			Value: labelPair.GetValue(),
		}
	}

	return labels
}

func withLabel(labels prometheus_labels.Labels, name, value string) prometheus_labels.Labels {
	return append(labels, prometheus_labels.Label{
		Name:  name,
		Value: value,
	})
}
