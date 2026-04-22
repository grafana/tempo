// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"
	"regexp"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

const invalidRegexErrMsg = "the regex pattern supplied to %s '%q' is not a valid pattern: %w"

type dynamicRegex[K any] struct {
	funcName string
	getter   ottl.StringGetter[K]
	value    *regexp.Regexp
}

// newDynamicRegex creates a new dynamicRegex instance that handles both literal and dynamic regex patterns.
// If the pattern is a literal value, it compiles the regex immediately and caches it.
// If the pattern is dynamic, it defers compilation until runtime.
func newDynamicRegex[K any](funcName string, getter ottl.StringGetter[K]) (*dynamicRegex[K], error) {
	if pattern, isLiteral := ottl.GetLiteralValue(getter); isLiteral {
		r, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf(invalidRegexErrMsg, funcName, pattern, err)
		}
		return &dynamicRegex[K]{
			funcName: funcName,
			getter:   getter,
			value:    r,
		}, nil
	}
	return &dynamicRegex[K]{funcName: funcName, getter: getter}, nil
}

// compile returns a compiled regex pattern. If the pattern was pre-compiled (literal), it returns the cached version.
// Otherwise, it retrieves the pattern value at runtime and compiles it.
func (l *dynamicRegex[K]) compile(ctx context.Context, tCtx K) (*regexp.Regexp, error) {
	if l.value != nil {
		return l.value, nil
	}
	pattern, err := l.getter.Get(ctx, tCtx)
	if err != nil {
		return nil, fmt.Errorf(invalidRegexErrMsg, l.funcName, pattern, err)
	}
	return regexp.Compile(pattern)
}
