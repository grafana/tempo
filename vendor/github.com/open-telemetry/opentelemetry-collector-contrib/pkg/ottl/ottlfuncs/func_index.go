// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type indexArguments[K any] struct {
	Target ottl.Getter[K]
	Value  ottl.Getter[K]
}

// NewIndexFactory returns a factory for the Index OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#index
func NewIndexFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Index", &indexArguments[K]{}, createIndexFunction[K])
}

func createIndexFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*indexArguments[K])

	if !ok {
		return nil, errors.New("IndexFactory args must be of type *indexArguments[K]")
	}

	return index(ottl.NewValueComparator(), args.Target, args.Value), nil
}

func index[K any](valueComparator ottl.ValueComparator, target, value ottl.Getter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		targetVal, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		valueVal, err := value.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		if pv, ok := valueVal.(pcommon.Value); ok {
			valueVal = pv.AsRaw()
		}

		switch s := targetVal.(type) {
		case string:
			v, ok := valueVal.(string)
			if !ok {
				return nil, errors.New("invalid value type for Index function, value must be a string")
			}
			return int64(strings.Index(s, v)), nil
		default:
			sg := ottl.StandardPSliceGetter[K]{
				Getter: func(_ context.Context, _ K) (any, error) {
					return targetVal, nil
				},
			}
			slice, err := sg.Get(ctx, tCtx)
			if err != nil {
				return nil, err
			}
			for i, v := range slice.All() {
				if valueComparator.Equal(v.AsRaw(), valueVal) {
					return int64(i), nil
				}
			}
			return int64(-1), nil
		}
	}
}
