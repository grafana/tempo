// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"context"
)

var _ boolExpr[any] = (*literalExpr[any, bool])(nil)

func newLiteralExpr[K, V any](val V) *literalExpr[K, V] {
	return &literalExpr[K, V]{val: val}
}

type literalExpr[K any, V any] struct {
	val V
}

// Eval evaluates an OTTL condition
func (e *literalExpr[K, V]) Eval(context.Context, K) (V, error) {
	return e.val, nil
}

//nolint:unused
func (*literalExpr[K, V]) unexported() {}

func (e *literalExpr[K, V]) getValue() V {
	return e.val
}
