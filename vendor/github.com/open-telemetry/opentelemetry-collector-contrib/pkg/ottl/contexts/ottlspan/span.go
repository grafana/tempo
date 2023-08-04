// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ottlspan // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ottlcommon"
)

var _ ottlcommon.ResourceContext = TransformContext{}
var _ ottlcommon.InstrumentationScopeContext = TransformContext{}

type TransformContext struct {
	span                 ptrace.Span
	instrumentationScope pcommon.InstrumentationScope
	resource             pcommon.Resource
	cache                pcommon.Map
}

type Option func(*ottl.Parser[TransformContext])

func NewTransformContext(span ptrace.Span, instrumentationScope pcommon.InstrumentationScope, resource pcommon.Resource) TransformContext {
	return TransformContext{
		span:                 span,
		instrumentationScope: instrumentationScope,
		resource:             resource,
		cache:                pcommon.NewMap(),
	}
}

func (tCtx TransformContext) GetSpan() ptrace.Span {
	return tCtx.span
}

func (tCtx TransformContext) GetInstrumentationScope() pcommon.InstrumentationScope {
	return tCtx.instrumentationScope
}

func (tCtx TransformContext) GetResource() pcommon.Resource {
	return tCtx.resource
}

func (tCtx TransformContext) getCache() pcommon.Map {
	return tCtx.cache
}

func NewParser(functions map[string]interface{}, telemetrySettings component.TelemetrySettings, options ...Option) (ottl.Parser[TransformContext], error) {
	p, err := ottl.NewParser[TransformContext](
		functions,
		parsePath,
		telemetrySettings,
		ottl.WithEnumParser[TransformContext](parseEnum),
	)
	if err != nil {
		return ottl.Parser[TransformContext]{}, err
	}
	for _, opt := range options {
		opt(&p)
	}
	return p, nil
}

type StatementsOption func(*ottl.Statements[TransformContext])

func WithErrorMode(errorMode ottl.ErrorMode) StatementsOption {
	return func(s *ottl.Statements[TransformContext]) {
		ottl.WithErrorMode[TransformContext](errorMode)(s)
	}
}

func NewStatements(statements []*ottl.Statement[TransformContext], telemetrySettings component.TelemetrySettings, options ...StatementsOption) ottl.Statements[TransformContext] {
	s := ottl.NewStatements(statements, telemetrySettings)
	for _, op := range options {
		op(&s)
	}
	return s
}

func parseEnum(val *ottl.EnumSymbol) (*ottl.Enum, error) {
	if val != nil {
		if enum, ok := ottlcommon.SpanSymbolTable[*val]; ok {
			return &enum, nil
		}
		return nil, fmt.Errorf("enum symbol, %s, not found", *val)
	}
	return nil, fmt.Errorf("enum symbol not provided")
}

func parsePath(val *ottl.Path) (ottl.GetSetter[TransformContext], error) {
	if val != nil && len(val.Fields) > 0 {
		return newPathGetSetter(val.Fields)
	}
	return nil, fmt.Errorf("bad path %v", val)
}

func newPathGetSetter(path []ottl.Field) (ottl.GetSetter[TransformContext], error) {
	switch path[0].Name {
	case "cache":
		mapKey := path[0].MapKey
		if mapKey == nil {
			return accessCache(), nil
		}
		return accessCacheKey(mapKey), nil
	case "resource":
		return ottlcommon.ResourcePathGetSetter[TransformContext](path[1:])
	case "instrumentation_scope":
		return ottlcommon.ScopePathGetSetter[TransformContext](path[1:])
	default:
		return ottlcommon.SpanPathGetSetter[TransformContext](path)
	}
}

func accessCache() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return tCtx.getCache(), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if m, ok := val.(pcommon.Map); ok {
				m.CopyTo(tCtx.getCache())
			}
			return nil
		},
	}
}

func accessCacheKey(mapKey *string) ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return ottlcommon.GetMapValue(tCtx.getCache(), *mapKey), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			ottlcommon.SetMapValue(tCtx.getCache(), *mapKey, val)
			return nil
		},
	}
}
