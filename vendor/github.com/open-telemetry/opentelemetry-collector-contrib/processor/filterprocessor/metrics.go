// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterconfig"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filtermatcher"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filtermetric"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterset"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottldatapoint"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlmetric"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlresource"
)

type filterMetricProcessor struct {
	skipResourceExpr  expr.BoolExpr[ottlresource.TransformContext]
	skipMetricExpr    expr.BoolExpr[ottlmetric.TransformContext]
	skipDataPointExpr expr.BoolExpr[ottldatapoint.TransformContext]
	telemetry         *filterProcessorTelemetry
	logger            *zap.Logger
}

func newFilterMetricProcessor(set processor.CreateSettings, cfg *Config) (*filterMetricProcessor, error) {
	var err error
	fsp := &filterMetricProcessor{
		logger: set.Logger,
	}

	fpt, err := newfilterProcessorTelemetry(set)
	if err != nil {
		return nil, fmt.Errorf("error creating filter processor telemetry: %w", err)
	}
	fsp.telemetry = fpt

	if cfg.Metrics.MetricConditions != nil || cfg.Metrics.DataPointConditions != nil {
		if cfg.Metrics.MetricConditions != nil {
			fsp.skipMetricExpr, err = filterottl.NewBoolExprForMetric(cfg.Metrics.MetricConditions, filterottl.StandardMetricFuncs(), cfg.ErrorMode, set.TelemetrySettings)
			if err != nil {
				return nil, err
			}
		}

		if cfg.Metrics.DataPointConditions != nil {
			fsp.skipDataPointExpr, err = filterottl.NewBoolExprForDataPoint(cfg.Metrics.DataPointConditions, filterottl.StandardDataPointFuncs(), cfg.ErrorMode, set.TelemetrySettings)
			if err != nil {
				return nil, err
			}
		}

		return fsp, nil
	}

	fsp.skipResourceExpr, err = newSkipResExpr(cfg.Metrics.Include, cfg.Metrics.Exclude)
	if err != nil {
		return nil, err
	}

	fsp.skipMetricExpr, err = filtermetric.NewSkipExpr(cfg.Metrics.Include, cfg.Metrics.Exclude)
	if err != nil {
		return nil, err
	}

	includeMatchType := ""
	var includeExpressions []string
	var includeMetricNames []string
	var includeResourceAttributes []filterconfig.Attribute
	if cfg.Metrics.Include != nil {
		includeMatchType = string(cfg.Metrics.Include.MatchType)
		includeExpressions = cfg.Metrics.Include.Expressions
		includeMetricNames = cfg.Metrics.Include.MetricNames
		includeResourceAttributes = cfg.Metrics.Include.ResourceAttributes
	}

	excludeMatchType := ""
	var excludeExpressions []string
	var excludeMetricNames []string
	var excludeResourceAttributes []filterconfig.Attribute
	if cfg.Metrics.Exclude != nil {
		excludeMatchType = string(cfg.Metrics.Exclude.MatchType)
		excludeExpressions = cfg.Metrics.Exclude.Expressions
		excludeMetricNames = cfg.Metrics.Exclude.MetricNames
		excludeResourceAttributes = cfg.Metrics.Exclude.ResourceAttributes
	}

	set.Logger.Info(
		"Metric filter configured",
		zap.String("include match_type", includeMatchType),
		zap.Strings("include expressions", includeExpressions),
		zap.Strings("include metric names", includeMetricNames),
		zap.Any("include metrics with resource attributes", includeResourceAttributes),
		zap.String("exclude match_type", excludeMatchType),
		zap.Strings("exclude expressions", excludeExpressions),
		zap.Strings("exclude metric names", excludeMetricNames),
		zap.Any("exclude metrics with resource attributes", excludeResourceAttributes),
	)

	return fsp, nil
}

