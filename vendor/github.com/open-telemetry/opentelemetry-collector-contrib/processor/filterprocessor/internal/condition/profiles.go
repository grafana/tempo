// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package condition // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor/internal/condition"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.uber.org/multierr"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlprofile"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlresource"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlscope"
)

type ProfilesConsumer struct {
	resourceExpr expr.BoolExpr[*ottlresource.TransformContext]
	scopeExpr    expr.BoolExpr[*ottlscope.TransformContext]
	profileExpr  expr.BoolExpr[*ottlprofile.TransformContext]
}

// parsedProfileConditions is the type R for ParserCollection[R] that holds parsed OTTL conditions
type parsedProfileConditions struct {
	resourceConditions []*ottl.Condition[*ottlresource.TransformContext]
	scopeConditions    []*ottl.Condition[*ottlscope.TransformContext]
	profileConditions  []*ottl.Condition[*ottlprofile.TransformContext]
	telemetrySettings  component.TelemetrySettings
	errorMode          ottl.ErrorMode
}

func (pc ProfilesConsumer) ConsumeProfiles(ctx context.Context, pd pprofile.Profiles) error {
	var condErr error
	pd.ResourceProfiles().RemoveIf(func(rp pprofile.ResourceProfiles) bool {
		if pc.resourceExpr != nil {
			rCtx := ottlresource.NewTransformContextPtr(rp.Resource(), rp)
			rCond, err := pc.resourceExpr.Eval(ctx, rCtx)
			rCtx.Close()
			if err != nil {
				condErr = multierr.Append(condErr, err)
				return false
			}
			if rCond {
				return true
			}
		}

		if pc.scopeExpr == nil && pc.profileExpr == nil {
			return rp.ScopeProfiles().Len() == 0
		}

		rp.ScopeProfiles().RemoveIf(func(sp pprofile.ScopeProfiles) bool {
			if pc.scopeExpr != nil {
				sCtx := ottlscope.NewTransformContextPtr(sp.Scope(), rp.Resource(), sp)
				sCond, err := pc.scopeExpr.Eval(ctx, sCtx)
				sCtx.Close()
				if err != nil {
					condErr = multierr.Append(condErr, err)
					return false
				}
				if sCond {
					return true
				}
			}

			if pc.profileExpr != nil {
				sp.Profiles().RemoveIf(func(profile pprofile.Profile) bool {
					tCtx := ottlprofile.NewTransformContextPtr(rp, sp, profile, pd.Dictionary())
					cond, err := pc.profileExpr.Eval(ctx, tCtx)
					tCtx.Close()
					if err != nil {
						condErr = multierr.Append(condErr, err)
						return false
					}
					return cond
				})
			}
			return sp.Profiles().Len() == 0
		})
		return rp.ScopeProfiles().Len() == 0
	})

	if pd.ResourceProfiles().Len() == 0 {
		return processorhelper.ErrSkipProcessingData
	}
	return condErr
}

func newProfileConditionsFromResource(rc []*ottl.Condition[*ottlresource.TransformContext], telemetrySettings component.TelemetrySettings, errorMode ottl.ErrorMode) parsedProfileConditions {
	return parsedProfileConditions{
		resourceConditions: rc,
		telemetrySettings:  telemetrySettings,
		errorMode:          errorMode,
	}
}

func newProfileConditionsFromScope(sc []*ottl.Condition[*ottlscope.TransformContext], telemetrySettings component.TelemetrySettings, errorMode ottl.ErrorMode) parsedProfileConditions {
	return parsedProfileConditions{
		scopeConditions:   sc,
		telemetrySettings: telemetrySettings,
		errorMode:         errorMode,
	}
}

func newProfilesConsumer(ppc *parsedProfileConditions) ProfilesConsumer {
	var rExpr expr.BoolExpr[*ottlresource.TransformContext]
	var sExpr expr.BoolExpr[*ottlscope.TransformContext]
	var pExpr expr.BoolExpr[*ottlprofile.TransformContext]

	if len(ppc.resourceConditions) > 0 {
		cs := ottlresource.NewConditionSequence(ppc.resourceConditions, ppc.telemetrySettings, ottlresource.WithConditionSequenceErrorMode(ppc.errorMode))
		rExpr = &cs
	}

	if len(ppc.scopeConditions) > 0 {
		cs := ottlscope.NewConditionSequence(ppc.scopeConditions, ppc.telemetrySettings, ottlscope.WithConditionSequenceErrorMode(ppc.errorMode))
		sExpr = &cs
	}

	if len(ppc.profileConditions) > 0 {
		cs := ottlprofile.NewConditionSequence(ppc.profileConditions, ppc.telemetrySettings, ottlprofile.WithConditionSequenceErrorMode(ppc.errorMode))
		pExpr = &cs
	}

	return ProfilesConsumer{
		resourceExpr: rExpr,
		scopeExpr:    sExpr,
		profileExpr:  pExpr,
	}
}

