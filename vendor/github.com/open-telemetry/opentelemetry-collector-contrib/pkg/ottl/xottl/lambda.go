// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/xottl"

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/lambda"
)

// LambdaExpression is a parsed OTTL lambda expression. OTTL functions accept it by declaring an
// argument of type *LambdaExpression[K]; the OTTL parser only builds lambda arguments when the
// ottl.functions.enableLambda feature gate is enabled.
//
// Call ValidateArity once in the function factory with the number of arguments to bind, then for
// each outer invocation call Activate with the evaluation context. Use SetArg on the returned
// [LambdaActivation] to bind positional arguments, Eval to run the body (possibly multiple times
// with different arguments), and Close when finished.
type LambdaExpression[K any] = lambda.LambdaExpression[K]

// LambdaActivation is a local activation of a [LambdaExpression] produced by its Activate method.
type LambdaActivation[K any] = lambda.LambdaActivation[K]

// NewTestingLambdaExpression creates a LambdaExpression with a value body for use in tests.
// eval is called with resolveBinding to resolve local identifier values from the active scope.
// Params named "_" are blank placeholders that are never bound.
func NewTestingLambdaExpression[K any](
	params []string,
	eval func(ctx context.Context, tCtx K, resolveBinding func(string) any) (any, error),
) *LambdaExpression[K] {
	return lambda.New(params, func(ctx context.Context, tCtx K) (any, error) {
		resolveBinding := func(name string) any {
			v, err := lambda.ResolveBinding(ctx, name)
			if err != nil {
				return nil
			}
			return v
		}
		return eval(ctx, tCtx, resolveBinding)
	})
}
