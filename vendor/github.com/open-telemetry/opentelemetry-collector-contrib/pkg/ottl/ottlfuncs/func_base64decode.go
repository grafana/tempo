// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"encoding/base64"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type Base64DecodeArguments[K any] struct {
	Target ottl.StringGetter[K]
}

func NewBase64DecodeFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Base64Decode", &Base64DecodeArguments[K]{}, createBase64DecodeFunction[K])
}

func createBase64DecodeFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*Base64DecodeArguments[K])

	if !ok {
		return nil, errors.New("Base64DecodeFactory args must be of type *Base64DecodeArguments[K]")
	}

	return Base64Decode(args.Target)
}

func Base64Decode[K any](target ottl.StringGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		base64string, err := base64.StdEncoding.DecodeString(val)
		if err != nil {
			return nil, err
		}
		return string(base64string), nil
	}, nil
}
