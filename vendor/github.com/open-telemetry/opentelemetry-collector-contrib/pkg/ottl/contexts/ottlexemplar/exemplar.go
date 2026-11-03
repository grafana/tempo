// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlexemplar // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlexemplar"

import (
	"errors"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap/zapcore"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcache"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcommon"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxdatapoint"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxexemplar"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxmetric"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxotelcol"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxresource"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxscope"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/logging"
)

var tcPool = sync.Pool{
	New: func() any {
		return &TransformContext{cache: pcommon.NewMap()}
	},
}

// ContextName is the name of the context for exemplars.
const ContextName = ctxexemplar.Name

var (
	_ ctxresource.Context     = (*TransformContext)(nil)
	_ ctxscope.Context        = (*TransformContext)(nil)
	_ ctxmetric.Context       = (*TransformContext)(nil)
	_ ctxdatapoint.Context    = (*TransformContext)(nil)
	_ ctxexemplar.Context     = (*TransformContext)(nil)
	_ zapcore.ObjectMarshaler = (*TransformContext)(nil)
)

// TransformContext represents an Exemplar and all its hierarchy.
type TransformContext struct {
	resourceMetrics pmetric.ResourceMetrics
	scopeMetrics    pmetric.ScopeMetrics
	metric          pmetric.Metric
	dataPoint       any
	exemplar        pmetric.Exemplar
	cache           pcommon.Map
	externalCache   *pcommon.Map
}

// MarshalLogObject serializes the TransformContext into a zapcore.ObjectEncoder for logging.
func (tCtx *TransformContext) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	err := encoder.AddObject("resource", logging.Resource(tCtx.GetResource()))
	err = errors.Join(err, encoder.AddObject("scope", logging.InstrumentationScope(tCtx.GetInstrumentationScope())))
	err = errors.Join(err, encoder.AddObject("metric", logging.Metric(tCtx.metric)))
	err = errors.Join(err, encoder.AddObject("datapoint", logging.DataPoint(tCtx.dataPoint)))
	err = errors.Join(err, encoder.AddObject("exemplar", logging.Exemplar(tCtx.exemplar)))
	err = errors.Join(err, encoder.AddObject("cache", logging.Map(getCache(tCtx))))
	return err
}

// TransformContextOption represents an option for configuring a TransformContext.
type TransformContextOption func(*TransformContext)

// WithCache sets an external shared cache on the TransformContext.
// When set, the cache is shared across multiple TransformContext instances.
// Experimental: *NOTE* this option is subject to change or removal in the future.
func WithCache(cache *pcommon.Map) TransformContextOption {
	return func(tCtx *TransformContext) {
		if cache != nil {
			tCtx.externalCache = cache
		}
	}
}

// NewTransformContext returns a new TransformContext from a pool of contexts.
// Caller must call TransformContext.Close on the returned TransformContext.
func NewTransformContext(resourceMetrics pmetric.ResourceMetrics, scopeMetrics pmetric.ScopeMetrics, metric pmetric.Metric, dataPoint any, exemplar pmetric.Exemplar, options ...TransformContextOption) *TransformContext {
	tCtx := tcPool.Get().(*TransformContext)
	tCtx.resourceMetrics = resourceMetrics
	tCtx.scopeMetrics = scopeMetrics
	tCtx.metric = metric
	tCtx.dataPoint = dataPoint
	tCtx.exemplar = exemplar
	for _, opt := range options {
		opt(tCtx)
	}
	return tCtx
}

// Close returns the TransformContext to the pool.
// After this function returns this instance cannot be used.
func (tCtx *TransformContext) Close() {
	tCtx.resourceMetrics = pmetric.ResourceMetrics{}
	tCtx.scopeMetrics = pmetric.ScopeMetrics{}
	tCtx.metric = pmetric.Metric{}
	tCtx.dataPoint = nil
	tCtx.exemplar = pmetric.Exemplar{}
	tCtx.cache.Clear()
	tCtx.externalCache = nil
	tcPool.Put(tCtx)
}

// GetExemplar returns the exemplar from the TransformContext.
func (tCtx *TransformContext) GetExemplar() pmetric.Exemplar {
	return tCtx.exemplar
}

// GetDataPoint returns the parent datapoint from the TransformContext.
func (tCtx *TransformContext) GetDataPoint() any {
	return tCtx.dataPoint
}

// GetMetric returns the metric from the TransformContext.
func (tCtx *TransformContext) GetMetric() pmetric.Metric {
	return tCtx.metric
}

// GetInstrumentationScope returns the instrumentation scope from the TransformContext.
func (tCtx *TransformContext) GetInstrumentationScope() pcommon.InstrumentationScope {
	return tCtx.scopeMetrics.Scope()
}

// GetResource returns the resource from the TransformContext.
func (tCtx *TransformContext) GetResource() pcommon.Resource {
	return tCtx.resourceMetrics.Resource()
}

// GetScopeSchemaURLItem returns the schema URL item for the scope from the TransformContext.
func (tCtx *TransformContext) GetScopeSchemaURLItem() ottl.SchemaURLItem {
	return tCtx.scopeMetrics
}

// GetResourceSchemaURLItem returns the schema URL item for the resource from the TransformContext.
func (tCtx *TransformContext) GetResourceSchemaURLItem() ottl.SchemaURLItem {
	return tCtx.resourceMetrics
}

// GetMetrics returns the metric slice from the TransformContext.
func (tCtx *TransformContext) GetMetrics() pmetric.MetricSlice {
	return tCtx.scopeMetrics.Metrics()
}

// EnablePathContextNames enables the support for path's context names on statements.
// When this option is configured, all statement's paths must have a valid context prefix,
// otherwise an error is reported.
func EnablePathContextNames() ottl.Option[*TransformContext] {
	return func(p *ottl.Parser[*TransformContext]) {
		ottl.WithPathContextNames[*TransformContext]([]string{
			ctxexemplar.Name,
			ctxdatapoint.Name,
			ctxresource.Name,
			ctxscope.LegacyName,
			ctxscope.Name,
			ctxmetric.Name,
			ctxotelcol.Name,
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

// NewParser creates a new exemplar parser with the provided functions and options.
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

func parseEnum(_ *ottl.EnumSymbol) (*ottl.Enum, error) {
	return nil, errors.New("exemplar context does not support enums")
}

func getCache(tCtx *TransformContext) pcommon.Map {
	if tCtx.externalCache != nil {
		return *tCtx.externalCache
	}
	return tCtx.cache
}

func pathExpressionParser(cacheGetter ctxcache.Getter[*TransformContext]) ottl.PathExpressionParser[*TransformContext] {
	return ctxcommon.PathExpressionParser(
		ctxexemplar.Name,
		ctxexemplar.DocRef,
		cacheGetter,
		map[string]ottl.PathExpressionParser[*TransformContext]{
			ctxresource.Name:    ctxresource.PathGetSetter[*TransformContext],
			ctxscope.Name:       ctxscope.PathGetSetter[*TransformContext],
			ctxscope.LegacyName: ctxscope.PathGetSetter[*TransformContext],
			ctxmetric.Name:      ctxmetric.PathGetSetter[*TransformContext],
			ctxdatapoint.Name:   ctxdatapoint.PathGetSetter[*TransformContext],
			ctxexemplar.Name:    ctxexemplar.PathGetSetter[*TransformContext],
			ctxotelcol.Name:     ctxotelcol.PathGetSetter[*TransformContext],
		},
	)
}
