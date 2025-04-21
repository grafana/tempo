// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlresource // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlresource"

import (
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap/zapcore"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcache"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcommon"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxresource"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/logging"
)

// Experimental: *NOTE* this constant is subject to change or removal in the future.
const ContextName = ctxresource.Name

var (
	_ ctxresource.Context     = (*TransformContext)(nil)
	_ zapcore.ObjectMarshaler = (*TransformContext)(nil)
)

type TransformContext struct {
	resource      pcommon.Resource
	cache         pcommon.Map
	schemaURLItem ctxcommon.SchemaURLItem
}

func (tCtx TransformContext) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	err := encoder.AddObject("resource", logging.Resource(tCtx.resource))
	err = errors.Join(err, encoder.AddObject("cache", logging.Map(tCtx.cache)))
	return err
}

type TransformContextOption func(*TransformContext)

func NewTransformContext(resource pcommon.Resource, schemaURLItem ctxcommon.SchemaURLItem, options ...TransformContextOption) TransformContext {
	tc := TransformContext{
		resource:      resource,
		cache:         pcommon.NewMap(),
		schemaURLItem: schemaURLItem,
	}
	for _, opt := range options {
		opt(&tc)
	}
	return tc
}

func (tCtx TransformContext) GetResource() pcommon.Resource {
	return tCtx.resource
}

func (tCtx TransformContext) GetResourceSchemaURLItem() ctxcommon.SchemaURLItem {
	return tCtx.schemaURLItem
}

// EnablePathContextNames enables the support to path's context names on statements.
// When this option is configured, all statement's paths must have a valid context prefix,
// otherwise an error is reported.
//
// Experimental: *NOTE* this option is subject to change or removal in the future.
func EnablePathContextNames() ottl.Option[TransformContext] {
	return func(p *ottl.Parser[TransformContext]) {
		ottl.WithPathContextNames[TransformContext]([]string{ContextName})(p)
	}
}

type StatementSequenceOption func(*ottl.StatementSequence[TransformContext])

func WithStatementSequenceErrorMode(errorMode ottl.ErrorMode) StatementSequenceOption {
	return func(s *ottl.StatementSequence[TransformContext]) {
		ottl.WithStatementSequenceErrorMode[TransformContext](errorMode)(s)
	}
}

func NewStatementSequence(statements []*ottl.Statement[TransformContext], telemetrySettings component.TelemetrySettings, options ...StatementSequenceOption) ottl.StatementSequence[TransformContext] {
	s := ottl.NewStatementSequence(statements, telemetrySettings)
	for _, op := range options {
		op(&s)
	}
	return s
}

type ConditionSequenceOption func(*ottl.ConditionSequence[TransformContext])

func WithConditionSequenceErrorMode(errorMode ottl.ErrorMode) ConditionSequenceOption {
	return func(c *ottl.ConditionSequence[TransformContext]) {
		ottl.WithConditionSequenceErrorMode[TransformContext](errorMode)(c)
	}
}

func NewConditionSequence(conditions []*ottl.Condition[TransformContext], telemetrySettings component.TelemetrySettings, options ...ConditionSequenceOption) ottl.ConditionSequence[TransformContext] {
	c := ottl.NewConditionSequence(conditions, telemetrySettings)
	for _, op := range options {
		op(&c)
	}
	return c
}

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
	return nil, errors.New("resource context does not provide Enum support")
}

func getCache(tCtx TransformContext) pcommon.Map {
	return tCtx.cache
}

func pathExpressionParser(cacheGetter ctxcache.Getter[TransformContext]) ottl.PathExpressionParser[TransformContext] {
	return ctxcommon.PathExpressionParser(
		ctxresource.Name,
		ctxresource.DocRef,
		cacheGetter,
		map[string]ottl.PathExpressionParser[TransformContext]{
			ctxresource.Name: ctxresource.PathGetSetter[TransformContext],
		})
}
