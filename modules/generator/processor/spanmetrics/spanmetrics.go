package spanmetrics

import (
	"context"
	"sync"
	"time"

	"github.com/grafana/tempo/modules/generator/validation"
	"github.com/prometheus/client_golang/prometheus"

	gen "github.com/grafana/tempo/modules/generator/processor"
	processor_util "github.com/grafana/tempo/modules/generator/processor/util"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/cache/reclaimable"
	"github.com/grafana/tempo/pkg/spanfilter"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	tempo_util "github.com/grafana/tempo/pkg/util"
)

const (
	metricCallsTotal      = "traces_spanmetrics_calls_total"
	metricDurationSeconds = "traces_spanmetrics_latency"
	metricSizeTotal       = "traces_spanmetrics_size_total"
	targetInfo            = "traces_target_info"

	serviceNameKey       = "service.name"
	serviceNamespaceKey  = "service.namespace"
	serviceInstanceIDKey = "service.instance.id"
)

type Processor struct {
	Cfg Config

	registry registry.Registry

	spanMetricsCallsTotal      registry.Counter
	spanMetricsDurationSeconds registry.Histogram
	spanMetricsSizeTotal       registry.Counter
	spanMetricsTargetInfo      registry.Gauge

	filter                 *spanfilter.SpanFilter
	filteredSpansCounter   prometheus.Counter
	invalidUTF8Counter     prometheus.Counter
	sanitizeCache          reclaimable.Cache[string, string]
	targetInfoExcluded     map[string]struct{}
	dimensionLabels        []string
	dimensionMappingLabels []string
	usesSpanMultiplier     bool

	// sampler is nil unless per-series sampling is configured.
	sampler *seriesSampler
	// scratchPool holds the per-push buffers the sampled path uses. Only
	// touched when sampler is non-nil, so the unsampled path is unaffected.
	scratchPool sync.Pool
	// mappingSourceOffsets[i]:mappingSourceOffsets[i+1] indexes the resolved
	// source values of Cfg.DimensionMappings[i] inside spanScratch.mappings.
	mappingSourceOffsets []int

	// for testing
	now func() time.Time
}

// spanScratch holds the buffers the sampled path needs for one push. The
// sampling key and the label set read the same resolved attribute values, so
// they are resolved once into here and shared. Pooled because PushSpans may run
// concurrently for a single tenant.
type spanScratch struct {
	key        samplerKey
	dimensions []string
	mappings   []string
}

