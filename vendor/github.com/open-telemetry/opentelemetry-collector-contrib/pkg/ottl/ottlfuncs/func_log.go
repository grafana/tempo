// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type logArguments[K any] struct {
	Target ottl.FloatLikeGetter[K]
}

// NewLogFactory returns a factory for the Log OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#log
func NewLogFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Log", &logArguments[K]{}, createLogFunction[K])
}

func createLogFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*logArguments[K])

	if !ok {
		return nil, errors.New("LogFactory args must be of type *logArguments[K]")
	}

	return logFunc(args.Target), nil
}

func logFunc[K any](target ottl.FloatLikeGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		value, ok, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errors.New("invalid input: <nil>")
		}

		if value <= 0 {
			return nil, fmt.Errorf("invalid input: expected number greater than zero but got %v", value)
		}
		return math.Log(value), nil
	}
}
