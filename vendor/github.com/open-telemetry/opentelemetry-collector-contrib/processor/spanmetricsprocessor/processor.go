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

package spanmetricsprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanmetricsprocessor"

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/model/pdata"
	conventions "go.opentelemetry.io/collector/model/semconv/v1.5.0"
	"go.uber.org/zap"
)

const (
	serviceNameKey     = conventions.AttributeServiceName
	operationKey       = "operation"   // OpenTelemetry non-standard constant.
	spanKindKey        = "span.kind"   // OpenTelemetry non-standard constant.
	statusCodeKey      = "status.code" // OpenTelemetry non-standard constant.
	metricKeySeparator = string(byte(0))
	traceIDKey         = "trace_id"
)

var (
	maxDuration   = time.Duration(math.MaxInt64)
	maxDurationMs = durationToMillis(maxDuration)

	defaultLatencyHistogramBucketsMs = []float64{
		2, 4, 6, 8, 10, 50, 100, 200, 400, 800, 1000, 1400, 2000, 5000, 10_000, 15_000, maxDurationMs,
	}
)

type exemplarData struct {
	traceID pdata.TraceID
	value   float64
}

type metricKey string

type processorImp struct {
	lock   sync.RWMutex
	logger *zap.Logger
	config Config

	metricsExporter component.MetricsExporter
	nextConsumer    consumer.Traces

	// Additional dimensions to add to metrics.
	dimensions []Dimension

	// The starting time of the data points.
	startTime time.Time

	// Call & Error counts.
	callSum map[metricKey]int64

	// Latency histogram.
	latencyCount         map[metricKey]uint64
	latencySum           map[metricKey]float64
	latencyBucketCounts  map[metricKey][]uint64
	latencyBounds        []float64
	latencyExemplarsData map[metricKey][]exemplarData

	// A cache of dimension key-value maps keyed by a unique identifier formed by a concatenation of its values:
	// e.g. { "foo/barOK": { "serviceName": "foo", "operation": "/bar", "status_code": "OK" }}
	metricKeyToDimensions map[metricKey]pdata.AttributeMap
}

func newProcessor(logger *zap.Logger, config config.Processor, nextConsumer consumer.Traces) (*processorImp, error) {
	logger.Info("Building spanmetricsprocessor")
	pConfig := config.(*Config)

	bounds := defaultLatencyHistogramBucketsMs
	if pConfig.LatencyHistogramBuckets != nil {
		bounds = mapDurationsToMillis(pConfig.LatencyHistogramBuckets)

		// "Catch-all" bucket.
		if bounds[len(bounds)-1] != maxDurationMs {
			bounds = append(bounds, maxDurationMs)
		}
	}

	if err := validateDimensions(pConfig.Dimensions); err != nil {
		return nil, err
	}

	return &processorImp{
		logger:                logger,
		config:                *pConfig,
		startTime:             time.Now(),
		callSum:               make(map[metricKey]int64),
		latencyBounds:         bounds,
		latencySum:            make(map[metricKey]float64),
		latencyCount:          make(map[metricKey]uint64),
		latencyBucketCounts:   make(map[metricKey][]uint64),
		latencyExemplarsData:  make(map[metricKey][]exemplarData),
		nextConsumer:          nextConsumer,
		dimensions:            pConfig.Dimensions,
		metricKeyToDimensions: make(map[metricKey]pdata.AttributeMap),
	}, nil
}

// durationToMillis converts the given duration to the number of milliseconds it represents.
// Note that this can return sub-millisecond (i.e. < 1ms) values as well.
func durationToMillis(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / float64(time.Millisecond.Nanoseconds())
}

func mapDurationsToMillis(vs []time.Duration) []float64 {
	vsm := make([]float64, len(vs))
	for i, v := range vs {
		vsm[i] = durationToMillis(v)
	}
	return vsm
}

// validateDimensions checks duplicates for reserved dimensions and additional dimensions. Considering
// the usage of Prometheus related exporters, we also validate the dimensions after sanitization.
func validateDimensions(dimensions []Dimension) error {
	labelNames := make(map[string]struct{})
	for _, key := range []string{serviceNameKey, spanKindKey, statusCodeKey} {
		labelNames[key] = struct{}{}
		labelNames[sanitize(key)] = struct{}{}
	}
	labelNames[operationKey] = struct{}{}

	for _, key := range dimensions {
		if _, ok := labelNames[key.Name]; ok {
			return fmt.Errorf("duplicate dimension name %s", key.Name)
		}
		labelNames[key.Name] = struct{}{}

		sanitizedName := sanitize(key.Name)
		if sanitizedName == key.Name {
			continue
		}
		if _, ok := labelNames[sanitizedName]; ok {
			return fmt.Errorf("duplicate dimension name %s after sanitization", sanitizedName)
		}
		labelNames[sanitizedName] = struct{}{}
	}

	return nil
}

