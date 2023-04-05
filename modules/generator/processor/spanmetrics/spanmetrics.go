package spanmetrics

import (
	"context"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/prometheus/util/strutil"

	gen "github.com/grafana/tempo/modules/generator/processor"
	processor_util "github.com/grafana/tempo/modules/generator/processor/util"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/spanfilter"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	tempo_util "github.com/grafana/tempo/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	metricCallsTotal      = "traces_spanmetrics_calls_total"
	metricDurationSeconds = "traces_spanmetrics_latency"
	metricSizeTotal       = "traces_spanmetrics_size_total"
	targetInfo            = "target_info"
)

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

	// for testing
	now func() time.Time
}

func New(cfg Config, registry registry.Registry, spanDiscardCounter prometheus.Counter) (gen.Processor, error) {
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

	for _, d := range cfg.Dimensions {
		labels = append(labels, sanitizeLabelNameWithCollisions(d))
	}

	for _, m := range cfg.DimensionMappings {
		labels = append(labels, m.Name)
	}
	
	p := &Processor{
		Cfg:                        cfg,
		registry:                   registry,
		spanMetricsTargetInfo:      registry.NewGauge(targetInfo, []string{}),
		now:                        time.Now,
		labels:                     labels,
	}

	if cfg.Subprocessors[Latency] {
		p.spanMetricsDurationSeconds = registry.NewHistogram(metricDurationSeconds, labels, cfg.HistogramBuckets)
	}
	if cfg.Subprocessors[Count] {
		p.spanMetricsCallsTotal = registry.NewCounter(metricCallsTotal, labels)
	}
	if cfg.Subprocessors[Size] {
		p.spanMetricsSizeTotal = registry.NewCounter(metricSizeTotal, labels)
	}

	filter, err := spanfilter.NewSpanFilter(cfg.FilterPolicies)
	if err != nil {
		return nil, err
	}

	p.Cfg = cfg
	p.registry = registry
	p.now = time.Now
	p.filteredSpansCounter = spanDiscardCounter
	p.filter = filter
	return p, nil
}

func (p *Processor) Name() string {
	return Name
}

func (p *Processor) PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) {
	span, _ := opentracing.StartSpanFromContext(ctx, "spanmetrics.PushSpans")
	defer span.Finish()

	p.aggregateMetrics(req.Batches)
}

func (p *Processor) Shutdown(_ context.Context) {
}

func (p *Processor) aggregateMetrics(resourceSpans []*v1_trace.ResourceSpans) {
	for _, rs := range resourceSpans {
		// already extract job name & instance id, so we only have to do it once per batch of spans
		svcName, _ := processor_util.FindServiceName(rs.Resource.Attributes)
		jobName := processor_util.GetJobValue(rs.Resource.Attributes)
		instanceID, _ := processor_util.FindInstanceID(rs.Resource.Attributes)
		resourceLabels := make([]string, 0)
		if p.Cfg.EnableTargetInfo {
			resourceLabels = processor_util.GetTargetInfoAttributes(rs.Resource.Attributes)
		}
		for _, ils := range rs.ScopeSpans {
			for _, span := range ils.Spans {
				if p.filter.ApplyFilterPolicy(rs.Resource, span) {
					p.aggregateMetricsForSpan(svcName, jobName, instanceID, rs.Resource, span, resourceLabels)
					continue
				}
				p.filteredSpansCounter.Inc()
			}
		}
	}
}

func (p *Processor) aggregateMetricsForSpan(svcName string, jobName string, instanceID string, rs *v1.Resource, span *v1_trace.Span, resourceLabels []string) {
	latencySeconds := float64(span.GetEndTimeUnixNano()-span.GetStartTimeUnixNano()) / float64(time.Second.Nanoseconds())

	labelValues := make([]string, 0, 4+len(p.Cfg.Dimensions))
	targetInfoLabelValues := make([]string, 0, len(resourceLabels))
	labels := make([]string, len(p.labels))
	targetInfoLabels := make([]string, len(resourceLabels))
	copy(labels, p.labels)
	copy(targetInfoLabels, resourceLabels)

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
	if jobName != "" {
		labels = append(labels, dimJob)
		labelValues = append(labelValues, jobName)
	}
	//  add instance label only if job is not blank
	if instanceID != "" {
		labels = append(labels, dimInstance)
		labelValues = append(labelValues, instanceID)
	}

	//update labels to add job and/or instance if they're not blank
	p.spanMetricsCallsTotal.UpdateLabels(labels)
	p.spanMetricsDurationSeconds.UpdateLabels(labels)
	p.spanMetricsSizeTotal.UpdateLabels(labels)

	spanMultiplier := processor_util.GetSpanMultiplier(p.Cfg.SpanMultiplierKey, span)

	registryLabelValues := p.registry.NewLabelValues(labelValues)

	if p.Cfg.Subprocessors[Count] {
		p.spanMetricsCallsTotal.Inc(registryLabelValues, 1*spanMultiplier)
	}

	p.spanMetricsSizeTotal.Inc(registryLabelValues, float64(span.Size())*spanMultiplier)

	if p.Cfg.Subprocessors[Latency] {
		p.spanMetricsDurationSeconds.ObserveWithExemplar(registryLabelValues, latencySeconds, tempo_util.TraceIDToHexString(span.TraceId), spanMultiplier)
	}

	if p.Cfg.Subprocessors[Size] {
		p.spanMetricsSizeTotal.Inc(registryLabelValues, float64(span.Size()))
	}

	// update target_info label values
	if p.Cfg.EnableTargetInfo {
		resourceAttributesCount := len(targetInfoLabels)
		for _, label := range targetInfoLabels {
			value, _ := processor_util.FindAttributeValue(label, rs.Attributes, span.Attributes)
			targetInfoLabelValues = append(targetInfoLabelValues, value)
		}

		// add joblabel to target info only if job is not blank
		if jobName != "" && p.Cfg.EnableTargetInfo {
			targetInfoLabels = append(targetInfoLabels, dimJob)
			targetInfoLabelValues = append(targetInfoLabelValues, jobName)
		}
		//  add instance label to target info only if job is not blank
		if instanceID != "" && p.Cfg.EnableTargetInfo {
			targetInfoLabels = append(targetInfoLabels, dimInstance)
			targetInfoLabelValues = append(targetInfoLabelValues, instanceID)
		}

		//update labels to add job and/or instance if they're not blank
		p.spanMetricsTargetInfo.UpdateLabels(targetInfoLabels)

		targetInfoRegistryLabelValues := p.registry.NewLabelValues(targetInfoLabelValues)

		// only register target info if at least (job or instance) AND one other attribute are present
		if resourceAttributesCount > 0 && len(targetInfoLabels) > resourceAttributesCount {
			p.spanMetricsTargetInfo.Set(targetInfoRegistryLabelValues, 1)
		}
	}

}

func sanitizeLabelNameWithCollisions(name string) string {
	sanitized := strutil.SanitizeLabelName(name)

	if isIntrinsicDimension(sanitized) {
		return "__" + sanitized
	}

	return sanitized
}

func isIntrinsicDimension(name string) bool {
	return name == dimJob ||
		name == dimSpanName ||
		name == dimSpanKind ||
		name == dimStatusCode ||
		name == dimStatusMessage ||
		name == dimInstance
}
