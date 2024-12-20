// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package noop provides an implementation of the OpenTelemetry metric API that
// produces no telemetry and minimizes used computation resources.
//
// Using this package to implement the OpenTelemetry metric API will
// effectively disable OpenTelemetry.
//
// This implementation can be embedded in other implementations of the
// OpenTelemetry metric API. Doing so will mean the implementation defaults to
// no operation for methods it does not implement.

// Adapted from "go.opentelemetry.io/otel/metric/noop"

package receiver

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/embedded"
	"go.opentelemetry.io/otel/metric/noop"
)

const (
	// These metrics are defined here: https://github.com/open-telemetry/opentelemetry-collector/blob/release/v0.116.x/receiver/receiverhelper/internal/metadata/generated_telemetry.go
	otelcolAcceptedSpansMetricName = "otelcol_receiver_accepted_spans"
	otelcolRefusedSpansMetricName  = "otelcol_receiver_refused_spans"
)

var (
	// Compile-time check this implements the OpenTelemetry API.

	_ metric.MeterProvider = MeterProvider{}
	_ metric.Meter         = Meter{}
	_ metric.Int64Counter  = Int64Counter{}
)

type metrics struct {
	receiverAcceptedSpans *prometheus.CounterVec
	receiverRefusedSpans  *prometheus.CounterVec
}

func newMetrics(reg prometheus.Registerer) *metrics {
	return &metrics{
		receiverAcceptedSpans: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "receiver_accepted_spans",
			Help:      "Number of spans successfully pushed into the pipeline.",
		}, []string{"receiver", "transport"}),
		receiverRefusedSpans: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "receiver_refused_spans",
			Help:      "Number of spans that could not be pushed into the pipeline.",
		}, []string{"receiver", "transport"}),
	}
}

// MeterProvider is an OpenTelemetry No-Op MeterProvider.
type MeterProvider struct {
	embedded.MeterProvider
	metrics *metrics
}

// NewMeterProvider returns a MeterProvider that does not record any telemetry.
func NewMeterProvider(reg prometheus.Registerer) MeterProvider {
	return MeterProvider{
		metrics: newMetrics(reg),
	}
}

// Meter returns an OpenTelemetry Meter that does not record any telemetry.
func (mp MeterProvider) Meter(string, ...metric.MeterOption) metric.Meter {
	return &Meter{
		metrics: mp.metrics,
	}
}

// Meter is an OpenTelemetry No-Op Meter.
type Meter struct {
	// embed the noop Meter, this provides noop implementations for all methods we don't implement ourselves
	noop.Meter
	metrics *metrics
}

// Int64Counter returns a Counter used to record int64 measurements
func (m Meter) Int64Counter(name string, _ ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	switch name {
	case otelcolAcceptedSpansMetricName, otelcolRefusedSpansMetricName:
		return Int64Counter{Name: name, metrics: m.metrics}, nil
	default:
		return noop.Int64Counter{}, nil
	}
}

// Int64Counter is an OpenTelemetry Counter used to record int64 measurements.
type Int64Counter struct {
	embedded.Int64Counter
	Name    string
	metrics *metrics
}

func (r Int64Counter) Add(_ context.Context, value int64, options ...metric.AddOption) {
	// don't do anything for metrics that we don't care
	if r.Name == "" {
		return
	}
	attributes := metric.NewAddConfig(options).Attributes()
	var receiver string
	var transport string

	// attributes are sorted by key, therefore we can expect that "resource" to comes always first. This can be broken in the future.
	kv, found := attributes.Get(0)
	if found {
		receiver = kv.Value.AsString()
	}

	kv, found = attributes.Get(1)
	if found {
		transport = kv.Value.AsString()
	}

	switch r.Name {
	case otelcolAcceptedSpansMetricName:
		r.metrics.receiverAcceptedSpans.WithLabelValues(receiver, transport).Add(float64(value))
	case otelcolRefusedSpansMetricName:
		r.metrics.receiverRefusedSpans.WithLabelValues(receiver, transport).Add(float64(value))
	}
}
