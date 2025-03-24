// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/twmb/murmur3"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type Murmur3Hash128Arguments[K any] struct {
	Target ottl.StringGetter[K]
}

func NewMurmur3Hash128Factory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Murmur3Hash128", &Murmur3Hash128Arguments[K]{}, createMurmur3Hash128Function[K])
}

func createMurmur3Hash128Function[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*Murmur3Hash128Arguments[K])

	if !ok {
		return nil, fmt.Errorf("Murmur3Hash128Factory args must be of type *Murmur3Hash128Arguments[K]")
	}

	return murmur3Hash128(args.Target), nil
}

func murmur3Hash128[K any](target ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		h1, h2 := murmur3.Sum128([]byte(val))
		b := make([]byte, 16)
		binary.LittleEndian.PutUint64(b[:8], h1)
		binary.LittleEndian.PutUint64(b[8:], h2)
		return hex.EncodeToString(b), nil
	}
}