// Start implements the component.Component interface.
func (p *processorImp) Start(ctx context.Context, host component.Host) error {
	p.logger.Info("Starting spanmetricsprocessor")
	exporters := host.GetExporters()

	var availableMetricsExporters []string

	// The available list of exporters come from any configured metrics pipelines' exporters.
	for k, exp := range exporters[config.MetricsDataType] {
		metricsExp, ok := exp.(component.MetricsExporter)
		if !ok {
			return fmt.Errorf("the exporter %q isn't a metrics exporter", k.String())
		}

		availableMetricsExporters = append(availableMetricsExporters, k.String())

		p.logger.Debug("Looking for spanmetrics exporter from available exporters",
			zap.String("spanmetrics-exporter", p.config.MetricsExporter),
			zap.Any("available-exporters", availableMetricsExporters),
		)
		if k.String() == p.config.MetricsExporter {
			p.metricsExporter = metricsExp
			p.logger.Info("Found exporter", zap.String("spanmetrics-exporter", p.config.MetricsExporter))
			break
		}
	}
	if p.metricsExporter == nil {
		return fmt.Errorf("failed to find metrics exporter: '%s'; please configure metrics_exporter from one of: %+v",
			p.config.MetricsExporter, availableMetricsExporters)
	}
	p.logger.Info("Started spanmetricsprocessor")
	return nil
}

// Shutdown implements the component.Component interface.
func (p *processorImp) Shutdown(ctx context.Context) error {
	p.logger.Info("Shutting down spanmetricsprocessor")
	return nil
}

// Capabilities implements the consumer interface.
func (p *processorImp) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeTraces implements the consumer.Traces interface.
// It aggregates the trace data to generate metrics, forwarding these metrics to the discovered metrics exporter.
// The original input trace data will be forwarded to the next consumer, unmodified.
func (p *processorImp) ConsumeTraces(ctx context.Context, traces pdata.Traces) error {
	p.aggregateMetrics(traces)

	m := p.buildMetrics()

	// Firstly, export metrics to avoid being impacted by downstream trace processor errors/latency.
	if err := p.metricsExporter.ConsumeMetrics(ctx, *m); err != nil {
		return err
	}

	// Forward trace data unmodified.
	return p.nextConsumer.ConsumeTraces(ctx, traces)
}

// buildMetrics collects the computed raw metrics data, builds the metrics object and
// writes the raw metrics data into the metrics object.
func (p *processorImp) buildMetrics() *pdata.Metrics {
	m := pdata.NewMetrics()
	ilm := m.ResourceMetrics().AppendEmpty().InstrumentationLibraryMetrics().AppendEmpty()
	ilm.InstrumentationLibrary().SetName("spanmetricsprocessor")

	p.lock.RLock()
	p.collectCallMetrics(ilm)
	p.collectLatencyMetrics(ilm)
	p.lock.RUnlock()

	return &m
}

// collectLatencyMetrics collects the raw latency metrics, writing the data
// into the given instrumentation library metrics.
func (p *processorImp) collectLatencyMetrics(ilm pdata.InstrumentationLibraryMetrics) {
	for key := range p.latencyCount {
		mLatency := ilm.Metrics().AppendEmpty()
		mLatency.SetDataType(pdata.MetricDataTypeHistogram)
		mLatency.SetName("latency")
		mLatency.Histogram().SetAggregationTemporality(pdata.MetricAggregationTemporalityCumulative)

		timestamp := pdata.NewTimestampFromTime(time.Now())

		dpLatency := mLatency.Histogram().DataPoints().AppendEmpty()
		dpLatency.SetStartTimestamp(pdata.NewTimestampFromTime(p.startTime))
		dpLatency.SetTimestamp(timestamp)
		dpLatency.SetExplicitBounds(p.latencyBounds)
		dpLatency.SetBucketCounts(p.latencyBucketCounts[key])
		dpLatency.SetCount(p.latencyCount[key])
		dpLatency.SetSum(p.latencySum[key])

		setLatencyExemplars(p.latencyExemplarsData[key], timestamp, dpLatency.Exemplars())

		p.metricKeyToDimensions[key].CopyTo(dpLatency.Attributes())
	}
}

