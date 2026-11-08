// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package funcutil // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs/internal/funcutil"

import (
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/ottlcommon"
)

// EvaluateBiFunction executes a lambda with two positional arguments and returns its result.
// R is the type of lambda's evaluation result.
func EvaluateBiFunction[K, R any](
	tCtx K,
	lambda *ottl.LambdaActivation[K],
	v1 any,
	v2 any,
) (R, error) {
	return EvaluateFunction[K, R](tCtx, lambda, v1, v2)
}

// EvaluateBiPredicate evaluates a lambda bi-predicate with two arguments and returns the result.
func EvaluateBiPredicate[K any](
	tCtx K,
	lambda *ottl.LambdaActivation[K],
	v1 any,
	v2 any,
) (bool, error) {
	return EvaluateFunction[K, bool](tCtx, lambda, v1, v2)
}

// EvaluateFunction evaluates a lambda with the given arguments and returns the result.
// Arguments are set via [SetLambdaArgs]. If the number of arguments exceeds the number of
// formals, the function panics.
func EvaluateFunction[K, R any](tCtx K, lambda *ottl.LambdaActivation[K], args ...any) (R, error) {
	err := SetLambdaArgs[K](lambda, args...)
	if err != nil {
		return *new(R), err
	}
	return EvaluateLambdaActivation[K, R](tCtx, lambda)
}

// SetLambdaArgs sets all positional arguments for a [ottl.LambdaActivation] in a single call.
// Bound arguments (those whose formal is not a blank identifier "_") are normalized via
// [ottlcommon.NormalizeValue] before passing to [ottl.LambdaActivation.SetArg]; unbound
// arguments are set to nil since the lambda body discards them. Panics if the number of
// arguments exceeds the number of formals.
func SetLambdaArgs[K any](lambda *ottl.LambdaActivation[K], args ...any) error {
	for i, v := range args {
		var normalizedValue any
		if lambda.IsArgBound(i) {
			normalizedValue = ottlcommon.NormalizeValue(v)
		}
		err := lambda.SetArg(i, normalizedValue)
		if err != nil {
			return err
		}
	}
	return nil
}

// EvaluateLambdaActivation evaluates a lambda activation with the given context and returns the result.
// R is the type of lambda's evaluation result.
// All lambda arguments must be set before calling this function.
func EvaluateLambdaActivation[K, R any](
	tCtx K,
	lambda *ottl.LambdaActivation[K],
) (R, error) {
	eval, err := lambda.Eval(tCtx)
	if err != nil {
		return *new(R), err
	}
	//nolint:gocritic // we want R to be returned as-is even if it's a pcommon.Value
	switch typedVal := eval.(type) {
	case R:
		return typedVal, nil
	case pcommon.Value:
		if res, ok := typedVal.AsRaw().(R); ok {
			return res, nil
		}
	}
	return *new(R), fmt.Errorf("lambda expression must return a value of type %T, got %T", *new(R), eval)
}
