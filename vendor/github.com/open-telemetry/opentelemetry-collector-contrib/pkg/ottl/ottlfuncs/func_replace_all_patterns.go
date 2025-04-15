// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
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
	Target            ottl.PMapGetter[K]
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
		return nil, fmt.Errorf("ReplaceAllPatternsFactory args must be of type *ReplaceAllPatternsArguments[K]")
	}

	return replaceAllPatterns(args.Target, args.Mode, args.RegexPattern, args.Replacement, args.Function, args.ReplacementFormat)
}

func replaceAllPatterns[K any](target ottl.PMapGetter[K], mode string, regexPattern string, replacement ottl.StringGetter[K], fn ottl.Optional[ottl.FunctionGetter[K]], replacementFormat ottl.Optional[ottl.StringGetter[K]]) (ottl.ExprFunc[K], error) {
	compiledPattern, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("the regex pattern supplied to replace_all_patterns is not a valid pattern: %w", err)
	}
	if mode != modeValue && mode != modeKey {
		return nil, fmt.Errorf("invalid mode %v, must be either 'key' or 'value'", mode)
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		var replacementVal string
		if err != nil {
			return nil, err
		}
		replacementVal, err = replacement.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		updated := pcommon.NewMap()
		updated.EnsureCapacity(val.Len())
	AttributeLoop:
		for key, originalValue := range val.All() {
			switch mode {
			case modeValue:
				if originalValue.Type() == pcommon.ValueTypeStr && compiledPattern.MatchString(originalValue.Str()) {
					if !fn.IsEmpty() {
						updatedString, err := applyOptReplaceFunction(ctx, tCtx, compiledPattern, fn, originalValue.Str(), replacementVal, replacementFormat)
						if err != nil {
							break AttributeLoop
						}
						updated.PutStr(key, updatedString)
					} else {
						updatedString := compiledPattern.ReplaceAllString(originalValue.Str(), replacementVal)
						updated.PutStr(key, updatedString)
					}
				} else {
					originalValue.CopyTo(updated.PutEmpty(key))
				}
			case modeKey:
				if compiledPattern.MatchString(key) {
					if !fn.IsEmpty() {
						updatedString, err := applyOptReplaceFunction(ctx, tCtx, compiledPattern, fn, key, replacementVal, replacementFormat)
						if err != nil {
							break AttributeLoop
						}
						updated.PutStr(key, updatedString)
					} else {
						updatedKey := compiledPattern.ReplaceAllString(key, replacementVal)
						originalValue.CopyTo(updated.PutEmpty(updatedKey))
					}
				} else {
					originalValue.CopyTo(updated.PutEmpty(key))
				}
			}
		}
		updated.MoveTo(val)
		return nil, nil
	}, nil
}
