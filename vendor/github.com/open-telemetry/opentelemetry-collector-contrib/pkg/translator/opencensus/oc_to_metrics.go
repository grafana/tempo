// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opencensus // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/opencensus"

import (
	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// OCToMetrics converts OC data format to pmetric.Metrics,
// may be used only by OpenCensus receiver and exporter implementations.
func OCToMetrics(node *occommon.Node, resource *ocresource.Resource, metrics []*ocmetrics.Metric) pmetric.Metrics {
	dest := pmetric.NewMetrics()
	if node == nil && resource == nil && len(metrics) == 0 {
		return dest
	}

	rms := dest.ResourceMetrics()
	initialRmsLen := rms.Len()

	if len(metrics) == 0 {
		// At least one of the md.Node or md.Resource is not nil. Set the resource and return.
		ocNodeResourceToInternal(node, resource, rms.AppendEmpty().Resource())
		return dest
	}

	// We may need to split OC metrics into several ResourceMetrics. OC metrics can have a
	// Resource field inside them set to nil which indicates they use the Resource
	// specified in "md.Resource", or they can have the Resource field inside them set
	// to non-nil which indicates they have overridden Resource field and "md.Resource"
	// does not apply to those metrics.
	//
	// Each OC metric that has its own Resource field set to non-nil must be placed in a
	// separate ResourceMetrics instance, containing only that metric. All other OC Metrics
	// that have nil Resource field must be placed in one other ResourceMetrics instance,
	// which will gets its Resource field from "md.Resource".
	//
	// We will end up with with one or more ResourceMetrics like this:
	//
	// ResourceMetrics           ResourceMetrics  ResourceMetrics
	// +-------+-------+---+-------+ +--------------+ +--------------+
	// |Metric1|Metric2|...|MetricM| |Metric        | |Metric        | ...
	// +-------+-------+---+-------+ +--------------+ +--------------+

	// Count the number of metrics that have nil Resource and need to be combined
	// in one slice.
	combinedMetricCount := 0
	distinctResourceCount := 0
	for _, ocMetric := range metrics {
		if ocMetric == nil {
			// Skip nil metrics.
			continue
		}
		if ocMetric.Resource == nil {
			combinedMetricCount++
		} else {
			distinctResourceCount++
		}
	}
	// Total number of resources is equal to:
	// initial + numMetricsWithResource + (optional) 1
	resourceCount := initialRmsLen + distinctResourceCount
	if combinedMetricCount > 0 {
		// +1 for all metrics with nil resource
		resourceCount++
	}
	rms.EnsureCapacity(resourceCount)

	// Translate "combinedMetrics" first

	if combinedMetricCount > 0 {
		rm0 := rms.AppendEmpty()
		ocNodeResourceToInternal(node, resource, rm0.Resource())

		// Allocate a slice for metrics that need to be combined into first ResourceMetrics.
		ilms := rm0.ScopeMetrics()
		combinedMetrics := ilms.AppendEmpty().Metrics()
		combinedMetrics.EnsureCapacity(combinedMetricCount)

		for _, ocMetric := range metrics {
			if ocMetric == nil {
				// Skip nil metrics.
				continue
			}

			if ocMetric.Resource != nil {
				continue // Those are processed separately below.
			}

			// Add the metric to the "combinedMetrics". combinedMetrics length is equal
			// to combinedMetricCount. The loop above that calculates combinedMetricCount
			// has exact same conditions as we have here in this loop.
			ocMetricToMetrics(ocMetric, combinedMetrics.AppendEmpty())
		}
	}

	// Translate distinct metrics

	for _, ocMetric := range metrics {
		if ocMetric == nil {
			// Skip nil metrics.
			continue
		}

		if ocMetric.Resource == nil {
			continue // Already processed above.
		}

		// This metric has a different Resource and must be placed in a different
		// ResourceMetrics instance. Create a separate ResourceMetrics item just for this metric
		// and store at resourceMetricIdx.
		ocMetricToResourceMetrics(ocMetric, node, rms.AppendEmpty())
	}
	return dest
}

func ocMetricToResourceMetrics(ocMetric *ocmetrics.Metric, node *occommon.Node, out pmetric.ResourceMetrics) {
	ocNodeResourceToInternal(node, ocMetric.Resource, out.Resource())
	ilms := out.ScopeMetrics()
	ocMetricToMetrics(ocMetric, ilms.AppendEmpty().Metrics().AppendEmpty())
}

func ocMetricToMetrics(ocMetric *ocmetrics.Metric, metric pmetric.Metric) {
	ocDescriptor := ocMetric.GetMetricDescriptor()
	if ocDescriptor == nil {
		pmetric.NewMetric().CopyTo(metric)
		return
	}

	dataType, valType := descriptorTypeToMetrics(ocDescriptor.Type, metric)
	if dataType == pmetric.MetricTypeEmpty {
		pmetric.NewMetric().CopyTo(metric)
		return
	}

	metric.SetDescription(ocDescriptor.GetDescription())
	metric.SetName(ocDescriptor.GetName())
	metric.SetUnit(ocDescriptor.GetUnit())

	setDataPoints(ocMetric, metric, valType)
}

func descriptorTypeToMetrics(t ocmetrics.MetricDescriptor_Type, metric pmetric.Metric) (pmetric.MetricType, pmetric.NumberDataPointValueType) {
	switch t {
	case ocmetrics.MetricDescriptor_GAUGE_INT64:
		metric.SetEmptyGauge()
		return pmetric.MetricTypeGauge, pmetric.NumberDataPointValueTypeInt
	case ocmetrics.MetricDescriptor_GAUGE_DOUBLE:
		metric.SetEmptyGauge()
		return pmetric.MetricTypeGauge, pmetric.NumberDataPointValueTypeDouble
	case ocmetrics.MetricDescriptor_CUMULATIVE_INT64:
		sum := metric.SetEmptySum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		return pmetric.MetricTypeSum, pmetric.NumberDataPointValueTypeInt
	case ocmetrics.MetricDescriptor_CUMULATIVE_DOUBLE:
		sum := metric.SetEmptySum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		return pmetric.MetricTypeSum, pmetric.NumberDataPointValueTypeDouble
	case ocmetrics.MetricDescriptor_CUMULATIVE_DISTRIBUTION:
		histo := metric.SetEmptyHistogram()
		histo.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		return pmetric.MetricTypeHistogram, pmetric.NumberDataPointValueTypeEmpty
	case ocmetrics.MetricDescriptor_SUMMARY:
		metric.SetEmptySummary()
		// no temporality specified for summary metric
		return pmetric.MetricTypeSummary, pmetric.NumberDataPointValueTypeEmpty
	}
	return pmetric.MetricTypeEmpty, pmetric.NumberDataPointValueTypeEmpty
}

// setDataPoints converts OC timeseries to internal datapoints based on metric type
func setDataPoints(ocMetric *ocmetrics.Metric, metric pmetric.Metric, valType pmetric.NumberDataPointValueType) {
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		fillNumberDataPoint(ocMetric, metric.Gauge().DataPoints(), valType)
	case pmetric.MetricTypeSum:
		fillNumberDataPoint(ocMetric, metric.Sum().DataPoints(), valType)
	case pmetric.MetricTypeHistogram:
		fillDoubleHistogramDataPoint(ocMetric, metric.Histogram().DataPoints())
	case pmetric.MetricTypeSummary:
		fillDoubleSummaryDataPoint(ocMetric, metric.Summary().DataPoints())
	}
}

