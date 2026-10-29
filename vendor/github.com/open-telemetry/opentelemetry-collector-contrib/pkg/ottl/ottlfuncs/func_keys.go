// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type KeysArguments[K any] struct {
	Target ottl.PMapGetter[K]
}

func NewKeysFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Keys", &KeysArguments[K]{}, createKeysFunction[K])
}

func createKeysFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*KeysArguments[K])

	if !ok {
		return nil, errors.New("KeysFactory args must be of type *KeysArguments[K]")
	}

	return keys(args.Target), nil
}

func keys[K any](target ottl.PMapGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		m, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		output := pcommon.NewSlice()
		output.EnsureCapacity(m.Len())
		for key := range m.All() {
			output.AppendEmpty().SetStr(key)
		}

		return output, nil
	}
}
