// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type UnixSecondsArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

func NewUnixSecondsFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("UnixSeconds", &UnixSecondsArguments[K]{}, createUnixSecondsFunction[K])
}
func createUnixSecondsFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*UnixSecondsArguments[K])

	if !ok {
		return nil, fmt.Errorf("UnixSecondsFactory args must be of type *UnixSecondsArguments[K]")
	}

	return UnixSeconds(args.Time)
}

func UnixSeconds[K any](inputTime ottl.TimeGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := inputTime.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return t.Unix(), nil
	}, nil
}
