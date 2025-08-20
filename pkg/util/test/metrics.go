package test

import (
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func GetCounterValue(metric prometheus.Counter) (float64, error) {
	m := &dto.Metric{}
	err := metric.Write(m)
	if err != nil {
		return 0, err
	}
	return m.Counter.GetValue(), nil
}

func GetGaugeValue(metric prometheus.Gauge) (float64, error) {
	m := &dto.Metric{}
	err := metric.Write(m)
	if err != nil {
		return 0, err
	}
	return m.Gauge.GetValue(), nil
}

func GetGaugeVecValue(metric *prometheus.GaugeVec, labels ...string) (float64, error) {
	m := &dto.Metric{}
	err := metric.WithLabelValues(labels...).Write(m)
	if err != nil {
		return 0, err
	}
	return m.Gauge.GetValue(), nil
}

func GetCounterVecValue(metric *prometheus.CounterVec, label string) (float64, error) {
	m := &dto.Metric{}
	if err := metric.WithLabelValues(label).Write(m); err != nil {
		return 0, err
	}
	return m.Counter.GetValue(), nil
}

func MustGetCounterValue(metric prometheus.Counter) float64 {
	value, err := GetCounterValue(metric)
	if err != nil {
		panic(err)
	}
	return value
}
