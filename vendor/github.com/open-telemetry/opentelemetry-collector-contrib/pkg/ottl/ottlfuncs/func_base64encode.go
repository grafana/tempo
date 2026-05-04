// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type Base64EncodeArguments[K any] struct {
	Target  ottl.StringGetter[K]
	Variant ottl.Optional[ottl.StringGetter[K]]
}

func NewBase64EncodeFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Base64Encode", &Base64EncodeArguments[K]{}, createBase64EncodeFunction[K])
}

func createBase64EncodeFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*Base64EncodeArguments[K])
	if !ok {
		return nil, errors.New("Base64EncodeFactory args must be of type *Base64EncodeArguments[K]")
	}

	return base64Encode(args.Target, args.Variant), nil
}

func base64Encode[K any](target ottl.StringGetter[K], variant ottl.Optional[ottl.StringGetter[K]]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		str, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		data := []byte(str)

		variantVal := "base64"
		if !variant.IsEmpty() {
			variantGetter := variant.Get()
			variantVal, err = variantGetter.Get(ctx, tCtx)
			if err != nil {
				return nil, fmt.Errorf("failed to get base64 variant: %w", err)
			}
		}

		switch variantVal {
		case "base64":
			return base64.StdEncoding.EncodeToString(data), nil
		case "base64-raw":
			return base64.RawStdEncoding.EncodeToString(data), nil
		case "base64-url":
			return base64.URLEncoding.EncodeToString(data), nil
		case "base64-raw-url":
			return base64.RawURLEncoding.EncodeToString(data), nil
		default:
			return nil, fmt.Errorf("unsupported base64 variant: %s", variantVal)
		}
	}
}
