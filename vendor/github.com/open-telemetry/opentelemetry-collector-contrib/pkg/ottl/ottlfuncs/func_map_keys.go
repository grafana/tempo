// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/xpdata"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs/internal/funcutil"
)

type MapKeysArguments[K any] struct {
	Source    ottl.PMapGetter[K]
	KeyMapper *ottl.LambdaExpression[K]
}

func NewMapKeysFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("MapKeys", &MapKeysArguments[K]{}, createMapKeysFunction[K])
}

func createMapKeysFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*MapKeysArguments[K])
	if !ok {
		return nil, errors.New("MapKeysFactory args must be of type *MapKeysArguments[K]")
	}
	return mapKeys(args.Source, args.KeyMapper)
}

func mapKeys[K any](source ottl.PMapGetter[K], keyMapper *ottl.LambdaExpression[K]) (ottl.ExprFunc[K], error) {
	err := keyMapper.ValidateArity(2)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		sourceVal, err := source.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		lb, err := keyMapper.Activate(ctx)
		if err != nil {
			return nil, err
		}
		defer lb.Close()

		var builder xpdata.MapBuilder
		builder.EnsureCapacity(sourceVal.Len())
		seenKeys := make(map[string]pcommon.Value, sourceVal.Len())
		for k, v := range sourceVal.All() {
			newKey, err := funcutil.EvaluateBiFunction[K, string](tCtx, lb, k, v)
			if err != nil {
				return pcommon.Map{}, fmt.Errorf("error while evaluating lambda function on map item (%s, %v): %w", k, v, err)
			}
			if seenValue, ok := seenKeys[newKey]; ok {
				v.CopyTo(seenValue)
				continue
			}
			newValue := builder.AppendEmpty(newKey)
			v.CopyTo(newValue)
			seenKeys[newKey] = newValue
		}

		res := pcommon.NewMap()
		builder.UnsafeIntoMap(res)
		return res, nil
	}, nil
}
