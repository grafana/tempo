// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type SplitArguments[K any] struct {
	Target    ottl.StringGetter[K]
	Delimiter string
}

func NewSplitFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Split", &SplitArguments[K]{}, createSplitFunction[K])
}

func createSplitFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*SplitArguments[K])

	if !ok {
		return nil, fmt.Errorf("SplitFactory args must be of type *SplitArguments[K]")
	}

	return split(args.Target, args.Delimiter), nil
}

func split[K any](target ottl.StringGetter[K], delimiter string) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return strings.Split(val, delimiter), nil
	}
}
