// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package spanpruningprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/collector/processor/xprocessor"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor/internal/metadata"
)

var processorCapabilities = consumer.Capabilities{MutatesData: true}

// NewFactory returns a new factory for the Span Pruning processor.
func NewFactory() processor.Factory {
	return xprocessor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		xprocessor.WithTraces(createTracesProcessor, metadata.TracesStability),
		xprocessor.WithDeprecatedTypeAlias(metadata.DeprecatedType),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		MinSpansToAggregate:        5,
		MaxParentDepth:             1,
		AggregationAttributePrefix: "aggregation.",
		AggregationHistogramBuckets: []time.Duration{
			5 * time.Millisecond,
			10 * time.Millisecond,
			25 * time.Millisecond,
			50 * time.Millisecond,
			100 * time.Millisecond,
			250 * time.Millisecond,
			500 * time.Millisecond,
			time.Second,
			2500 * time.Millisecond,
			5 * time.Second,
			10 * time.Second,
		},
		EnableOutlierAnalysis: false,
		OutlierAnalysis: OutlierAnalysisConfig{
			Method:                         OutlierMethodIQR,
			IQRMultiplier:                  1.5,
			MADMultiplier:                  3.0,
			MinGroupSize:                   7,
			CorrelationMinOccurrence:       0.75,
			CorrelationMaxNormalOccurrence: 0.25,
			MaxCorrelatedAttributes:        5,
			PreserveOutliers:               false,
			MaxPreservedOutliers:           2,
			PreserveOnlyWithCorrelation:    false,
			MinOutlierThresholdPercent:     0.1,
		},
		EnableExemplarSampling: false,
		ExemplarSampling: ExemplarSamplingConfig{
			PrecisionMultiplier: 1.0,
		},
	}
}

func createTracesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	pCfg := cfg.(*Config)

	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	// Compile OTTL conditions if configured.
	var conditions *ottl.ConditionSequence[*ottlspan.TransformContext]
	if len(pCfg.Conditions) > 0 {
		conditions, err = filterottl.NewBoolExprForSpan(
			pCfg.Conditions,
			filterottl.StandardSpanFuncs(),
			ottl.PropagateError,
			set.TelemetrySettings,
		)
		if err != nil {
			return nil, err
		}
		set.Logger.Info("OTTL conditions configured", zap.Int("count", len(pCfg.Conditions)))
	}

	p, err := newSpanPruningProcessor(set, pCfg, telemetryBuilder, conditions)
	if err != nil {
		return nil, err
	}

	return processorhelper.NewTraces(
		ctx,
		set,
		cfg,
		nextConsumer,
		p.processTraces,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithShutdown(p.shutdown),
	)
}
