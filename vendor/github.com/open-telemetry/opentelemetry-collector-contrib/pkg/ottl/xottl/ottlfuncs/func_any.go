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

type anyArguments[K any] struct {
	Source    ottl.Getter[K]
	Predicate *xottl.LambdaExpression[K]
}

// NewAnyFactory returns a factory for the Any OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/xottl/ottlfuncs/README.md#any
//
// The function requires the ottl.functions.enableLambda feature gate to be enabled.
func NewAnyFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Any", &anyArguments[K]{}, createAnyFunction[K], ottl.WithExperimental[K]())
}

func createAnyFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*anyArguments[K])
	if !ok {
		return nil, errors.New("AnyFactory args must be of type *anyArguments[K]")
	}
	return anyMatch(args.Source, args.Predicate)
}

func anyMatch[K any](source ottl.Getter[K], predicate *xottl.LambdaExpression[K]) (ottl.ExprFunc[K], error) {
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
			return anyMapValueMatch(tCtx, typedVal, lb)
		case pcommon.Slice:
			return anySliceValueMatch(tCtx, typedVal, lb)
		default:
			return nil, fmt.Errorf("unsupported type: %T", typedVal)
		}
	}, nil
}

func anySliceValueMatch[K any](tCtx K, source pcommon.Slice, lambda *xottl.LambdaActivation[K]) (bool, error) {
	for i, v := range source.All() {
		match, err := funcutil.EvaluateBiPredicate(tCtx, lambda, int64(i), v)
		if err != nil {
			return false, fmt.Errorf("error while evaluating lambda function on slice item (%d, %v): %w", i, v, err)
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}

func anyMapValueMatch[K any](tCtx K, source pcommon.Map, lambda *xottl.LambdaActivation[K]) (bool, error) {
	for k, v := range source.All() {
		match, err := funcutil.EvaluateBiPredicate(tCtx, lambda, k, v)
		if err != nil {
			return false, fmt.Errorf("error while evaluating lambda function on map item (%s, %v): %w", k, v, err)
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}
