// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
)

// NewIsRootSpanFactory returns a factory for the IsRootSpan OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#isrootspan
func NewIsRootSpanFactory() ottl.Factory[*ottlspan.TransformContext] {
	return ottl.NewFactory("IsRootSpan", nil, createIsRootSpanFunction)
}

func createIsRootSpanFunction(_ ottl.FunctionContext, _ ottl.Arguments) (ottl.ExprFunc[*ottlspan.TransformContext], error) {
	return isRootSpan()
}

func isRootSpan() (ottl.ExprFunc[*ottlspan.TransformContext], error) {
	return func(_ context.Context, tCtx *ottlspan.TransformContext) (any, error) {
		return tCtx.GetSpan().ParentSpanID().IsEmpty(), nil
	}, nil
}
