// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterconfig"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterlog"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlresource"
)

type filterLogProcessor struct {
	skipResourceExpr  expr.BoolExpr[ottlresource.TransformContext]
	skipLogRecordExpr expr.BoolExpr[ottllog.TransformContext]
	telemetry         *filterTelemetry
	logger            *zap.Logger
}

func newFilterLogsProcessor(set processor.Settings, cfg *Config) (*filterLogProcessor, error) {
	flp := &filterLogProcessor{
		logger: set.Logger,
	}

	fpt, err := newFilterTelemetry(set, pipeline.SignalLogs)
	if err != nil {
		return nil, fmt.Errorf("error creating filter processor telemetry: %w", err)
	}
	flp.telemetry = fpt

	if cfg.Logs.ResourceConditions != nil || cfg.Logs.LogConditions != nil {
		if cfg.Logs.ResourceConditions != nil {
			flp.skipResourceExpr, err = filterottl.NewBoolExprForResource(cfg.Logs.ResourceConditions, cfg.resourceFunctions, cfg.ErrorMode, set.TelemetrySettings)
			if err != nil {
				return nil, err
			}
		}

		if cfg.Logs.LogConditions != nil {
			flp.skipLogRecordExpr, err = filterottl.NewBoolExprForLog(cfg.Logs.LogConditions, cfg.logFunctions, cfg.ErrorMode, set.TelemetrySettings)
			if err != nil {
				return nil, err
			}
		}
		return flp, nil
	}

	cfgMatch := filterconfig.MatchConfig{}
	if cfg.Logs.Include != nil && !cfg.Logs.Include.isEmpty() {
		cfgMatch.Include = cfg.Logs.Include.matchProperties()
	}

	if cfg.Logs.Exclude != nil && !cfg.Logs.Exclude.isEmpty() {
		cfgMatch.Exclude = cfg.Logs.Exclude.matchProperties()
	}

	skipExpr, err := filterlog.NewSkipExpr(&cfgMatch)
	if err != nil {
		return nil, fmt.Errorf("failed to build skip matcher: %w", err)
	}
	flp.skipLogRecordExpr = skipExpr

	return flp, nil
}

func (flp *filterLogProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	if flp.skipResourceExpr == nil && flp.skipLogRecordExpr == nil {
		return ld, nil
	}

	logCountBeforeFilters := ld.LogRecordCount()

	var errors error
	ld.ResourceLogs().RemoveIf(func(rl plog.ResourceLogs) bool {
		resource := rl.Resource()
		if flp.skipResourceExpr != nil {
			skip, err := flp.skipResourceExpr.Eval(ctx, ottlresource.NewTransformContext(resource, rl))
			if err != nil {
				errors = multierr.Append(errors, err)
				return false
			}
			if skip {
				return true
			}
		}
		if flp.skipLogRecordExpr == nil {
			return rl.ScopeLogs().Len() == 0
		}
		rl.ScopeLogs().RemoveIf(func(sl plog.ScopeLogs) bool {
			scope := sl.Scope()
			lrs := sl.LogRecords()
			lrs.RemoveIf(func(lr plog.LogRecord) bool {
				skip, err := flp.skipLogRecordExpr.Eval(ctx, ottllog.NewTransformContext(lr, scope, resource, sl, rl))
				if err != nil {
					errors = multierr.Append(errors, err)
					return false
				}
				return skip
			})

			return sl.LogRecords().Len() == 0
		})
		return rl.ScopeLogs().Len() == 0
	})

	logCountAfterFilters := ld.LogRecordCount()
	flp.telemetry.record(ctx, int64(logCountBeforeFilters-logCountAfterFilters))

	if errors != nil {
		flp.logger.Error("failed processing logs", zap.Error(errors))
		return ld, errors
	}
	if ld.ResourceLogs().Len() == 0 {
		return ld, processorhelper.ErrSkipProcessingData
	}
	return ld, nil
}
