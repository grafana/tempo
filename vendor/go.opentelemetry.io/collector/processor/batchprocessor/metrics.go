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

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"context"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	"go.opentelemetry.io/otel/metric/unit"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	otelview "go.opentelemetry.io/otel/sdk/metric/view"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/obsreport"
)

const (
	scopeName = "go.opentelemetry.io/collector/processor/batchprocessor"
)

var (
	processorTagKey          = tag.MustNewKey(obsmetrics.ProcessorKey)
	statBatchSizeTriggerSend = stats.Int64("batch_size_trigger_send", "Number of times the batch was sent due to a size trigger", stats.UnitDimensionless)
	statTimeoutTriggerSend   = stats.Int64("timeout_trigger_send", "Number of times the batch was sent due to a timeout trigger", stats.UnitDimensionless)
	statBatchSendSize        = stats.Int64("batch_send_size", "Number of units in the batch", stats.UnitDimensionless)
	statBatchSendSizeBytes   = stats.Int64("batch_send_size_bytes", "Number of bytes in batch that was sent", stats.UnitBytes)
)

type trigger int

const (
	triggerTimeout trigger = iota
	triggerBatchSize
)

// MetricViews returns the metrics views related to batching
func MetricViews() []*view.View {
	processorTagKeys := []tag.Key{processorTagKey}

	countBatchSizeTriggerSendView := &view.View{
		Name:        obsreport.BuildProcessorCustomMetricName(typeStr, statBatchSizeTriggerSend.Name()),
		Measure:     statBatchSizeTriggerSend,
		Description: statBatchSizeTriggerSend.Description(),
		TagKeys:     processorTagKeys,
		Aggregation: view.Sum(),
	}

	countTimeoutTriggerSendView := &view.View{
		Name:        obsreport.BuildProcessorCustomMetricName(typeStr, statTimeoutTriggerSend.Name()),
		Measure:     statTimeoutTriggerSend,
		Description: statTimeoutTriggerSend.Description(),
		TagKeys:     processorTagKeys,
		Aggregation: view.Sum(),
	}

	distributionBatchSendSizeView := &view.View{
		Name:        obsreport.BuildProcessorCustomMetricName(typeStr, statBatchSendSize.Name()),
		Measure:     statBatchSendSize,
		Description: statBatchSendSize.Description(),
		TagKeys:     processorTagKeys,
		Aggregation: view.Distribution(10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000, 100000),
	}

	distributionBatchSendSizeBytesView := &view.View{
		Name:        obsreport.BuildProcessorCustomMetricName(typeStr, statBatchSendSizeBytes.Name()),
		Measure:     statBatchSendSizeBytes,
		Description: statBatchSendSizeBytes.Description(),
		TagKeys:     processorTagKeys,
		Aggregation: view.Distribution(10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000,
			100_000, 200_000, 300_000, 400_000, 500_000, 600_000, 700_000, 800_000, 900_000,
			1000_000, 2000_000, 3000_000, 4000_000, 5000_000, 6000_000, 7000_000, 8000_000, 9000_000),
	}

	return []*view.View{
		countBatchSizeTriggerSendView,
		countTimeoutTriggerSendView,
		distributionBatchSendSizeView,
		distributionBatchSendSizeBytesView,
	}
}

func OtelMetricsViews() ([]otelview.View, error) {
	var views []otelview.View
	var err error

	v, err := otelview.New(
		otelview.MatchInstrumentName(obsreport.BuildProcessorCustomMetricName(typeStr, "batch_send_size")),
		otelview.WithSetAggregation(aggregation.ExplicitBucketHistogram{
			Boundaries: []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000, 100000},
		}),
	)
	if err != nil {
		return nil, err
	}
	views = append(views, v)

	v, err = otelview.New(
		otelview.MatchInstrumentName(obsreport.BuildProcessorCustomMetricName(typeStr, "batch_send_size_bytes")),
		otelview.WithSetAggregation(aggregation.ExplicitBucketHistogram{
			Boundaries: []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000,
				100_000, 200_000, 300_000, 400_000, 500_000, 600_000, 700_000, 800_000, 900_000,
				1000_000, 2000_000, 3000_000, 4000_000, 5000_000, 6000_000, 7000_000, 8000_000, 9000_000},
		}),
	)
	if err != nil {
		return nil, err
	}
	views = append(views, v)

	return views, nil
}

