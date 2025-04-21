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

type LogArguments[K any] struct {
	Target ottl.FloatLikeGetter[K]
}

func NewLogFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Log", &LogArguments[K]{}, createLogFunction[K])
}

func createLogFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*LogArguments[K])

	if !ok {
		return nil, errors.New("LogFactory args must be of type *LogArguments[K]")
	}

	return logFunc(args.Target), nil
}

func logFunc[K any](target ottl.FloatLikeGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		value, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		if value == nil {
			return nil, fmt.Errorf("invalid input: %v", value)
		}

		if *value <= 0 {
			return nil, fmt.Errorf("invalid input: expected number greater than zero but got %v", *value)
		}
		return math.Log(*value), nil
	}
}
