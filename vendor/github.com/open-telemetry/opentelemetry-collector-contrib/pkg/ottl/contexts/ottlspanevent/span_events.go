// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlspanevent // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspanevent"

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap/zapcore"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcache"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcommon"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxresource"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxscope"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxspanevent"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/logging"
)

var tcPool = sync.Pool{
	New: func() any {
		return &TransformContext{cache: pcommon.NewMap()}
	},
}

// ContextName is the name of the context for span events.
// Experimental: *NOTE* this constant is subject to change or removal in the future.
const ContextName = ctxspanevent.Name

var _ zapcore.ObjectMarshaler = (*TransformContext)(nil)

// TransformContext represents a span event and its associated hierarchy.
type TransformContext struct {
	resourceSpans ptrace.ResourceSpans
	scopeSpans    ptrace.ScopeSpans
	span          ptrace.Span
	spanEvent     ptrace.SpanEvent
	cache         pcommon.Map
	eventIndex    *int64
}

// MarshalLogObject serializes the TransformContext into a zapcore.ObjectEncoder for logging.
func (tCtx *TransformContext) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	err := encoder.AddObject("resource", logging.Resource(tCtx.GetResource()))
	err = errors.Join(err, encoder.AddObject("scope", logging.InstrumentationScope(tCtx.GetInstrumentationScope())))
	err = errors.Join(err, encoder.AddObject("span", logging.Span(tCtx.span)))
	err = errors.Join(err, encoder.AddObject("spanevent", logging.SpanEvent(tCtx.spanEvent)))
	err = errors.Join(err, encoder.AddObject("cache", logging.Map(tCtx.cache)))
	if tCtx.eventIndex != nil {
		encoder.AddInt64("event_index", *tCtx.eventIndex)
	}
	return err
}

// TransformContextOption represents an option for configuring a TransformContext.
type TransformContextOption func(*TransformContext)

// NewTransformContextPtr returns a new TransformContext with the provided parameters from a pool of contexts.
// Caller must call TransformContext.Close on the returned TransformContext.
func NewTransformContextPtr(resourceSpans ptrace.ResourceSpans, scopeSpans ptrace.ScopeSpans, span ptrace.Span, spanEvent ptrace.SpanEvent, options ...TransformContextOption) *TransformContext {
	tCtx := tcPool.Get().(*TransformContext)
	tCtx.resourceSpans = resourceSpans
	tCtx.scopeSpans = scopeSpans
	tCtx.span = span
	tCtx.spanEvent = spanEvent
	for _, opt := range options {
		opt(tCtx)
	}
	return tCtx
}

// Close the current TransformContext.
// After this function returns this instance cannot be used.
func (tCtx *TransformContext) Close() {
	tCtx.resourceSpans = ptrace.ResourceSpans{}
	tCtx.scopeSpans = ptrace.ScopeSpans{}
	tCtx.span = ptrace.Span{}
	tCtx.spanEvent = ptrace.SpanEvent{}
	tCtx.cache.Clear()
	tCtx.eventIndex = nil
	tcPool.Put(tCtx)
}

// WithEventIndex sets the index of the SpanEvent within the span, to make it accessible via the event_index property of its context.
// The index must be greater than or equal to zero, otherwise the given value will not be applied.
func WithEventIndex(eventIndex int64) TransformContextOption {
	return func(p *TransformContext) {
		p.eventIndex = &eventIndex
	}
}

// GetSpanEvent returns the span event from the TransformContext.
func (tCtx *TransformContext) GetSpanEvent() ptrace.SpanEvent {
	return tCtx.spanEvent
}

// GetSpan returns the span from the TransformContext.
func (tCtx *TransformContext) GetSpan() ptrace.Span {
	return tCtx.span
}

// GetInstrumentationScope returns the instrumentation scope from the TransformContext.
func (tCtx *TransformContext) GetInstrumentationScope() pcommon.InstrumentationScope {
	return tCtx.scopeSpans.Scope()
}

// GetResource returns the resource from the TransformContext.
func (tCtx *TransformContext) GetResource() pcommon.Resource {
	return tCtx.resourceSpans.Resource()
}

// GetScopeSchemaURLItem returns the schema URL item for the scope from the TransformContext.
func (tCtx *TransformContext) GetScopeSchemaURLItem() ctxcommon.SchemaURLItem {
	return tCtx.scopeSpans
}

// GetResourceSchemaURLItem returns the schema URL item for the resource from the TransformContext.
func (tCtx *TransformContext) GetResourceSchemaURLItem() ctxcommon.SchemaURLItem {
	return tCtx.resourceSpans
}

// GetEventIndex returns the event index from the TransformContext.
// If the event index is not set or invalid, an error is returned.
func (tCtx *TransformContext) GetEventIndex() (int64, error) {
	if tCtx.eventIndex != nil {
		if *tCtx.eventIndex < 0 {
			return 0, errors.New("found invalid value for 'event_index'")
		}
		return *tCtx.eventIndex, nil
	}
	return 0, errors.New("no 'event_index' property has been set")
}

