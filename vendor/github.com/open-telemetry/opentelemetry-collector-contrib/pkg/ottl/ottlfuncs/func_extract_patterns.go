// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ExtractPatternsArguments[K any] struct {
	Target  ottl.StringGetter[K]
	Pattern ottl.StringGetter[K]
}

func NewExtractPatternsFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ExtractPatterns", &ExtractPatternsArguments[K]{}, createExtractPatternsFunction[K])
}

func createExtractPatternsFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ExtractPatternsArguments[K])

	if !ok {
		return nil, errors.New("ExtractPatternsFactory args must be of type *ExtractPatternsArguments[K]")
	}

	return extractPatterns(args.Target, args.Pattern)
}

func extractPatterns[K any](target, pattern ottl.StringGetter[K]) (ottl.ExprFunc[K], error) {
	compiledPattern, err := newDynamicRegex("ExtractPatterns", pattern)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		cp, err := compiledPattern.compile(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		namedCaptureGroups := 0
		for _, groupName := range cp.SubexpNames() {
			if groupName != "" {
				namedCaptureGroups++
			}
		}

		if namedCaptureGroups == 0 {
			return nil, errors.New("at least 1 named capture group must be supplied in the given regex")
		}

		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		matches := cp.FindStringSubmatch(val)
		if matches == nil {
			return pcommon.NewMap(), nil
		}

		result := pcommon.NewMap()
		for i, subexp := range cp.SubexpNames() {
			if i == 0 {
				// Skip whole match
				continue
			}
			if subexp != "" {
				result.PutStr(subexp, matches[i])
			}
		}
		return result, err
	}, nil
}
