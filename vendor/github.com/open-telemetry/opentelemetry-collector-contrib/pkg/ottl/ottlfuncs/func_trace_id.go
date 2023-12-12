// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type TraceIDArguments[K any] struct {
	Bytes []byte
}

func NewTraceIDFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("TraceID", &TraceIDArguments[K]{}, createTraceIDFunction[K])
}

func createTraceIDFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*TraceIDArguments[K])

	if !ok {
		return nil, fmt.Errorf("TraceIDFactory args must be of type *TraceIDArguments[K]")
	}

	return traceID[K](args.Bytes)
}

func traceID[K any](bytes []byte) (ottl.ExprFunc[K], error) {
	if len(bytes) != 16 {
		return nil, errors.New("traces ids must be 16 bytes")
	}
	var idArr [16]byte
	copy(idArr[:16], bytes)
	id := pcommon.TraceID(idArr)
	return func(context.Context, K) (any, error) {
		return id, nil
	}, nil
}
