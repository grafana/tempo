// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

const (
	modeKey   = "key"
	modeValue = "value"
)

type ReplaceAllPatternsArguments[K any] struct {
	Target            ottl.PMapGetSetter[K]
	Mode              string
	RegexPattern      string
	Replacement       ottl.StringGetter[K]
	Function          ottl.Optional[ottl.FunctionGetter[K]]
	ReplacementFormat ottl.Optional[ottl.StringGetter[K]]
}

func NewReplaceAllPatternsFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("replace_all_patterns", &ReplaceAllPatternsArguments[K]{}, createReplaceAllPatternsFunction[K])
}

func createReplaceAllPatternsFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ReplaceAllPatternsArguments[K])

	if !ok {
		return nil, errors.New("ReplaceAllPatternsFactory args must be of type *ReplaceAllPatternsArguments[K]")
	}

	return replaceAllPatterns(args.Target, args.Mode, args.RegexPattern, args.Replacement, args.Function, args.ReplacementFormat)
}

func replaceAllPatterns[K any](target ottl.PMapGetSetter[K], mode, regexPattern string, replacement ottl.StringGetter[K], fn ottl.Optional[ottl.FunctionGetter[K]], replacementFormat ottl.Optional[ottl.StringGetter[K]]) (ottl.ExprFunc[K], error) {
	compiledPattern, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("the regex pattern supplied to replace_all_patterns is not a valid pattern: %w", err)
	}
	if mode != modeValue && mode != modeKey {
		return nil, fmt.Errorf("invalid mode %v, must be either 'key' or 'value'", mode)
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		var replacementVal string
		replacementVal, err = replacement.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		switch mode {
		case modeValue:
			for _, value := range val.All() {
				if value.Type() != pcommon.ValueTypeStr || !compiledPattern.MatchString(value.Str()) {
					continue
				}
				if !fn.IsEmpty() {
					updatedString, err := applyOptReplaceFunction(ctx, tCtx, compiledPattern, fn, value.Str(), replacementVal, replacementFormat)
					if err != nil {
						continue
					}
					value.SetStr(updatedString)
				} else {
					value.SetStr(compiledPattern.ReplaceAllString(value.Str(), replacementVal))
				}
			}
		case modeKey:
			// Because we are changing the keys we cannot do in-place update, but we can move values to the
			// updated map and then move back the updated map to the initial map to avoid a copy in the target.Set,
			// because the pcommon.Map.CopyTo will not do a copy if it is the same object in this case val.
			updated := pcommon.NewMap()
			updated.EnsureCapacity(val.Len())
			for key, value := range val.All() {
				if !compiledPattern.MatchString(key) {
					value.MoveTo(updated.PutEmpty(key))
					continue
				}
				if !fn.IsEmpty() {
					updatedKey, err := applyOptReplaceFunction(ctx, tCtx, compiledPattern, fn, key, replacementVal, replacementFormat)
					if err != nil {
						continue
					}
					value.MoveTo(updated.PutEmpty(updatedKey))
				} else {
					value.MoveTo(updated.PutEmpty(compiledPattern.ReplaceAllString(key, replacementVal)))
				}
			}
			updated.MoveTo(val)
		}
		return nil, target.Set(ctx, tCtx, val)
	}, nil
}
