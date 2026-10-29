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

// optionalLiteral is the equivalent of literal for getters whose Get returns (T, bool, error),
// caching the pre-computed value and ok result.
type optionalLiteral[K any, T any] struct {
	value T
	ok    bool
}

func newOptionalLiteral[K, T any](value T, ok bool) *optionalLiteral[K, T] {
	return &optionalLiteral[K, T]{value: value, ok: ok}
}

func (l *optionalLiteral[K, T]) Get(context.Context, K) (T, bool, error) {
	return l.value, l.ok, nil
}

func (*optionalLiteral[K, T]) isLiteral() {}

func isLiteralGetter(getter any) bool {
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

// TryGetLiteralValue retrieves the literal value from the given getter whose Get returns
// (V, bool, error), such as the "Like" getters.
// It returns the value, whether a value was found (false if the underlying value was nil),
// and whether the getter is a literal getter. If the getter is not a literal getter, it
// returns the zero value of V, false, and false.
func TryGetLiteralValue[K, V any](getter interface {
	Get(ctx context.Context, tCtx K) (V, bool, error)
},
) (V, bool, bool) {
	if _, isLiteral := getter.(literalGetter); !isLiteral {
		return *new(V), false, false
	}

	val, found, err := getter.Get(context.Background(), *new(K))
	if err != nil {
		return *new(V), false, false
	}

	return val, found, true
}
