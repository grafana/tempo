// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"encoding/hex"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type hexArguments[K any] struct {
	Target ottl.ByteSliceLikeGetter[K]
}

// NewHexFactory returns a factory for the Hex OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#hex
func NewHexFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Hex", &hexArguments[K]{}, createHexFunction[K])
}

func createHexFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*hexArguments[K])

	if !ok {
		return nil, errors.New("HexFactory args must be of type *hexArguments[K]")
	}

	return hexString(args.Target), nil
}

func hexString[K any](target ottl.ByteSliceLikeGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		value, _, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return hex.EncodeToString(value), nil
	}
}
