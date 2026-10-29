// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"

	"github.com/twmb/murmur3"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type Murmur3HashArguments[K any] struct {
	Target ottl.StringGetter[K]
}

func NewMurmur3HashFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Murmur3Hash", &Murmur3HashArguments[K]{}, createMurmur3HashFunction[K])
}

func createMurmur3HashFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*Murmur3HashArguments[K])

	if !ok {
		return nil, errors.New("Murmur3HashFactory args must be of type *Murmur3HashArguments[K]")
	}

	return murmur3Hash(args.Target), nil
}

func murmur3Hash[K any](target ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		h := murmur3.Sum32([]byte(val))
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, h)
		return hex.EncodeToString(b), nil
	}
}
