// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlprofile // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlprofile"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.uber.org/zap/zapcore"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcache"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcommon"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxprofile"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxresource"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxscope"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/logging"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/logprofile"
)

// ContextName is the name of the context for profiles.
// Experimental: *NOTE* this constant is subject to change or removal in the future.
const ContextName = ctxprofile.Name

var (
	_ ctxresource.Context     = TransformContext{}
	_ ctxscope.Context        = TransformContext{}
	_ ctxprofile.Context      = TransformContext{}
	_ zapcore.ObjectMarshaler = TransformContext{}
)

// MarshalLogObject serializes the profile into a zapcore.ObjectEncoder for logging.
func (tCtx TransformContext) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	err := encoder.AddObject("resource", logging.Resource(tCtx.resource))
	err = errors.Join(err, encoder.AddObject("scope", logging.InstrumentationScope(tCtx.instrumentationScope)))
	err = errors.Join(err, encoder.AddObject("profile", logprofile.Profile{Profile: tCtx.profile, Dictionary: tCtx.dictionary}))
	err = errors.Join(err, encoder.AddObject("cache", logging.Map(tCtx.cache)))
	return err
}

// TransformContext represents a profile and its associated hierarchy.
type TransformContext struct {
	profile              pprofile.Profile
	dictionary           pprofile.ProfilesDictionary
	instrumentationScope pcommon.InstrumentationScope
	resource             pcommon.Resource
	cache                pcommon.Map
	scopeProfiles        pprofile.ScopeProfiles
	resourceProfiles     pprofile.ResourceProfiles
}

// TransformContextOption represents an option for configuring a TransformContext.
type TransformContextOption func(*TransformContext)

// NewTransformContext creates a new TransformContext with the provided parameters.
func NewTransformContext(profile pprofile.Profile, dictionary pprofile.ProfilesDictionary, instrumentationScope pcommon.InstrumentationScope, resource pcommon.Resource, scopeProfiles pprofile.ScopeProfiles, resourceProfiles pprofile.ResourceProfiles, options ...TransformContextOption) TransformContext {
	tc := TransformContext{
		profile:              profile,
		dictionary:           dictionary,
		instrumentationScope: instrumentationScope,
		resource:             resource,
		cache:                pcommon.NewMap(),
		scopeProfiles:        scopeProfiles,
		resourceProfiles:     resourceProfiles,
	}
	for _, opt := range options {
		opt(&tc)
	}
	return tc
}

// WithCache sets the cache for the TransformContext.
// Experimental: *NOTE* this option is subject to change or removal in the future.
func WithCache(cache *pcommon.Map) TransformContextOption {
	return func(p *TransformContext) {
		if cache != nil {
			p.cache = *cache
		}
	}
}

// GetProfile returns the profile from the TransformContext.
func (tCtx TransformContext) GetProfile() pprofile.Profile {
	return tCtx.profile
}

// GetInstrumentationScope returns the instrumentation scope from the TransformContext.
func (tCtx TransformContext) GetInstrumentationScope() pcommon.InstrumentationScope {
	return tCtx.instrumentationScope
}

// GetResource returns the resource from the TransformContext.
func (tCtx TransformContext) GetResource() pcommon.Resource {
	return tCtx.resource
}

// GetScopeSchemaURLItem returns the scope schema URL item from the TransformContext.
func (tCtx TransformContext) GetScopeSchemaURLItem() ctxcommon.SchemaURLItem {
	return tCtx.scopeProfiles
}

// GetResourceSchemaURLItem returns the resource schema URL item from the TransformContext.
func (tCtx TransformContext) GetResourceSchemaURLItem() ctxcommon.SchemaURLItem {
	return tCtx.resourceProfiles
}

// NewParser creates a new profile parser with the provided functions and options.
func NewParser(functions map[string]ottl.Factory[TransformContext], telemetrySettings component.TelemetrySettings, options ...ottl.Option[TransformContext]) (ottl.Parser[TransformContext], error) {
	return ctxcommon.NewParser(
		functions,
		telemetrySettings,
		pathExpressionParser(getCache),
		parseEnum,
		options...,
	)
}

// EnablePathContextNames enables the support for path's context names on statements.
// When this option is configured, all statement's paths must have a valid context prefix,
// otherwise an error is reported.
//
// Experimental: *NOTE* this option is subject to change or removal in the future.
func EnablePathContextNames() ottl.Option[TransformContext] {
	return func(p *ottl.Parser[TransformContext]) {
		ottl.WithPathContextNames[TransformContext]([]string{
			ctxprofile.Name,
			ctxscope.LegacyName,
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

func parseEnum(val *ottl.EnumSymbol) (*ottl.Enum, error) {
	if val != nil {
		return nil, fmt.Errorf("enum symbol, %s, not found", *val)
	}
	return nil, errors.New("enum symbol not provided")
}

func getCache(tCtx TransformContext) pcommon.Map {
	return tCtx.cache
}

func pathExpressionParser(cacheGetter ctxcache.Getter[TransformContext]) ottl.PathExpressionParser[TransformContext] {
	return ctxcommon.PathExpressionParser(
		ctxprofile.Name,
		ctxprofile.DocRef,
		cacheGetter,
		map[string]ottl.PathExpressionParser[TransformContext]{
			ctxresource.Name:    ctxresource.PathGetSetter[TransformContext],
			ctxscope.Name:       ctxscope.PathGetSetter[TransformContext],
			ctxscope.LegacyName: ctxscope.PathGetSetter[TransformContext],
			ctxprofile.Name:     ctxprofile.PathGetSetter[TransformContext],
		})
}
