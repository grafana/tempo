// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type unixArguments[K any] struct {
	Seconds     ottl.IntGetter[K]
	Nanoseconds ottl.Optional[ottl.IntGetter[K]]
}

// NewUnixFactory returns a factory for the Unix OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#unix
func NewUnixFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Unix", &unixArguments[K]{}, createUnixFunction[K])
}

func createUnixFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*unixArguments[K])

	if !ok {
		return nil, errors.New("UnixFactory args must be of type *unixArguments[K]")
	}

	return unix(args.Seconds, args.Nanoseconds), nil
}

func unix[K any](seconds ottl.IntGetter[K], nanoseconds ottl.Optional[ottl.IntGetter[K]]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		sec, err := seconds.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		var nsec int64

		if !nanoseconds.IsEmpty() {
			nsec, err = nanoseconds.Get().Get(ctx, tCtx)
			if err != nil {
				return nil, err
			}
		}

		return time.Unix(sec, nsec), nil
	}
}
