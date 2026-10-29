// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"encoding/hex"
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

const traceIDFuncName = "TraceID"

type TraceIDArguments[K any] struct {
	Target ottl.ByteSliceLikeGetter[K]
}

func NewTraceIDFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory(traceIDFuncName, &TraceIDArguments[K]{}, createTraceIDFunction[K])
}

func createTraceIDFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*TraceIDArguments[K])

	if !ok {
		return nil, errors.New("TraceIDFactory args must be of type *TraceIDArguments[K]")
	}

	return traceID[K](args.Target)
}

func traceID[K any](target ottl.ByteSliceLikeGetter[K]) (ottl.ExprFunc[K], error) {
	return newIDExprFunc(traceIDFuncName, target, decodeHexToTraceID)
}

func decodeHexToTraceID(b []byte) (pcommon.TraceID, error) {
	var id pcommon.TraceID
	if _, err := hex.Decode(id[:], b); err != nil {
		return pcommon.TraceID{}, err
	}
	return id, nil
}
