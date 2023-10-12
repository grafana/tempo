// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlresource // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlresource"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal"
)

var _ internal.ResourceContext = TransformContext{}

type TransformContext struct {
	resource pcommon.Resource
	cache    pcommon.Map
}

type Option func(*ottl.Parser[TransformContext])

func NewTransformContext(resource pcommon.Resource) TransformContext {
	return TransformContext{
		resource: resource,
		cache:    pcommon.NewMap(),
	}
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

func parseEnum(_ *ottl.EnumSymbol) (*ottl.Enum, error) {
	return nil, fmt.Errorf("resource context does not provide Enum support")
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
	default:
		return internal.ResourcePathGetSetter[TransformContext](path)
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
