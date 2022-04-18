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

package opencensus // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/opencensus"

import (
	"sort"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/golang/protobuf/ptypes/wrappers"
	"go.opentelemetry.io/collector/model/pdata"
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

// ResourceMetricsToOC may be used only by OpenCensus receiver and exporter implementations.
// Deprecated: Use pdata.Metrics.
// TODO: move this function to OpenCensus package.
func ResourceMetricsToOC(rm pdata.ResourceMetrics) (*occommon.Node, *ocresource.Resource, []*ocmetrics.Metric) {
	node, resource := internalResourceToOC(rm.Resource())
	ilms := rm.InstrumentationLibraryMetrics()
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

func metricToOC(metric pdata.Metric) *ocmetrics.Metric {
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

func collectLabelKeysAndValueType(metric pdata.Metric) *labelKeysAndType {
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
	switch metric.DataType() {
	case pdata.MetricDataTypeGauge:
		allNumberDataPointValueInt = collectLabelKeysNumberDataPoints(metric.Gauge().DataPoints(), keySet)
	case pdata.MetricDataTypeSum:
		allNumberDataPointValueInt = collectLabelKeysNumberDataPoints(metric.Sum().DataPoints(), keySet)
	case pdata.MetricDataTypeHistogram:
		collectLabelKeysHistogramDataPoints(metric.Histogram().DataPoints(), keySet)
	case pdata.MetricDataTypeSummary:
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
func collectLabelKeysNumberDataPoints(dps pdata.NumberDataPointSlice, keySet map[string]struct{}) bool {
	allInt := true
	for i := 0; i < dps.Len(); i++ {
		addLabelKeys(keySet, dps.At(i).Attributes())
		if dps.At(i).ValueType() != pdata.MetricValueTypeInt {
			allInt = false
		}
	}
	return allInt
}

func collectLabelKeysHistogramDataPoints(dhdp pdata.HistogramDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < dhdp.Len(); i++ {
		addLabelKeys(keySet, dhdp.At(i).Attributes())
	}
}

func collectLabelKeysSummaryDataPoints(dhdp pdata.SummaryDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < dhdp.Len(); i++ {
		addLabelKeys(keySet, dhdp.At(i).Attributes())
	}
}

func addLabelKeys(keySet map[string]struct{}, attributes pdata.AttributeMap) {
	attributes.Range(func(k string, v pdata.AttributeValue) bool {
		keySet[k] = struct{}{}
		return true
	})
}

func descriptorTypeToOC(metric pdata.Metric, allNumberDataPointValueInt bool) ocmetrics.MetricDescriptor_Type {
	switch metric.DataType() {
	case pdata.MetricDataTypeGauge:
		return gaugeType(allNumberDataPointValueInt)
	case pdata.MetricDataTypeSum:
		sd := metric.Sum()
		if sd.IsMonotonic() && sd.AggregationTemporality() == pdata.MetricAggregationTemporalityCumulative {
			return cumulativeType(allNumberDataPointValueInt)
		}
		return gaugeType(allNumberDataPointValueInt)
	case pdata.MetricDataTypeHistogram:
		hd := metric.Histogram()
		if hd.AggregationTemporality() == pdata.MetricAggregationTemporalityCumulative {
			return ocmetrics.MetricDescriptor_CUMULATIVE_DISTRIBUTION
		}
		return ocmetrics.MetricDescriptor_GAUGE_DISTRIBUTION
	case pdata.MetricDataTypeSummary:
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

func dataPointsToTimeseries(metric pdata.Metric, labelKeys *labelKeysAndType) []*ocmetrics.TimeSeries {
	switch metric.DataType() {
	case pdata.MetricDataTypeGauge:
		return numberDataPointsToOC(metric.Gauge().DataPoints(), labelKeys)
	case pdata.MetricDataTypeSum:
		return numberDataPointsToOC(metric.Sum().DataPoints(), labelKeys)
	case pdata.MetricDataTypeHistogram:
		return doubleHistogramPointToOC(metric.Histogram().DataPoints(), labelKeys)
	case pdata.MetricDataTypeSummary:
		return doubleSummaryPointToOC(metric.Summary().DataPoints(), labelKeys)
	}

	return nil
}

func numberDataPointsToOC(dps pdata.NumberDataPointSlice, labelKeys *labelKeysAndType) []*ocmetrics.TimeSeries {
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
		case pdata.MetricValueTypeInt:
			point.Value = &ocmetrics.Point_Int64Value{
				Int64Value: dp.IntVal(),
			}
		case pdata.MetricValueTypeDouble:
			point.Value = &ocmetrics.Point_DoubleValue{
				DoubleValue: dp.DoubleVal(),
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

func doubleHistogramPointToOC(dps pdata.HistogramDataPointSlice, labelKeys *labelKeysAndType) []*ocmetrics.TimeSeries {
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

func histogramExplicitBoundsToOC(bounds []float64) *ocmetrics.DistributionValue_BucketOptions {
	if len(bounds) == 0 {
		return nil
	}

	return &ocmetrics.DistributionValue_BucketOptions{
		Type: &ocmetrics.DistributionValue_BucketOptions_Explicit_{
			Explicit: &ocmetrics.DistributionValue_BucketOptions_Explicit{
				Bounds: bounds,
			},
		},
	}
}

func histogramBucketsToOC(bcts []uint64) []*ocmetrics.DistributionValue_Bucket {
	if len(bcts) == 0 {
		return nil
	}

	ocBuckets := make([]*ocmetrics.DistributionValue_Bucket, 0, len(bcts))
	for _, bucket := range bcts {
		ocBuckets = append(ocBuckets, &ocmetrics.DistributionValue_Bucket{
			Count: int64(bucket),
		})
	}
	return ocBuckets
}

func doubleSummaryPointToOC(dps pdata.SummaryDataPointSlice, labelKeys *labelKeysAndType) []*ocmetrics.TimeSeries {
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

func summaryPercentilesToOC(qtls pdata.ValueAtQuantileSlice) []*ocmetrics.SummaryValue_Snapshot_ValueAtPercentile {
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

func exemplarsToOC(bounds []float64, ocBuckets []*ocmetrics.DistributionValue_Bucket, exemplars pdata.ExemplarSlice) {
	if exemplars.Len() == 0 {
		return
	}

	for i := 0; i < exemplars.Len(); i++ {
		exemplar := exemplars.At(i)
		var val float64
		switch exemplar.ValueType() {
		case pdata.MetricValueTypeInt:
			val = float64(exemplar.IntVal())
		case pdata.MetricValueTypeDouble:
			val = exemplar.DoubleVal()
		}
		pos := 0
		for ; pos < len(bounds); pos++ {
			if val > bounds[pos] {
				continue
			}
			break
		}
		ocBuckets[pos].Exemplar = exemplarToOC(exemplar.FilteredAttributes(), val, exemplar.Timestamp())
	}
}

func exemplarToOC(filteredLabels pdata.AttributeMap, value float64, timestamp pdata.Timestamp) *ocmetrics.DistributionValue_Exemplar {
	var labels map[string]string
	if filteredLabels.Len() != 0 {
		labels = make(map[string]string, filteredLabels.Len())
		filteredLabels.Range(func(k string, v pdata.AttributeValue) bool {
			labels[k] = v.AsString()
			return true
		})
	}

	return &ocmetrics.DistributionValue_Exemplar{
		Value:       value,
		Timestamp:   timestampAsTimestampPb(timestamp),
		Attachments: labels,
	}
}

func attributeValuesToOC(labels pdata.AttributeMap, labelKeys *labelKeysAndType) []*ocmetrics.LabelValue {
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
	labels.Range(func(k string, v pdata.AttributeValue) bool {
		// Find the appropriate label value that we need to update
		keyIndex := labelKeys.keyIndices[k]
		labelValue := labelValues[keyIndex]

		// Update label value
		labelValue.Value = v.AsString()
		labelValue.HasValue = true
		return true
	})

	return labelValues
}
