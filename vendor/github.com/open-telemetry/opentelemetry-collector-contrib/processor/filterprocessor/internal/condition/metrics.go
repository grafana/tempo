// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package condition // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor/internal/condition"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.uber.org/multierr"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottldatapoint"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlmetric"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlresource"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlscope"
)

type MetricsConsumer struct {
	resourceExpr  expr.BoolExpr[*ottlresource.TransformContext]
	scopeExpr     expr.BoolExpr[*ottlscope.TransformContext]
	metricExpr    expr.BoolExpr[*ottlmetric.TransformContext]
	dataPointExpr expr.BoolExpr[*ottldatapoint.TransformContext]
}

// parsedMetricConditions is the type R for ParserCollection[R] that holds parsed OTTL conditions
type parsedMetricConditions struct {
	resourceConditions  []*ottl.Condition[*ottlresource.TransformContext]
	scopeConditions     []*ottl.Condition[*ottlscope.TransformContext]
	metricConditions    []*ottl.Condition[*ottlmetric.TransformContext]
	dataPointConditions []*ottl.Condition[*ottldatapoint.TransformContext]
	telemetrySettings   component.TelemetrySettings
	errorMode           ottl.ErrorMode
}

func (mc MetricsConsumer) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	var condErr error
	md.ResourceMetrics().RemoveIf(func(rm pmetric.ResourceMetrics) bool {
		if mc.resourceExpr != nil {
			rCtx := ottlresource.NewTransformContextPtr(rm.Resource(), rm)
			rCond, rErr := mc.resourceExpr.Eval(ctx, rCtx)
			rCtx.Close()
			if rErr != nil {
				condErr = multierr.Append(condErr, rErr)
				return false
			}
			if rCond {
				return true
			}
		}

		if mc.scopeExpr == nil && mc.metricExpr == nil && mc.dataPointExpr == nil {
			return rm.ScopeMetrics().Len() == 0
		}

		rm.ScopeMetrics().RemoveIf(func(sm pmetric.ScopeMetrics) bool {
			if mc.scopeExpr != nil {
				sCtx := ottlscope.NewTransformContextPtr(sm.Scope(), rm.Resource(), sm)
				sCond, sErr := mc.scopeExpr.Eval(ctx, sCtx)
				sCtx.Close()
				if sErr != nil {
					condErr = multierr.Append(condErr, sErr)
					return false
				}
				if sCond {
					return true
				}
			}

			if mc.metricExpr == nil && mc.dataPointExpr == nil {
				return sm.Metrics().Len() == 0
			}

			sm.Metrics().RemoveIf(func(metric pmetric.Metric) bool {
				if mc.metricExpr != nil {
					tCtx := ottlmetric.NewTransformContextPtr(rm, sm, metric)
					mCond, err := mc.metricExpr.Eval(ctx, tCtx)
					tCtx.Close()
					if err != nil {
						condErr = multierr.Append(condErr, err)
						return false
					}
					if mCond {
						return true
					}
				}

				if mc.dataPointExpr != nil {
					//exhaustive:enforce
					switch metric.Type() {
					case pmetric.MetricTypeSum:
						err := mc.handleNumberDataPoints(ctx, rm, sm, metric, metric.Sum().DataPoints())
						if err != nil {
							condErr = multierr.Append(condErr, err)
							return false
						}
						return metric.Sum().DataPoints().Len() == 0
					case pmetric.MetricTypeGauge:
						err := mc.handleNumberDataPoints(ctx, rm, sm, metric, metric.Gauge().DataPoints())
						if err != nil {
							condErr = multierr.Append(condErr, err)
							return false
						}
						return metric.Gauge().DataPoints().Len() == 0
					case pmetric.MetricTypeHistogram:
						err := mc.handleHistogramDataPoints(ctx, rm, sm, metric, metric.Histogram().DataPoints())
						if err != nil {
							condErr = multierr.Append(condErr, err)
							return false
						}
						return metric.Histogram().DataPoints().Len() == 0
					case pmetric.MetricTypeExponentialHistogram:
						err := mc.handleExponentialHistogramDataPoints(ctx, rm, sm, metric, metric.ExponentialHistogram().DataPoints())
						if err != nil {
							condErr = multierr.Append(condErr, err)
							return false
						}
						return metric.ExponentialHistogram().DataPoints().Len() == 0
					case pmetric.MetricTypeSummary:
						err := mc.handleSummaryDataPoints(ctx, rm, sm, metric, metric.Summary().DataPoints())
						if err != nil {
							condErr = multierr.Append(condErr, err)
							return false
						}
						return metric.Summary().DataPoints().Len() == 0
					default:
						return false
					}
				}
				return false
			})
			return sm.Metrics().Len() == 0
		})
		return rm.ScopeMetrics().Len() == 0
	})

	if md.ResourceMetrics().Len() == 0 {
		return processorhelper.ErrSkipProcessingData
	}
	return condErr
}

