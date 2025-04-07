// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opencensus // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/opencensus"

import (
	"sort"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/golang/protobuf/ptypes/wrappers"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type labelKeysAndType struct {
	// ordered OC label keys
	keys []*ocmetrics.LabelKey
	// map from a label key literal
	// to its index in the slice above
	keyIndices map[string]int
	// true if the metric is a scalar (number data points) and all int vales:
	allNumberDataPointValueInt bool
}

// ResourceMetricsToOC converts pmetric.ResourceMetrics to OC data format,
// may be used only by OpenCensus receiver and exporter implementations.
func ResourceMetricsToOC(rm pmetric.ResourceMetrics) (*occommon.Node, *ocresource.Resource, []*ocmetrics.Metric) {
	node, resource := internalResourceToOC(rm.Resource())
	ilms := rm.ScopeMetrics()
	if ilms.Len() == 0 {
		return node, resource, nil
	}
	// Approximate the number of the metrics as the number of the metrics in the first
	// instrumentation library info.
	ocMetrics := make([]*ocmetrics.Metric, 0, ilms.At(0).Metrics().Len())
	for i := 0; i < ilms.Len(); i++ {
		ilm := ilms.At(i)
		// TODO: Handle instrumentation library name and version.
		metrics := ilm.Metrics()
		for j := 0; j < metrics.Len(); j++ {
			ocMetrics = append(ocMetrics, metricToOC(metrics.At(j)))
		}
	}
	if len(ocMetrics) == 0 {
		ocMetrics = nil
	}
	return node, resource, ocMetrics
}

func metricToOC(metric pmetric.Metric) *ocmetrics.Metric {
	lblKeys := collectLabelKeysAndValueType(metric)
	return &ocmetrics.Metric{
		MetricDescriptor: &ocmetrics.MetricDescriptor{
			Name:        metric.Name(),
			Description: metric.Description(),
			Unit:        metric.Unit(),
			Type:        descriptorTypeToOC(metric, lblKeys.allNumberDataPointValueInt),
			LabelKeys:   lblKeys.keys,
		},
		Timeseries: dataPointsToTimeseries(metric, lblKeys),
		Resource:   nil,
	}
}

func collectLabelKeysAndValueType(metric pmetric.Metric) *labelKeysAndType {
	// NOTE: Internal data structure and OpenCensus have different representations of labels:
	// - OC has a single "global" ordered list of label keys per metric in the MetricDescriptor;
	// then, every data point has an ordered list of label values matching the key index.
	// - Internally labels are stored independently as key-value storage for each point.
	//
	// So what we do in this translator:
	// - Scan all points and their labels to find all label keys used across the metric,
	// sort them and set in the MetricDescriptor.
	// - For each point we generate an ordered list of label values,
	// matching the order of label keys returned here (see `labelValuesToOC` function).
	// - If the value for particular label key is missing in the point, we set it to default
	// to preserve 1:1 matching between label keys and values.

	// First, collect a set of all labels present in the metric
	keySet := make(map[string]struct{})
	allNumberDataPointValueInt := false
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		allNumberDataPointValueInt = collectLabelKeysNumberDataPoints(metric.Gauge().DataPoints(), keySet)
	case pmetric.MetricTypeSum:
		allNumberDataPointValueInt = collectLabelKeysNumberDataPoints(metric.Sum().DataPoints(), keySet)
	case pmetric.MetricTypeHistogram:
		collectLabelKeysHistogramDataPoints(metric.Histogram().DataPoints(), keySet)
	case pmetric.MetricTypeSummary:
		collectLabelKeysSummaryDataPoints(metric.Summary().DataPoints(), keySet)
	}

	lkt := &labelKeysAndType{
		allNumberDataPointValueInt: allNumberDataPointValueInt,
	}
	if len(keySet) == 0 {
		return lkt
	}

	// Sort keys: while not mandatory, this helps to make the
	// output OC metric deterministic and easy to test, i.e.
	// the same set of labels will always produce
	// OC labels in the alphabetically sorted order.
	sortedKeys := make([]string, 0, len(keySet))
	for key := range keySet {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	// Construct a resulting list of label keys
	lkt.keys = make([]*ocmetrics.LabelKey, 0, len(sortedKeys))
	// Label values will have to match keys by index
	// so this map will help with fast lookups.
	lkt.keyIndices = make(map[string]int, len(sortedKeys))
	for i, key := range sortedKeys {
		lkt.keys = append(lkt.keys, &ocmetrics.LabelKey{
			Key: key,
		})
		lkt.keyIndices[key] = i
	}

	return lkt
}