// processMetrics filters the given metrics based off the filterMetricProcessor's filters.
func (fmp *filterMetricProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	if fmp.skipResourceExpr == nil && fmp.skipMetricExpr == nil && fmp.skipDataPointExpr == nil {
		return md, nil
	}

	metricDataPointCountBeforeFilters := md.DataPointCount()

	var errors error
	md.ResourceMetrics().RemoveIf(func(rmetrics pmetric.ResourceMetrics) bool {
		resource := rmetrics.Resource()
		if fmp.skipResourceExpr != nil {
			skip, err := fmp.skipResourceExpr.Eval(ctx, ottlresource.NewTransformContext(resource))
			if err != nil {
				errors = multierr.Append(errors, err)
				return false
			}
			if skip {
				return true
			}
		}
		rmetrics.ScopeMetrics().RemoveIf(func(smetrics pmetric.ScopeMetrics) bool {
			scope := smetrics.Scope()
			smetrics.Metrics().RemoveIf(func(metric pmetric.Metric) bool {
				if fmp.skipMetricExpr != nil {
					skip, err := fmp.skipMetricExpr.Eval(ctx, ottlmetric.NewTransformContext(metric, smetrics.Metrics(), scope, resource))
					if err != nil {
						errors = multierr.Append(errors, err)
					}
					if skip {
						return true
					}
				}
				if fmp.skipDataPointExpr != nil {
					//exhaustive:enforce
					switch metric.Type() {
					case pmetric.MetricTypeSum:
						errors = multierr.Append(errors, fmp.handleNumberDataPoints(ctx, metric.Sum().DataPoints(), metric, smetrics.Metrics(), scope, resource))
						return metric.Sum().DataPoints().Len() == 0
					case pmetric.MetricTypeGauge:
						errors = multierr.Append(errors, fmp.handleNumberDataPoints(ctx, metric.Gauge().DataPoints(), metric, smetrics.Metrics(), scope, resource))
						return metric.Gauge().DataPoints().Len() == 0
					case pmetric.MetricTypeHistogram:
						errors = multierr.Append(errors, fmp.handleHistogramDataPoints(ctx, metric.Histogram().DataPoints(), metric, smetrics.Metrics(), scope, resource))
						return metric.Histogram().DataPoints().Len() == 0
					case pmetric.MetricTypeExponentialHistogram:
						errors = multierr.Append(errors, fmp.handleExponetialHistogramDataPoints(ctx, metric.ExponentialHistogram().DataPoints(), metric, smetrics.Metrics(), scope, resource))
						return metric.ExponentialHistogram().DataPoints().Len() == 0
					case pmetric.MetricTypeSummary:
						errors = multierr.Append(errors, fmp.handleSummaryDataPoints(ctx, metric.Summary().DataPoints(), metric, smetrics.Metrics(), scope, resource))
						return metric.Summary().DataPoints().Len() == 0
					default:
						return false
					}
				}
				return false
			})
			return smetrics.Metrics().Len() == 0
		})
		return rmetrics.ScopeMetrics().Len() == 0
	})

	metricDataPointCountAfterFilters := md.DataPointCount()
	fmp.telemetry.record(triggerMetricDataPointsDropped, int64(metricDataPointCountBeforeFilters-metricDataPointCountAfterFilters))

	if errors != nil {
		fmp.logger.Error("failed processing metrics", zap.Error(errors))
		return md, errors
	}
	if md.ResourceMetrics().Len() == 0 {
		return md, processorhelper.ErrSkipProcessingData
	}
	return md, nil
}

func newSkipResExpr(include *filterconfig.MetricMatchProperties, exclude *filterconfig.MetricMatchProperties) (expr.BoolExpr[ottlresource.TransformContext], error) {
	if filtermetric.UseOTTLBridge.IsEnabled() {
		mp := filterconfig.MatchConfig{}

		if include != nil {
			mp.Include = &filterconfig.MatchProperties{
				Config: filterset.Config{
					MatchType:    filterset.MatchType(include.MatchType),
					RegexpConfig: include.RegexpConfig,
				},
				Resources: include.ResourceAttributes,
			}
		}

		if exclude != nil {
			mp.Exclude = &filterconfig.MatchProperties{
				Config: filterset.Config{
					MatchType:    filterset.MatchType(exclude.MatchType),
					RegexpConfig: exclude.RegexpConfig,
				},
				Resources: exclude.ResourceAttributes,
			}
		}

		return filterottl.NewResourceSkipExprBridge(&mp)
	}

	var matchers []expr.BoolExpr[ottlresource.TransformContext]
	inclExpr, err := newResExpr(include)
	if err != nil {
		return nil, err
	}
	if inclExpr != nil {
		matchers = append(matchers, expr.Not(inclExpr))
	}
	exclExpr, err := newResExpr(exclude)
	if err != nil {
		return nil, err
	}
	if exclExpr != nil {
		matchers = append(matchers, exclExpr)
	}
	return expr.Or(matchers...), nil
}