func New(cfg Config, reg registry.Registry, filteredSpansCounter, invalidUTF8Counter prometheus.Counter) (gen.Processor, error) {
	var configuredIntrinsicDimensions []string

	if cfg.IntrinsicDimensions.Service {
		configuredIntrinsicDimensions = append(configuredIntrinsicDimensions, gen.DimService)
	}
	if cfg.IntrinsicDimensions.SpanName {
		configuredIntrinsicDimensions = append(configuredIntrinsicDimensions, gen.DimSpanName)
	}
	if cfg.IntrinsicDimensions.SpanKind {
		configuredIntrinsicDimensions = append(configuredIntrinsicDimensions, gen.DimSpanKind)
	}
	if cfg.IntrinsicDimensions.StatusCode {
		configuredIntrinsicDimensions = append(configuredIntrinsicDimensions, gen.DimStatusCode)
	}
	if cfg.IntrinsicDimensions.StatusMessage {
		configuredIntrinsicDimensions = append(configuredIntrinsicDimensions, gen.DimStatusMessage)
	}

	c := reclaimable.New(validation.SanitizeLabelName, 10000)

	err := validation.ValidateDimensions(cfg.Dimensions, configuredIntrinsicDimensions, cfg.DimensionMappings, c.Get)
	if err != nil {
		return nil, err
	}

	dimensionLabels := make([]string, len(cfg.Dimensions))
	for i, d := range cfg.Dimensions {
		dimensionLabels[i] = validation.SanitizeLabelNameWithCollisions(d, validation.SupportedIntrinsicDimensionsSet, c.Get)
	}

	dimensionMappingLabels := make([]string, len(cfg.DimensionMappings))
	for i, m := range cfg.DimensionMappings {
		dimensionMappingLabels[i] = validation.SanitizeLabelNameWithCollisions(m.Name, validation.SupportedIntrinsicDimensionsSet, c.Get)
	}

	mappingSourceOffsets := make([]int, len(cfg.DimensionMappings)+1)
	for i, m := range cfg.DimensionMappings {
		mappingSourceOffsets[i+1] = mappingSourceOffsets[i] + len(m.SourceLabel)
	}

	p := &Processor{
		Cfg:                    cfg,
		registry:               reg,
		spanMetricsTargetInfo:  reg.NewGauge(targetInfo),
		now:                    time.Now,
		filteredSpansCounter:   filteredSpansCounter,
		invalidUTF8Counter:     invalidUTF8Counter,
		sanitizeCache:          c,
		targetInfoExcluded:     excludedDimensionsSet(cfg.TargetInfoExcludedDimensions),
		dimensionLabels:        dimensionLabels,
		dimensionMappingLabels: dimensionMappingLabels,
		usesSpanMultiplier:     cfg.SpanMultiplierKey != "" || cfg.EnableTraceStateSpanMultiplier,
		sampler:                newSeriesSampler(cfg.MaxSpansPerSeriesPerSecond),
		mappingSourceOffsets:   mappingSourceOffsets,
	}
	p.scratchPool.New = func() any {
		return &spanScratch{
			dimensions: make([]string, len(cfg.Dimensions)),
			mappings:   make([]string, mappingSourceOffsets[len(mappingSourceOffsets)-1]),
		}
	}

	if cfg.Subprocessors[Latency] {
		p.spanMetricsDurationSeconds = reg.NewHistogram(metricDurationSeconds, cfg.HistogramBuckets, cfg.HistogramOverride)
	}
	if cfg.Subprocessors[Count] {
		p.spanMetricsCallsTotal = reg.NewCounter(metricCallsTotal)
	}
	if cfg.Subprocessors[Size] {
		p.spanMetricsSizeTotal = reg.NewCounter(metricSizeTotal)
	}

	filter, err := spanfilter.NewSpanFilter(cfg.FilterPolicies)
	if err != nil {
		return nil, err
	}

	p.filter = filter
	return p, nil
}

func (p *Processor) Name() string {
	return gen.SpanMetricsName
}

func (p *Processor) PushSpans(_ context.Context, req *tempopb.PushSpansRequest) {
	p.aggregateMetrics(req.Batches)
}

func (p *Processor) Shutdown(_ context.Context) {
}

func (p *Processor) aggregateMetrics(resourceSpans []*v1_trace.ResourceSpans) {
	// One timestamp per push keeps time.Now off the per-span path; lastUpdated
	// feeds staleness checks on the order of minutes, so per-batch granularity
	// is more than enough.
	updateTimeMs := p.now().UnixMilli()

	// One scratch per push, not per span: the sampled path reuses its buffers
	// across every span in the request.
	var scratch *spanScratch
	if p.sampler != nil {
		scratch = p.scratchPool.Get().(*spanScratch)
		defer p.scratchPool.Put(scratch)
	}

	for _, rs := range resourceSpans {
		// already extract job name & instance id, so we only have to do it once per batch of spans
		svcName, jobName, instanceID := processor_util.FindServiceLabels(rs.Resource.Attributes)
		targetInfoLabelsValid := true
		targetInfoLabelsBuilt := false
		for _, ils := range rs.ScopeSpans {
			for _, span := range ils.Spans {
				if !p.filter.ApplyFilterPolicy(rs.Resource, span) {
					p.filteredSpansCounter.Inc()
					continue
				}
				if !p.aggregateMetricsForSpan(scratch, svcName, jobName, instanceID, rs.Resource, span, updateTimeMs) {
					continue
				}
				if p.Cfg.EnableTargetInfo {
					// Register target_info only after a span's primary labels have been
					// validated and its series updated. This preserves the pre-optimization
					// ordering: span metrics claim active-series/entity limiter capacity
					// before target_info, and a resource whose accepted spans all carry
					// invalid UTF-8 labels emits no target_info at all. The labels are
					// still built and registered at most once per resource batch.
					if !targetInfoLabelsBuilt {
						targetInfoLabelsValid = p.buildAndSetTargetInfoLabels(rs.Resource.Attributes, jobName, instanceID, updateTimeMs)
						targetInfoLabelsBuilt = true
					}
					if !targetInfoLabelsValid {
						// Pre-optimization behavior: every span that reached the target_info
						// step with invalid target_info labels counted one discarded update.
						p.invalidUTF8Counter.Inc()
					}
				}
			}
		}
	}
}

