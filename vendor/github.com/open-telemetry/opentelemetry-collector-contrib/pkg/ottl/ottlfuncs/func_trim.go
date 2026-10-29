// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type trimArguments[K any] struct {
	Target      ottl.StringGetter[K]
	Replacement ottl.Optional[string]
}

// NewTrimFactory returns a factory for the Trim OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#trim
func NewTrimFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Trim", &trimArguments[K]{}, createTrimFunction[K])
}

func createTrimFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*trimArguments[K])

	if !ok {
		return nil, errors.New("TrimFactory args must be of type *trimArguments[K]")
	}

	return trim(args.Target, args.Replacement), nil
}

func trim[K any](target ottl.StringGetter[K], replacement ottl.Optional[string]) ottl.ExprFunc[K] {
	replacementString := " "
	if !replacement.IsEmpty() {
		replacementString = replacement.Get()
	}
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return strings.Trim(val, replacementString), nil
	}
}