type ProfileParserCollection ottl.ParserCollection[parsedProfileConditions]

type ProfileParserCollectionOption ottl.ParserCollectionOption[parsedProfileConditions]

func WithProfileParser(functions map[string]ottl.Factory[*ottlprofile.TransformContext]) ProfileParserCollectionOption {
	return func(pc *ottl.ParserCollection[parsedProfileConditions]) error {
		profileParser, err := ottlprofile.NewParser(functions, pc.Settings, ottlprofile.EnablePathContextNames())
		if err != nil {
			return err
		}
		return ottl.WithParserCollectionContext(ottlprofile.ContextName, &profileParser, ottl.WithConditionConverter(convertProfileConditions))(pc)
	}
}

func WithProfileErrorMode(errorMode ottl.ErrorMode) ProfileParserCollectionOption {
	return ProfileParserCollectionOption(ottl.WithParserCollectionErrorMode[parsedProfileConditions](errorMode))
}

func WithProfileCommonParsers(functions map[string]ottl.Factory[*ottlresource.TransformContext]) ProfileParserCollectionOption {
	return ProfileParserCollectionOption(withCommonParsers(functions, newProfileConditionsFromResource, newProfileConditionsFromScope))
}

func NewProfileParserCollection(settings component.TelemetrySettings, options ...ProfileParserCollectionOption) (*ProfileParserCollection, error) {
	pcOptions := []ottl.ParserCollectionOption[parsedProfileConditions]{
		ottl.EnableParserCollectionModifiedPathsLogging[parsedProfileConditions](true),
	}

	for _, option := range options {
		pcOptions = append(pcOptions, ottl.ParserCollectionOption[parsedProfileConditions](option))
	}

	pc, err := ottl.NewParserCollection(settings, pcOptions...)
	if err != nil {
		return nil, err
	}

	ppc := ProfileParserCollection(*pc)
	return &ppc, nil
}

func convertProfileConditions(pc *ottl.ParserCollection[parsedProfileConditions], conditions ottl.ConditionsGetter, parsedConditions []*ottl.Condition[*ottlprofile.TransformContext]) (parsedProfileConditions, error) {
	contextConditions, err := toContextConditions(conditions)
	if err != nil {
		return parsedProfileConditions{}, err
	}

	errorMode := getErrorMode(pc, contextConditions)
	return parsedProfileConditions{
		profileConditions: parsedConditions,
		telemetrySettings: pc.Settings,
		errorMode:         errorMode,
	}, nil
}

func (ppc *ProfileParserCollection) ParseContextConditions(contextConditions ContextConditions) (ProfilesConsumer, error) {
	pc := ottl.ParserCollection[parsedProfileConditions](*ppc)
	if contextConditions.Context != "" {
		pConditions, err := pc.ParseConditionsWithContext(string(contextConditions.Context), contextConditions, true)
		if err != nil {
			return ProfilesConsumer{}, err
		}
		return newProfilesConsumer(&pConditions), nil
	}

	var rConditions []*ottl.Condition[*ottlresource.TransformContext]
	var sConditions []*ottl.Condition[*ottlscope.TransformContext]
	var pConditions []*ottl.Condition[*ottlprofile.TransformContext]

	for _, cc := range contextConditions.Conditions {
		profConditions, err := pc.ParseConditions(ContextConditions{Conditions: []string{cc}})
		if err != nil {
			return ProfilesConsumer{}, err
		}

		if len(profConditions.resourceConditions) > 0 {
			rConditions = append(rConditions, profConditions.resourceConditions...)
		}
		if len(profConditions.scopeConditions) > 0 {
			sConditions = append(sConditions, profConditions.scopeConditions...)
		}
		if len(profConditions.profileConditions) > 0 {
			pConditions = append(pConditions, profConditions.profileConditions...)
		}
	}

	aggregatedConditions := parsedProfileConditions{
		resourceConditions: rConditions,
		scopeConditions:    sConditions,
		profileConditions:  pConditions,
		telemetrySettings:  pc.Settings,
		errorMode:          getErrorMode[parsedProfileConditions](&pc, &contextConditions),
	}

	return newProfilesConsumer(&aggregatedConditions), nil
}
