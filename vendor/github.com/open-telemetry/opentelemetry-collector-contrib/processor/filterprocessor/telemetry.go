// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor/internal/metadata"
)

type filterTelemetry struct {
	attr    metric.MeasurementOption
	counter metric.Int64Counter
}

func newFilterTelemetry(set processor.Settings, signal pipeline.Signal) (*filterTelemetry, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	var counter metric.Int64Counter
	switch signal {
	case pipeline.SignalMetrics:
		counter = telemetryBuilder.ProcessorFilterDatapointsFiltered
	case pipeline.SignalLogs:
		counter = telemetryBuilder.ProcessorFilterLogsFiltered
	case pipeline.SignalTraces:
		counter = telemetryBuilder.ProcessorFilterSpansFiltered
	case xpipeline.SignalProfiles:
		counter = telemetryBuilder.ProcessorFilterProfilesFiltered
	default:
		return nil, fmt.Errorf("unsupported signal type: %v", signal)
	}

	return &filterTelemetry{
		attr:    metric.WithAttributeSet(attribute.NewSet(attribute.String(metadata.Type.String(), set.ID.String()))),
		counter: counter,
	}, nil
}

func (fpt *filterTelemetry) record(ctx context.Context, dropped int64) {
	fpt.counter.Add(ctx, dropped, fpt.attr)
}
