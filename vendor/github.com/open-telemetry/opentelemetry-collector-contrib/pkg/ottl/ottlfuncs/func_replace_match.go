// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"github.com/gobwas/glob"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ReplaceMatchArguments[K any] struct {
	Target      ottl.GetSetter[K]    `ottlarg:"0"`
	Pattern     string               `ottlarg:"1"`
	Replacement ottl.StringGetter[K] `ottlarg:"2"`
}

func NewReplaceMatchFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("replace_match", &ReplaceMatchArguments[K]{}, createReplaceMatchFunction[K])
}

func createReplaceMatchFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ReplaceMatchArguments[K])

	if !ok {
		return nil, fmt.Errorf("ReplaceMatchFactory args must be of type *ReplaceMatchArguments[K]")
	}

	return replaceMatch(args.Target, args.Pattern, args.Replacement)
}

func replaceMatch[K any](target ottl.GetSetter[K], pattern string, replacement ottl.StringGetter[K]) (ottl.ExprFunc[K], error) {
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
		if val == nil {
			return nil, nil
		}
		if valStr, ok := val.(string); ok {
			if glob.Match(valStr) {
				err = target.Set(ctx, tCtx, replacementVal)
				if err != nil {
					return nil, err
				}
			}
		}
		return nil, nil
	}, nil
}
