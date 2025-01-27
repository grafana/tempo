// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlresource // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlresource"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap/zapcore"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/logging"
)

const (
	// Experimental: *NOTE* this constant is subject to change or removal in the future.
	ContextName = internal.ResourceContextName
)

var (
	_ internal.ResourceContext = (*TransformContext)(nil)
	_ zapcore.ObjectMarshaler  = (*TransformContext)(nil)
)

type TransformContext struct {
	resource      pcommon.Resource
	cache         pcommon.Map
	schemaURLItem internal.SchemaURLItem
}

func (tCtx TransformContext) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	err := encoder.AddObject("resource", logging.Resource(tCtx.resource))
	err = errors.Join(err, encoder.AddObject("cache", logging.Map(tCtx.cache)))
	return err
}

type Option func(*ottl.Parser[TransformContext])

type TransformContextOption func(*TransformContext)

func NewTransformContext(resource pcommon.Resource, schemaURLItem internal.SchemaURLItem, options ...TransformContextOption) TransformContext {
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

// Experimental: *NOTE* this option is subject to change or removal in the future.
func WithCache(cache *pcommon.Map) TransformContextOption {
	return func(p *TransformContext) {
		if cache != nil {
			p.cache = *cache
		}
	}
}

func (tCtx TransformContext) GetResource() pcommon.Resource {
	return tCtx.resource
}

func (tCtx TransformContext) getCache() pcommon.Map {
	return tCtx.cache
}

func (tCtx TransformContext) GetResourceSchemaURLItem() internal.SchemaURLItem {
	return tCtx.schemaURLItem
}

func NewParser(functions map[string]ottl.Factory[TransformContext], telemetrySettings component.TelemetrySettings, options ...Option) (ottl.Parser[TransformContext], error) {
	pep := pathExpressionParser{telemetrySettings}
	p, err := ottl.NewParser[TransformContext](
		functions,
		pep.parsePath,
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

// EnablePathContextNames enables the support to path's context names on statements.
// When this option is configured, all statement's paths must have a valid context prefix,
// otherwise an error is reported.
//
// Experimental: *NOTE* this option is subject to change or removal in the future.
func EnablePathContextNames() Option {
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

func parseEnum(_ *ottl.EnumSymbol) (*ottl.Enum, error) {
	return nil, fmt.Errorf("resource context does not provide Enum support")
}

type pathExpressionParser struct {
	telemetrySettings component.TelemetrySettings
}

func (pep *pathExpressionParser) parsePath(path ottl.Path[TransformContext]) (ottl.GetSetter[TransformContext], error) {
	if path == nil {
		return nil, fmt.Errorf("path cannot be nil")
	}
	if path.Context() != "" && path.Context() != ContextName {
		return nil, internal.FormatDefaultErrorMessage(path.Context(), path.String(), "Resource", internal.ResourceContextRef)
	}

	switch path.Name() {
	case "cache":
		if path.Keys() == nil {
			return accessCache(), nil
		}
		return accessCacheKey(path.Keys()), nil
	default:
		return internal.ResourcePathGetSetter[TransformContext](ContextName, path)
	}
}

func accessCache() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(_ context.Context, tCtx TransformContext) (any, error) {
			return tCtx.getCache(), nil
		},
		Setter: func(_ context.Context, tCtx TransformContext, val any) error {
			if m, ok := val.(pcommon.Map); ok {
				m.CopyTo(tCtx.getCache())
			}
			return nil
		},
	}
}

func accessCacheKey(key []ottl.Key[TransformContext]) ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (any, error) {
			return internal.GetMapValue[TransformContext](ctx, tCtx, tCtx.getCache(), key)
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val any) error {
			return internal.SetMapValue[TransformContext](ctx, tCtx, tCtx.getCache(), key, val)
		},
	}
}
