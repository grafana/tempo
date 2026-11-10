// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type keepMatchingKeysArguments[K any] struct {
	Target  ottl.PMapGetSetter[K]
	Pattern ottl.StringGetter[K]
}

// NewKeepMatchingKeysFactory returns a factory for the keep_matching_keys OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#keep_matching_keys
func NewKeepMatchingKeysFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("keep_matching_keys", &keepMatchingKeysArguments[K]{}, createKeepMatchingKeysFunction[K])
}

func createKeepMatchingKeysFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*keepMatchingKeysArguments[K])

	if !ok {
		return nil, errors.New("KeepMatchingKeysFactory args must be of type *keepMatchingKeysArguments[K")
	}

	return keepMatchingKeys(args.Target, args.Pattern)
}

func keepMatchingKeys[K any](target ottl.PMapGetSetter[K], pattern ottl.StringGetter[K]) (ottl.ExprFunc[K], error) {
	compiledPattern, err := newDynamicRegex("keep_matching_keys", pattern)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, tCtx K) (any, error) {
		cp, err := compiledPattern.compile(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		val.RemoveIf(func(key string, _ pcommon.Value) bool {
			return !cp.MatchString(key)
		})
		return nil, target.Set(ctx, tCtx, val)
	}, nil
}
