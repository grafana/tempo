// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package condition // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor/internal/condition"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.uber.org/multierr"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlresource"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlscope"
)

type LogsConsumer struct {
	resourceExpr expr.BoolExpr[*ottlresource.TransformContext]
	scopeExpr    expr.BoolExpr[*ottlscope.TransformContext]
	logExpr      expr.BoolExpr[*ottllog.TransformContext]
}

// parsedLogConditions is the type R for ParserCollection[R] that holds parsed OTTL conditions
type parsedLogConditions struct {
	resourceConditions []*ottl.Condition[*ottlresource.TransformContext]
	scopeConditions    []*ottl.Condition[*ottlscope.TransformContext]
	logConditions      []*ottl.Condition[*ottllog.TransformContext]
	telemetrySettings  component.TelemetrySettings
	errorMode          ottl.ErrorMode
}

func (lc LogsConsumer) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	var condErr error
	ld.ResourceLogs().RemoveIf(func(rlogs plog.ResourceLogs) bool {
		if lc.resourceExpr != nil {
			rCtx := ottlresource.NewTransformContextPtr(rlogs.Resource(), rlogs)
			rCond, rErr := lc.resourceExpr.Eval(ctx, rCtx)
			rCtx.Close()
			if rErr != nil {
				condErr = multierr.Append(condErr, rErr)
				return false
			}
			if rCond {
				return true
			}
		}

		if lc.scopeExpr == nil && lc.logExpr == nil {
			return rlogs.ScopeLogs().Len() == 0
		}

		rlogs.ScopeLogs().RemoveIf(func(slogs plog.ScopeLogs) bool {
			if lc.scopeExpr != nil {
				sCtx := ottlscope.NewTransformContextPtr(slogs.Scope(), rlogs.Resource(), slogs)
				sCond, sErr := lc.scopeExpr.Eval(ctx, sCtx)
				sCtx.Close()
				if sErr != nil {
					condErr = multierr.Append(condErr, sErr)
					return false
				}
				if sCond {
					return true
				}
			}

			if lc.logExpr != nil {
				slogs.LogRecords().RemoveIf(func(log plog.LogRecord) bool {
					tCtx := ottllog.NewTransformContextPtr(rlogs, slogs, log)
					cond, err := lc.logExpr.Eval(ctx, tCtx)
					tCtx.Close()
					if err != nil {
						condErr = multierr.Append(condErr, err)
						return false
					}
					return cond
				})
			}
			return slogs.LogRecords().Len() == 0
		})
		return rlogs.ScopeLogs().Len() == 0
	})

	if ld.ResourceLogs().Len() == 0 {
		return processorhelper.ErrSkipProcessingData
	}
	return condErr
}

func newLogConditionsFromResource(rc []*ottl.Condition[*ottlresource.TransformContext], telemetrySettings component.TelemetrySettings, errorMode ottl.ErrorMode) parsedLogConditions {
	return parsedLogConditions{
		resourceConditions: rc,
		telemetrySettings:  telemetrySettings,
		errorMode:          errorMode,
	}
}

func newLogConditionsFromScope(sc []*ottl.Condition[*ottlscope.TransformContext], telemetrySettings component.TelemetrySettings, errorMode ottl.ErrorMode) parsedLogConditions {
	return parsedLogConditions{
		scopeConditions:   sc,
		telemetrySettings: telemetrySettings,
		errorMode:         errorMode,
	}
}

func newLogsConsumer(lc *parsedLogConditions) LogsConsumer {
	var rExpr expr.BoolExpr[*ottlresource.TransformContext]
	var sExpr expr.BoolExpr[*ottlscope.TransformContext]
	var lExpr expr.BoolExpr[*ottllog.TransformContext]

	if len(lc.resourceConditions) > 0 {
		cs := ottlresource.NewConditionSequence(lc.resourceConditions, lc.telemetrySettings, ottlresource.WithConditionSequenceErrorMode(lc.errorMode))
		rExpr = &cs
	}

	if len(lc.scopeConditions) > 0 {
		cs := ottlscope.NewConditionSequence(lc.scopeConditions, lc.telemetrySettings, ottlscope.WithConditionSequenceErrorMode(lc.errorMode))
		sExpr = &cs
	}

	if len(lc.logConditions) > 0 {
		cs := ottllog.NewConditionSequence(lc.logConditions, lc.telemetrySettings, ottllog.WithConditionSequenceErrorMode(lc.errorMode))
		lExpr = &cs
	}

	return LogsConsumer{
		resourceExpr: rExpr,
		scopeExpr:    sExpr,
		logExpr:      lExpr,
	}
}

