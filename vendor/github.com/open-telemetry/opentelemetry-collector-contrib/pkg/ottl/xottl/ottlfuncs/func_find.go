// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/xottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/ottlcommon"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/xottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/xottl/ottlfuncs/internal/funcutil"
)

type findArguments[K any] struct {
	Source    ottl.Getter[K]
	Predicate *xottl.LambdaExpression[K]
	Mapper    ottl.Optional[*xottl.LambdaExpression[K]]
}

// NewFindFactory returns a factory for the Find OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/xottl/ottlfuncs/README.md#find
//
// The function requires the ottl.functions.enableLambda feature gate to be enabled.
func NewFindFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Find", &findArguments[K]{}, createFindFunction[K], ottl.WithExperimental[K]())
}

func createFindFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*findArguments[K])
	if !ok {
		return nil, errors.New("FindFactory args must be of type *findArguments[K]")
	}
	return find(args.Source, args.Predicate, &args.Mapper)
}

func find[K any](source ottl.Getter[K], predicate *xottl.LambdaExpression[K], mapper *ottl.Optional[*xottl.LambdaExpression[K]]) (ottl.ExprFunc[K], error) {
	err := predicate.ValidateArity(2)
	if err != nil {
		return nil, fmt.Errorf("invalid predicate: %w", err)
	}

	if !mapper.IsEmpty() {
		err = mapper.Get().ValidateArity(2)
		if err != nil {
			return nil, fmt.Errorf("invalid mapper: %w", err)
		}
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

		var valueMapper *xottl.LambdaActivation[K]
		if !mapper.IsEmpty() {
			m := mapper.Get()
			valueMapper, err = m.Activate(ctx)
			if err != nil {
				return nil, err
			}
			defer valueMapper.Close()
		}

		switch typedVal := sourceVal.(type) {
		case pcommon.Map:
			return findMapValue(tCtx, typedVal, lb, valueMapper)
		case pcommon.Slice:
			return findSliceValue(tCtx, typedVal, lb, valueMapper)
		default:
			return nil, fmt.Errorf("unsupported type: %T", typedVal)
		}
	}, nil
}

func findSliceValue[K any](tCtx K, source pcommon.Slice, lambda, mapper *xottl.LambdaActivation[K]) (any, error) {
	for i, v := range source.All() {
		match, err := funcutil.EvaluateBiPredicate(tCtx, lambda, int64(i), v)
		if err != nil {
			return false, fmt.Errorf("error while evaluating lambda function on slice item (%d, %v): %w", i, v, err)
		}
		if match {
			return formatFindResult(tCtx, int64(i), v, mapper)
		}
	}
	return nil, nil
}

func findMapValue[K any](tCtx K, source pcommon.Map, lambda, mapper *xottl.LambdaActivation[K]) (any, error) {
	for k, v := range source.All() {
		match, err := funcutil.EvaluateBiPredicate(tCtx, lambda, k, v)
		if err != nil {
			return false, fmt.Errorf("error while evaluating lambda function on map item (%s, %v): %w", k, v, err)
		}
		if match {
			return formatFindResult(tCtx, k, v, mapper)
		}
	}
	return nil, nil
}

func formatFindResult[K any](tCtx K, k any, v pcommon.Value, mapper *xottl.LambdaActivation[K]) (any, error) {
	if mapper == nil {
		return ottlcommon.GetValue(v), nil
	}
	result, err := funcutil.EvaluateBiFunction[K, any](tCtx, mapper, k, v)
	if err != nil {
		return false, fmt.Errorf("error while evaluating mapper lambda function on item (%v, %v): %w", k, v, err)
	}
	return ottlcommon.NormalizeValue(result), nil
}
