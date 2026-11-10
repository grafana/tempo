// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type sHA256Arguments[K any] struct {
	Target ottl.StringGetter[K]
}

// NewSHA256Factory returns a factory for the SHA256 OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#sha256
func NewSHA256Factory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("SHA256", &sHA256Arguments[K]{}, createSHA256Function[K])
}

func createSHA256Function[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*sHA256Arguments[K])

	if !ok {
		return nil, errors.New("SHA256Factory args must be of type *sHA256Arguments[K]")
	}

	return sha256HashString(args.Target), nil
}

func sha256HashString[K any](target ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		hash := sha256.New()
		_, err = hash.Write([]byte(val))
		if err != nil {
			return nil, err
		}
		hashValue := hex.EncodeToString(hash.Sum(nil))
		return hashValue, nil
	}
}