type batchProcessorTelemetry struct {
	level    configtelemetry.Level
	detailed bool
	useOtel  bool

	exportCtx context.Context

	processorAttr        []attribute.KeyValue
	batchSizeTriggerSend syncint64.Counter
	timeoutTriggerSend   syncint64.Counter
	batchSendSize        syncint64.Histogram
	batchSendSizeBytes   syncint64.Histogram
}

func newBatchProcessorTelemetry(mp metric.MeterProvider, cfg *Config, level configtelemetry.Level, registry *featuregate.Registry) (*batchProcessorTelemetry, error) {
	exportCtx, err := tag.New(context.Background(), tag.Insert(processorTagKey, cfg.ID().String()))
	if err != nil {
		return nil, err
	}

	bpt := &batchProcessorTelemetry{
		useOtel:       registry.IsEnabled(obsreportconfig.UseOtelForInternalMetricsfeatureGateID),
		processorAttr: []attribute.KeyValue{attribute.String(obsmetrics.ProcessorKey, cfg.ID().String())},
		exportCtx:     exportCtx,
		level:         level,
		detailed:      level == configtelemetry.LevelDetailed,
	}

	err = bpt.createOtelMetrics(mp)
	if err != nil {
		return nil, err
	}

	return bpt, nil
}

func (bpt *batchProcessorTelemetry) createOtelMetrics(mp metric.MeterProvider) error {
	if !bpt.useOtel {
		return nil
	}

	var err error
	meter := mp.Meter(scopeName)

	bpt.batchSizeTriggerSend, err = meter.SyncInt64().Counter(
		obsreport.BuildProcessorCustomMetricName(typeStr, "batch_size_trigger_send"),
		instrument.WithDescription("Number of times the batch was sent due to a size trigger"),
		instrument.WithUnit(unit.Dimensionless),
	)
	if err != nil {
		return err
	}

	bpt.timeoutTriggerSend, err = meter.SyncInt64().Counter(
		obsreport.BuildProcessorCustomMetricName(typeStr, "timeout_trigger_send"),
		instrument.WithDescription("Number of times the batch was sent due to a timeout trigger"),
		instrument.WithUnit(unit.Dimensionless),
	)
	if err != nil {
		return err
	}

	bpt.batchSendSize, err = meter.SyncInt64().Histogram(
		obsreport.BuildProcessorCustomMetricName(typeStr, "batch_send_size"),
		instrument.WithDescription("Number of units in the batch"),
		instrument.WithUnit(unit.Dimensionless),
	)
	if err != nil {
		return err
	}

	bpt.batchSendSizeBytes, err = meter.SyncInt64().Histogram(
		obsreport.BuildProcessorCustomMetricName(typeStr, "batch_send_size_bytes"),
		instrument.WithDescription("Number of bytes in batch that was sent"),
		instrument.WithUnit(unit.Bytes),
	)
	if err != nil {
		return err
	}

	return nil
}

func (bpt *batchProcessorTelemetry) record(trigger trigger, sent, bytes int64) {
	if bpt.useOtel {
		bpt.recordWithOtel(trigger, sent, bytes)
	} else {
		bpt.recordWithOC(trigger, sent, bytes)
	}
}

func (bpt *batchProcessorTelemetry) recordWithOC(trigger trigger, sent, bytes int64) {
	var triggerMeasure *stats.Int64Measure
	switch trigger {
	case triggerBatchSize:
		triggerMeasure = statBatchSizeTriggerSend
	case triggerTimeout:
		triggerMeasure = statTimeoutTriggerSend
	}

	stats.Record(bpt.exportCtx, triggerMeasure.M(1), statBatchSendSize.M(sent))
	if bpt.detailed {
		stats.Record(bpt.exportCtx, statBatchSendSizeBytes.M(bytes))
	}
}

func (bpt *batchProcessorTelemetry) recordWithOtel(trigger trigger, sent int64, bytes int64) {
	switch trigger {
	case triggerBatchSize:
		bpt.batchSizeTriggerSend.Add(bpt.exportCtx, 1, bpt.processorAttr...)
	case triggerTimeout:
		bpt.timeoutTriggerSend.Add(bpt.exportCtx, 1, bpt.processorAttr...)
	}

	bpt.batchSendSize.Record(bpt.exportCtx, sent, bpt.processorAttr...)
	if bpt.detailed {
		bpt.batchSendSizeBytes.Record(bpt.exportCtx, bytes, bpt.processorAttr...)
	}
}
