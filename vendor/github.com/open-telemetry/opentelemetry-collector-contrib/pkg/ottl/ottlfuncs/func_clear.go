// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ClearArguments[K any] struct {
	Target ottl.GetSetter[K]
}

func NewClearFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("clear", &ClearArguments[K]{}, createClearFunction[K])
}

func createClearFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ClearArguments[K])

	if !ok {
		return nil, errors.New("ClearFactory args must be of type *ClearArguments[K]")
	}

	return clearFunc(args.Target), nil
}

func clearFunc[K any](target ottl.GetSetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		targetVal, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, fmt.Errorf("error getting target value to infer zero value in clear: %w", err)
		}

		var val any
		if targetVal != nil {
			switch targetVal.(type) {
			case pcommon.Map:
				val = pcommon.NewMap()
			case pcommon.Slice:
				val = pcommon.NewSlice()
			case pcommon.Value:
				val = pcommon.NewValueEmpty()
			default:
				zero := reflect.Zero(reflect.TypeOf(targetVal))
				if !zero.CanInterface() {
					return nil, fmt.Errorf("cannot infer zero value for type %T in clear", targetVal)
				}
				val = zero.Interface()
			}
		}

		err = target.Set(ctx, tCtx, val)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}
}
