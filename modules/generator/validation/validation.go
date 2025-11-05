package validation

import (
	"errors"
	"fmt"
	"time"

	"github.com/grafana/tempo/modules/generator/processor/hostinfo"
	"github.com/grafana/tempo/modules/generator/processor/localblocks"
	"github.com/grafana/tempo/modules/generator/processor/servicegraphs"
	"github.com/grafana/tempo/modules/generator/processor/spanmetrics"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/prometheus/common/model"
)

var SupportedProcessors = []string{
	servicegraphs.Name,
	spanmetrics.Name,
	localblocks.Name,
	spanmetrics.Count.String(),
	spanmetrics.Latency.String(),
	spanmetrics.Size.String(),
	hostinfo.Name,
}

var SupportedProcessorsSet map[string]struct{}

var SupportedHistogramModesSet map[string]struct{}

func init() {
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
	if !model.IsValidMetricName(model.LabelValue(metricName)) {
		return errors.New("metric_name is invalid")
	}
	return nil
}
