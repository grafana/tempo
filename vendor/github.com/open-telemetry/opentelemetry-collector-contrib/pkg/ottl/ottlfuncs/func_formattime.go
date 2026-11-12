// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/timeutils"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type formatTimeArguments[K any] struct {
	Time   ottl.TimeGetter[K]
	Format string
}

// NewFormatTimeFactory returns a factory for the FormatTime OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#formattime
func NewFormatTimeFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("FormatTime", &formatTimeArguments[K]{}, createFormatTimeFunction[K])
}

func createFormatTimeFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*formatTimeArguments[K])

	if !ok {
		return nil, errors.New("FormatTimeFactory args must be of type *formatTimeArguments[K]")
	}

	return formatTime(args.Time, args.Format)
}

func formatTime[K any](timeValue ottl.TimeGetter[K], format string) (ottl.ExprFunc[K], error) {
	if format == "" {
		return nil, errors.New("format cannot be nil")
	}

	if err := timeutils.ValidateStrptime(format); err != nil {
		return nil, err
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := timeValue.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		return timeutils.FormatStrptime(format, t)
	}, nil
}