func fillAttributesMap(ocLabelsKeys []*ocmetrics.LabelKey, ocLabelValues []*ocmetrics.LabelValue, attributesMap pcommon.Map) {
	if len(ocLabelsKeys) == 0 || len(ocLabelValues) == 0 {
		return
	}

	lablesCount := len(ocLabelsKeys)

	// Handle invalid length of OC label values list
	if len(ocLabelValues) < lablesCount {
		lablesCount = len(ocLabelValues)
	}

	attributesMap.EnsureCapacity(lablesCount)
	for i := 0; i < lablesCount; i++ {
		if !ocLabelValues[i].GetHasValue() {
			continue
		}
		attributesMap.PutStr(ocLabelsKeys[i].Key, ocLabelValues[i].Value)
	}
}

func fillNumberDataPoint(ocMetric *ocmetrics.Metric, dps pmetric.NumberDataPointSlice, valType pmetric.NumberDataPointValueType) {
	ocPointsCount := getPointsCount(ocMetric)
	dps.EnsureCapacity(ocPointsCount)
	ocLabelsKeys := ocMetric.GetMetricDescriptor().GetLabelKeys()
	for _, timeseries := range ocMetric.GetTimeseries() {
		if timeseries == nil {
			continue
		}
		startTimestamp := pcommon.NewTimestampFromTime(timeseries.GetStartTimestamp().AsTime())

		for _, point := range timeseries.GetPoints() {
			if point == nil {
				continue
			}

			dp := dps.AppendEmpty()
			dp.SetStartTimestamp(startTimestamp)
			dp.SetTimestamp(pcommon.NewTimestampFromTime(point.GetTimestamp().AsTime()))
			fillAttributesMap(ocLabelsKeys, timeseries.LabelValues, dp.Attributes())
			switch valType {
			case pmetric.NumberDataPointValueTypeInt:
				dp.SetIntValue(point.GetInt64Value())
			case pmetric.NumberDataPointValueTypeDouble:
				dp.SetDoubleValue(point.GetDoubleValue())
			}
		}
	}
}