// aggregateMetricsForSpan updates the enabled span metric series for a single
// span. It reports whether the span's primary label set was valid UTF-8;
// callers gate target_info registration on this, independent of which
// subprocessors are enabled.
//
// scratch is non-nil exactly when per-series sampling is enabled, in which case
// a span the sampler skips returns early. Skipped spans report true: their
// label set was never built, so nothing is known about its UTF-8 validity, and
// reporting false would suppress target_info for the whole resource batch.
// target_info is per resource rather than per span and is never sampled.
func (p *Processor) aggregateMetricsForSpan(scratch *spanScratch, svcName string, jobName string, instanceID string, rs *v1.Resource, span *v1_trace.Span, updateTimeMs int64) bool {
	samplingMultiplier := 1.0
	if scratch != nil {
		samplingMultiplier = p.sampleSpan(scratch, svcName, jobName, instanceID, rs, span, updateTimeMs)
		if samplingMultiplier == 0 {
			return true
		}
	}

	builder := p.registry.NewLabelBuilder()

	if p.Cfg.IntrinsicDimensions.Service {
		builder.Add(gen.DimService, svcName)
	}
	if p.Cfg.IntrinsicDimensions.SpanName {
		builder.Add(gen.DimSpanName, span.GetName())
	}
	if p.Cfg.IntrinsicDimensions.SpanKind {
		builder.Add(gen.DimSpanKind, spanKindString(span.GetKind()))
	}
	if p.Cfg.IntrinsicDimensions.StatusCode {
		builder.Add(gen.DimStatusCode, statusCodeString(span.GetStatus().GetCode()))
	}
	if p.Cfg.IntrinsicDimensions.StatusMessage {
		builder.Add(gen.DimStatusMessage, span.GetStatus().GetMessage())
	}

	// The sampled path already resolved every dimension value to build the
	// sampling key, so it reads them back instead of scanning the attributes a
	// second time. If there is a collision, for example deployment.environment
	// and deployment_environment, both sanitized to deployment_environment, we
	// just take the last one configured.
	if scratch != nil {
		for i := range p.Cfg.Dimensions {
			builder.Add(p.dimensionLabels[i], scratch.dimensions[i])
		}
	} else {
		for i, d := range p.Cfg.Dimensions {
			value, _ := processor_util.FindAttributeValue(d, rs.Attributes, span.Attributes)
			builder.Add(p.dimensionLabels[i], value)
		}
	}

	for i, m := range p.Cfg.DimensionMappings {
		// Plain concatenation: source lists are short (1-3 entries), where
		// strings.Builder allocates more than simple concatenation.
		values := ""
		for j, source := range m.SourceLabel {
			value := ""
			if scratch != nil {
				value = scratch.mappings[p.mappingSourceOffsets[i]+j]
			} else {
				value, _ = processor_util.FindAttributeValue(source, rs.Attributes, span.Attributes)
			}
			if value != "" {
				if values == "" {
					values = value
				} else {
					values = values + m.Join + value
				}
			}
		}
		builder.Add(p.dimensionMappingLabels[i], values)
	}

	// add job label only if job is not blank and target_info is enabled
	if jobName != "" && p.Cfg.EnableTargetInfo {
		builder.Add(gen.DimJob, jobName)
	}
	// add instance label only if instance is not blank and enabled and target_info is enabled
	if instanceID != "" && p.Cfg.EnableTargetInfo && p.Cfg.EnableInstanceLabel {
		builder.Add(gen.DimInstance, instanceID)
	}

	spanMultiplier := samplingMultiplier
	if p.usesSpanMultiplier {
		spanMultiplier *= processor_util.GetSpanMultiplier(p.Cfg.SpanMultiplierKey, span, rs, p.Cfg.EnableTraceStateSpanMultiplier)
	}

	registryLabelValues, validUTF8 := builder.CloseAndBorrowLabels()
	if !validUTF8 {
		p.invalidUTF8Counter.Inc()
		return false
	}
	defer registryLabelValues.Release()

	if p.Cfg.Subprocessors[Count] {
		p.spanMetricsCallsTotal.IncBorrowed(registryLabelValues, 1*spanMultiplier, updateTimeMs)
	}

	if p.Cfg.Subprocessors[Latency] {
		// Spans with negative latency are treated as zero.
		latencySeconds := 0.0
		if start, end := span.GetStartTimeUnixNano(), span.GetEndTimeUnixNano(); start < end {
			latencySeconds = float64(end-start) / float64(time.Second.Nanoseconds())
		}
		p.spanMetricsDurationSeconds.ObserveBorrowed(registryLabelValues, latencySeconds, span.TraceId, spanMultiplier, updateTimeMs)
	}

	if p.Cfg.Subprocessors[Size] {
		// size_total deliberately ignores the span multiplier, but it still has
		// to be scaled by the sampling multiplier: the skipped spans' bytes are
		// never observed anywhere else.
		p.spanMetricsSizeTotal.IncBorrowed(registryLabelValues, float64(span.Size())*samplingMultiplier, updateTimeMs)
	}

	return true
}

