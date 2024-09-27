package spanmetrics

import (
	"context"
	"time"

	"github.com/prometheus/prometheus/util/strutil"
	"go.opentelemetry.io/otel"

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
	targetInfo            = "traces_target_info"
)

var tracer = otel.Tracer("modules/generator/processor/spanmetrics")

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

func New(cfg Config, reg registry.Registry, spanDiscardCounter prometheus.Counter) (gen.Processor, error) {
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
		labels = append(labels, sanitizeLabelNameWithCollisions(m.Name))
	}

	p := &Processor{
		Cfg:                   cfg,
		registry:              reg,
		spanMetricsTargetInfo: reg.NewGauge(targetInfo),
		now:                   time.Now,
		labels:                labels,
		filteredSpansCounter:  spanDiscardCounter,
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

	p.filteredSpansCounter = spanDiscardCounter
	p.filter = filter
	return p, nil
}

func (p *Processor) Name() string {
	return Name
}

func (p *Processor) PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) {
	_, span := tracer.Start(ctx, "spanmetrics.PushSpans")
	defer span.End()

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
		resourceLabels := make([]string, 0) // TODO move outside the loop and reuse?
		resourceValues := make([]string, 0) // TODO don't allocate unless needed?

		if p.Cfg.EnableTargetInfo {
			resourceLabels, resourceValues = processor_util.GetTargetInfoAttributesValues(rs.Resource.Attributes, p.Cfg.TargetInfoExcludedDimensions)
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

	spanMultiplier := processor_util.GetSpanMultiplier(p.Cfg.SpanMultiplierKey, span)

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
		for index, label := range targetInfoLabels {
			// sanitize label name
			targetInfoLabels[index] = sanitizeLabelNameWithCollisions(label)
		}

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

func sanitizeLabelNameWithCollisions(name string) string {
	sanitized := strutil.SanitizeLabelName(name)

	if isIntrinsicDimension(sanitized) {
		return "__" + sanitized
	}

	return sanitized
}

func isIntrinsicDimension(name string) bool {
	return processor_util.Contains(name, []string{dimJob, dimSpanName, dimSpanKind, dimStatusCode, dimStatusMessage, dimInstance})
}
