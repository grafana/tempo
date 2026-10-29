// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type valuesArguments[K any] struct {
	Target ottl.PMapGetter[K]
}

// NewValuesFactory returns a factory for the Values OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#values
func NewValuesFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Values", &valuesArguments[K]{}, createValuesFunction[K])
}

func createValuesFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*valuesArguments[K])
	if !ok {
		return nil, errors.New("ValuesFactory args must be of type *valuesArguments[K]")
	}

	return values(args.Target), nil
}

func values[K any](target ottl.PMapGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		m, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		output := pcommon.NewSlice()
		output.EnsureCapacity(m.Len())

		for _, val := range m.All() {
			val.CopyTo(output.AppendEmpty())
		}

		return output, nil
	}
}
