// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlscope // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlscope"

import (
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap/zapcore"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcache"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcommon"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxresource"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxscope"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/logging"
)

// ContextName is the name of the context for instrumentation scopes.
// Experimental: *NOTE* this constant is subject to change or removal in the future.
const ContextName = ctxscope.Name

var (
	_ ctxresource.Context     = (*TransformContext)(nil)
	_ ctxscope.Context        = (*TransformContext)(nil)
	_ zapcore.ObjectMarshaler = (*TransformContext)(nil)
)

// TransformContext represents an instrumentation scope and its associated hierarchy.
type TransformContext struct {
	instrumentationScope pcommon.InstrumentationScope
	resource             pcommon.Resource
	cache                pcommon.Map
	schemaURLItem        ctxcommon.SchemaURLItem
}

// MarshalLogObject serializes the TransformContext into a zapcore.ObjectEncoder for logging.
func (tCtx TransformContext) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	err := encoder.AddObject("resource", logging.Resource(tCtx.resource))
	err = errors.Join(err, encoder.AddObject("scope", logging.InstrumentationScope(tCtx.instrumentationScope)))
	err = errors.Join(err, encoder.AddObject("cache", logging.Map(tCtx.cache)))
	return err
}

// TransformContextOption represents an option for configuring a TransformContext.
type TransformContextOption func(*TransformContext)

// NewTransformContext creates a new TransformContext with the provided parameters.
func NewTransformContext(instrumentationScope pcommon.InstrumentationScope, resource pcommon.Resource, schemaURLItem ctxcommon.SchemaURLItem, options ...TransformContextOption) TransformContext {
	tc := TransformContext{
		instrumentationScope: instrumentationScope,
		resource:             resource,
		cache:                pcommon.NewMap(),
		schemaURLItem:        schemaURLItem,
	}
	for _, opt := range options {
		opt(&tc)
	}
	return tc
}

// GetInstrumentationScope returns the instrumentation scope from the TransformContext.
func (tCtx TransformContext) GetInstrumentationScope() pcommon.InstrumentationScope {
	return tCtx.instrumentationScope
}

// GetResource returns the resource from the TransformContext.
func (tCtx TransformContext) GetResource() pcommon.Resource {
	return tCtx.resource
}

// GetScopeSchemaURLItem returns the schema URL item for the scope from the TransformContext.
func (tCtx TransformContext) GetScopeSchemaURLItem() ctxcommon.SchemaURLItem {
	return tCtx.schemaURLItem
}

// GetResourceSchemaURLItem returns the schema URL item for the resource from the TransformContext.
func (tCtx TransformContext) GetResourceSchemaURLItem() ctxcommon.SchemaURLItem {
	return tCtx.schemaURLItem
}

// EnablePathContextNames enables the support for path's context names on statements.
// When this option is configured, all statement's paths must have a valid context prefix,
// otherwise an error is reported.
//
// Experimental: *NOTE* this option is subject to change or removal in the future.
func EnablePathContextNames() ottl.Option[TransformContext] {
	return func(p *ottl.Parser[TransformContext]) {
		ottl.WithPathContextNames[TransformContext]([]string{
			ContextName,
			ctxresource.Name,
		})(p)
	}
}

// StatementSequenceOption represents an option for configuring a statement sequence.
type StatementSequenceOption func(*ottl.StatementSequence[TransformContext])

// WithStatementSequenceErrorMode sets the error mode for a statement sequence.
func WithStatementSequenceErrorMode(errorMode ottl.ErrorMode) StatementSequenceOption {
	return func(s *ottl.StatementSequence[TransformContext]) {
		ottl.WithStatementSequenceErrorMode[TransformContext](errorMode)(s)
	}
}

// NewStatementSequence creates a new statement sequence with the provided statements and options.
func NewStatementSequence(statements []*ottl.Statement[TransformContext], telemetrySettings component.TelemetrySettings, options ...StatementSequenceOption) ottl.StatementSequence[TransformContext] {
	s := ottl.NewStatementSequence(statements, telemetrySettings)
	for _, op := range options {
		op(&s)
	}
	return s
}

// ConditionSequenceOption represents an option for configuring a condition sequence.
type ConditionSequenceOption func(*ottl.ConditionSequence[TransformContext])

// WithConditionSequenceErrorMode sets the error mode for a condition sequence.
func WithConditionSequenceErrorMode(errorMode ottl.ErrorMode) ConditionSequenceOption {
	return func(c *ottl.ConditionSequence[TransformContext]) {
		ottl.WithConditionSequenceErrorMode[TransformContext](errorMode)(c)
	}
}

// NewConditionSequence creates a new condition sequence with the provided conditions and options.
func NewConditionSequence(conditions []*ottl.Condition[TransformContext], telemetrySettings component.TelemetrySettings, options ...ConditionSequenceOption) ottl.ConditionSequence[TransformContext] {
	c := ottl.NewConditionSequence(conditions, telemetrySettings)
	for _, op := range options {
		op(&c)
	}
	return c
}

// NewParser creates a new scope parser with the provided functions and options.
func NewParser(
	functions map[string]ottl.Factory[TransformContext],
	telemetrySettings component.TelemetrySettings,
	options ...ottl.Option[TransformContext],
) (ottl.Parser[TransformContext], error) {
	return ctxcommon.NewParser(
		functions,
		telemetrySettings,
		pathExpressionParser(getCache),
		parseEnum,
		options...,
	)
}

func parseEnum(_ *ottl.EnumSymbol) (*ottl.Enum, error) {
	return nil, errors.New("instrumentation scope context does not provide Enum support")
}

func getCache(tCtx TransformContext) pcommon.Map {
	return tCtx.cache
}

func pathExpressionParser(cacheGetter ctxcache.Getter[TransformContext]) ottl.PathExpressionParser[TransformContext] {
	return ctxcommon.PathExpressionParser(
		ctxscope.Name,
		ctxscope.DocRef,
		cacheGetter,
		map[string]ottl.PathExpressionParser[TransformContext]{
			ctxresource.Name:    ctxresource.PathGetSetter[TransformContext],
			ctxscope.Name:       ctxscope.PathGetSetter[TransformContext],
			ctxscope.LegacyName: ctxscope.PathGetSetter[TransformContext],
		})
}
