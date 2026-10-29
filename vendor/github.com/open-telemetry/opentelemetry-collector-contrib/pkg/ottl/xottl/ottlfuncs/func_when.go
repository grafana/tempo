// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/xottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/xottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/xottl/ottlfuncs/internal/funcutil"
)

type whenArguments[K any] struct {
	Condition  *xottl.LambdaExpression[K]
	TrueValue  ottl.Getter[K]
	FalseValue ottl.Getter[K]
}

// NewWhenFactory returns a factory for the When OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/xottl/ottlfuncs/README.md#when
//
// The function requires the ottl.functions.enableLambda feature gate to be enabled.
func NewWhenFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("When", &whenArguments[K]{}, createWhenFunction[K], ottl.WithExperimental[K]())
}

func createWhenFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*whenArguments[K])
	if !ok {
		return nil, errors.New("WhenFactory args must be of type *whenArguments[K]")
	}
	return whenFunction(args.Condition, args.TrueValue, args.FalseValue)
}

func whenFunction[K any](condition *xottl.LambdaExpression[K], trueValueGetter, falseValueGetter ottl.Getter[K]) (ottl.ExprFunc[K], error) {
	err := condition.ValidateArity(0)
	if err != nil {
		return nil, err
	}

	var trueValue any
	var falseValue any
	if tv, ok := ottl.GetLiteralValue(trueValueGetter); ok {
		trueValue = tv
	}
	if fv, ok := ottl.GetLiteralValue(falseValueGetter); ok {
		falseValue = fv
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		lb, err := condition.Activate(ctx)
		if err != nil {
			return nil, err
		}
		defer lb.Close()

		match, err := funcutil.EvaluateLambdaActivation[K, bool](tCtx, lb)
		if err != nil {
			return nil, fmt.Errorf("error while evaluating lambda function: %w", err)
		}

		if match {
			if trueValue != nil {
				return trueValue, nil
			}
			return trueValueGetter.Get(ctx, tCtx)
		}

		if falseValue != nil {
			return falseValue, nil
		}

		return falseValueGetter.Get(ctx, tCtx)
	}, nil
}