func (mc MetricsConsumer) handleNumberDataPoints(ctx context.Context, rm pmetric.ResourceMetrics, sm pmetric.ScopeMetrics, m pmetric.Metric, dps pmetric.NumberDataPointSlice) error {
	var errors error
	dps.RemoveIf(func(datapoint pmetric.NumberDataPoint) bool {
		tCtx := ottldatapoint.NewTransformContextPtr(rm, sm, m, datapoint)
		cond, err := mc.dataPointExpr.Eval(ctx, tCtx)
		tCtx.Close()
		if err != nil {
			errors = multierr.Append(errors, err)
			return false
		}
		return cond
	})
	return errors
}

func (mc MetricsConsumer) handleHistogramDataPoints(ctx context.Context, rm pmetric.ResourceMetrics, sm pmetric.ScopeMetrics, m pmetric.Metric, dps pmetric.HistogramDataPointSlice) error {
	var errors error
	dps.RemoveIf(func(dp pmetric.HistogramDataPoint) bool {
		tCtx := ottldatapoint.NewTransformContextPtr(rm, sm, m, dp)
		cond, err := mc.dataPointExpr.Eval(ctx, tCtx)
		tCtx.Close()
		if err != nil {
			errors = multierr.Append(errors, err)
			return false
		}
		return cond
	})
	return errors
}

func (mc MetricsConsumer) handleExponentialHistogramDataPoints(ctx context.Context, rm pmetric.ResourceMetrics, sm pmetric.ScopeMetrics, m pmetric.Metric, dps pmetric.ExponentialHistogramDataPointSlice) error {
	var errors error
	dps.RemoveIf(func(dp pmetric.ExponentialHistogramDataPoint) bool {
		tCtx := ottldatapoint.NewTransformContextPtr(rm, sm, m, dp)
		cond, err := mc.dataPointExpr.Eval(ctx, tCtx)
		tCtx.Close()
		if err != nil {
			errors = multierr.Append(errors, err)
			return false
		}
		return cond
	})
	return errors
}

func (mc MetricsConsumer) handleSummaryDataPoints(ctx context.Context, rm pmetric.ResourceMetrics, sm pmetric.ScopeMetrics, m pmetric.Metric, dps pmetric.SummaryDataPointSlice) error {
	var errors error
	dps.RemoveIf(func(dp pmetric.SummaryDataPoint) bool {
		tCtx := ottldatapoint.NewTransformContextPtr(rm, sm, m, dp)
		cond, err := mc.dataPointExpr.Eval(ctx, tCtx)
		tCtx.Close()
		if err != nil {
			errors = multierr.Append(errors, err)
			return false
		}
		return cond
	})
	return errors
}

