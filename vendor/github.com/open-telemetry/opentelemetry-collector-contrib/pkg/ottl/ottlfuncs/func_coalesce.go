// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type CoalesceArguments[K any] struct {
	Values []ottl.Getter[K]
}

func NewCoalesceFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Coalesce", &CoalesceArguments[K]{}, createCoalesceFunction[K])
}

func createCoalesceFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*CoalesceArguments[K])
	if !ok {
		return nil, errors.New("CoalesceFactory args must be of type *CoalesceArguments[K]")
	}

	if len(args.Values) == 0 {
		return nil, errors.New("Coalesce requires at least one argument")
	}

	return coalesce(args.Values), nil
}

func coalesce[K any](values []ottl.Getter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		for _, val := range values {
			v, err := val.Get(ctx, tCtx)
			if err != nil {
				return nil, err
			}
			if v != nil {
				return v, nil
			}
		}
		return nil, nil
	}
}
