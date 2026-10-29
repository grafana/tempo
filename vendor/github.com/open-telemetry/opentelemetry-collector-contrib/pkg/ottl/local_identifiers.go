// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"context"
	"fmt"
	"slices"
)

// LocalIdentifierDecl represents a named or blank parameter in a local scope.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
type LocalIdentifierDecl interface {
	// Name returns the identifier's name as a string.
	Name() string
	// IsBlank indicates whether the identifier is a blank ("_") placeholder.
	IsBlank() bool
}

// localBindingsKey is a [context.Context] key used for storing the active *localActivation
// in the context during evaluation.
type localActivationKey struct{}

// localActivation is a runtime scope frame. Frames link to their parent to support nested scopes
// without copying bindings on each entry.
type localActivation struct {
	parent   *localActivation
	bindings map[string]any
}

// resolve retrieves a local identifier value from the current or parent activation bindings.
func (a *localActivation) resolve(binding string) (any, bool) {
	for cur := a; cur != nil; cur = cur.parent {
		if v, ok := cur.bindings[binding]; ok {
			return v, true
		}
	}
	return nil, false
}

// pushLocalActivation pushes a new localActivation onto the context stack.
// If an existing activation is present, it sets it as the parent of the new activation.
func pushLocalActivation(ctx context.Context, activation *localActivation) context.Context {
	if parent, ok := ctx.Value(localActivationKey{}).(*localActivation); ok {
		activation.parent = parent
	} else {
		activation.parent = nil
	}
	return context.WithValue(ctx, localActivationKey{}, activation)
}

// localScopeFrame is the set of local identifier lexemes declared in one scope frame. (parse time only)
type localScopeFrame map[string]struct{}

// localScopeStack tracks which names are in scope while parsing a subtree (parse time only).
// Lambdas and other constructs that introduce local identifiers push and pop frames on this stack.
// Frames record only local declarations, context data paths are not tracked here and are still
// resolved through PathExpressionParser like any other path.
// No values are stored here, it is only used to determine if a given identifier is in scope.
type localScopeStack []localScopeFrame

func (s *localScopeStack) push(frame localScopeFrame) {
	*s = append(*s, frame)
}

func (s *localScopeStack) pop() {
	*s = (*s)[:len(*s)-1]
}

func (s *localScopeStack) empty() bool {
	return len(*s) == 0
}

func (s *localScopeStack) inScope(lexeme string) bool {
	if s.empty() {
		return false
	}
	for _, v := range slices.Backward(*s) {
		if _, ok := v[lexeme]; ok {
			return true
		}
	}
	return false
}

func localIdentifiersDeclToFrame(params []localIdentifierDecl) localScopeFrame {
	frame := make(localScopeFrame, len(params))
	for _, param := range params {
		if !param.IsBlank() {
			frame[param.Name()] = struct{}{}
		}
	}
	return frame
}

// withLocalScope runs fn while the frame is on the parseContext.localScopes stack.
func (p *parseContext[K]) withLocalScope(frame localScopeFrame, fn func() error) error {
	p.localScopes.push(frame)
	defer p.localScopes.pop()
	return fn()
}

type localIdentifierGetter[K any] struct {
	identifier *basePath[K]
}

func (p *parseContext[K]) newLocalIdentifierGetter(identifier *basePath[K]) (GetSetter[K], error) {
	if !identifier.localIdentifier {
		return nil, fmt.Errorf("%q is not a valid local identifier", identifier.originalText)
	}
	if p.localScopes.empty() {
		return nil, fmt.Errorf("local identifier %q is only valid inside a scoped context", identifier.name)
	}
	if !p.localScopes.inScope(identifier.name) {
		return nil, fmt.Errorf("local identifier %q is not defined in the local scope", identifier.name)
	}
	return &localIdentifierGetter[K]{identifier: identifier}, nil
}

func (g *localIdentifierGetter[K]) Get(ctx context.Context, tCtx K) (any, error) {
	v, err := resolveLocalIdentifierBinding(ctx, g.identifier.name)
	if err != nil {
		return nil, err
	}
	if len(g.identifier.keys) > 0 {
		val, err := getIndexedValue[K](ctx, tCtx, v, g.identifier.keys)
		if err != nil {
			return nil, fmt.Errorf("cannot index local identifier %q: %w", g.identifier.name, err)
		}
		return val, nil
	}
	return v, nil
}

func (g *localIdentifierGetter[K]) Set(context.Context, K, any) error {
	return fmt.Errorf("local identifier %q cannot be set", g.identifier.originalText)
}

func countNonBlankIdentifiers(params []LocalIdentifierDecl) int {
	count := 0
	for _, param := range params {
		if !param.IsBlank() {
			count++
		}
	}
	return count
}

func resolveLocalIdentifierBinding(ctx context.Context, name string) (any, error) {
	activation, ok := ctx.Value(localActivationKey{}).(*localActivation)
	if !ok {
		return nil, fmt.Errorf("local identifier %q evaluated outside of an active local scope", name)
	}
	v, ok := activation.resolve(name)
	if !ok {
		return nil, fmt.Errorf("missing value for local identifier %q", name)
	}
	return v, nil
}

func makeLocalIdentifiers(args ...string) []LocalIdentifierDecl {
	res := make([]LocalIdentifierDecl, len(args))
	for i, v := range args {
		lid := localIdentifierDecl(v)
		res[i] = &lid
	}
	return res
}