func fillDoubleHistogramDataPoint(ocMetric *ocmetrics.Metric, dps pmetric.HistogramDataPointSlice) {
	ocPointsCount := getPointsCount(ocMetric)
	dps.EnsureCapacity(ocPointsCount)
	ocLabelsKeys := ocMetric.GetMetricDescriptor().GetLabelKeys()
	for _, timeseries := range ocMetric.GetTimeseries() {
		if timeseries == nil {
			continue
		}
		startTimestamp := pcommon.NewTimestampFromTime(timeseries.GetStartTimestamp().AsTime())

		for _, point := range timeseries.GetPoints() {
			if point == nil {
				continue
			}

			dp := dps.AppendEmpty()
			dp.SetStartTimestamp(startTimestamp)
			dp.SetTimestamp(pcommon.NewTimestampFromTime(point.GetTimestamp().AsTime()))
			fillAttributesMap(ocLabelsKeys, timeseries.LabelValues, dp.Attributes())
			distributionValue := point.GetDistributionValue()
			dp.SetSum(distributionValue.GetSum())
			dp.SetCount(uint64(distributionValue.GetCount()))
			ocHistogramBucketsToMetrics(distributionValue.GetBuckets(), dp)
			dp.ExplicitBounds().FromRaw(distributionValue.GetBucketOptions().GetExplicit().GetBounds())
		}
	}
}

func fillDoubleSummaryDataPoint(ocMetric *ocmetrics.Metric, dps pmetric.SummaryDataPointSlice) {
	ocPointsCount := getPointsCount(ocMetric)
	dps.EnsureCapacity(ocPointsCount)
	ocLabelsKeys := ocMetric.GetMetricDescriptor().GetLabelKeys()
	for _, timeseries := range ocMetric.GetTimeseries() {
		if timeseries == nil {
			continue
		}
		startTimestamp := pcommon.NewTimestampFromTime(timeseries.GetStartTimestamp().AsTime())

		for _, point := range timeseries.GetPoints() {
			if point == nil {
				continue
			}

			dp := dps.AppendEmpty()
			dp.SetStartTimestamp(startTimestamp)
			dp.SetTimestamp(pcommon.NewTimestampFromTime(point.GetTimestamp().AsTime()))
			fillAttributesMap(ocLabelsKeys, timeseries.LabelValues, dp.Attributes())
			summaryValue := point.GetSummaryValue()
			dp.SetSum(summaryValue.GetSum().GetValue())
			dp.SetCount(uint64(summaryValue.GetCount().GetValue()))
			ocSummaryPercentilesToMetrics(summaryValue.GetSnapshot().GetPercentileValues(), dp)
		}
	}
}

func ocHistogramBucketsToMetrics(ocBuckets []*ocmetrics.DistributionValue_Bucket, dp pmetric.HistogramDataPoint) {
	if len(ocBuckets) == 0 {
		return
	}
	buckets := make([]uint64, len(ocBuckets))
	for i := range buckets {
		buckets[i] = uint64(ocBuckets[i].GetCount())
		if ocBuckets[i].GetExemplar() != nil {
			exemplar := dp.Exemplars().AppendEmpty()
			exemplarToMetrics(ocBuckets[i].GetExemplar(), exemplar)
		}
	}
	dp.BucketCounts().FromRaw(buckets)
}

func ocSummaryPercentilesToMetrics(ocPercentiles []*ocmetrics.SummaryValue_Snapshot_ValueAtPercentile, dp pmetric.SummaryDataPoint) {
	if len(ocPercentiles) == 0 {
		return
	}

	quantiles := pmetric.NewSummaryDataPointValueAtQuantileSlice()
	quantiles.EnsureCapacity(len(ocPercentiles))

	for _, percentile := range ocPercentiles {
		quantile := quantiles.AppendEmpty()
		quantile.SetQuantile(percentile.GetPercentile() / 100)
		quantile.SetValue(percentile.GetValue())
	}

	quantiles.CopyTo(dp.QuantileValues())
}

func exemplarToMetrics(ocExemplar *ocmetrics.DistributionValue_Exemplar, exemplar pmetric.Exemplar) {
	if ocExemplar.GetTimestamp() != nil {
		exemplar.SetTimestamp(pcommon.NewTimestampFromTime(ocExemplar.GetTimestamp().AsTime()))
	}
	ocAttachments := ocExemplar.GetAttachments()
	exemplar.SetDoubleValue(ocExemplar.GetValue())
	filteredAttributes := exemplar.FilteredAttributes()
	filteredAttributes.EnsureCapacity(len(ocAttachments))
	for k, v := range ocAttachments {
		filteredAttributes.PutStr(k, v)
	}
}

func getPointsCount(ocMetric *ocmetrics.Metric) int {
	timeseriesSlice := ocMetric.GetTimeseries()
	var count int
	for _, timeseries := range timeseriesSlice {
		count += len(timeseries.GetPoints())
	}
	return count
}
