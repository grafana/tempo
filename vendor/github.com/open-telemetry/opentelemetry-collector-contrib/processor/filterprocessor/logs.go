// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterconfig"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterlog"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
)

type filterLogProcessor struct {
	skipExpr expr.BoolExpr[ottllog.TransformContext]
	logger   *zap.Logger
}

func newFilterLogsProcessor(set component.TelemetrySettings, cfg *Config) (*filterLogProcessor, error) {
	flp := &filterLogProcessor{
		logger: set.Logger,
	}
	if cfg.Logs.LogConditions != nil {
		skipExpr, err := filterottl.NewBoolExprForLog(cfg.Logs.LogConditions, filterottl.StandardLogFuncs(), cfg.ErrorMode, set)
		if err != nil {
			return nil, err
		}
		flp.skipExpr = skipExpr
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
	flp.skipExpr = skipExpr

	return flp, nil
}

func (flp *filterLogProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	if flp.skipExpr == nil {
		return ld, nil
	}

	var errors error
	ld.ResourceLogs().RemoveIf(func(rl plog.ResourceLogs) bool {
		resource := rl.Resource()
		rl.ScopeLogs().RemoveIf(func(sl plog.ScopeLogs) bool {
			scope := sl.Scope()
			lrs := sl.LogRecords()
			lrs.RemoveIf(func(lr plog.LogRecord) bool {
				skip, err := flp.skipExpr.Eval(ctx, ottllog.NewTransformContext(lr, scope, resource))
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

	if errors != nil {
		flp.logger.Error("failed processing logs", zap.Error(errors))
		return ld, errors
	}
	if ld.ResourceLogs().Len() == 0 {
		return ld, processorhelper.ErrSkipProcessingData
	}
	return ld, nil
}
