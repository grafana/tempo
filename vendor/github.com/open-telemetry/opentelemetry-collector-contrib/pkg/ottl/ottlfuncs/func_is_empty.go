// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"reflect"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type IsEmptyArguments[K any] struct {
	Target ottl.Getter[K]
}

func NewIsEmptyFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("IsEmpty", &IsEmptyArguments[K]{}, createIsEmptyFunction[K])
}

func createIsEmptyFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*IsEmptyArguments[K])

	if !ok {
		return nil, errors.New("IsEmptyFactory args must be of type *IsEmptyArguments[K]")
	}

	return isEmpty(args.Target), nil
}

func isEmpty[K any](target ottl.Getter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return false, err
		}
		if val == nil {
			return true, nil
		}
		switch v := val.(type) {
		case pcommon.Value:
			return v.Type() == pcommon.ValueTypeEmpty, nil
		case pcommon.Map:
			return v.Len() == 0, nil
		case pcommon.Slice:
			return v.Len() == 0, nil
		default:
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Pointer {
				if rv.IsNil() {
					return true, nil
				}
				rv = rv.Elem()
			}
			switch rv.Kind() {
			case reflect.Slice, reflect.Map:
				return rv.Len() == 0, nil
			default:
				return rv.IsZero(), nil
			}
		}
	}
}
