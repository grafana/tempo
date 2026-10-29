// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type StringifyAllArguments[K any] struct {
	Target ottl.PMapGetSetter[K]
}

func NewStringifyAllFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("stringify_all", &StringifyAllArguments[K]{}, createStringifyAllFunction[K])
}

func createStringifyAllFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*StringifyAllArguments[K])
	if !ok {
		return nil, errors.New("StringifyAllFactory args must be of type *StringifyAllArguments[K]")
	}

	return stringifyAll(args.Target), nil
}

func stringifyAll[K any](target ottl.PMapGetSetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		for _, value := range val.All() {
			if value.Type() != pcommon.ValueTypeStr {
				value.SetStr(value.AsString())
			}
		}
		return nil, target.Set(ctx, tCtx, val)
	}
}
