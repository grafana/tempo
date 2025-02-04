// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal"

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

var _ ottl.Path[any] = &TestPath[any]{}

type TestPath[K any] struct {
	C        string
	N        string
	KeySlice []ottl.Key[K]
	NextPath *TestPath[K]
	FullPath string
}

func (p *TestPath[K]) Name() string {
	return p.N
}

func (p *TestPath[K]) Context() string {
	return p.C
}

func (p *TestPath[K]) Next() ottl.Path[K] {
	if p.NextPath == nil {
		return nil
	}
	return p.NextPath
}

func (p *TestPath[K]) Keys() []ottl.Key[K] {
	return p.KeySlice
}

func (p *TestPath[K]) String() string {
	if p.FullPath != "" {
		return p.FullPath
	}
	return p.N
}

var _ ottl.Key[any] = &TestKey[any]{}

type TestKey[K any] struct {
	S *string
	I *int64
	G ottl.Getter[K]
}

func (k *TestKey[K]) String(_ context.Context, _ K) (*string, error) {
	return k.S, nil
}

func (k *TestKey[K]) Int(_ context.Context, _ K) (*int64, error) {
	return k.I, nil
}

func (k *TestKey[K]) ExpressionGetter(_ context.Context, _ K) (ottl.Getter[K], error) {
	return k.G, nil
}
