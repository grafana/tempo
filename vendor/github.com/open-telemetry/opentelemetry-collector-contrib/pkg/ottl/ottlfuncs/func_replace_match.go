// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"

	"github.com/gobwas/glob"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ReplaceMatchArguments[K any] struct {
	Target            ottl.GetSetter[K]
	Pattern           ottl.StringGetter[K]
	Replacement       ottl.StringGetter[K]
	Function          ottl.Optional[ottl.FunctionGetter[K]]
	ReplacementFormat ottl.Optional[ottl.StringGetter[K]]
}

type replaceMatchFuncArgs[K any] struct {
	Input ottl.StringGetter[K]
}

func NewReplaceMatchFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("replace_match", &ReplaceMatchArguments[K]{}, createReplaceMatchFunction[K])
}

func createReplaceMatchFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ReplaceMatchArguments[K])

	if !ok {
		return nil, errors.New("ReplaceMatchFactory args must be of type *ReplaceMatchArguments[K]")
	}

	return replaceMatch(args.Target, args.Pattern, args.Replacement, args.Function, args.ReplacementFormat)
}

func replaceMatch[K any](target ottl.GetSetter[K], pattern, replacement ottl.StringGetter[K], fn ottl.Optional[ottl.FunctionGetter[K]], replacementFormat ottl.Optional[ottl.StringGetter[K]]) (ottl.ExprFunc[K], error) {
	literalPattern, ok := ottl.GetLiteralValue(pattern)
	var compiledPattern glob.Glob
	var err error
	if ok {
		compiledPattern, err = glob.Compile(literalPattern)
		if err != nil {
			return nil, fmt.Errorf(invalidRegexErrMsg, "replace_match", literalPattern, err)
		}
	}
	return func(ctx context.Context, tCtx K) (any, error) {
		cp := compiledPattern
		if cp == nil {
			patternVal, err := pattern.Get(ctx, tCtx)
			if err != nil {
				return nil, err
			}
			cp, err = glob.Compile(patternVal)
			if err != nil {
				return nil, fmt.Errorf(invalidRegexErrMsg, "replace_all_matches", patternVal, err)
			}
		}
		val, err := target.Get(ctx, tCtx)
		var replacementVal string
		if err != nil {
			return nil, err
		}
		if fn.IsEmpty() {
			replacementVal, err = replacement.Get(ctx, tCtx)
			if err != nil {
				return nil, err
			}
		} else {
			fnVal := fn.Get()
			replacementExpr, errNew := fnVal.Get(&replaceMatchFuncArgs[K]{Input: replacement})
			if errNew != nil {
				return nil, errNew
			}
			replacementValRaw, errNew := replacementExpr.Eval(ctx, tCtx)
			if errNew != nil {
				return nil, errNew
			}
			replacementValStr, ok := replacementValRaw.(string)
			if !ok {
				return nil, errors.New("replacement value is not a string")
			}
			replacementVal, err = applyReplaceFormat(ctx, tCtx, replacementFormat, replacementValStr)
			if err != nil {
				return nil, err
			}
		}
		if err != nil {
			return nil, err
		}
		if val == nil {
			return nil, nil
		}
		if valStr, ok := val.(string); ok {
			if cp.Match(valStr) {
				err = target.Set(ctx, tCtx, replacementVal)
				if err != nil {
					return nil, err
				}
			}
		}
		return nil, nil
	}, nil
}
