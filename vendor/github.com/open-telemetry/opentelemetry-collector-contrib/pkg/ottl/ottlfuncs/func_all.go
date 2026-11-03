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

type AllArguments[K any] struct {
	Source    ottl.Getter[K]
	Predicate *ottl.LambdaExpression[K]
}

func NewAllFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("All", &AllArguments[K]{}, createAllFunction[K])
}

func createAllFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*AllArguments[K])
	if !ok {
		return nil, errors.New("AllFactory args must be of type *AllArguments[K]")
	}
	return allMatch(args.Source, args.Predicate)
}

func allMatch[K any](source ottl.Getter[K], predicate *ottl.LambdaExpression[K]) (ottl.ExprFunc[K], error) {
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
			return allMapValuesMatch(tCtx, typedVal, lb)
		case pcommon.Slice:
			return allSliceValuesMatch(tCtx, typedVal, lb)
		default:
			return nil, fmt.Errorf("unsupported type: %T", typedVal)
		}
	}, nil
}

func allSliceValuesMatch[K any](tCtx K, source pcommon.Slice, lambda *ottl.LambdaActivation[K]) (bool, error) {
	for i, v := range source.All() {
		match, err := funcutil.EvaluateBiPredicate(tCtx, lambda, int64(i), v)
		if err != nil {
			return false, fmt.Errorf("error while evaluating lambda function on slice item (%d, %v): %w", i, v, err)
		}
		if !match {
			return false, nil
		}
	}
	return true, nil
}

func allMapValuesMatch[K any](tCtx K, source pcommon.Map, lambda *ottl.LambdaActivation[K]) (bool, error) {
	for k, v := range source.All() {
		match, err := funcutil.EvaluateBiPredicate(tCtx, lambda, k, v)
		if err != nil {
			return false, fmt.Errorf("error while evaluating lambda function on map item (%s, %v): %w", k, v, err)
		}
		if !match {
			return false, nil
		}
	}
	return true, nil
}