type resExpr filtermatcher.AttributesMatcher

func (r resExpr) Eval(_ context.Context, tCtx ottlresource.TransformContext) (bool, error) {
	return filtermatcher.AttributesMatcher(r).Match(tCtx.GetResource().Attributes()), nil
}

func newResExpr(mp *filterconfig.MetricMatchProperties) (expr.BoolExpr[ottlresource.TransformContext], error) {
	if mp == nil {
		return nil, nil
	}
	attributeMatcher, err := filtermatcher.NewAttributesMatcher(
		filterset.Config{
			MatchType:    filterset.MatchType(mp.MatchType),
			RegexpConfig: mp.RegexpConfig,
		},
		mp.ResourceAttributes,
	)
	if err != nil {
		return nil, err
	}
	if attributeMatcher == nil {
		return nil, err
	}
	return resExpr(attributeMatcher), nil
}

func (fmp *filterMetricProcessor) handleNumberDataPoints(ctx context.Context, dps pmetric.NumberDataPointSlice, metric pmetric.Metric, metrics pmetric.MetricSlice, is pcommon.InstrumentationScope, resource pcommon.Resource) error {
	var errors error
	dps.RemoveIf(func(datapoint pmetric.NumberDataPoint) bool {
		skip, err := fmp.skipDataPointExpr.Eval(ctx, ottldatapoint.NewTransformContext(datapoint, metric, metrics, is, resource))
		if err != nil {
			errors = multierr.Append(errors, err)
			return false
		}
		return skip
	})
	return errors
}

func (fmp *filterMetricProcessor) handleHistogramDataPoints(ctx context.Context, dps pmetric.HistogramDataPointSlice, metric pmetric.Metric, metrics pmetric.MetricSlice, is pcommon.InstrumentationScope, resource pcommon.Resource) error {
	var errors error
	dps.RemoveIf(func(datapoint pmetric.HistogramDataPoint) bool {
		skip, err := fmp.skipDataPointExpr.Eval(ctx, ottldatapoint.NewTransformContext(datapoint, metric, metrics, is, resource))
		if err != nil {
			errors = multierr.Append(errors, err)
			return false
		}
		return skip
	})
	return errors
}

func (fmp *filterMetricProcessor) handleExponetialHistogramDataPoints(ctx context.Context, dps pmetric.ExponentialHistogramDataPointSlice, metric pmetric.Metric, metrics pmetric.MetricSlice, is pcommon.InstrumentationScope, resource pcommon.Resource) error {
	var errors error
	dps.RemoveIf(func(datapoint pmetric.ExponentialHistogramDataPoint) bool {
		skip, err := fmp.skipDataPointExpr.Eval(ctx, ottldatapoint.NewTransformContext(datapoint, metric, metrics, is, resource))
		if err != nil {
			errors = multierr.Append(errors, err)
			return false
		}
		return skip
	})
	return errors
}

func (fmp *filterMetricProcessor) handleSummaryDataPoints(ctx context.Context, dps pmetric.SummaryDataPointSlice, metric pmetric.Metric, metrics pmetric.MetricSlice, is pcommon.InstrumentationScope, resource pcommon.Resource) error {
	var errors error
	dps.RemoveIf(func(datapoint pmetric.SummaryDataPoint) bool {
		skip, err := fmp.skipDataPointExpr.Eval(ctx, ottldatapoint.NewTransformContext(datapoint, metric, metrics, is, resource))
		if err != nil {
			errors = multierr.Append(errors, err)
			return false
		}
		return skip
	})
	return errors
}
