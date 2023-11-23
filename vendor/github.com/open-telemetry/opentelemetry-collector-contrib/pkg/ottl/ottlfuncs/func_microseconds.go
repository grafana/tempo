// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type MicrosecondsArguments[K any] struct {
	Duration ottl.DurationGetter[K]
}

func NewMicrosecondsFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Microseconds", &MicrosecondsArguments[K]{}, createMicrosecondsFunction[K])
}
func createMicrosecondsFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*MicrosecondsArguments[K])

	if !ok {
		return nil, fmt.Errorf("MicrosecondsFactory args must be of type *MicrosecondsArguments[K]")
	}

	return Microseconds(args.Duration)
}

func Microseconds[K any](duration ottl.DurationGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		d, err := duration.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return d.Microseconds(), nil
	}, nil
}