// collectCallMetrics collects the raw call count metrics, writing the data
// into the given instrumentation library metrics.
func (p *processorImp) collectCallMetrics(ilm pdata.InstrumentationLibraryMetrics) {
	for key := range p.callSum {
		mCalls := ilm.Metrics().AppendEmpty()
		mCalls.SetDataType(pdata.MetricDataTypeSum)
		mCalls.SetName("calls_total")
		mCalls.Sum().SetIsMonotonic(true)
		mCalls.Sum().SetAggregationTemporality(pdata.MetricAggregationTemporalityCumulative)

		dpCalls := mCalls.Sum().DataPoints().AppendEmpty()
		dpCalls.SetStartTimestamp(pdata.NewTimestampFromTime(p.startTime))
		dpCalls.SetTimestamp(pdata.NewTimestampFromTime(time.Now()))
		dpCalls.SetIntVal(p.callSum[key])

		p.metricKeyToDimensions[key].CopyTo(dpCalls.Attributes())
	}
}

// aggregateMetrics aggregates the raw metrics from the input trace data.
// Each metric is identified by a key that is built from the service name
// and span metadata such as operation, kind, status_code and any additional
// dimensions the user has configured.
func (p *processorImp) aggregateMetrics(traces pdata.Traces) {
	for i := 0; i < traces.ResourceSpans().Len(); i++ {
		rspans := traces.ResourceSpans().At(i)
		r := rspans.Resource()

		attr, ok := r.Attributes().Get(conventions.AttributeServiceName)
		if !ok {
			continue
		}
		serviceName := attr.StringVal()
		p.aggregateMetricsForServiceSpans(rspans, serviceName)
	}
}

func (p *processorImp) aggregateMetricsForServiceSpans(rspans pdata.ResourceSpans, serviceName string) {
	ilsSlice := rspans.InstrumentationLibrarySpans()
	for j := 0; j < ilsSlice.Len(); j++ {
		ils := ilsSlice.At(j)
		spans := ils.Spans()
		for k := 0; k < spans.Len(); k++ {
			span := spans.At(k)
			p.aggregateMetricsForSpan(serviceName, span, rspans.Resource().Attributes())
		}
	}
}

func (p *processorImp) aggregateMetricsForSpan(serviceName string, span pdata.Span, resourceAttr pdata.AttributeMap) {
	latencyInMilliseconds := float64(span.EndTimestamp()-span.StartTimestamp()) / float64(time.Millisecond.Nanoseconds())

	// Binary search to find the latencyInMilliseconds bucket index.
	index := sort.SearchFloat64s(p.latencyBounds, latencyInMilliseconds)

	key := buildKey(serviceName, span, p.dimensions, resourceAttr)

	p.lock.Lock()
	p.cache(serviceName, span, key, resourceAttr)
	p.updateCallMetrics(key)
	p.updateLatencyMetrics(key, latencyInMilliseconds, index)
	p.updateLatencyExemplars(key, latencyInMilliseconds, index, span.TraceID())
	p.lock.Unlock()
}

// updateCallMetrics increments the call count for the given metric key.
func (p *processorImp) updateCallMetrics(key metricKey) {
	p.callSum[key]++
}

// updateLatencyExemplars sets the histogram exemplars for the given metric key and bucket index.
func (p *processorImp) updateLatencyExemplars(key metricKey, value float64, index int, traceID pdata.TraceID) {
	if _, ok := p.latencyExemplarsData[key]; !ok {
		p.latencyExemplarsData[key] = make([]exemplarData, len(p.latencyBounds))
	}
	p.latencyExemplarsData[key][index] = exemplarData{
		traceID: traceID,
		value:   value,
	}
}

// updateLatencyMetrics increments the histogram counts for the given metric key and bucket index.
func (p *processorImp) updateLatencyMetrics(key metricKey, latency float64, index int) {
	if _, ok := p.latencyBucketCounts[key]; !ok {
		p.latencyBucketCounts[key] = make([]uint64, len(p.latencyBounds))
	}
	p.latencySum[key] += latency
	p.latencyCount[key]++
	p.latencyBucketCounts[key][index]++
}

func (p *processorImp) buildDimensionKVs(serviceName string, span pdata.Span, optionalDims []Dimension, resourceAttrs pdata.AttributeMap) pdata.AttributeMap {
	dims := pdata.NewAttributeMap()
	dims.UpsertString(serviceNameKey, serviceName)
	dims.UpsertString(operationKey, span.Name())
	dims.UpsertString(spanKindKey, span.Kind().String())
	dims.UpsertString(statusCodeKey, span.Status().Code().String())
	for _, d := range optionalDims {
		if v, ok := getDimensionValue(d, span.Attributes(), resourceAttrs); ok {
			dims.Upsert(d.Name, v)
		}
	}
	return dims
}

