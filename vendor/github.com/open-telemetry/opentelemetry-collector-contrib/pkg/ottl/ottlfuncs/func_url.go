// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/parseutils"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type URLArguments[K any] struct {
	URI ottl.StringGetter[K]
}

func NewURLFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("URL", &URLArguments[K]{}, createURIFunction[K])
}

func createURIFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*URLArguments[K])
	if !ok {
		return nil, errors.New("URLFactory args must be of type *URLArguments[K]")
	}

	return url(args.URI), nil //revive:disable-line:var-naming
}

func url[K any](uriSource ottl.StringGetter[K]) ottl.ExprFunc[K] { //revive:disable-line:var-naming
	return func(ctx context.Context, tCtx K) (any, error) {
		urlString, err := uriSource.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		if urlString == "" {
			return nil, errors.New("url cannot be empty")
		}

		return parseutils.ParseURI(urlString, true)
	}
}
