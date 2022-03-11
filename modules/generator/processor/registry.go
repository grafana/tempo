package processor

import (
	"fmt"
	"math"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	prometheus_model "github.com/prometheus/client_model/go"
	"github.com/prometheus/prometheus/model/exemplar"
	prometheus_labels "github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
)

var (
	activeSeries = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_active_series",
		Help:      "Number of active series in the metrics registry",
	}, []string{"tenant"})
)

// Registry is a prometheus.Registerer that can gather metrics and push them directly into a
// Prometheus storage.Appender.
//
// Currently, only counters and histograms are supported. Descriptions are ignored.
type Registry struct {
	prometheus.Registerer

	gatherer     prometheus.Gatherer
	activeSeries prometheus.Gauge

	// for testing
	now func() time.Time
}

func NewRegistry(externalLabels map[string]string, tenant string) Registry {
	registry := prometheus.NewRegistry()

	registerer := prometheus.WrapRegistererWith(externalLabels, registry)

	return Registry{
		Registerer:   registerer,
		gatherer:     registry,
		activeSeries: activeSeries.WithLabelValues(tenant),
		now:          time.Now,
	}
}

func (r *Registry) Gather(appender storage.Appender) (err error) {
	activeSeriesCount := 0.0
	defer func() {
		if err == nil {
			r.activeSeries.Set(activeSeriesCount)
		}
	}()

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
				labels = appendWithLabel(labels, "__name__", metricFamily.GetName())

				_, err := appender.Append(0, labels, timestamp, metric.GetCounter().GetValue())
				if err != nil {
					return err
				}
				activeSeriesCount++
			}

		case prometheus_model.MetricType_HISTOGRAM:
			for _, metric := range metricFamily.GetMetric() {
				labels := labelPairsToLabels(metric.Label)

				histogram := metric.GetHistogram()

				// _count
				countLabels := copyWithLabel(labels, "__name__", fmt.Sprintf("%s_count", metricFamily.GetName()))
				_, err := appender.Append(0, countLabels, timestamp, float64(histogram.GetSampleCount()))
				if err != nil {
					return err
				}
				activeSeriesCount++

				// _sum
				sumLabels := copyWithLabel(labels, "__name__", fmt.Sprintf("%s_sum", metricFamily.GetName()))
				_, err = appender.Append(0, sumLabels, timestamp, histogram.GetSampleSum())
				if err != nil {
					return err
				}
				activeSeriesCount++

				addedInfBucket := false

				// _bucket
				bucketLabels := copyWithLabel(labels, "__name__", fmt.Sprintf("%s_bucket", metricFamily.GetName()))
				for _, bucket := range histogram.GetBucket() {

					if bucket.GetUpperBound() == math.Inf(1) {
						addedInfBucket = true
					}

					bucketWithLeLabels := copyWithLabel(bucketLabels, "le", fmt.Sprintf("%g", bucket.GetUpperBound()))
					ref, err := appender.Append(0, bucketWithLeLabels, timestamp, float64(bucket.GetCumulativeCount()))
					if err != nil {
						return err
					}
					activeSeriesCount++

					e := bucket.GetExemplar()
					if e != nil {
						_, err = appender.AppendExemplar(ref, bucketWithLeLabels, exemplar.Exemplar{
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
					bucketInfLabels := copyWithLabel(bucketLabels, "le", "+Inf")
					_, err = appender.Append(0, bucketInfLabels, timestamp, float64(histogram.GetSampleCount()))
					if err != nil {
						return err
					}
					activeSeriesCount++
				}
			}

		default:
			return fmt.Errorf("metric type %s is not supported by Registry", metricFamily.GetType())
		}
	}

	return nil
}

// SetTimeNow is used for stubbing time.Now in testing
func (r *Registry) SetTimeNow(now func() time.Time) {
	r.now = now
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

func appendWithLabel(labels prometheus_labels.Labels, name, value string) prometheus_labels.Labels {
	return append(labels, prometheus_labels.Label{
		Name:  name,
		Value: value,
	})
}

func copyWithLabel(labels prometheus_labels.Labels, name, value string) prometheus_labels.Labels {
	labelsCopy := make(prometheus_labels.Labels, len(labels), len(labels)+1)
	copy(labelsCopy, labels)

	return appendWithLabel(labelsCopy, name, value)
}
