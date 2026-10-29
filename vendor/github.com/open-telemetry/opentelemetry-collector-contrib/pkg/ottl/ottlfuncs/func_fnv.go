// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"hash/fnv"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type fnvArguments[K any] struct {
	Target ottl.StringGetter[K]
}

// NewFnvFactory returns a factory for the FNV OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#fnv
func NewFnvFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("FNV", &fnvArguments[K]{}, createFnvFunction[K])
}

func createFnvFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*fnvArguments[K])

	if !ok {
		return nil, errors.New("FNVFactory args must be of type *fnvArguments[K]")
	}

	return fnvHashString(args.Target), nil
}

func fnvHashString[K any](target ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		hash := fnv.New64a()
		_, err = hash.Write([]byte(val))
		if err != nil {
			return nil, err
		}
		hashValue := hash.Sum64()
		return int64(hashValue), nil
	}
}