type LogParserCollection ottl.ParserCollection[parsedLogConditions]

type LogParserCollectionOption ottl.ParserCollectionOption[parsedLogConditions]

func WithLogParser(functions map[string]ottl.Factory[*ottllog.TransformContext]) LogParserCollectionOption {
	return func(pc *ottl.ParserCollection[parsedLogConditions]) error {
		logParser, err := ottllog.NewParser(functions, pc.Settings, ottllog.EnablePathContextNames())
		if err != nil {
			return err
		}
		return ottl.WithParserCollectionContext(ottllog.ContextName, &logParser, ottl.WithConditionConverter(convertLogConditions))(pc)
	}
}

func WithLogErrorMode(errorMode ottl.ErrorMode) LogParserCollectionOption {
	return LogParserCollectionOption(ottl.WithParserCollectionErrorMode[parsedLogConditions](errorMode))
}

func WithLogCommonParsers(functions map[string]ottl.Factory[*ottlresource.TransformContext]) LogParserCollectionOption {
	return LogParserCollectionOption(withCommonParsers(functions, newLogConditionsFromResource, newLogConditionsFromScope))
}

func NewLogParserCollection(settings component.TelemetrySettings, options ...LogParserCollectionOption) (*LogParserCollection, error) {
	pcOptions := []ottl.ParserCollectionOption[parsedLogConditions]{
		ottl.EnableParserCollectionModifiedPathsLogging[parsedLogConditions](true),
	}

	for _, option := range options {
		pcOptions = append(pcOptions, ottl.ParserCollectionOption[parsedLogConditions](option))
	}

	pc, err := ottl.NewParserCollection(settings, pcOptions...)
	if err != nil {
		return nil, err
	}

	lpc := LogParserCollection(*pc)
	return &lpc, nil
}

func convertLogConditions(pc *ottl.ParserCollection[parsedLogConditions], conditions ottl.ConditionsGetter, parsedConditions []*ottl.Condition[*ottllog.TransformContext]) (parsedLogConditions, error) {
	contextConditions, err := toContextConditions(conditions)
	if err != nil {
		return parsedLogConditions{}, err
	}

	errorMode := getErrorMode(pc, contextConditions)
	return parsedLogConditions{
		logConditions:     parsedConditions,
		telemetrySettings: pc.Settings,
		errorMode:         errorMode,
	}, nil
}

// ParseContextConditions parses the given ContextConditions and returns a LogsConsumer.
// For undefined context, each condition is parsed independently.
// Conditions are then grouped by their inferred context (resource, scope, log).
// The conditions group's error mode takes precedence over the processor-level error mode.
func (lpc *LogParserCollection) ParseContextConditions(contextConditions ContextConditions) (LogsConsumer, error) {
	pc := ottl.ParserCollection[parsedLogConditions](*lpc)

	if contextConditions.Context != "" {
		lc, err := pc.ParseConditionsWithContext(string(contextConditions.Context), contextConditions, true)
		if err != nil {
			return LogsConsumer{}, err
		}
		return newLogsConsumer(&lc), nil
	}

	var rConditions []*ottl.Condition[*ottlresource.TransformContext]
	var sConditions []*ottl.Condition[*ottlscope.TransformContext]
	var lConditions []*ottl.Condition[*ottllog.TransformContext]

	for _, cc := range contextConditions.GetConditions() {
		lc, err := pc.ParseConditions(ContextConditions{Conditions: []string{cc}})
		if err != nil {
			return LogsConsumer{}, err
		}

		if len(lc.resourceConditions) > 0 {
			rConditions = append(rConditions, lc.resourceConditions...)
		}
		if len(lc.scopeConditions) > 0 {
			sConditions = append(sConditions, lc.scopeConditions...)
		}
		if len(lc.logConditions) > 0 {
			lConditions = append(lConditions, lc.logConditions...)
		}
	}

	aggregatedConditions := parsedLogConditions{
		resourceConditions: rConditions,
		scopeConditions:    sConditions,
		logConditions:      lConditions,
		telemetrySettings:  pc.Settings,
		errorMode:          getErrorMode[parsedLogConditions](&pc, &contextConditions),
	}

	return newLogsConsumer(&aggregatedConditions), nil
}