func newMetricConditionsFromResource(rc []*ottl.Condition[*ottlresource.TransformContext], telemetrySettings component.TelemetrySettings, errorMode ottl.ErrorMode) parsedMetricConditions {
	return parsedMetricConditions{
		resourceConditions: rc,
		telemetrySettings:  telemetrySettings,
		errorMode:          errorMode,
	}
}

func newMetricConditionsFromScope(sc []*ottl.Condition[*ottlscope.TransformContext], telemetrySettings component.TelemetrySettings, errorMode ottl.ErrorMode) parsedMetricConditions {
	return parsedMetricConditions{
		scopeConditions:   sc,
		telemetrySettings: telemetrySettings,
		errorMode:         errorMode,
	}
}

func newMetricsConsumer(mc *parsedMetricConditions) MetricsConsumer {
	var rExpr expr.BoolExpr[*ottlresource.TransformContext]
	var sExpr expr.BoolExpr[*ottlscope.TransformContext]
	var mExpr expr.BoolExpr[*ottlmetric.TransformContext]
	var dExpr expr.BoolExpr[*ottldatapoint.TransformContext]

	if len(mc.resourceConditions) > 0 {
		cs := ottlresource.NewConditionSequence(mc.resourceConditions, mc.telemetrySettings, ottlresource.WithConditionSequenceErrorMode(mc.errorMode))
		rExpr = &cs
	}

	if len(mc.scopeConditions) > 0 {
		cs := ottlscope.NewConditionSequence(mc.scopeConditions, mc.telemetrySettings, ottlscope.WithConditionSequenceErrorMode(mc.errorMode))
		sExpr = &cs
	}

	if len(mc.metricConditions) > 0 {
		cs := ottlmetric.NewConditionSequence(mc.metricConditions, mc.telemetrySettings, ottlmetric.WithConditionSequenceErrorMode(mc.errorMode))
		mExpr = &cs
	}

	if len(mc.dataPointConditions) > 0 {
		cs := ottldatapoint.NewConditionSequence(mc.dataPointConditions, mc.telemetrySettings, ottldatapoint.WithConditionSequenceErrorMode(mc.errorMode))
		dExpr = &cs
	}

	return MetricsConsumer{
		resourceExpr:  rExpr,
		scopeExpr:     sExpr,
		metricExpr:    mExpr,
		dataPointExpr: dExpr,
	}
}

type MetricParserCollection ottl.ParserCollection[parsedMetricConditions]

type MetricParserCollectionOption ottl.ParserCollectionOption[parsedMetricConditions]

func WithMetricParser(functions map[string]ottl.Factory[*ottlmetric.TransformContext]) MetricParserCollectionOption {
	return func(pc *ottl.ParserCollection[parsedMetricConditions]) error {
		metricParser, err := ottlmetric.NewParser(functions, pc.Settings, ottlmetric.EnablePathContextNames())
		if err != nil {
			return err
		}
		return ottl.WithParserCollectionContext(ottlmetric.ContextName, &metricParser, ottl.WithConditionConverter(convertMetricConditions))(pc)
	}
}

func WithDataPointParser(functions map[string]ottl.Factory[*ottldatapoint.TransformContext]) MetricParserCollectionOption {
	return func(pc *ottl.ParserCollection[parsedMetricConditions]) error {
		dataPointParser, err := ottldatapoint.NewParser(functions, pc.Settings, ottldatapoint.EnablePathContextNames())
		if err != nil {
			return err
		}
		return ottl.WithParserCollectionContext(ottldatapoint.ContextName, &dataPointParser, ottl.WithConditionConverter(convertDataPointConditions))(pc)
	}
}

func WithMetricErrorMode(errorMode ottl.ErrorMode) MetricParserCollectionOption {
	return MetricParserCollectionOption(ottl.WithParserCollectionErrorMode[parsedMetricConditions](errorMode))
}

func WithMetricCommonParsers(functions map[string]ottl.Factory[*ottlresource.TransformContext]) MetricParserCollectionOption {
	return MetricParserCollectionOption(withCommonParsers(functions, newMetricConditionsFromResource, newMetricConditionsFromScope))
}

