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

type SpanIDArguments[K any] struct {
	Bytes []byte
}

func NewSpanIDFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("SpanID", &SpanIDArguments[K]{}, createSpanIDFunction[K])
}

func createSpanIDFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*SpanIDArguments[K])

	if !ok {
		return nil, fmt.Errorf("SpanIDFactory args must be of type *SpanIDArguments[K]")
	}

	return spanID[K](args.Bytes)
}

func spanID[K any](bytes []byte) (ottl.ExprFunc[K], error) {
	if len(bytes) != 8 {
		return nil, errors.New("span ids must be 8 bytes")
	}
	var idArr [8]byte
	copy(idArr[:8], bytes)
	id := pcommon.SpanID(idArr)
	return func(context.Context, K) (any, error) {
		return id, nil
	}, nil
}