// collectLabelKeysNumberDataPoints returns true if all values are int.
func collectLabelKeysNumberDataPoints(dps pmetric.NumberDataPointSlice, keySet map[string]struct{}) bool {
	allInt := true
	for i := 0; i < dps.Len(); i++ {
		addLabelKeys(keySet, dps.At(i).Attributes())
		if dps.At(i).ValueType() != pmetric.NumberDataPointValueTypeInt {
			allInt = false
		}
	}
	return allInt
}

func collectLabelKeysHistogramDataPoints(dhdp pmetric.HistogramDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < dhdp.Len(); i++ {
		addLabelKeys(keySet, dhdp.At(i).Attributes())
	}
}

func collectLabelKeysSummaryDataPoints(dhdp pmetric.SummaryDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < dhdp.Len(); i++ {
		addLabelKeys(keySet, dhdp.At(i).Attributes())
	}
}

func addLabelKeys(keySet map[string]struct{}, attributes pcommon.Map) {
	for k := range attributes.All() {
		keySet[k] = struct{}{}
	}
}

func descriptorTypeToOC(metric pmetric.Metric, allNumberDataPointValueInt bool) ocmetrics.MetricDescriptor_Type {
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		return gaugeType(allNumberDataPointValueInt)
	case pmetric.MetricTypeSum:
		sd := metric.Sum()
		if sd.IsMonotonic() && sd.AggregationTemporality() == pmetric.AggregationTemporalityCumulative {
			return cumulativeType(allNumberDataPointValueInt)
		}
		return gaugeType(allNumberDataPointValueInt)
	case pmetric.MetricTypeHistogram:
		hd := metric.Histogram()
		if hd.AggregationTemporality() == pmetric.AggregationTemporalityCumulative {
			return ocmetrics.MetricDescriptor_CUMULATIVE_DISTRIBUTION
		}
		return ocmetrics.MetricDescriptor_GAUGE_DISTRIBUTION
	case pmetric.MetricTypeSummary:
		return ocmetrics.MetricDescriptor_SUMMARY
	}
	return ocmetrics.MetricDescriptor_UNSPECIFIED
}

func gaugeType(allNumberDataPointValueInt bool) ocmetrics.MetricDescriptor_Type {
	if allNumberDataPointValueInt {
		return ocmetrics.MetricDescriptor_GAUGE_INT64
	}
	return ocmetrics.MetricDescriptor_GAUGE_DOUBLE
}

func cumulativeType(allNumberDataPointValueInt bool) ocmetrics.MetricDescriptor_Type {
	if allNumberDataPointValueInt {
		return ocmetrics.MetricDescriptor_CUMULATIVE_INT64
	}
	return ocmetrics.MetricDescriptor_CUMULATIVE_DOUBLE
}

func dataPointsToTimeseries(metric pmetric.Metric, labelKeys *labelKeysAndType) []*ocmetrics.TimeSeries {
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		return numberDataPointsToOC(metric.Gauge().DataPoints(), labelKeys)
	case pmetric.MetricTypeSum:
		return numberDataPointsToOC(metric.Sum().DataPoints(), labelKeys)
	case pmetric.MetricTypeHistogram:
		return doubleHistogramPointToOC(metric.Histogram().DataPoints(), labelKeys)
	case pmetric.MetricTypeSummary:
		return doubleSummaryPointToOC(metric.Summary().DataPoints(), labelKeys)
	}

	return nil
}

