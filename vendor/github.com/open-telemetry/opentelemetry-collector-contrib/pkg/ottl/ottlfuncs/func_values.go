// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ValuesArguments[K any] struct {
	Target ottl.PMapGetter[K]
}

func NewValuesFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Values", &ValuesArguments[K]{}, createValuesFunction[K])
}

func createValuesFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ValuesArguments[K])
	if !ok {
		return nil, errors.New("ValuesFactory args must be of type *ValuesArguments[K]")
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