func NewMetricParserCollection(settings component.TelemetrySettings, options ...MetricParserCollectionOption) (*MetricParserCollection, error) {
	pcOptions := []ottl.ParserCollectionOption[parsedMetricConditions]{
		ottl.EnableParserCollectionModifiedPathsLogging[parsedMetricConditions](true),
	}

	for _, option := range options {
		pcOptions = append(pcOptions, ottl.ParserCollectionOption[parsedMetricConditions](option))
	}

	pc, err := ottl.NewParserCollection(settings, pcOptions...)
	if err != nil {
		return nil, err
	}

	mpc := MetricParserCollection(*pc)
	return &mpc, nil
}

func convertMetricConditions(pc *ottl.ParserCollection[parsedMetricConditions], conditions ottl.ConditionsGetter, parsedConditions []*ottl.Condition[*ottlmetric.TransformContext]) (parsedMetricConditions, error) {
	contextConditions, err := toContextConditions(conditions)
	if err != nil {
		return parsedMetricConditions{}, err
	}

	errorMode := getErrorMode(pc, contextConditions)
	return parsedMetricConditions{
		metricConditions:  parsedConditions,
		telemetrySettings: pc.Settings,
		errorMode:         errorMode,
	}, nil
}

func convertDataPointConditions(pc *ottl.ParserCollection[parsedMetricConditions], conditions ottl.ConditionsGetter, parsedConditions []*ottl.Condition[*ottldatapoint.TransformContext]) (parsedMetricConditions, error) {
	contextConditions, err := toContextConditions(conditions)
	if err != nil {
		return parsedMetricConditions{}, err
	}

	errorMode := getErrorMode(pc, contextConditions)
	return parsedMetricConditions{
		dataPointConditions: parsedConditions,
		telemetrySettings:   pc.Settings,
		errorMode:           errorMode,
	}, nil
}

func (mpc *MetricParserCollection) ParseContextConditions(contextConditions ContextConditions) (MetricsConsumer, error) {
	pc := ottl.ParserCollection[parsedMetricConditions](*mpc)
	if contextConditions.Context != "" {
		mc, err := pc.ParseConditionsWithContext(string(contextConditions.Context), contextConditions, true)
		if err != nil {
			return MetricsConsumer{}, err
		}
		return newMetricsConsumer(&mc), nil
	}

	var rConditions []*ottl.Condition[*ottlresource.TransformContext]
	var sConditions []*ottl.Condition[*ottlscope.TransformContext]
	var mConditions []*ottl.Condition[*ottlmetric.TransformContext]
	var dConditions []*ottl.Condition[*ottldatapoint.TransformContext]

	for _, cc := range contextConditions.GetConditions() {
		mc, err := pc.ParseConditions(ContextConditions{Conditions: []string{cc}})
		if err != nil {
			return MetricsConsumer{}, err
		}

		if len(mc.resourceConditions) > 0 {
			rConditions = append(rConditions, mc.resourceConditions...)
		}
		if len(mc.scopeConditions) > 0 {
			sConditions = append(sConditions, mc.scopeConditions...)
		}
		if len(mc.metricConditions) > 0 {
			mConditions = append(mConditions, mc.metricConditions...)
		}
		if len(mc.dataPointConditions) > 0 {
			dConditions = append(dConditions, mc.dataPointConditions...)
		}
	}

	aggregatedConditions := parsedMetricConditions{
		resourceConditions:  rConditions,
		scopeConditions:     sConditions,
		metricConditions:    mConditions,
		dataPointConditions: dConditions,
		telemetrySettings:   pc.Settings,
		errorMode:           getErrorMode[parsedMetricConditions](&pc, &contextConditions),
	}

	return newMetricsConsumer(&aggregatedConditions), nil
}
