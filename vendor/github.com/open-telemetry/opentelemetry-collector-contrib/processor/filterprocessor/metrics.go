// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pipeline"
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
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor/internal/condition"
)

type filterMetricProcessor struct {
	consumers         []condition.MetricsConsumer
	skipResourceExpr  expr.BoolExpr[*ottlresource.TransformContext]
	skipMetricExpr    expr.BoolExpr[*ottlmetric.TransformContext]
	skipDataPointExpr expr.BoolExpr[*ottldatapoint.TransformContext]
	telemetry         *filterTelemetry
	logger            *zap.Logger
}

func newFilterMetricProcessor(set processor.Settings, cfg *Config) (*filterMetricProcessor, error) {
	var err error
	fsp := &filterMetricProcessor{
		logger: set.Logger,
	}

	fpt, err := newFilterTelemetry(set, pipeline.SignalMetrics)
	if err != nil {
		return nil, fmt.Errorf("error creating filter processor telemetry: %w", err)
	}
	fsp.telemetry = fpt

	if len(cfg.MetricConditions) > 0 {
		pc, collectionErr := cfg.newMetricParserCollection(set.TelemetrySettings)
		if collectionErr != nil {
			return nil, collectionErr
		}
		var errs error
		for _, cs := range cfg.MetricConditions {
			consumer, parseErr := pc.ParseContextConditions(cs)
			errs = multierr.Append(errs, parseErr)
			fsp.consumers = append(fsp.consumers, consumer)
		}
		if errs != nil {
			return nil, errs
		}
		return fsp, nil
	}

	if cfg.Metrics.ResourceConditions != nil || cfg.Metrics.MetricConditions != nil || cfg.Metrics.DataPointConditions != nil {
		if cfg.Metrics.ResourceConditions != nil {
			fsp.skipResourceExpr, err = filterottl.NewBoolExprForResource(cfg.Metrics.ResourceConditions, cfg.resourceFunctions, cfg.ErrorMode, set.TelemetrySettings)
			if err != nil {
				return nil, err
			}
		}

		if cfg.Metrics.MetricConditions != nil {
			fsp.skipMetricExpr, err = filterottl.NewBoolExprForMetric(cfg.Metrics.MetricConditions, cfg.metricFunctions, cfg.ErrorMode, set.TelemetrySettings)
			if err != nil {
				return nil, err
			}
		}

		if cfg.Metrics.DataPointConditions != nil {
			fsp.skipDataPointExpr, err = filterottl.NewBoolExprForDataPoint(cfg.Metrics.DataPointConditions, cfg.dataPointFunctions, cfg.ErrorMode, set.TelemetrySettings)
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
	if fmp.skipResourceExpr == nil && fmp.skipMetricExpr == nil && fmp.skipDataPointExpr == nil && len(fmp.consumers) == 0 {
		return md, nil
	}

	metricDataPointCountBeforeFilters := md.DataPointCount()

	var errs error
	var processedMetrics pmetric.Metrics
	if len(fmp.consumers) > 0 {
		processedMetrics, errs = fmp.processConditions(ctx, md)
	} else {
		processedMetrics, errs = fmp.processSkipExpression(ctx, md)
	}

	metricDataPointCountAfterFilters := processedMetrics.DataPointCount()
	fmp.telemetry.record(ctx, int64(metricDataPointCountBeforeFilters-metricDataPointCountAfterFilters))

	if errs != nil && !errors.Is(errs, processorhelper.ErrSkipProcessingData) {
		fmp.logger.Error("failed processing metrics", zap.Error(errs))
		return processedMetrics, errs
	}

	if processedMetrics.ResourceMetrics().Len() == 0 {
		return processedMetrics, processorhelper.ErrSkipProcessingData
	}
	return processedMetrics, nil
}

func (fmp *filterMetricProcessor) processConditions(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	var errs error
	for _, consumer := range fmp.consumers {
		err := consumer.ConsumeMetrics(ctx, md)
		if err != nil {
			errs = multierr.Append(errs, err)
		}
	}
	return md, errs
}

func (fmp *filterMetricProcessor) processSkipExpression(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	var errs error
	md.ResourceMetrics().RemoveIf(func(rm pmetric.ResourceMetrics) bool {
		if fmp.skipResourceExpr != nil {
			tCtx := ottlresource.NewTransformContextPtr(rm.Resource(), rm)
			skip, err := fmp.skipResourceExpr.Eval(ctx, tCtx)
			tCtx.Close()
			if err != nil {
				errs = multierr.Append(errs, err)
				return false
			}
			if skip {
				return true
			}
		}
		if fmp.skipMetricExpr == nil && fmp.skipDataPointExpr == nil {
			return rm.ScopeMetrics().Len() == 0
		}
		rm.ScopeMetrics().RemoveIf(func(smetrics pmetric.ScopeMetrics) bool {
			smetrics.Metrics().RemoveIf(func(metric pmetric.Metric) bool {
				if fmp.skipMetricExpr != nil {
					tCtx := ottlmetric.NewTransformContextPtr(rm, smetrics, metric)
					skip, err := fmp.skipMetricExpr.Eval(ctx, tCtx)
					tCtx.Close()
					if err != nil {
						errs = multierr.Append(errs, err)
						return false
					}
					if skip {
						return true
					}
				}
				if fmp.skipDataPointExpr != nil {
					//exhaustive:enforce
					switch metric.Type() {
					case pmetric.MetricTypeSum:
						errs = multierr.Append(errs, fmp.handleNumberDataPoints(ctx, rm, smetrics, metric, metric.Sum().DataPoints()))
						return metric.Sum().DataPoints().Len() == 0
					case pmetric.MetricTypeGauge:
						errs = multierr.Append(errs, fmp.handleNumberDataPoints(ctx, rm, smetrics, metric, metric.Gauge().DataPoints()))
						return metric.Gauge().DataPoints().Len() == 0
					case pmetric.MetricTypeHistogram:
						errs = multierr.Append(errs, fmp.handleHistogramDataPoints(ctx, rm, smetrics, metric, metric.Histogram().DataPoints()))
						return metric.Histogram().DataPoints().Len() == 0
					case pmetric.MetricTypeExponentialHistogram:
						errs = multierr.Append(errs, fmp.handleExponentialHistogramDataPoints(ctx, rm, smetrics, metric, metric.ExponentialHistogram().DataPoints()))
						return metric.ExponentialHistogram().DataPoints().Len() == 0
					case pmetric.MetricTypeSummary:
						errs = multierr.Append(errs, fmp.handleSummaryDataPoints(ctx, rm, smetrics, metric, metric.Summary().DataPoints()))
						return metric.Summary().DataPoints().Len() == 0
					default:
						return false
					}
				}
				return false
			})
			return smetrics.Metrics().Len() == 0
		})
		return rm.ScopeMetrics().Len() == 0
	})
	return md, errs
}

func newSkipResExpr(include, exclude *filterconfig.MetricMatchProperties) (expr.BoolExpr[*ottlresource.TransformContext], error) {
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

	var matchers []expr.BoolExpr[*ottlresource.TransformContext]
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

func (r resExpr) Eval(_ context.Context, tCtx *ottlresource.TransformContext) (bool, error) {
	return filtermatcher.AttributesMatcher(r).Match(tCtx.GetResource().Attributes()), nil
}

func newResExpr(mp *filterconfig.MetricMatchProperties) (expr.BoolExpr[*ottlresource.TransformContext], error) {
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

func (fmp *filterMetricProcessor) handleNumberDataPoints(ctx context.Context, rm pmetric.ResourceMetrics, sm pmetric.ScopeMetrics, m pmetric.Metric, dps pmetric.NumberDataPointSlice) error {
	var errs error
	dps.RemoveIf(func(dp pmetric.NumberDataPoint) bool {
		tCtx := ottldatapoint.NewTransformContextPtr(rm, sm, m, dp)
		defer tCtx.Close()
		skip, err := fmp.skipDataPointExpr.Eval(ctx, tCtx)
		if err != nil {
			errs = multierr.Append(errs, err)
			return false
		}
		return skip
	})
	return errs
}

func (fmp *filterMetricProcessor) handleHistogramDataPoints(ctx context.Context, rm pmetric.ResourceMetrics, sm pmetric.ScopeMetrics, m pmetric.Metric, dps pmetric.HistogramDataPointSlice) error {
	var errs error
	dps.RemoveIf(func(dp pmetric.HistogramDataPoint) bool {
		tCtx := ottldatapoint.NewTransformContextPtr(rm, sm, m, dp)
		defer tCtx.Close()
		skip, err := fmp.skipDataPointExpr.Eval(ctx, tCtx)
		if err != nil {
			errs = multierr.Append(errs, err)
			return false
		}
		return skip
	})
	return errs
}

func (fmp *filterMetricProcessor) handleExponentialHistogramDataPoints(ctx context.Context, rm pmetric.ResourceMetrics, sm pmetric.ScopeMetrics, m pmetric.Metric, dps pmetric.ExponentialHistogramDataPointSlice) error {
	var errs error
	dps.RemoveIf(func(dp pmetric.ExponentialHistogramDataPoint) bool {
		tCtx := ottldatapoint.NewTransformContextPtr(rm, sm, m, dp)
		defer tCtx.Close()
		skip, err := fmp.skipDataPointExpr.Eval(ctx, tCtx)
		if err != nil {
			errs = multierr.Append(errs, err)
			return false
		}
		return skip
	})
	return errs
}

func (fmp *filterMetricProcessor) handleSummaryDataPoints(ctx context.Context, rm pmetric.ResourceMetrics, sm pmetric.ScopeMetrics, m pmetric.Metric, dps pmetric.SummaryDataPointSlice) error {
	var errs error
	dps.RemoveIf(func(dp pmetric.SummaryDataPoint) bool {
		tCtx := ottldatapoint.NewTransformContextPtr(rm, sm, m, dp)
		defer tCtx.Close()
		skip, err := fmp.skipDataPointExpr.Eval(ctx, tCtx)
		if err != nil {
			errs = multierr.Append(errs, err)
			return false
		}
		return skip
	})
	return errs
}
