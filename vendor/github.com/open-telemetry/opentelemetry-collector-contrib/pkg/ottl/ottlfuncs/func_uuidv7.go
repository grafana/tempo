// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"

	guuid "github.com/google/uuid"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

func uuidV7[K any]() (ottl.ExprFunc[K], error) {
	return func(_ context.Context, _ K) (any, error) {
		u, err := guuid.NewV7()
		if err != nil {
			return nil, err
		}
		return u.String(), nil
	}, nil
}

func createUUIDv7Function[K any](_ ottl.FunctionContext, _ ottl.Arguments) (ottl.ExprFunc[K], error) {
	return uuidV7[K]()
}

func NewUUIDv7Factory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("UUIDv7", nil, createUUIDv7Function[K])
}
