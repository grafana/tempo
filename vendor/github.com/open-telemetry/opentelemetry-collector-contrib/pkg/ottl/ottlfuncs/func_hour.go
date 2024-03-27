// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type HourArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

func NewHourFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Hour", &HourArguments[K]{}, createHourFunction[K])
}
func createHourFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*HourArguments[K])

	if !ok {
		return nil, fmt.Errorf("HourFactory args must be of type *HourArguments[K]")
	}

	return Hour(args.Time)
}

func Hour[K any](t ottl.TimeGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		time, err := t.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return int64(time.Hour()), nil
	}, nil
}
