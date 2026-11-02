// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/ottlcommon"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs/internal/funcutil"
)

type FindArguments[K any] struct {
	Source    ottl.Getter[K]
	Predicate *ottl.LambdaExpression[K]
	Mapper    ottl.Optional[*ottl.LambdaExpression[K]]
}

func NewFindFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Find", &FindArguments[K]{}, createFindFunction[K])
}

func createFindFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*FindArguments[K])
	if !ok {
		return nil, errors.New("FindFactory args must be of type *FindArguments[K]")
	}
	return find(args.Source, args.Predicate, &args.Mapper)
}

func find[K any](source ottl.Getter[K], predicate *ottl.LambdaExpression[K], mapper *ottl.Optional[*ottl.LambdaExpression[K]]) (ottl.ExprFunc[K], error) {
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

		var valueMapper *ottl.LambdaActivation[K]
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

func findSliceValue[K any](tCtx K, source pcommon.Slice, lambda, mapper *ottl.LambdaActivation[K]) (any, error) {
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

func findMapValue[K any](tCtx K, source pcommon.Map, lambda, mapper *ottl.LambdaActivation[K]) (any, error) {
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

func formatFindResult[K any](tCtx K, k any, v pcommon.Value, mapper *ottl.LambdaActivation[K]) (any, error) {
	if mapper == nil {
		return ottlcommon.GetValue(v), nil
	}
	result, err := funcutil.EvaluateBiFunction[K, any](tCtx, mapper, k, v)
	if err != nil {
		return false, fmt.Errorf("error while evaluating mapper lambda function on item (%v, %v): %w", k, v, err)
	}
	return ottlcommon.NormalizeValue(result), nil
}
