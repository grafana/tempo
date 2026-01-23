package spanmetrics

import (
	"context"
	"slices"
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
)

type Processor struct {
	Cfg Config

	registry registry.Registry

	spanMetricsCallsTotal      registry.Counter
	spanMetricsDurationSeconds registry.Histogram
	spanMetricsSizeTotal       registry.Counter
	spanMetricsTargetInfo      registry.Gauge

	filter               *spanfilter.SpanFilter
	filteredSpansCounter prometheus.Counter
	invalidUTF8Counter   prometheus.Counter
	sanitizeCache        reclaimable.Cache[string, string]

	// for testing
	now func() time.Time
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

	p := &Processor{
		Cfg:                   cfg,
		registry:              reg,
		spanMetricsTargetInfo: reg.NewGauge(targetInfo),
		now:                   time.Now,
		filteredSpansCounter:  filteredSpansCounter,
		invalidUTF8Counter:    invalidUTF8Counter,
		sanitizeCache:         c,
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
	resourceLabels := make([]string, 0)
	resourceValues := make([]string, 0)
	for _, rs := range resourceSpans {
		// already extract job name & instance id, so we only have to do it once per batch of spans
		svcName, _ := processor_util.FindServiceName(rs.Resource.Attributes)
		jobName := processor_util.GetJobValue(rs.Resource.Attributes)
		instanceID, _ := processor_util.FindInstanceID(rs.Resource.Attributes)
		if p.Cfg.EnableTargetInfo {
			getTargetInfoAttributesValues(&resourceLabels, &resourceValues, rs.Resource.Attributes, p.Cfg.TargetInfoExcludedDimensions, p.sanitizeCache.Get)
		}
		for _, ils := range rs.ScopeSpans {
			for _, span := range ils.Spans {
				if p.filter.ApplyFilterPolicy(rs.Resource, span) {
					p.aggregateMetricsForSpan(svcName, jobName, instanceID, rs.Resource, span, resourceLabels, resourceValues)
					continue
				}
				p.filteredSpansCounter.Inc()
			}
		}
	}
}

func (p *Processor) aggregateMetricsForSpan(svcName string, jobName string, instanceID string, rs *v1.Resource, span *v1_trace.Span, resourceLabels []string, resourceValues []string) {
	// Spans with negative latency are treated as zero.
	latencySeconds := 0.0
	if start, end := span.GetStartTimeUnixNano(), span.GetEndTimeUnixNano(); start < end {
		latencySeconds = float64(end-start) / float64(time.Second.Nanoseconds())
	}

	builder := p.registry.NewLabelBuilder()
	targetInfoBuilder := p.registry.NewLabelBuilder()
	for i := range resourceLabels {
		targetInfoBuilder.Add(resourceLabels[i], resourceValues[i])
	}

	if p.Cfg.IntrinsicDimensions.Service {
		builder.Add(gen.DimService, svcName)
	}
	if p.Cfg.IntrinsicDimensions.SpanName {
		builder.Add(gen.DimSpanName, span.GetName())
	}
	if p.Cfg.IntrinsicDimensions.SpanKind {
		builder.Add(gen.DimSpanKind, span.GetKind().String())
	}
	if p.Cfg.IntrinsicDimensions.StatusCode {
		builder.Add(gen.DimStatusCode, span.GetStatus().GetCode().String())
	}
	if p.Cfg.IntrinsicDimensions.StatusMessage {
		builder.Add(gen.DimStatusMessage, span.GetStatus().GetMessage())
	}

	for _, d := range p.Cfg.Dimensions {
		value, _ := processor_util.FindAttributeValue(d, rs.Attributes, span.Attributes)
		label := validation.SanitizeLabelNameWithCollisions(d, validation.SupportedIntrinsicDimensionsSet, p.sanitizeCache.Get)
		builder.Add(label, value)
	}

	for _, m := range p.Cfg.DimensionMappings {
		values := ""
		for _, s := range m.SourceLabel {
			if value, _ := processor_util.FindAttributeValue(s, rs.Attributes, span.Attributes); value != "" {
				if values == "" {
					values += value
				} else {
					values = values + m.Join + value
				}
			}
		}
		label := validation.SanitizeLabelNameWithCollisions(m.Name, validation.SupportedIntrinsicDimensionsSet, p.sanitizeCache.Get)
		builder.Add(label, values)
	}

	// add job label only if job is not blank and target_info is enabled
	if jobName != "" && p.Cfg.EnableTargetInfo {
		builder.Add(gen.DimJob, jobName)
	}
	//  add instance label only if instance is not blank and enabled and target_info is enabled
	if instanceID != "" && p.Cfg.EnableTargetInfo && p.Cfg.EnableInstanceLabel {
		builder.Add(gen.DimInstance, instanceID)
	}

	spanMultiplier := processor_util.GetSpanMultiplier(p.Cfg.SpanMultiplierKey, span, rs)

	registryLabelValues, validUTF8 := builder.CloseAndBuildLabels()
	if !validUTF8 {
		p.invalidUTF8Counter.Inc()
		return
	}

	if p.Cfg.Subprocessors[Count] {
		p.spanMetricsCallsTotal.Inc(registryLabelValues, 1*spanMultiplier)
	}

	if p.Cfg.Subprocessors[Latency] {
		p.spanMetricsDurationSeconds.ObserveWithExemplar(registryLabelValues, latencySeconds, tempo_util.TraceIDToHexString(span.TraceId), spanMultiplier)
	}

	if p.Cfg.Subprocessors[Size] {
		p.spanMetricsSizeTotal.Inc(registryLabelValues, float64(span.Size()))
	}

	// update target_info label values
	if p.Cfg.EnableTargetInfo {
		// TODO - The resource labels only need to be sanitized once
		// TODO - attribute names are stable across applications
		//        so let's cache the result of previous sanitizations
		resourceAttributesCount := len(resourceLabels)

		// add joblabel to target info only if job is not blank
		if jobName != "" {
			targetInfoBuilder.Add(gen.DimJob, jobName)
		}
		//  add instance label to target info only if instance is not blank and enabled
		if instanceID != "" && p.Cfg.EnableInstanceLabel {
			targetInfoBuilder.Add(gen.DimInstance, instanceID)
		}

		targetInfoRegistryLabelValues, validUTF8 := targetInfoBuilder.CloseAndBuildLabels()
		if !validUTF8 {
			p.invalidUTF8Counter.Inc()
			return
		}

		// only register target info if at least (job or instance) AND one other attribute are present
		// TODO - We can move this check to the top
		if resourceAttributesCount > 0 && targetInfoRegistryLabelValues.Len() > resourceAttributesCount {
			p.spanMetricsTargetInfo.SetForTargetInfo(targetInfoRegistryLabelValues, 1)
		}
	}
}

func getTargetInfoAttributesValues(keys, values *[]string, attributes []*v1_common.KeyValue, exclude []string, sanitizeFn validation.SanitizeFn) {
	// TODO allocate with known length, or take new params for existing buffers
	*keys = (*keys)[:0]
	*values = (*values)[:0]
	for _, attrs := range attributes {
		// ignoring job and instance
		key := attrs.Key
		// Skip empty string keys, which are out of spec but
		// technically possible in the proto. These will cause
		// issues downstream for metrics datasources
		if key == "" || (key[0] >= '0' && key[0] <= '9') {
			continue
		}
		if key != "service.name" && key != "service.namespace" && key != "service.instance.id" && !slices.Contains(exclude, key) {
			*keys = append(*keys, validation.SanitizeLabelNameWithCollisions(key, targetInfoIntrinsicLabelsSet, sanitizeFn))
			value := tempo_util.StringifyAnyValue(attrs.Value)
			*values = append(*values, value)
		}
	}
}
