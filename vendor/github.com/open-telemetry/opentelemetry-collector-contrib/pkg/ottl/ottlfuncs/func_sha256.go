// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type SHA256Arguments[K any] struct {
	Target ottl.StringGetter[K] `ottlarg:"0"`
}

func NewSHA256Factory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("SHA256", &SHA256Arguments[K]{}, createSHA256Function[K])
}

func createSHA256Function[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*SHA256Arguments[K])

	if !ok {
		return nil, fmt.Errorf("SHA256Factory args must be of type *SHA256Arguments[K]")
	}

	return SHA256HashString(args.Target)
}

func SHA256HashString[K any](target ottl.StringGetter[K]) (ottl.ExprFunc[K], error) {

	return func(ctx context.Context, tCtx K) (interface{}, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		hash := sha256.New()
		_, err = hash.Write([]byte(val))
		if err != nil {
			return nil, err
		}
		hashValue := hex.EncodeToString(hash.Sum(nil))
		return hashValue, nil
	}, nil
}
