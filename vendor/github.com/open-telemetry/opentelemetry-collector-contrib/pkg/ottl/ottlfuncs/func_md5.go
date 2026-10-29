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

type MD5Arguments[K any] struct {
	Target ottl.StringGetter[K]
}

func NewMD5Factory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("MD5", &MD5Arguments[K]{}, createMD5Function[K])
}

func createMD5Function[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*MD5Arguments[K])

	if !ok {
		return nil, errors.New("MD5Factory args must be of type *MD5Arguments[K]")
	}

	return MD5HashString(args.Target)
}

func MD5HashString[K any](target ottl.StringGetter[K]) (ottl.ExprFunc[K], error) {
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
	}, nil
}
