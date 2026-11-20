// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"crypto/fips140"
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
		// SHA-1 is not a FIPS 140-3 approved algorithm for new uses. The digest
		// is used as a plain hash of telemetry data, not for any security
		// function, so it runs outside strict enforcement (GODEBUG=fips140=only).
		// Without that, this converter would panic in FIPS 140-only mode.
		var sum [sha1.Size]byte
		fips140.WithoutEnforcement(func() {
			sum = sha1.Sum([]byte(val)) // #nosec
		})
		return hex.EncodeToString(sum[:]), nil
	}
}
