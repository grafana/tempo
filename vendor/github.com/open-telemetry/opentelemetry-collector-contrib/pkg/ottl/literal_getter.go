// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"context"
)

// literalGetter is an optional interface that allows Getter implementations to indicate
// if they support literal values retrieval.
type literalGetter interface {
	isLiteral()
}

type literal[K any, T any] struct {
	value T
}

func newLiteral[K, T any](value T) *literal[K, T] {
	return &literal[K, T]{value: value}
}

func (l *literal[K, T]) Get(context.Context, K) (T, error) {
	return l.value, nil
}

func (*literal[K, T]) isLiteral() {}

func isLiteralGetter[K, V any](getter typedGetter[K, V]) bool {
	_, isLiteral := getter.(literalGetter)
	return isLiteral
}

// GetLiteralValue retrieves the literal value from the given getter.
// If the getter is not a literal getter, or if the value it's currently holding is not a
// literal value, it returns the zero value of V and false.
func GetLiteralValue[K, V any](getter typedGetter[K, V]) (V, bool) {
	if !isLiteralGetter(getter) {
		return *new(V), false
	}

	val, err := getter.Get(context.Background(), *new(K))
	if err != nil {
		return *new(V), false
	}

	return val, true
}
