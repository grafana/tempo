// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
)

func NewIsRootSpanFactory() ottl.Factory[ottlspan.TransformContext] {
	return ottl.NewFactory("IsRootSpan", nil, createIsRootSpanFunction)
}

func createIsRootSpanFunction(_ ottl.FunctionContext, _ ottl.Arguments) (ottl.ExprFunc[ottlspan.TransformContext], error) {
	return isRootSpan()
}

func isRootSpan() (ottl.ExprFunc[ottlspan.TransformContext], error) {
	return func(_ context.Context, tCtx ottlspan.TransformContext) (any, error) {
		return tCtx.GetSpan().ParentSpanID().IsEmpty(), nil
	}, nil
}
