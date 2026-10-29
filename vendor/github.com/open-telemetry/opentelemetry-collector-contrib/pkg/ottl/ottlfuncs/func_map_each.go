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
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/ottlcommon"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs/internal/funcutil"
)

type MapEachArguments[K any] struct {
	Source ottl.Getter[K]
	Mapper *ottl.LambdaExpression[K]
}

func NewMapEachFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("MapEach", &MapEachArguments[K]{}, createMapEachFunction[K])
}

func createMapEachFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*MapEachArguments[K])
	if !ok {
		return nil, errors.New("MapEachFactory args must be of type *MapEachArguments[K]")
	}
	return mapEach(args.Source, args.Mapper)
}

func mapEach[K any](source ottl.Getter[K], mapper *ottl.LambdaExpression[K]) (ottl.ExprFunc[K], error) {
	err := mapper.ValidateArity(2)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		sourceVal, err := funcutil.GetSliceOrMapValue(ctx, tCtx, source)
		if err != nil {
			return nil, err
		}

		lb, err := mapper.Activate(ctx)
		if err != nil {
			return nil, err
		}
		defer lb.Close()

		switch typedVal := sourceVal.(type) {
		case pcommon.Map:
			return mapMapValues(tCtx, typedVal, lb)
		case pcommon.Slice:
			return mapSliceValues(tCtx, typedVal, lb)
		default:
			return nil, fmt.Errorf("unsupported type: %T", typedVal)
		}
	}, nil
}

func mapMapValues[K any](tCtx K, source pcommon.Map, lb *ottl.LambdaActivation[K]) (pcommon.Map, error) {
	var builder xpdata.MapBuilder
	builder.EnsureCapacity(source.Len())
	for k, v := range source.All() {
		val, err := funcutil.EvaluateBiFunction[K, any](tCtx, lb, k, v)
		if err != nil {
			return pcommon.Map{}, fmt.Errorf("error while evaluating lambda function on map item (%s, %v): %w", k, v, err)
		}
		err = ottlcommon.CopyValueTo(val, builder.AppendEmpty(k))
		if err != nil {
			return pcommon.Map{}, fmt.Errorf("error while converting lambda function result on map item (%s, %v): %w", k, val, err)
		}
	}
	res := pcommon.NewMap()
	builder.UnsafeIntoMap(res)
	return res, nil
}

func mapSliceValues[K any](tCtx K, source pcommon.Slice, lb *ottl.LambdaActivation[K]) (pcommon.Slice, error) {
	res := pcommon.NewSlice()
	res.EnsureCapacity(source.Len())
	for i, v := range source.All() {
		val, err := funcutil.EvaluateBiFunction[K, any](tCtx, lb, int64(i), v)
		if err != nil {
			return pcommon.Slice{}, fmt.Errorf("error while evaluating lambda function on slice item (%d, %v): %w", i, v, err)
		}
		err = ottlcommon.CopyValueTo(val, res.AppendEmpty())
		if err != nil {
			return pcommon.Slice{}, fmt.Errorf("error while converting lambda function result on slice item (%d, %v): %w", i, val, err)
		}
	}
	return res, nil
}
