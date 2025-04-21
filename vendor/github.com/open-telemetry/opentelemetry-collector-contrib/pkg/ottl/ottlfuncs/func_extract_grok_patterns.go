// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/elastic/go-grok"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ExtractGrokPatternsArguments[K any] struct {
	Target             ottl.StringGetter[K]
	Pattern            string
	NamedCapturesOnly  ottl.Optional[bool]
	PatternDefinitions ottl.Optional[[]string]
}

func NewExtractGrokPatternsFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ExtractGrokPatterns", &ExtractGrokPatternsArguments[K]{}, createExtractGrokPatternsFunction[K])
}

func createExtractGrokPatternsFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ExtractGrokPatternsArguments[K])

	if !ok {
		return nil, errors.New("ExtractGrokPatternsFactory args must be of type *ExtractGrokPatternsArguments[K]")
	}

	return extractGrokPatterns(args.Target, args.Pattern, args.NamedCapturesOnly, args.PatternDefinitions)
}

func extractGrokPatterns[K any](target ottl.StringGetter[K], pattern string, nco ottl.Optional[bool], patternDefinitions ottl.Optional[[]string]) (ottl.ExprFunc[K], error) {
	g, err := grok.NewComplete()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize grok parser: %w", err)
	}
	namedCapturesOnly := !nco.IsEmpty() && nco.Get()

	if !patternDefinitions.IsEmpty() {
		for i, patternDefinition := range patternDefinitions.Get() {
			// split pattern in format key=val
			parts := strings.SplitN(patternDefinition, "=", 2)
			if len(parts) == 1 {
				trimmedPattern := patternDefinition
				if len(patternDefinition) > 20 {
					trimmedPattern = fmt.Sprintf("%s...", patternDefinition[:17]) // keep whole string 20 characters long including ...
				}
				return nil, fmt.Errorf("pattern %q supplied to ExtractGrokPatterns at index %d has incorrect format, expecting PATTERNNAME=pattern definition", trimmedPattern, i)
			}

			if strings.ContainsRune(parts[0], ':') {
				return nil, fmt.Errorf("pattern ID %q should not contain ':'", parts[0])
			}

			err = g.AddPattern(parts[0], parts[1])
			if err != nil {
				return nil, fmt.Errorf("failed to add pattern %q=%q: %w", parts[0], parts[1], err)
			}
		}
	}
	err = g.Compile(pattern, namedCapturesOnly)
	if err != nil {
		return nil, fmt.Errorf("the pattern supplied to ExtractGrokPatterns is not a valid pattern: %w", err)
	}

	if namedCapturesOnly && !g.HasCaptureGroups() {
		return nil, errors.New("at least 1 named capture group must be supplied in the given regex")
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		matches, err := g.ParseTypedString(val)
		if err != nil {
			return nil, err
		}

		result := pcommon.NewMap()
		for k, v := range matches {
			switch val := v.(type) {
			case bool:
				result.PutBool(k, val)
			case float64:
				result.PutDouble(k, val)
			case int:
				result.PutInt(k, int64(val))
			case string:
				result.PutStr(k, val)
			}
		}

		return result, err
	}, nil
}