func numberDataPointsToOC(dps pmetric.NumberDataPointSlice, labelKeys *labelKeysAndType) []*ocmetrics.TimeSeries {
	if dps.Len() == 0 {
		return nil
	}
	timeseries := make([]*ocmetrics.TimeSeries, 0, dps.Len())
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		point := &ocmetrics.Point{
			Timestamp: timestampAsTimestampPb(dp.Timestamp()),
		}
		switch dp.ValueType() {
		case pmetric.NumberDataPointValueTypeInt:
			point.Value = &ocmetrics.Point_Int64Value{
				Int64Value: dp.IntValue(),
			}
		case pmetric.NumberDataPointValueTypeDouble:
			point.Value = &ocmetrics.Point_DoubleValue{
				DoubleValue: dp.DoubleValue(),
			}
		}
		ts := &ocmetrics.TimeSeries{
			StartTimestamp: timestampAsTimestampPb(dp.StartTimestamp()),
			LabelValues:    attributeValuesToOC(dp.Attributes(), labelKeys),
			Points:         []*ocmetrics.Point{point},
		}
		timeseries = append(timeseries, ts)
	}
	return timeseries
}

func doubleHistogramPointToOC(dps pmetric.HistogramDataPointSlice, labelKeys *labelKeysAndType) []*ocmetrics.TimeSeries {
	if dps.Len() == 0 {
		return nil
	}
	timeseries := make([]*ocmetrics.TimeSeries, 0, dps.Len())
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		buckets := histogramBucketsToOC(dp.BucketCounts())
		exemplarsToOC(dp.ExplicitBounds(), buckets, dp.Exemplars())

		ts := &ocmetrics.TimeSeries{
			StartTimestamp: timestampAsTimestampPb(dp.StartTimestamp()),
			LabelValues:    attributeValuesToOC(dp.Attributes(), labelKeys),
			Points: []*ocmetrics.Point{
				{
					Timestamp: timestampAsTimestampPb(dp.Timestamp()),
					Value: &ocmetrics.Point_DistributionValue{
						DistributionValue: &ocmetrics.DistributionValue{
							Count:                 int64(dp.Count()),
							Sum:                   dp.Sum(),
							SumOfSquaredDeviation: 0,
							BucketOptions:         histogramExplicitBoundsToOC(dp.ExplicitBounds()),
							Buckets:               buckets,
						},
					},
				},
			},
		}
		timeseries = append(timeseries, ts)
	}
	return timeseries
}

func histogramExplicitBoundsToOC(bounds pcommon.Float64Slice) *ocmetrics.DistributionValue_BucketOptions {
	if bounds.Len() == 0 {
		return nil
	}

	return &ocmetrics.DistributionValue_BucketOptions{
		Type: &ocmetrics.DistributionValue_BucketOptions_Explicit_{
			Explicit: &ocmetrics.DistributionValue_BucketOptions_Explicit{
				Bounds: bounds.AsRaw(),
			},
		},
	}
}

func histogramBucketsToOC(bcts pcommon.UInt64Slice) []*ocmetrics.DistributionValue_Bucket {
	if bcts.Len() == 0 {
		return nil
	}

	ocBuckets := make([]*ocmetrics.DistributionValue_Bucket, 0, bcts.Len())
	for i := 0; i < bcts.Len(); i++ {
		ocBuckets = append(ocBuckets, &ocmetrics.DistributionValue_Bucket{
			Count: int64(bcts.At(i)),
		})
	}
	return ocBuckets
}