// sampleSpan resolves the values that decide the span's series into scratch and
// asks the sampler for this span's multiplier, 0 meaning skip. The resolved
// values are left in scratch for aggregateMetricsForSpan to reuse.
//
// The components mirror the label set built by aggregateMetricsForSpan exactly.
// Keying on more than the label set would hand each series several budgets;
// keying on less would make several series share one, and neither is what the
// per-series accuracy target asks for.
func (p *Processor) sampleSpan(scratch *spanScratch, svcName string, jobName string, instanceID string, rs *v1.Resource, span *v1_trace.Span, updateTimeMs int64) float64 {
	p.buildSamplerKey(scratch, svcName, jobName, instanceID, rs, span)
	return p.sampler.sample(scratch.key.sum(), updateTimeMs)
}

// buildSamplerKey fills scratch.key with the span's series-defining values and
// scratch.dimensions/mappings with the attribute values it resolved on the way.
func (p *Processor) buildSamplerKey(scratch *spanScratch, svcName string, jobName string, instanceID string, rs *v1.Resource, span *v1_trace.Span) {
	key := &scratch.key
	key.reset()

	if p.Cfg.IntrinsicDimensions.Service {
		key.addString(svcName)
	}
	if p.Cfg.IntrinsicDimensions.SpanName {
		key.addString(span.GetName())
	}
	if p.Cfg.IntrinsicDimensions.SpanKind {
		key.addString(spanKindString(span.GetKind()))
	}
	if p.Cfg.IntrinsicDimensions.StatusCode {
		key.addString(statusCodeString(span.GetStatus().GetCode()))
	}
	if p.Cfg.IntrinsicDimensions.StatusMessage {
		key.addString(span.GetStatus().GetMessage())
	}

	for i, d := range p.Cfg.Dimensions {
		value, _ := processor_util.FindAttributeValue(d, rs.Attributes, span.Attributes)
		scratch.dimensions[i] = value
		key.addString(value)
	}

	// The mapping label is the join of its source values, but the sources
	// discriminate series just as well and hashing them avoids building the
	// joined string for a span that is about to be skipped.
	for i, m := range p.Cfg.DimensionMappings {
		for j, source := range m.SourceLabel {
			value, _ := processor_util.FindAttributeValue(source, rs.Attributes, span.Attributes)
			scratch.mappings[p.mappingSourceOffsets[i]+j] = value
			key.addString(value)
		}
	}

	if jobName != "" && p.Cfg.EnableTargetInfo {
		key.addString(jobName)
	}
	if instanceID != "" && p.Cfg.EnableTargetInfo && p.Cfg.EnableInstanceLabel {
		key.addString(instanceID)
	}
}

