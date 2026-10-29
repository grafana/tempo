// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"encoding/hex"
	"errors"

	"github.com/zeebo/xxh3"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type XXH3Arguments[K any] struct {
	Target ottl.StringGetter[K]
}

func NewXXH3Factory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("XXH3", &XXH3Arguments[K]{}, createXXH3Function[K])
}

func createXXH3Function[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*XXH3Arguments[K])

	if !ok {
		return nil, errors.New("XXH3Factory args must be of type *XXH3Arguments[K]")
	}

	return xxh3HashString(args.Target), nil
}

func xxh3HashString[K any](target ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		hash := xxh3.New()
		_, err = hash.WriteString(val)
		if err != nil {
			return nil, err
		}
		hashValue := hex.EncodeToString(hash.Sum(nil))
		return hashValue, nil
	}
}
