// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/timeutils"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type FormatTimeArguments[K any] struct {
	Time   ottl.TimeGetter[K]
	Format string
}

func NewFormatTimeFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("FormatTime", &FormatTimeArguments[K]{}, createFormatTimeFunction[K])
}

func createFormatTimeFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*FormatTimeArguments[K])

	if !ok {
		return nil, errors.New("FormatTimeFactory args must be of type *FormatTimeArguments[K]")
	}

	return FormatTime(args.Time, args.Format)
}

func FormatTime[K any](timeValue ottl.TimeGetter[K], format string) (ottl.ExprFunc[K], error) {
	if format == "" {
		return nil, errors.New("format cannot be nil")
	}

	gotimeFormat, err := timeutils.StrptimeToGotime(format)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := timeValue.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		return t.Format(gotimeFormat), nil
	}, nil
}
