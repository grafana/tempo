// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"encoding/hex"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type HexArguments[K any] struct {
	Target ottl.ByteSliceLikeGetter[K]
}

func NewHexFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Hex", &HexArguments[K]{}, createHexFunction[K])
}

func createHexFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*HexArguments[K])

	if !ok {
		return nil, errors.New("HexFactory args must be of type *HexArguments[K]")
	}

	return Hex(args.Target)
}

func Hex[K any](target ottl.ByteSliceLikeGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		value, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return hex.EncodeToString(value), nil
	}, nil
}
