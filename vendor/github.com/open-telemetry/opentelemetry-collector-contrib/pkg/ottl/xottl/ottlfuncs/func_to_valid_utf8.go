// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/xottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

const defaultToValidUTF8Replacement = "\uFFFD"

type toValidUTF8Arguments[K any] struct {
	Target      ottl.StringGetter[K]
	Replacement ottl.Optional[string]
}

// NewToValidUTF8Factory returns a factory for the ToValidUTF8 OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/xottl/ottlfuncs/README.md#tovalidutf8
func NewToValidUTF8Factory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ToValidUTF8", &toValidUTF8Arguments[K]{}, createToValidUTF8Function[K], ottl.WithExperimental[K]())
}

func createToValidUTF8Function[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*toValidUTF8Arguments[K])
	if !ok {
		return nil, errors.New("ToValidUTF8Factory args must be of type *toValidUTF8Arguments[K]")
	}

	return toValidUTF8(args.Target, args.Replacement)
}

func toValidUTF8[K any](target ottl.StringGetter[K], replacement ottl.Optional[string]) (ottl.ExprFunc[K], error) {
	repl := replacement.GetOr(defaultToValidUTF8Replacement)
	if !utf8.ValidString(repl) {
		return nil, fmt.Errorf("invalid replacement for ToValidUTF8 function, %q is not valid UTF-8", repl)
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		if utf8.ValidString(val) {
			return val, nil
		}
		return strings.ToValidUTF8(val, repl), nil
	}, nil
}
