// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package lambda // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/lambda"

import (
	"context"
	"fmt"
)

type localActivationKey struct{}

// activation is a runtime scope frame. Frames link to their parent to support nested scopes
// without copying bindings on each entry.
type localActivation struct {
	parent   *localActivation
	bindings map[string]any
}

func (a *localActivation) resolve(name string) (any, bool) {
	for cur := a; cur != nil; cur = cur.parent {
		if v, ok := cur.bindings[name]; ok {
			return v, true
		}
	}
	return nil, false
}

func pushLocalActivation(ctx context.Context, a *localActivation) context.Context {
	if parent, ok := ctx.Value(localActivationKey{}).(*localActivation); ok {
		a.parent = parent
	} else {
		a.parent = nil
	}
	return context.WithValue(ctx, localActivationKey{}, a)
}

// WithBindings returns a copy of ctx with a new activation frame holding bindings, nested inside any active frame.
func WithBindings(ctx context.Context, bindings map[string]any) context.Context {
	return pushLocalActivation(ctx, &localActivation{bindings: bindings})
}

// ResolveBinding returns the value bound to name by the innermost lambda activation in ctx that binds it.
func ResolveBinding(ctx context.Context, name string) (any, error) {
	a, ok := ctx.Value(localActivationKey{}).(*localActivation)
	if !ok {
		return nil, fmt.Errorf("local identifier %q evaluated outside of an active local scope", name)
	}
	v, ok := a.resolve(name)
	if !ok {
		return nil, fmt.Errorf("missing value for local identifier %q", name)
	}
	return v, nil
}
