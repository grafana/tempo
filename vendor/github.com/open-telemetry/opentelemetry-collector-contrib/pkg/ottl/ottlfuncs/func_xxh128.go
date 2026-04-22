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

type XXH128Arguments[K any] struct {
	Target ottl.StringGetter[K]
}

func NewXXH128Factory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("XXH128", &XXH128Arguments[K]{}, createXXH128Function[K])
}

func createXXH128Function[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*XXH128Arguments[K])

	if !ok {
		return nil, errors.New("XXH128Factory args must be of type *XXH128Arguments[K]")
	}

	return xxh128HashString(args.Target), nil
}

func xxh128HashString[K any](target ottl.StringGetter[K]) ottl.ExprFunc[K] {
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
		b := hash.Sum128().Bytes()
		hashValue := hex.EncodeToString(b[:])
		return hashValue, nil
	}
}
