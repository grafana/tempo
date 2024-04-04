// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"

import (
	"context"

	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor/internal/metadata"
)

type trigger int

const (
	triggerMetricDataPointsDropped trigger = iota
	triggerLogsDropped
	triggerSpansDropped
)

type filterProcessorTelemetry struct {
	exportCtx context.Context

	processorAttr []attribute.KeyValue

	datapointsFiltered metric.Int64Counter
	logsFiltered       metric.Int64Counter
	spansFiltered      metric.Int64Counter
}

func newfilterProcessorTelemetry(set processor.CreateSettings) (*filterProcessorTelemetry, error) {
	processorID := set.ID.String()

	fpt := &filterProcessorTelemetry{
		processorAttr: []attribute.KeyValue{attribute.String(metadata.Type.String(), processorID)},
		exportCtx:     context.Background(),
	}

	counter, err := metadata.Meter(set.TelemetrySettings).Int64Counter(
		processorhelper.BuildCustomMetricName(metadata.Type.String(), "datapoints.filtered"),
		metric.WithDescription("Number of metric data points dropped by the filter processor"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}
	fpt.datapointsFiltered = counter

	counter, err = metadata.Meter(set.TelemetrySettings).Int64Counter(
		processorhelper.BuildCustomMetricName(metadata.Type.String(), "logs.filtered"),
		metric.WithDescription("Number of logs dropped by the filter processor"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}
	fpt.logsFiltered = counter

	counter, err = metadata.Meter(set.TelemetrySettings).Int64Counter(
		processorhelper.BuildCustomMetricName(metadata.Type.String(), "spans.filtered"),
		metric.WithDescription("Number of spans dropped by the filter processor"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}
	fpt.spansFiltered = counter

	return fpt, nil
}

func (fpt *filterProcessorTelemetry) record(trigger trigger, dropped int64) {
	var triggerMeasure metric.Int64Counter
	switch trigger {
	case triggerMetricDataPointsDropped:
		triggerMeasure = fpt.datapointsFiltered
	case triggerLogsDropped:
		triggerMeasure = fpt.logsFiltered
	case triggerSpansDropped:
		triggerMeasure = fpt.spansFiltered
	}

	triggerMeasure.Add(fpt.exportCtx, dropped, metric.WithAttributes(fpt.processorAttr...))
}
