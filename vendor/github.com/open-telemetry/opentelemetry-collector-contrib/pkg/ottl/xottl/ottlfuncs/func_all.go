// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/xottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/xottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/xottl/ottlfuncs/internal/funcutil"
)

type allArguments[K any] struct {
	Source    ottl.Getter[K]
	Predicate *xottl.LambdaExpression[K]
}

// NewAllFactory returns a factory for the All OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/xottl/ottlfuncs/README.md#all
//
// The function requires the ottl.functions.enableLambda feature gate to be enabled.
func NewAllFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("All", &allArguments[K]{}, createAllFunction[K], ottl.WithExperimental[K]())
}

func createAllFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*allArguments[K])
	if !ok {
		return nil, errors.New("AllFactory args must be of type *allArguments[K]")
	}
	return allMatch(args.Source, args.Predicate)
}

func allMatch[K any](source ottl.Getter[K], predicate *xottl.LambdaExpression[K]) (ottl.ExprFunc[K], error) {
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

func allSliceValuesMatch[K any](tCtx K, source pcommon.Slice, lambda *xottl.LambdaActivation[K]) (bool, error) {
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

func allMapValuesMatch[K any](tCtx K, source pcommon.Map, lambda *xottl.LambdaActivation[K]) (bool, error) {
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
