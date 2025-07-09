// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/timeutils"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type TimeArguments[K any] struct {
	Time     ottl.StringGetter[K]
	Format   string
	Location ottl.Optional[string]
	Locale   ottl.Optional[string]
}

func NewTimeFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Time", &TimeArguments[K]{}, createTimeFunction[K])
}

func createTimeFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*TimeArguments[K])

	if !ok {
		return nil, errors.New("TimeFactory args must be of type *TimeArguments[K]")
	}

	return Time(args.Time, args.Format, args.Location, args.Locale)
}

func Time[K any](inputTime ottl.StringGetter[K], format string, location ottl.Optional[string], locale ottl.Optional[string]) (ottl.ExprFunc[K], error) {
	if format == "" {
		return nil, errors.New("format cannot be nil")
	}
	gotimeFormat, err := timeutils.StrptimeToGotime(format)
	if err != nil {
		return nil, err
	}

	var defaultLocation *string
	if !location.IsEmpty() {
		l := location.Get()
		defaultLocation = &l
	}

	loc, err := timeutils.GetLocation(defaultLocation, &format)
	if err != nil {
		return nil, err
	}

	var inputTimeLocale *string
	if !locale.IsEmpty() {
		l := locale.Get()
		if err = timeutils.ValidateLocale(l); err != nil {
			return nil, err
		}
		inputTimeLocale = &l
	}

	ctimeSubstitutes := timeutils.GetStrptimeNativeSubstitutes(format)

	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := inputTime.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		if t == "" {
			return nil, errors.New("time cannot be nil")
		}
		var timestamp time.Time
		if inputTimeLocale != nil {
			timestamp, err = timeutils.ParseLocalizedGotime(gotimeFormat, t, loc, *inputTimeLocale)
		} else {
			timestamp, err = timeutils.ParseGotime(gotimeFormat, t, loc)
		}
		if err != nil {
			var timeErr *time.ParseError
			if errors.As(err, &timeErr) {
				return nil, timeutils.ToStrptimeParseError(timeErr, format, ctimeSubstitutes)
			}
			return nil, err
		}
		return timestamp, nil
	}, nil
}
