// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs/internal/funcutil"
)

type WhenArguments[K any] struct {
	Condition  *ottl.LambdaExpression[K]
	TrueValue  ottl.Getter[K]
	FalseValue ottl.Getter[K]
}

func NewWhenFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("When", &WhenArguments[K]{}, createWhenFunction[K])
}

func createWhenFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*WhenArguments[K])
	if !ok {
		return nil, errors.New("WhenFactory args must be of type *WhenArguments[K]")
	}
	return whenFunction(args.Condition, args.TrueValue, args.FalseValue)
}

func whenFunction[K any](condition *ottl.LambdaExpression[K], trueValueGetter, falseValueGetter ottl.Getter[K]) (ottl.ExprFunc[K], error) {
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
