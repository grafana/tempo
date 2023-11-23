// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

func now[K any]() (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		return time.Now(), nil
	}, nil
}

func createNowFunction[K any](_ ottl.FunctionContext, _ ottl.Arguments) (ottl.ExprFunc[K], error) {
	return now[K]()
}

func NewNowFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Now", nil, createNowFunction[K])
}
