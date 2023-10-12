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
	Target      ottl.PMapGetter[K]   `ottlarg:"0"`
	Pattern     string               `ottlarg:"1"`
	Replacement ottl.StringGetter[K] `ottlarg:"2"`
}

func NewReplaceAllMatchesFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("replace_all_matches", &ReplaceAllMatchesArguments[K]{}, createReplaceAllMatchesFunction[K])
}

func createReplaceAllMatchesFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ReplaceAllMatchesArguments[K])

	if !ok {
		return nil, fmt.Errorf("ReplaceAllMatchesFactory args must be of type *ReplaceAllMatchesArguments[K]")
	}

	return replaceAllMatches(args.Target, args.Pattern, args.Replacement)
}

func replaceAllMatches[K any](target ottl.PMapGetter[K], pattern string, replacement ottl.StringGetter[K]) (ottl.ExprFunc[K], error) {
	glob, err := glob.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("the pattern supplied to replace_match is not a valid pattern: %w", err)
	}
	return func(ctx context.Context, tCtx K) (interface{}, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		replacementVal, err := replacement.Get(ctx, tCtx)
		if err != nil {
			return nil, err
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
