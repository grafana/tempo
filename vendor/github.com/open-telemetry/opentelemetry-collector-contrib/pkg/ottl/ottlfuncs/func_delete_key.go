// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type DeleteKeyArguments[K any] struct {
	Target ottl.PMapGetSetter[K]
	Key    ottl.StringGetter[K]
}

func NewDeleteKeyFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("delete_key", &DeleteKeyArguments[K]{}, createDeleteKeyFunction[K])
}

func createDeleteKeyFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*DeleteKeyArguments[K])

	if !ok {
		return nil, errors.New("DeleteKeysFactory args must be of type *DeleteKeyArguments[K]")
	}

	return deleteKey(args.Target, args.Key), nil
}

func deleteKey[K any](target ottl.PMapGetSetter[K], key ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		keyVal, err := key.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		val.Remove(keyVal)
		return nil, target.Set(ctx, tCtx, val)
	}
}