func concatDimensionValue(metricKeyBuilder *strings.Builder, value string, prefixSep bool) {
	// It's worth noting that from pprof benchmarks, WriteString is the most expensive operation of this processor.
	// Specifically, the need to grow the underlying []byte slice to make room for the appended string.
	if prefixSep {
		metricKeyBuilder.WriteString(metricKeySeparator)
	}
	metricKeyBuilder.WriteString(value)
}

// buildKey builds the metric key from the service name and span metadata such as operation, kind, status_code and
// will attempt to add any additional dimensions the user has configured that match the span's attributes
// or resource attributes. If the dimension exists in both, the span's attributes, being the most specific, takes precedence.
//
// The metric key is a simple concatenation of dimension values, delimited by a null character.
func buildKey(serviceName string, span pdata.Span, optionalDims []Dimension, resourceAttrs pdata.AttributeMap) metricKey {
	var metricKeyBuilder strings.Builder
	concatDimensionValue(&metricKeyBuilder, serviceName, false)
	concatDimensionValue(&metricKeyBuilder, span.Name(), true)
	concatDimensionValue(&metricKeyBuilder, span.Kind().String(), true)
	concatDimensionValue(&metricKeyBuilder, span.Status().Code().String(), true)

	for _, d := range optionalDims {
		if v, ok := getDimensionValue(d, span.Attributes(), resourceAttrs); ok {
			concatDimensionValue(&metricKeyBuilder, v.AsString(), true)
		}
	}

	k := metricKey(metricKeyBuilder.String())
	return k
}

// getDimensionValue gets the dimension value for the given configured dimension.
// It searches through the span's attributes first, being the more specific;
// falling back to searching in resource attributes if it can't be found in the span.
// Finally, falls back to the configured default value if provided.
//
// The ok flag indicates if a dimension value was fetched in order to differentiate
// an empty string value from a state where no value was found.
func getDimensionValue(d Dimension, spanAttr pdata.AttributeMap, resourceAttr pdata.AttributeMap) (v pdata.AttributeValue, ok bool) {
	// The more specific span attribute should take precedence.
	if attr, exists := spanAttr.Get(d.Name); exists {
		return attr, true
	}
	if attr, exists := resourceAttr.Get(d.Name); exists {
		return attr, true
	}
	// Set the default if configured, otherwise this metric will have no value set for the dimension.
	if d.Default != nil {
		return pdata.NewAttributeValueString(*d.Default), true
	}
	return v, ok
}

// cache the dimension key-value map for the metricKey if there is a cache miss.
// This enables a lookup of the dimension key-value map when constructing the metric like so:
//   LabelsMap().InitFromMap(p.metricKeyToDimensions[key])
func (p *processorImp) cache(serviceName string, span pdata.Span, k metricKey, resourceAttrs pdata.AttributeMap) {
	if _, ok := p.metricKeyToDimensions[k]; !ok {
		p.metricKeyToDimensions[k] = p.buildDimensionKVs(serviceName, span, p.dimensions, resourceAttrs)
	}
}

// copied from prometheus-go-metric-exporter
// sanitize replaces non-alphanumeric characters with underscores in s.
func sanitize(s string) string {
	if len(s) == 0 {
		return s
	}

	// Note: No length limit for label keys because Prometheus doesn't
	// define a length limit, thus we should NOT be truncating label keys.
	// See https://github.com/orijtech/prometheus-go-metrics-exporter/issues/4.
	s = strings.Map(sanitizeRune, s)
	if unicode.IsDigit(rune(s[0])) {
		s = "key_" + s
	}
	if s[0] == '_' {
		s = "key" + s
	}
	return s
}

// copied from prometheus-go-metric-exporter
// sanitizeRune converts anything that is not a letter or digit to an underscore
func sanitizeRune(r rune) rune {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return r
	}
	// Everything else turns into an underscore
	return '_'
}

// setLatencyExemplars sets the histogram exemplars.
func setLatencyExemplars(exemplarsData []exemplarData, timestamp pdata.Timestamp, exemplars pdata.ExemplarSlice) {
	es := pdata.NewExemplarSlice()
	es.EnsureCapacity(len(exemplarsData))

	for _, ed := range exemplarsData {
		value := ed.value
		traceID := ed.traceID

		exemplar := es.AppendEmpty()

		if traceID.IsEmpty() {
			continue
		}

		exemplar.SetDoubleVal(value)
		exemplar.SetTimestamp(timestamp)
		exemplar.FilteredAttributes().Insert(traceIDKey, pdata.NewAttributeValueString(traceID.HexString()))
	}

	es.CopyTo(exemplars)
}
