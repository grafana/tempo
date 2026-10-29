// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"crypto/sha1" // #nosec
	"encoding/hex"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type sHA1Arguments[K any] struct {
	Target ottl.StringGetter[K]
}

// NewSHA1Factory returns a factory for the SHA1 OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#sha1
func NewSHA1Factory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("SHA1", &sHA1Arguments[K]{}, createSHA1Function[K])
}

func createSHA1Function[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*sHA1Arguments[K])

	if !ok {
		return nil, errors.New("SHA1Factory args must be of type *sHA1Arguments[K]")
	}

	return sha1HashString(args.Target), nil
}

func sha1HashString[K any](target ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		hash := sha1.New() // #nosec
		_, err = hash.Write([]byte(val))
		if err != nil {
			return nil, err
		}
		hashValue := hex.EncodeToString(hash.Sum(nil))
		return hashValue, nil
	}
}
