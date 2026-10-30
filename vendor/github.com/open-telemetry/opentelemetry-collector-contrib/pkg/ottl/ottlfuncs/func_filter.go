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

type FilterArguments[K any] struct {
	Source    ottl.Getter[K]
	Predicate *ottl.LambdaExpression[K]
}

func NewFilterFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Filter", &FilterArguments[K]{}, createFilterFunction[K])
}

func createFilterFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*FilterArguments[K])
	if !ok {
		return nil, errors.New("FilterFactory args must be of type *FilterArguments[K]")
	}
	return filter(args.Source, args.Predicate)
}

func filter[K any](source ottl.Getter[K], predicate *ottl.LambdaExpression[K]) (ottl.ExprFunc[K], error) {
	err := predicate.ValidateArity(2)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		sourceVal, err := funcutil.GetSliceOrMapValue(ctx, tCtx, source)
		if err != nil {
			return nil, err
		}

		lb, err := predicate.Activate(ctx)
		if err != nil {
			return nil, err
		}
		defer lb.Close()

		switch typedVal := sourceVal.(type) {
		case pcommon.Map:
			return filterMapValues(tCtx, typedVal, lb)
		case pcommon.Slice:
			return filterSliceValues(tCtx, typedVal, lb)
		default:
			return nil, fmt.Errorf("unsupported type: %T", typedVal)
		}
	}, nil
}

func filterSliceValues[K any](tCtx K, source pcommon.Slice, lambda *ottl.LambdaActivation[K]) (pcommon.Slice, error) {
	res := pcommon.NewSlice()
	res.EnsureCapacity(source.Len())
	for i, v := range source.All() {
		keep, err := funcutil.EvaluateBiPredicate(tCtx, lambda, int64(i), v)
		if err != nil {
			return pcommon.Slice{}, fmt.Errorf("error while evaluating lambda function on slice item (%d, %v): %w", i, v, err)
		}
		if keep {
			v.CopyTo(res.AppendEmpty())
		}
	}
	return res, nil
}

func filterMapValues[K any](tCtx K, source pcommon.Map, lambda *ottl.LambdaActivation[K]) (pcommon.Map, error) {
	var builder xpdata.MapBuilder
	builder.EnsureCapacity(source.Len())
	for k, v := range source.All() {
		keep, err := funcutil.EvaluateBiPredicate(tCtx, lambda, k, v)
		if err != nil {
			return pcommon.Map{}, fmt.Errorf("error while evaluating lambda function on map item (%s, %v): %w", k, v, err)
		}
		if keep {
			v.CopyTo(builder.AppendEmpty(k))
		}
	}
	res := pcommon.NewMap()
	builder.UnsafeIntoMap(res)
	return res, nil
}
