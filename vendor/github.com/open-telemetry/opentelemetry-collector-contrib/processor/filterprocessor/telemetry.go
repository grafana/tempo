// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"

import (
	"context"

	"go.opentelemetry.io/collector/processor"
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

	telemetryBuilder *metadata.TelemetryBuilder
}

func newfilterProcessorTelemetry(set processor.CreateSettings) (*filterProcessorTelemetry, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return &filterProcessorTelemetry{
		processorAttr:    []attribute.KeyValue{attribute.String(metadata.Type.String(), set.ID.String())},
		exportCtx:        context.Background(),
		telemetryBuilder: telemetryBuilder,
	}, nil
}

func (fpt *filterProcessorTelemetry) record(trigger trigger, dropped int64) {
	var triggerMeasure metric.Int64Counter
	switch trigger {
	case triggerMetricDataPointsDropped:
		triggerMeasure = fpt.telemetryBuilder.ProcessorFilterDatapointsFiltered
	case triggerLogsDropped:
		triggerMeasure = fpt.telemetryBuilder.ProcessorFilterLogsFiltered
	case triggerSpansDropped:
		triggerMeasure = fpt.telemetryBuilder.ProcessorFilterSpansFiltered
	}

	triggerMeasure.Add(fpt.exportCtx, dropped, metric.WithAttributes(fpt.processorAttr...))
}