// EnablePathContextNames enables the support for path's context names on statements.
// When this option is configured, all statement's paths must have a valid context prefix,
// otherwise an error is reported.
//
// Experimental: *NOTE* this option is subject to change or removal in the future.
func EnablePathContextNames() ottl.Option[*TransformContext] {
	return func(p *ottl.Parser[*TransformContext]) {
		ottl.WithPathContextNames[*TransformContext]([]string{
			ctxspanevent.Name,
			ctxspan.Name,
			ctxresource.Name,
			ctxscope.LegacyName,
			ctxscope.Name,
		})(p)
	}
}

// StatementSequenceOption represents an option for configuring a statement sequence.
type StatementSequenceOption func(*ottl.StatementSequence[*TransformContext])

// WithStatementSequenceErrorMode sets the error mode for a statement sequence.
func WithStatementSequenceErrorMode(errorMode ottl.ErrorMode) StatementSequenceOption {
	return func(s *ottl.StatementSequence[*TransformContext]) {
		ottl.WithStatementSequenceErrorMode[*TransformContext](errorMode)(s)
	}
}

// NewStatementSequence creates a new statement sequence with the provided statements and options.
func NewStatementSequence(statements []*ottl.Statement[*TransformContext], telemetrySettings component.TelemetrySettings, options ...StatementSequenceOption) ottl.StatementSequence[*TransformContext] {
	s := ottl.NewStatementSequence(statements, telemetrySettings)
	for _, op := range options {
		op(&s)
	}
	return s
}

// ConditionSequenceOption represents an option for configuring a condition sequence.
type ConditionSequenceOption func(*ottl.ConditionSequence[*TransformContext])

// WithConditionSequenceErrorMode sets the error mode for a condition sequence.
func WithConditionSequenceErrorMode(errorMode ottl.ErrorMode) ConditionSequenceOption {
	return func(c *ottl.ConditionSequence[*TransformContext]) {
		ottl.WithConditionSequenceErrorMode[*TransformContext](errorMode)(c)
	}
}

// NewConditionSequence creates a new condition sequence with the provided conditions and options.
func NewConditionSequence(conditions []*ottl.Condition[*TransformContext], telemetrySettings component.TelemetrySettings, options ...ConditionSequenceOption) ottl.ConditionSequence[*TransformContext] {
	c := ottl.NewConditionSequence(conditions, telemetrySettings)
	for _, op := range options {
		op(&c)
	}
	return c
}

// NewParser creates a new span event parser with the provided functions and options.
func NewParser(
	functions map[string]ottl.Factory[*TransformContext],
	telemetrySettings component.TelemetrySettings,
	options ...ottl.Option[*TransformContext],
) (ottl.Parser[*TransformContext], error) {
	return ctxcommon.NewParser(
		functions,
		telemetrySettings,
		pathExpressionParser(getCache),
		parseEnum,
		options...,
	)
}

func parseEnum(val *ottl.EnumSymbol) (*ottl.Enum, error) {
	if val != nil {
		if enum, ok := ctxspan.SymbolTable[*val]; ok {
			return &enum, nil
		}
		return nil, fmt.Errorf("enum symbol, %s, not found", *val)
	}
	return nil, errors.New("enum symbol not provided")
}

func getCache(tCtx *TransformContext) pcommon.Map {
	return tCtx.cache
}

func pathExpressionParser(cacheGetter ctxcache.Getter[*TransformContext]) ottl.PathExpressionParser[*TransformContext] {
	return ctxcommon.PathExpressionParser(
		ctxspanevent.Name,
		ctxspanevent.DocRef,
		cacheGetter,
		map[string]ottl.PathExpressionParser[*TransformContext]{
			ctxresource.Name:    ctxresource.PathGetSetter[*TransformContext],
			ctxscope.Name:       ctxscope.PathGetSetter[*TransformContext],
			ctxscope.LegacyName: ctxscope.PathGetSetter[*TransformContext],
			ctxspan.Name:        ctxspan.PathGetSetter[*TransformContext],
			ctxspanevent.Name:   spanEventGetSetterWithIndex,
		})
}

func spanEventGetSetterWithIndex(path ottl.Path[*TransformContext]) (ottl.GetSetter[*TransformContext], error) {
	if path.Name() == "event_index" {
		return accessSpanEventIndex(), nil
	}
	return ctxspanevent.PathGetSetter(path)
}

func accessSpanEventIndex() ottl.StandardGetSetter[*TransformContext] {
	return ottl.StandardGetSetter[*TransformContext]{
		Getter: func(_ context.Context, tCtx *TransformContext) (any, error) {
			return tCtx.GetEventIndex()
		},
		Setter: func(_ context.Context, _ *TransformContext, _ any) error {
			return errors.New("the 'event_index' path cannot be modified")
		},
	}
}
