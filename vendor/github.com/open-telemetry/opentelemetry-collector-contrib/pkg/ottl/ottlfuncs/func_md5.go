// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"crypto/md5" // #nosec
	"encoding/hex"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type mD5Arguments[K any] struct {
	Target ottl.StringGetter[K]
}

// NewMD5Factory returns a factory for the MD5 OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#md5
func NewMD5Factory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("MD5", &mD5Arguments[K]{}, createMD5Function[K])
}

func createMD5Function[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*mD5Arguments[K])

	if !ok {
		return nil, errors.New("MD5Factory args must be of type *mD5Arguments[K]")
	}

	return md5HashString(args.Target), nil
}

func md5HashString[K any](target ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		hash := md5.New() // #nosec
		_, err = hash.Write([]byte(val))
		if err != nil {
			return nil, err
		}
		return hex.EncodeToString(hash.Sum(nil)), nil
	}
}
