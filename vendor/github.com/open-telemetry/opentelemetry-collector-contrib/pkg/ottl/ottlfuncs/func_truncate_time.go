// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type TruncateTimeArguments[K any] struct {
	Time     ottl.TimeGetter[K]
	Duration ottl.DurationGetter[K]
}

func NewTruncateTimeFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("TruncateTime", &TruncateTimeArguments[K]{}, createTruncateTimeFunction[K])
}

func createTruncateTimeFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*TruncateTimeArguments[K])

	if !ok {
		return nil, errors.New("TimeFactory args must be of type *TruncateTimeArguments[K]")
	}

	return TruncateTime(args.Time, args.Duration)
}

func TruncateTime[K any](inputTime ottl.TimeGetter[K], inputDuration ottl.DurationGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := inputTime.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		d, err := inputDuration.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return t.Truncate(d), nil
	}, nil
}
