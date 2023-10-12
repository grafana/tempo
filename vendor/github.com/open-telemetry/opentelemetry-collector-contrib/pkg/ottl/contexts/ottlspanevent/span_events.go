// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlspanevent // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspanevent"

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal"
)

var _ internal.ResourceContext = TransformContext{}
var _ internal.InstrumentationScopeContext = TransformContext{}
var _ internal.SpanContext = TransformContext{}

type TransformContext struct {
	spanEvent            ptrace.SpanEvent
	span                 ptrace.Span
	instrumentationScope pcommon.InstrumentationScope
	resource             pcommon.Resource
	cache                pcommon.Map
}

type Option func(*ottl.Parser[TransformContext])

func NewTransformContext(spanEvent ptrace.SpanEvent, span ptrace.Span, instrumentationScope pcommon.InstrumentationScope, resource pcommon.Resource) TransformContext {
	return TransformContext{
		spanEvent:            spanEvent,
		span:                 span,
		instrumentationScope: instrumentationScope,
		resource:             resource,
		cache:                pcommon.NewMap(),
	}
}

func (tCtx TransformContext) GetSpanEvent() ptrace.SpanEvent {
	return tCtx.spanEvent
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

func NewParser(functions map[string]ottl.Factory[TransformContext], telemetrySettings component.TelemetrySettings, options ...Option) (ottl.Parser[TransformContext], error) {
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
		if enum, ok := internal.SpanSymbolTable[*val]; ok {
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
		mapKey := path[0].Keys
		if mapKey == nil {
			return accessCache(), nil
		}
		return accessCacheKey(mapKey), nil
	case "resource":
		return internal.ResourcePathGetSetter[TransformContext](path[1:])
	case "instrumentation_scope":
		return internal.ScopePathGetSetter[TransformContext](path[1:])
	case "span":
		return internal.SpanPathGetSetter[TransformContext](path[1:])
	case "time_unix_nano":
		return accessSpanEventTimeUnixNano(), nil
	case "name":
		return accessSpanEventName(), nil
	case "attributes":
		mapKey := path[0].Keys
		if mapKey == nil {
			return accessSpanEventAttributes(), nil
		}
		return accessSpanEventAttributesKey(mapKey), nil
	case "dropped_attributes_count":
		return accessSpanEventDroppedAttributeCount(), nil
	}

	return nil, fmt.Errorf("invalid scope path expression %v", path)
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

func accessCacheKey(keys []ottl.Key) ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return internal.GetMapValue(tCtx.getCache(), keys)
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			return internal.SetMapValue(tCtx.getCache(), keys, val)
		},
	}
}

func accessSpanEventTimeUnixNano() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return tCtx.GetSpanEvent().Timestamp().AsTime().UnixNano(), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newTimestamp, ok := val.(int64); ok {
				tCtx.GetSpanEvent().SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, newTimestamp)))
			}
			return nil
		},
	}
}

func accessSpanEventName() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return tCtx.GetSpanEvent().Name(), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newName, ok := val.(string); ok {
				tCtx.GetSpanEvent().SetName(newName)
			}
			return nil
		},
	}
}

func accessSpanEventAttributes() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return tCtx.GetSpanEvent().Attributes(), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if attrs, ok := val.(pcommon.Map); ok {
				attrs.CopyTo(tCtx.GetSpanEvent().Attributes())
			}
			return nil
		},
	}
}

func accessSpanEventAttributesKey(keys []ottl.Key) ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return internal.GetMapValue(tCtx.GetSpanEvent().Attributes(), keys)
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			return internal.SetMapValue(tCtx.GetSpanEvent().Attributes(), keys, val)
		},
	}
}

func accessSpanEventDroppedAttributeCount() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return int64(tCtx.GetSpanEvent().DroppedAttributesCount()), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newCount, ok := val.(int64); ok {
				tCtx.GetSpanEvent().SetDroppedAttributesCount(uint32(newCount))
			}
			return nil
		},
	}
}
