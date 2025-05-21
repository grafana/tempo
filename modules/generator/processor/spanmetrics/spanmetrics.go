package spanmetrics

import (
	"context"
	"fmt"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/util/strutil"

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

type sanitizeFn func(string) string

type Processor struct {
	Cfg Config

	registry registry.Registry

	spanMetricsCallsTotal      registry.Counter
	spanMetricsDurationSeconds registry.Histogram
	spanMetricsSizeTotal       registry.Counter
	spanMetricsTargetInfo      registry.Gauge
	labels                     []string

	filter               *spanfilter.SpanFilter
	filteredSpansCounter prometheus.Counter
	invalidUTF8Counter   prometheus.Counter
	sanitizeCache        reclaimable.Cache[string, string]

	// for testing
	now func() time.Time
}

func New(cfg Config, reg registry.Registry, filteredSpansCounter, invalidUTF8Counter prometheus.Counter) (gen.Processor, error) {
	labels := make([]string, 0, 4+len(cfg.Dimensions))

	if cfg.IntrinsicDimensions.Service {
		labels = append(labels, dimService)
	}
	if cfg.IntrinsicDimensions.SpanName {
		labels = append(labels, dimSpanName)
	}
	if cfg.IntrinsicDimensions.SpanKind {
		labels = append(labels, dimSpanKind)
	}
	if cfg.IntrinsicDimensions.StatusCode {
		labels = append(labels, dimStatusCode)
	}
	if cfg.IntrinsicDimensions.StatusMessage {
		labels = append(labels, dimStatusMessage)
	}

	c := reclaimable.New(strutil.SanitizeLabelName, 10000)

	for _, d := range cfg.Dimensions {
		labels = append(labels, SanitizeLabelNameWithCollisions(d, intrinsicLabels, c.Get))
	}

	for _, m := range cfg.DimensionMappings {
		labels = append(labels, SanitizeLabelNameWithCollisions(m.Name, intrinsicLabels, c.Get))
	}

	err := validateLabelValues(labels)
	if err != nil {
		return nil, err
	}

	p := &Processor{
		Cfg:                   cfg,
		registry:              reg,
		spanMetricsTargetInfo: reg.NewGauge(targetInfo),
		now:                   time.Now,
		labels:                labels,
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
	return Name
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
			GetTargetInfoAttributesValues(&resourceLabels, &resourceValues, rs.Resource.Attributes, p.Cfg.TargetInfoExcludedDimensions, intrinsicLabels, p.sanitizeCache.Get)
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

	labelValues := make([]string, 0, 4+len(p.Cfg.Dimensions))
	targetInfoLabelValues := make([]string, len(resourceLabels))
	labels := make([]string, len(p.labels))
	targetInfoLabels := make([]string, len(resourceLabels))
	copy(labels, p.labels)
	copy(targetInfoLabels, resourceLabels)
	copy(targetInfoLabelValues, resourceValues)

	// important: the order of labelValues must correspond to the order of labels / intrinsic dimensions
	if p.Cfg.IntrinsicDimensions.Service {
		labelValues = append(labelValues, svcName)
	}
	if p.Cfg.IntrinsicDimensions.SpanName {
		labelValues = append(labelValues, span.GetName())
	}
	if p.Cfg.IntrinsicDimensions.SpanKind {
		labelValues = append(labelValues, span.GetKind().String())
	}
	if p.Cfg.IntrinsicDimensions.StatusCode {
		labelValues = append(labelValues, span.GetStatus().GetCode().String())
	}
	if p.Cfg.IntrinsicDimensions.StatusMessage {
		labelValues = append(labelValues, span.GetStatus().GetMessage())
	}

	for _, d := range p.Cfg.Dimensions {
		value, _ := processor_util.FindAttributeValue(d, rs.Attributes, span.Attributes)
		labelValues = append(labelValues, value)
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
		labelValues = append(labelValues, values)
	}

	// add job label only if job is not blank
	if jobName != "" && p.Cfg.EnableTargetInfo {
		labels = append(labels, dimJob)
		labelValues = append(labelValues, jobName)
	}
	//  add instance label only if job is not blank
	if instanceID != "" && p.Cfg.EnableTargetInfo {
		labels = append(labels, dimInstance)
		labelValues = append(labelValues, instanceID)
	}

	spanMultiplier := processor_util.GetSpanMultiplier(p.Cfg.SpanMultiplierKey, span, rs)

	err := validateLabelValues(labelValues)
	if err != nil {
		p.invalidUTF8Counter.Inc()
		return
	}

	registryLabelValues := p.registry.NewLabelValueCombo(labels, labelValues)

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
		resourceAttributesCount := len(targetInfoLabels)

		// add joblabel to target info only if job is not blank
		if jobName != "" {
			targetInfoLabels = append(targetInfoLabels, dimJob)
			targetInfoLabelValues = append(targetInfoLabelValues, jobName)
		}
		//  add instance label to target info only if job is not blank
		if instanceID != "" {
			targetInfoLabels = append(targetInfoLabels, dimInstance)
			targetInfoLabelValues = append(targetInfoLabelValues, instanceID)
		}

		targetInfoRegistryLabelValues := p.registry.NewLabelValueCombo(targetInfoLabels, targetInfoLabelValues)

		// only register target info if at least (job or instance) AND one other attribute are present
		// TODO - We can move this check to the top
		if resourceAttributesCount > 0 && len(targetInfoLabels) > resourceAttributesCount {
			p.spanMetricsTargetInfo.SetForTargetInfo(targetInfoRegistryLabelValues, 1)
		}
	}
}

func validateLabelValues(v []string) error {
	for _, value := range v {
		if !utf8.ValidString(value) {
			return fmt.Errorf("invalid utf8 string: %s", value)
		}
	}
	return nil
}

func GetTargetInfoAttributesValues(keys, values *[]string, attributes []*v1_common.KeyValue, exclude, intrinsicLabels []string, sanitizeFn sanitizeFn) {
	// TODO allocate with known length, or take new params for existing buffers
	*keys = (*keys)[:0]
	*values = (*values)[:0]
	for _, attrs := range attributes {
		// ignoring job and instance
		key := attrs.Key
		// Skip empty string keys, which are out of spec but
		// technically possible in the proto. These will cause
		// issues downstream for metrics datasources
		if key == "" {
			continue
		}
		if key != "service.name" && key != "service.namespace" && key != "service.instance.id" && !slices.Contains(exclude, key) {
			*keys = append(*keys, SanitizeLabelNameWithCollisions(key, intrinsicLabels, sanitizeFn))
			value := tempo_util.StringifyAnyValue(attrs.Value)
			*values = append(*values, value)
		}
	}
}

func SanitizeLabelNameWithCollisions(name string, dimensions []string, sansanitizeFn sanitizeFn) string {
	sanitized := sansanitizeFn(name)

	// check if same label as intrinsics
	if slices.Contains(dimensions, sanitized) {
		return "__" + sanitized
	}

	return sanitized
}
