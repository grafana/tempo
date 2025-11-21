package validation

import (
	"errors"
	"fmt"
	"time"

	"github.com/grafana/tempo/modules/distributor/usage"
	"github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/sharedconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/util/strutil"
)

var SupportedProcessors = []string{
	processor.ServiceGraphsName,
	processor.SpanMetricsName,
	processor.LocalBlocksName,
	processor.SpanMetricsCountName,
	processor.SpanMetricsLatencyName,
	processor.SpanMetricsSizeName,
	processor.HostInfoName,
}

var SupportedIntrinsicDimensions = []string{processor.DimService, processor.DimSpanName, processor.DimSpanKind, processor.DimStatusCode, processor.DimStatusMessage}

var SupportedIntrinsicDimensionsSet map[string]struct{}

var SupportedProcessorsSet map[string]struct{}

var SupportedHistogramModesSet map[string]struct{}

func init() {
	SupportedIntrinsicDimensionsSet = make(map[string]struct{})
	for _, dim := range SupportedIntrinsicDimensions {
		SupportedIntrinsicDimensionsSet[dim] = struct{}{}
	}
	SupportedProcessorsSet = make(map[string]struct{})
	for _, p := range SupportedProcessors {
		SupportedProcessorsSet[p] = struct{}{}
	}
	SupportedHistogramModesSet = make(map[string]struct{})
	for mode := range registry.HistogramModeToValue {
		SupportedHistogramModesSet[mode] = struct{}{}
	}
}

func ValidateProcessor(processors string) error {
	if _, ok := SupportedProcessorsSet[processors]; !ok {
		return fmt.Errorf("metrics_generator.processor \"%s\" is not a known processor, valid values: %v", processors, SupportedProcessors)
	}
	return nil
}

func ValidateCollectionInterval(collectionInterval time.Duration) error {
	if collectionInterval < 15*time.Second || collectionInterval > 5*time.Minute {
		return fmt.Errorf("metrics_generator.collection_interval \"%s\" is outside acceptable range of 15s to 5m", collectionInterval.String())
	}
	return nil
}

func ValidateIngestionTimeRangeSlack(ingestionTimeRangeSlack time.Duration) error {
	if ingestionTimeRangeSlack < 0 || ingestionTimeRangeSlack > 12*time.Hour {
		return fmt.Errorf("metrics_generator.ingestion_time_range_slack \"%s\" is outside acceptable range of 0s to 12h", ingestionTimeRangeSlack.String())
	}
	return nil
}

func ValidateHistogramMode(mode string) error {
	if _, ok := SupportedHistogramModesSet[mode]; !ok {
		return fmt.Errorf("metrics_generator.generate_native_histograms \"%s\" is not a valid value, valid values: classic, native, both", mode)
	}
	return nil
}

func ValidateHostInfoHostIdentifiers(hostIdentifiers []string) error {
	if len(hostIdentifiers) == 0 {
		return errors.New("at least one value must be provided in host_identifiers")
	}
	return nil
}

func ValidateHostInfoMetricName(metricName string) error {
	if !model.UTF8Validation.IsValidLabelName(metricName) {
		return errors.New("metric_name is invalid")
	}
	return nil
}

func ValidateDimensions(dimensions []string, intrinsicDimensions []string, dimensionMappings []sharedconfig.DimensionMappings, sanitizeFn SanitizeFn) error {
	var labels []string
	labels = append(labels, intrinsicDimensions...)
	for _, d := range dimensions {
		labels = append(labels, SanitizeLabelNameWithCollisions(d, SupportedIntrinsicDimensionsSet, sanitizeFn))
	}

	for _, m := range dimensionMappings {
		labels = append(labels, SanitizeLabelNameWithCollisions(m.Name, SupportedIntrinsicDimensionsSet, sanitizeFn))
	}

	err := ValidateUTF8LabelValues(labels)
	if err != nil {
		return err
	}
	return nil
}

func ValidateTraceIDLabelName(traceIDLabelName string) error {
	if traceIDLabelName != SanitizeLabelName(traceIDLabelName) {
		return fmt.Errorf("trace_id_label_name \"%s\" is not a valid Prometheus label name", traceIDLabelName)
	}
	return nil
}

func ValidateHistogramBuckets(buckets []float64, field string) error {
	for i, bucket := range buckets {
		if i > 0 && bucket <= buckets[i-1] {
			return fmt.Errorf("%s must be strictly increasing: bucket[%d]=%g is <= bucket[%d]=%g", field, i, bucket, i-1, buckets[i-1])
		}
	}
	return nil
}

func ValidateNativeHistogramBucketFactor(factor float64) error {
	if factor <= 1 {
		return fmt.Errorf("metrics_generator.native_histogram_bucket_factor must be greater than 1")
	}
	return nil
}

func ValidateCostAttributionDimensions(dimensions map[string]string) error {
	seenLabels := make(map[string]string)

	// map is with key=tempo attribute, value=prometheus labelName
	for k, v := range dimensions {
		// build labelName in the similar way as usage.GetBuffersForDimensions
		attr, _ := usage.ParseDimensionKey(k) // extract attr so validate the duplicates with scope prefix
		labelName := v
		if labelName == "" {
			labelName = attr // The dimension is using default mapping, we map it to attribute
		}
		labelName = strutil.SanitizeFullLabelName(labelName) // sanitize label name

		// check for duplicate prometheus label names.
		// when we have duplicate labelNames, we randomly pick one so validate and don't allow duplicates.
		if originalKey, exists := seenLabels[labelName]; exists {
			return fmt.Errorf("cost_attribution.dimensions has duplicate label name: '%s', both '%s' and '%s' map to it", labelName, originalKey, k)
		}
		seenLabels[labelName] = k // put k as value so we can show configured keys in the error

		// creating a desc do the complete labelName validation
		desc := prometheus.NewDesc("test_desc", "test desc created for validation", []string{labelName}, nil)
		// try to create a metric and see if there are any error, we use same method in usage.Collect
		_, err := prometheus.NewConstMetric(desc, prometheus.CounterValue, float64(1), labelName)
		if err != nil {
			return fmt.Errorf("cost_attribution.dimensions config has invalid label name: '%s'", labelName)
		}
	}

	// no errors, we are good.
	return nil
}

func ValidateIntrinsicDimensions(intrinsicDimensions map[string]bool) error {
	for dim := range intrinsicDimensions {
		if _, ok := SupportedIntrinsicDimensionsSet[dim]; !ok {
			return fmt.Errorf("intrinsic dimension \"%s\" is not supported, valid values: %v", dim, SupportedIntrinsicDimensions)
		}
	}
	return nil
}
