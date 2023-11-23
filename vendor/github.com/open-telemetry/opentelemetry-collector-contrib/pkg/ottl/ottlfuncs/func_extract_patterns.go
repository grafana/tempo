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

type ExtractPatternsArguments[K any] struct {
	Target  ottl.StringGetter[K]
	Pattern string
}

func NewExtractPatternsFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ExtractPatterns", &ExtractPatternsArguments[K]{}, createExtractPatternsFunction[K])
}

func createExtractPatternsFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ExtractPatternsArguments[K])

	if !ok {
		return nil, fmt.Errorf("ExtractPatternsFactory args must be of type *ExtractPatternsArguments[K]")
	}

	return extractPatterns(args.Target, args.Pattern)
}

func extractPatterns[K any](target ottl.StringGetter[K], pattern string) (ottl.ExprFunc[K], error) {
	r, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("the pattern supplied to ExtractPatterns is not a valid pattern: %w", err)
	}

	namedCaptureGroups := 0
	for _, groupName := range r.SubexpNames() {
		if groupName != "" {
			namedCaptureGroups++
		}
	}

	if namedCaptureGroups == 0 {
		return nil, fmt.Errorf("at least 1 named capture group must be supplied in the given regex")
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		matches := r.FindStringSubmatch(val)
		if matches == nil {
			return pcommon.NewMap(), nil
		}

		result := pcommon.NewMap()
		for i, subexp := range r.SubexpNames() {
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
