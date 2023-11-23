// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type FnvArguments[K any] struct {
	Target ottl.StringGetter[K]
}

func NewFnvFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("FNV", &FnvArguments[K]{}, createFnvFunction[K])
}

func createFnvFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*FnvArguments[K])

	if !ok {
		return nil, fmt.Errorf("FNVFactory args must be of type *FnvArguments[K]")
	}

	return FNVHashString(args.Target)
}

func FNVHashString[K any](target ottl.StringGetter[K]) (ottl.ExprFunc[K], error) {

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
	}, nil
}
