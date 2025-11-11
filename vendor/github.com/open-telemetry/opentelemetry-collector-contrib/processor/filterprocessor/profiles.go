// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlprofile"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlresource"
)

type filterProfileProcessor struct {
	skipResourceExpr expr.BoolExpr[ottlresource.TransformContext]
	skipProfileExpr  expr.BoolExpr[ottlprofile.TransformContext]
	telemetry        *filterTelemetry
	logger           *zap.Logger
}

func newFilterProfilesProcessor(set processor.Settings, cfg *Config) (*filterProfileProcessor, error) {
	fpp := &filterProfileProcessor{
		logger: set.Logger,
	}

	fpt, err := newFilterTelemetry(set, xpipeline.SignalProfiles)
	if err != nil {
		return nil, fmt.Errorf("error creating filter processor telemetry: %w", err)
	}
	fpp.telemetry = fpt

	if cfg.Profiles.ResourceConditions != nil {
		fpp.skipResourceExpr, err = filterottl.NewBoolExprForResource(cfg.Profiles.ResourceConditions, cfg.resourceFunctions, cfg.ErrorMode, set.TelemetrySettings)
		if err != nil {
			return nil, err
		}
	}

	if cfg.Profiles.ProfileConditions != nil {
		fpp.skipProfileExpr, err = filterottl.NewBoolExprForProfile(cfg.Profiles.ProfileConditions, cfg.profileFunctions, cfg.ErrorMode, set.TelemetrySettings)
		if err != nil {
			return nil, err
		}
	}

	return fpp, nil
}

// processProfiles filters the given profile based off the filterSampleProcessor's filters.
func (fpp *filterProfileProcessor) processProfiles(ctx context.Context, pd pprofile.Profiles) (pprofile.Profiles, error) {
	if fpp.skipResourceExpr == nil && fpp.skipProfileExpr == nil {
		return pd, nil
	}

	sampleCountBeforeFilters := pd.SampleCount()
	dic := pd.Dictionary()

	var errors error
	pd.ResourceProfiles().RemoveIf(func(rp pprofile.ResourceProfiles) bool {
		resource := rp.Resource()
		if fpp.skipResourceExpr != nil {
			skip, err := fpp.skipResourceExpr.Eval(ctx, ottlresource.NewTransformContext(resource, rp))
			if err != nil {
				errors = multierr.Append(errors, err)
				return false
			}
			if skip {
				return true
			}
		}
		if fpp.skipProfileExpr == nil {
			return rp.ScopeProfiles().Len() == 0
		}
		rp.ScopeProfiles().RemoveIf(func(sp pprofile.ScopeProfiles) bool {
			scope := sp.Scope()
			sp.Profiles().RemoveIf(func(profile pprofile.Profile) bool {
				skip, err := fpp.skipProfileExpr.Eval(ctx, ottlprofile.NewTransformContext(profile, dic, scope, resource, sp, rp))
				if err != nil {
					errors = multierr.Append(errors, err)
					return false
				}
				if skip {
					return true
				}
				return false
			})
			return sp.Profiles().Len() == 0
		})
		return rp.ScopeProfiles().Len() == 0
	})

	sampleCountAfterFilters := pd.SampleCount()
	fpp.telemetry.record(ctx, int64(sampleCountBeforeFilters-sampleCountAfterFilters))

	if errors != nil {
		fpp.logger.Error("failed processing profiles", zap.Error(errors))
		return pd, errors
	}
	if pd.ResourceProfiles().Len() == 0 {
		return pd, processorhelper.ErrSkipProcessingData
	}
	return pd, nil
}
