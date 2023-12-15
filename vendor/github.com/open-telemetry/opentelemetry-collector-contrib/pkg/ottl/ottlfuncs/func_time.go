// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/timeutils"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type TimeArguments[K any] struct {
	Time   ottl.StringGetter[K]
	Format string
}

func NewTimeFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Time", &TimeArguments[K]{}, createTimeFunction[K])
}
func createTimeFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*TimeArguments[K])

	if !ok {
		return nil, fmt.Errorf("TimeFactory args must be of type *TimeArguments[K]")
	}

	return Time(args.Time, args.Format)
}

func Time[K any](inputTime ottl.StringGetter[K], format string) (ottl.ExprFunc[K], error) {
	if format == "" {
		return nil, fmt.Errorf("format cannot be nil")
	}
	loc, err := timeutils.GetLocation(nil, &format)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := inputTime.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		if t == "" {
			return nil, fmt.Errorf("time cannot be nil")
		}
		timestamp, err := timeutils.ParseStrptime(format, t, loc)
		if err != nil {
			return nil, err
		}
		return timestamp, nil
	}, nil
}
