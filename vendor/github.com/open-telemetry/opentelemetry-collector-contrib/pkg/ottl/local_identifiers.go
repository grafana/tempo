// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"context"
	"fmt"
	"slices"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/lambda"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/ottlerror"
)

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
		return nil, ottlerror.Errorf(identifier.pos, "%q is not a valid local identifier", identifier.originalText)
	}
	if p.localScopes.empty() {
		return nil, ottlerror.Errorf(identifier.pos, "local identifier %q is only valid inside a scoped context", identifier.name)
	}
	if !p.localScopes.inScope(identifier.name) {
		return nil, ottlerror.Errorf(identifier.pos, "local identifier %q is not defined in the local scope", identifier.name)
	}
	return &localIdentifierGetter[K]{identifier: identifier}, nil
}

func (g *localIdentifierGetter[K]) Get(ctx context.Context, tCtx K) (any, error) {
	v, err := lambda.ResolveBinding(ctx, g.identifier.name)
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
