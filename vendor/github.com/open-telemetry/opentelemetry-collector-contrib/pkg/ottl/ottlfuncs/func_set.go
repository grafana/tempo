// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"sync"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/metadata"
)

var warnOnce sync.Once

type SetArguments[K any] struct {
	Target ottl.Setter[K]
	Value  ottl.Getter[K]
}

func NewSetFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("set", &SetArguments[K]{}, createSetFunction[K])
}

func createSetFunction[K any](fCtx ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*SetArguments[K])

	if !ok {
		return nil, errors.New("SetFactory args must be of type *SetArguments[K]")
	}

	return set(args.Target, args.Value, fCtx), nil
}

func set[K any](target ottl.Setter[K], value ottl.Getter[K], fCtx ottl.FunctionContext) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := value.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		if val == nil {
			if !metadata.OttlSetAllowNilFeatureGate.IsEnabled() {
				warnOnce.Do(func() {
					fCtx.Set.Logger.Warn("The OTTL 'set' function silently ignored a nil value. This behavior is deprecated and will change in a future release. Enable the feature gate 'ottl.set.allowNil' to pass nil to the target.")
				})
				return nil, nil
			}
		}

		err = target.Set(ctx, tCtx, val)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}
}
