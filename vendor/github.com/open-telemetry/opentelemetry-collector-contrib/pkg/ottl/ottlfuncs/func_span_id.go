// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"encoding/hex"
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

const spanIDFuncName = "SpanID"

type spanIDArguments[K any] struct {
	Target ottl.ByteSliceLikeGetter[K]
}

// NewSpanIDFactory returns a factory for the SpanID OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#spanid
func NewSpanIDFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory(spanIDFuncName, &spanIDArguments[K]{}, createSpanIDFunction[K])
}

func createSpanIDFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*spanIDArguments[K])

	if !ok {
		return nil, errors.New("SpanIDFactory args must be of type *spanIDArguments[K]")
	}

	return spanID[K](args.Target)
}

func spanID[K any](target ottl.ByteSliceLikeGetter[K]) (ottl.ExprFunc[K], error) {
	return newIDExprFunc(spanIDFuncName, target, decodeHexToSpanID)
}

func decodeHexToSpanID(b []byte) (pcommon.SpanID, error) {
	var id pcommon.SpanID
	if _, err := hex.Decode(id[:], b); err != nil {
		return pcommon.SpanID{}, err
	}
	return id, nil
}
