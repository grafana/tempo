// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"

import (
	"context"
	"errors"
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
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor/internal/condition"
)

type filterProfileProcessor struct {
	consumers        []condition.ProfilesConsumer
	skipResourceExpr expr.BoolExpr[*ottlresource.TransformContext]
	skipProfileExpr  expr.BoolExpr[*ottlprofile.TransformContext]
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

	if len(cfg.ProfileConditions) > 0 {
		pc, collectionErr := cfg.newProfileParserCollection(set.TelemetrySettings)
		if collectionErr != nil {
			return nil, collectionErr
		}
		var errs error
		for _, cs := range cfg.ProfileConditions {
			consumer, parseErr := pc.ParseContextConditions(cs)
			errs = multierr.Append(errs, parseErr)
			fpp.consumers = append(fpp.consumers, consumer)
		}
		if errs != nil {
			return nil, errs
		}
		return fpp, nil
	}

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
	if fpp.skipResourceExpr == nil && fpp.skipProfileExpr == nil && len(fpp.consumers) == 0 {
		return pd, nil
	}

	sampleCountBeforeFilters := pd.SampleCount()
	var processedProfiles pprofile.Profiles
	var errs error
	if len(fpp.consumers) > 0 {
		processedProfiles, errs = fpp.processConditions(ctx, pd)
	} else {
		processedProfiles, errs = fpp.processSkipExpression(ctx, pd)
	}

	sampleCountAfterFilters := processedProfiles.SampleCount()
	fpp.telemetry.record(ctx, int64(sampleCountBeforeFilters-sampleCountAfterFilters))

	if errs != nil && !errors.Is(errs, processorhelper.ErrSkipProcessingData) {
		fpp.logger.Error("failed processing profiles", zap.Error(errs))
		return processedProfiles, errs
	}

	if processedProfiles.ResourceProfiles().Len() == 0 {
		return processedProfiles, processorhelper.ErrSkipProcessingData
	}
	return processedProfiles, nil
}

func (fpp *filterProfileProcessor) processSkipExpression(ctx context.Context, pd pprofile.Profiles) (pprofile.Profiles, error) {
	dic := pd.Dictionary()
	var errs error
	pd.ResourceProfiles().RemoveIf(func(rp pprofile.ResourceProfiles) bool {
		resource := rp.Resource()
		if fpp.skipResourceExpr != nil {
			tCtx := ottlresource.NewTransformContextPtr(resource, rp)
			skip, err := fpp.skipResourceExpr.Eval(ctx, tCtx)
			tCtx.Close()
			if err != nil {
				errs = multierr.Append(errs, err)
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
			sp.Profiles().RemoveIf(func(profile pprofile.Profile) bool {
				tCtx := ottlprofile.NewTransformContextPtr(rp, sp, profile, dic)
				defer tCtx.Close()
				skip, err := fpp.skipProfileExpr.Eval(ctx, tCtx)
				if err != nil {
					errs = multierr.Append(errs, err)
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
	return pd, errs
}

func (fpp *filterProfileProcessor) processConditions(ctx context.Context, pd pprofile.Profiles) (pprofile.Profiles, error) {
	var errs error
	for _, consumer := range fpp.consumers {
		err := consumer.ConsumeProfiles(ctx, pd)
		if err != nil {
			errs = multierr.Append(errs, err)
		}
	}
	return pd, errs
}
