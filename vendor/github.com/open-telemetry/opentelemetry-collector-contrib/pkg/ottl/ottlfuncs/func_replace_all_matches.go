// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"github.com/gobwas/glob"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ReplaceAllMatchesArguments[K any] struct {
	Target      ottl.PMapGetter[K]
	Pattern     string
	Replacement ottl.StringGetter[K]
	Function    ottl.Optional[ottl.FunctionGetter[K]]
}

type replaceAllMatchesFuncArgs[K any] struct {
	Input ottl.StringGetter[K]
}

func NewReplaceAllMatchesFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("replace_all_matches", &ReplaceAllMatchesArguments[K]{}, createReplaceAllMatchesFunction[K])
}

func createReplaceAllMatchesFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ReplaceAllMatchesArguments[K])

	if !ok {
		return nil, fmt.Errorf("ReplaceAllMatchesFactory args must be of type *ReplaceAllMatchesArguments[K]")
	}

	return replaceAllMatches(args.Target, args.Pattern, args.Replacement, args.Function)
}

func replaceAllMatches[K any](target ottl.PMapGetter[K], pattern string, replacement ottl.StringGetter[K], fn ottl.Optional[ottl.FunctionGetter[K]]) (ottl.ExprFunc[K], error) {
	glob, err := glob.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("the pattern supplied to replace_match is not a valid pattern: %w", err)
	}
	return func(ctx context.Context, tCtx K) (any, error) {
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
			replacementExpr, errNew := fnVal.Get(&replaceAllMatchesFuncArgs[K]{Input: replacement})
			if errNew != nil {
				return nil, errNew
			}
			replacementValRaw, errNew := replacementExpr.Eval(ctx, tCtx)
			if errNew != nil {
				return nil, errNew
			}
			replacementValStr, ok := replacementValRaw.(string)
			if !ok {
				return nil, fmt.Errorf("replacement value is not a string")
			}
			replacementVal = replacementValStr
		}
		val.Range(func(key string, value pcommon.Value) bool {
			if glob.Match(value.Str()) {
				value.SetStr(replacementVal)
			}
			return true
		})
		return nil, nil
	}, nil
}
