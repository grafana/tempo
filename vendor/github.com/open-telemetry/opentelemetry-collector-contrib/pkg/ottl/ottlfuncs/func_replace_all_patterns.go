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
	Target       ottl.PMapGetter[K]   `ottlarg:"0"`
	Mode         string               `ottlarg:"1"`
	RegexPattern string               `ottlarg:"2"`
	Replacement  ottl.StringGetter[K] `ottlarg:"3"`
}

func NewReplaceAllPatternsFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("replace_all_patterns", &ReplaceAllPatternsArguments[K]{}, createReplaceAllPatternsFunction[K])
}

func createReplaceAllPatternsFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ReplaceAllPatternsArguments[K])

	if !ok {
		return nil, fmt.Errorf("ReplaceAllPatternsFactory args must be of type *ReplaceAllPatternsArguments[K]")
	}

	return replaceAllPatterns(args.Target, args.Mode, args.RegexPattern, args.Replacement)
}

func replaceAllPatterns[K any](target ottl.PMapGetter[K], mode string, regexPattern string, replacement ottl.StringGetter[K]) (ottl.ExprFunc[K], error) {
	compiledPattern, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("the regex pattern supplied to replace_all_patterns is not a valid pattern: %w", err)
	}
	if mode != modeValue && mode != modeKey {
		return nil, fmt.Errorf("invalid mode %v, must be either 'key' or 'value'", mode)
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
		updated := pcommon.NewMap()
		updated.EnsureCapacity(val.Len())
		val.Range(func(key string, originalValue pcommon.Value) bool {
			switch mode {
			case modeValue:
				if compiledPattern.MatchString(originalValue.Str()) {
					updatedString := compiledPattern.ReplaceAllString(originalValue.Str(), replacementVal)
					updated.PutStr(key, updatedString)
				} else {
					originalValue.CopyTo(updated.PutEmpty(key))
				}
			case modeKey:
				if compiledPattern.MatchString(key) {
					updatedKey := compiledPattern.ReplaceAllString(key, replacementVal)
					originalValue.CopyTo(updated.PutEmpty(updatedKey))
				} else {
					originalValue.CopyTo(updated.PutEmpty(key))
				}
			}
			return true
		})
		updated.CopyTo(val)
		return nil, nil
	}, nil
}
