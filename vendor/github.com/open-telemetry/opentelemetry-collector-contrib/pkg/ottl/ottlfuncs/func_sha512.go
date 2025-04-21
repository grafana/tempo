// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type SHA512Arguments[K any] struct {
	Target ottl.StringGetter[K]
}

func NewSHA512Factory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("SHA512", &SHA512Arguments[K]{}, createSHA512Function[K])
}

func createSHA512Function[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*SHA512Arguments[K])

	if !ok {
		return nil, errors.New("SHA512Factory args must be of type *SHA512Arguments[K]")
	}

	return SHA512HashString(args.Target)
}

func SHA512HashString[K any](target ottl.StringGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		hash := sha512.New()
		_, err = hash.Write([]byte(val))
		if err != nil {
			return nil, err
		}
		hashValue := hex.EncodeToString(hash.Sum(nil))
		return hashValue, nil
	}, nil
}
