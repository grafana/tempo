// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs/internal/funcutil"
)

type ReduceArguments[K any] struct {
	Source      ottl.Getter[K]
	Seed        ottl.Getter[K]
	Accumulator *ottl.LambdaExpression[K]
}

func NewReduceFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Reduce", &ReduceArguments[K]{}, createReduceFunction[K])
}

func createReduceFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ReduceArguments[K])
	if !ok {
		return nil, errors.New("ReduceFactory args must be of type *ReduceArguments[K]")
	}
	return reduce(args.Source, args.Seed, args.Accumulator)
}

func reduce[K any](source, seed ottl.Getter[K], accumulator *ottl.LambdaExpression[K]) (ottl.ExprFunc[K], error) {
	err := accumulator.ValidateArity(3)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		sourceVal, err := funcutil.GetSliceOrMapValue(ctx, tCtx, source)
		if err != nil {
			return nil, err
		}

		seedVal, err := seed.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		lb, err := accumulator.Activate(ctx)
		if err != nil {
			return nil, err
		}
		defer lb.Close()

		switch typedVal := sourceVal.(type) {
		case pcommon.Map:
			return reduceMapValues(tCtx, typedVal, lb, seedVal)
		case pcommon.Slice:
			return reduceSliceValues(tCtx, typedVal, lb, seedVal)
		default:
			return nil, fmt.Errorf("unsupported type: %T", typedVal)
		}
	}, nil
}

func reduceMapValues[K any](tCtx K, source pcommon.Map, lb *ottl.LambdaActivation[K], seedVal any) (any, error) {
	if source.Len() == 0 {
		return seedVal, nil
	}
	var err error
	acc := seedVal
	for k, v := range source.All() {
		if acc, err = funcutil.EvaluateFunction[K, any](tCtx, lb, acc, k, v); err != nil {
			return nil, fmt.Errorf("error while evaluating accumulator function on map item (%s, %v): %w", k, v, err)
		}
	}
	return acc, nil
}

func reduceSliceValues[K any](tCtx K, source pcommon.Slice, lb *ottl.LambdaActivation[K], seedVal any) (any, error) {
	if source.Len() == 0 {
		return seedVal, nil
	}
	var err error
	acc := seedVal
	for i, v := range source.All() {
		if acc, err = funcutil.EvaluateFunction[K, any](tCtx, lb, acc, int64(i), v); err != nil {
			return nil, fmt.Errorf("error while evaluating accumulator function on slice item (%d, %v): %w", i, v, err)
		}
	}
	return acc, nil
}
