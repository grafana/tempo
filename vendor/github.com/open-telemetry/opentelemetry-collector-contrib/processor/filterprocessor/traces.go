// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspanevent"
)

type filterSpanProcessor struct {
	skipSpanExpr      expr.BoolExpr[ottlspan.TransformContext]
	skipSpanEventExpr expr.BoolExpr[ottlspanevent.TransformContext]
	telemetry         *filterProcessorTelemetry
	logger            *zap.Logger
}

func newFilterSpansProcessor(set processor.CreateSettings, cfg *Config) (*filterSpanProcessor, error) {
	var err error
	fsp := &filterSpanProcessor{
		logger: set.Logger,
	}

	fpt, err := newfilterProcessorTelemetry(set)
	if err != nil {
		return nil, fmt.Errorf("error creating filter processor telemetry: %w", err)
	}
	fsp.telemetry = fpt

	if cfg.Traces.SpanConditions != nil || cfg.Traces.SpanEventConditions != nil {
		if cfg.Traces.SpanConditions != nil {
			fsp.skipSpanExpr, err = filterottl.NewBoolExprForSpan(cfg.Traces.SpanConditions, filterottl.StandardSpanFuncs(), cfg.ErrorMode, set.TelemetrySettings)
			if err != nil {
				return nil, err
			}
		}
		if cfg.Traces.SpanEventConditions != nil {
			fsp.skipSpanEventExpr, err = filterottl.NewBoolExprForSpanEvent(cfg.Traces.SpanEventConditions, filterottl.StandardSpanEventFuncs(), cfg.ErrorMode, set.TelemetrySettings)
			if err != nil {
				return nil, err
			}
		}
		return fsp, nil
	}

	fsp.skipSpanExpr, err = filterspan.NewSkipExpr(&cfg.Spans)
	if err != nil {
		return nil, err
	}

	includeMatchType, excludeMatchType := "[None]", "[None]"
	if cfg.Spans.Include != nil {
		includeMatchType = string(cfg.Spans.Include.MatchType)
	}

	if cfg.Spans.Exclude != nil {
		excludeMatchType = string(cfg.Spans.Exclude.MatchType)
	}

	set.Logger.Info(
		"Span filter configured",
		zap.String("[Include] match_type", includeMatchType),
		zap.String("[Exclude] match_type", excludeMatchType),
	)

	return fsp, nil
}

// processTraces filters the given spans of a traces based off the filterSpanProcessor's filters.
func (fsp *filterSpanProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	if fsp.skipSpanExpr == nil && fsp.skipSpanEventExpr == nil {
		return td, nil
	}

	spanCountBeforeFilters := td.SpanCount()

	var errors error
	td.ResourceSpans().RemoveIf(func(rs ptrace.ResourceSpans) bool {
		resource := rs.Resource()
		rs.ScopeSpans().RemoveIf(func(ss ptrace.ScopeSpans) bool {
			scope := ss.Scope()
			ss.Spans().RemoveIf(func(span ptrace.Span) bool {
				if fsp.skipSpanExpr != nil {
					skip, err := fsp.skipSpanExpr.Eval(ctx, ottlspan.NewTransformContext(span, scope, resource))
					if err != nil {
						errors = multierr.Append(errors, err)
						return false
					}
					if skip {
						return true
					}
				}
				if fsp.skipSpanEventExpr != nil {
					span.Events().RemoveIf(func(spanEvent ptrace.SpanEvent) bool {
						skip, err := fsp.skipSpanEventExpr.Eval(ctx, ottlspanevent.NewTransformContext(spanEvent, span, scope, resource))
						if err != nil {
							errors = multierr.Append(errors, err)
							return false
						}
						return skip
					})
				}
				return false
			})
			return ss.Spans().Len() == 0
		})
		return rs.ScopeSpans().Len() == 0
	})

	spanCountAfterFilters := td.SpanCount()
	fsp.telemetry.record(triggerSpansDropped, int64(spanCountBeforeFilters-spanCountAfterFilters))

	if errors != nil {
		fsp.logger.Error("failed processing traces", zap.Error(errors))
		return td, errors
	}
	if td.ResourceSpans().Len() == 0 {
		return td, processorhelper.ErrSkipProcessingData
	}
	return td, nil
}