func (p *Processor) buildAndSetTargetInfoLabels(attributes []*v1_common.KeyValue, jobName string, instanceID string, updateTimeMs int64) bool {
	targetInfoBuilder := p.registry.NewInfoMetricLabelBuilder()
	for _, attr := range attributes {
		key := attr.Key
		// Skip empty string keys, which are out of spec but technically
		// possible in the proto, and keys starting with a digit. These will
		// cause issues downstream for metrics datasources.
		if key == "" || (key[0] >= '0' && key[0] <= '9') {
			continue
		}
		// The service.* attributes are represented by the job and instance
		// labels instead.
		if key == serviceNameKey || key == serviceNamespaceKey || key == serviceInstanceIDKey {
			continue
		}
		if _, excluded := p.targetInfoExcluded[key]; excluded {
			continue
		}
		targetInfoBuilder.Add(
			validation.SanitizeLabelNameWithCollisions(key, targetInfoIntrinsicLabelsSet, p.sanitizeCache.Get),
			tempo_util.StringifyAnyValue(attr.Value),
		)
	}

	identifyingLabels := 0
	// add job label to target info only if job is not blank
	if jobName != "" {
		targetInfoBuilder.Add(gen.DimJob, jobName)
		identifyingLabels++
	}
	// add instance label to target info only if instance is not blank and enabled
	if instanceID != "" && p.Cfg.EnableInstanceLabel {
		targetInfoBuilder.Add(gen.DimInstance, instanceID)
		identifyingLabels++
	}

	targetInfoLabels, validUTF8 := targetInfoBuilder.CloseAndBorrowLabels()
	if !validUTF8 {
		return false
	}
	defer targetInfoLabels.Release()
	// Only register target_info if it has at least (job or instance) AND one other
	// resource attribute in the built label set. We count from the built set because
	// the registry label builder drops empty-valued labels (Add(name, "") deletes name).
	if identifyingLabels > 0 && targetInfoLabels.Labels.Len() > identifyingLabels {
		p.spanMetricsTargetInfo.SetForTargetInfoBorrowed(targetInfoLabels, 1, updateTimeMs)
	}
	return true
}

func excludedDimensionsSet(exclude []string) map[string]struct{} {
	if len(exclude) == 0 {
		return nil
	}
	m := make(map[string]struct{}, len(exclude))
	for _, dim := range exclude {
		m[dim] = struct{}{}
	}
	return m
}

// spanKindString avoids the proto enum-name map lookup that Span_SpanKind.String
// performs on the per-span hot path. The cases must mirror the
// v1_trace.Span_SpanKind enum; unknown values fall back to the generated String.
func spanKindString(kind v1_trace.Span_SpanKind) string {
	switch kind {
	case v1_trace.Span_SPAN_KIND_UNSPECIFIED:
		return "SPAN_KIND_UNSPECIFIED"
	case v1_trace.Span_SPAN_KIND_INTERNAL:
		return "SPAN_KIND_INTERNAL"
	case v1_trace.Span_SPAN_KIND_SERVER:
		return "SPAN_KIND_SERVER"
	case v1_trace.Span_SPAN_KIND_CLIENT:
		return "SPAN_KIND_CLIENT"
	case v1_trace.Span_SPAN_KIND_PRODUCER:
		return "SPAN_KIND_PRODUCER"
	case v1_trace.Span_SPAN_KIND_CONSUMER:
		return "SPAN_KIND_CONSUMER"
	default:
		return kind.String()
	}
}

// statusCodeString avoids the proto enum-name map lookup that
// Status_StatusCode.String performs on the per-span hot path. The cases must
// mirror the v1_trace.Status_StatusCode enum; unknown values fall back to the
// generated String.
func statusCodeString(code v1_trace.Status_StatusCode) string {
	switch code {
	case v1_trace.Status_STATUS_CODE_UNSET:
		return "STATUS_CODE_UNSET"
	case v1_trace.Status_STATUS_CODE_OK:
		return "STATUS_CODE_OK"
	case v1_trace.Status_STATUS_CODE_ERROR:
		return "STATUS_CODE_ERROR"
	default:
		return code.String()
	}
}
