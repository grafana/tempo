package test

import (
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func GetCounterValue(metric prometheus.Counter) (float64, error) {
	var m = &dto.Metric{}
	err := metric.Write(m)
	if err != nil {
		return 0, err
	}
	return m.Counter.GetValue(), nil
}

func GetGaugeValue(metric prometheus.Gauge) (float64, error) {
	var m = &dto.Metric{}
	err := metric.Write(m)
	if err != nil {
		return 0, err
	}
	return m.Gauge.GetValue(), nil
}

func GetGaugeVecValue(metric *prometheus.GaugeVec, labels ...string) (float64, error) {
	var m = &dto.Metric{}
	err := metric.WithLabelValues(labels...).Write(m)
	if err != nil {
		return 0, err
	}
	return m.Gauge.GetValue(), nil
}

func GetCounterVecValue(metric *prometheus.CounterVec, label string) (float64, error) {
	var m = &dto.Metric{}
	if err := metric.WithLabelValues(label).Write(m); err != nil {
		return 0, err
	}
	return m.Counter.GetValue(), nil
}

func GetHistogramValue(m *prometheus.HistogramVec, labels ...string) (map[float64]uint64, error) {
	metric, err := m.MetricVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		return nil, err
	}
	var dtoMetric = &dto.Metric{}
	err = metric.Write(dtoMetric)
	if err != nil {
		return nil, err
	}

	ret := map[float64]uint64{}
	for _, b := range dtoMetric.Histogram.Bucket {
		ret[b.GetUpperBound()] = b.GetCumulativeCount()
	}
	ret[-1] = dtoMetric.Histogram.GetSampleCount()

	return ret, nil
}