func doubleSummaryPointToOC(dps pmetric.SummaryDataPointSlice, labelKeys *labelKeysAndType) []*ocmetrics.TimeSeries {
	if dps.Len() == 0 {
		return nil
	}
	timeseries := make([]*ocmetrics.TimeSeries, 0, dps.Len())
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		percentileValues := summaryPercentilesToOC(dp.QuantileValues())

		ts := &ocmetrics.TimeSeries{
			StartTimestamp: timestampAsTimestampPb(dp.StartTimestamp()),
			LabelValues:    attributeValuesToOC(dp.Attributes(), labelKeys),
			Points: []*ocmetrics.Point{
				{
					Timestamp: timestampAsTimestampPb(dp.Timestamp()),
					Value: &ocmetrics.Point_SummaryValue{
						SummaryValue: &ocmetrics.SummaryValue{
							Sum:   &wrappers.DoubleValue{Value: dp.Sum()},
							Count: &wrappers.Int64Value{Value: int64(dp.Count())},
							Snapshot: &ocmetrics.SummaryValue_Snapshot{
								PercentileValues: percentileValues,
							},
						},
					},
				},
			},
		}
		timeseries = append(timeseries, ts)
	}
	return timeseries
}

func summaryPercentilesToOC(qtls pmetric.SummaryDataPointValueAtQuantileSlice) []*ocmetrics.SummaryValue_Snapshot_ValueAtPercentile {
	if qtls.Len() == 0 {
		return nil
	}

	ocPercentiles := make([]*ocmetrics.SummaryValue_Snapshot_ValueAtPercentile, 0, qtls.Len())
	for i := 0; i < qtls.Len(); i++ {
		quantile := qtls.At(i)
		ocPercentiles = append(ocPercentiles, &ocmetrics.SummaryValue_Snapshot_ValueAtPercentile{
			Percentile: quantile.Quantile() * 100,
			Value:      quantile.Value(),
		})
	}
	return ocPercentiles
}

func exemplarsToOC(bounds pcommon.Float64Slice, ocBuckets []*ocmetrics.DistributionValue_Bucket, exemplars pmetric.ExemplarSlice) {
	if exemplars.Len() == 0 {
		return
	}

	for i := 0; i < exemplars.Len(); i++ {
		exemplar := exemplars.At(i)
		var val float64
		switch exemplar.ValueType() {
		case pmetric.ExemplarValueTypeInt:
			val = float64(exemplar.IntValue())
		case pmetric.ExemplarValueTypeDouble:
			val = exemplar.DoubleValue()
		}
		pos := 0
		for ; pos < bounds.Len(); pos++ {
			if val > bounds.At(pos) {
				continue
			}
			break
		}
		ocBuckets[pos].Exemplar = exemplarToOC(exemplar.FilteredAttributes(), val, exemplar.Timestamp())
	}
}

func exemplarToOC(filteredLabels pcommon.Map, value float64, timestamp pcommon.Timestamp) *ocmetrics.DistributionValue_Exemplar {
	var labels map[string]string
	if filteredLabels.Len() != 0 {
		labels = make(map[string]string, filteredLabels.Len())
		for k, v := range filteredLabels.All() {
			labels[k] = v.AsString()
		}
	}

	return &ocmetrics.DistributionValue_Exemplar{
		Value:       value,
		Timestamp:   timestampAsTimestampPb(timestamp),
		Attachments: labels,
	}
}

func attributeValuesToOC(labels pcommon.Map, labelKeys *labelKeysAndType) []*ocmetrics.LabelValue {
	if len(labelKeys.keys) == 0 {
		return nil
	}

	// Initialize label values with defaults
	// (The order matches key indices)
	labelValuesOrig := make([]ocmetrics.LabelValue, len(labelKeys.keys))
	labelValues := make([]*ocmetrics.LabelValue, len(labelKeys.keys))
	for i := 0; i < len(labelKeys.keys); i++ {
		labelValues[i] = &labelValuesOrig[i]
	}

	// Visit all defined labels in the point and override defaults with actual values
	for k, v := range labels.All() {
		// Find the appropriate label value that we need to update
		keyIndex := labelKeys.keyIndices[k]
		labelValue := labelValues[keyIndex]

		// Update label value
		labelValue.Value = v.AsString()
		labelValue.HasValue = true
	}

	return labelValues
}
