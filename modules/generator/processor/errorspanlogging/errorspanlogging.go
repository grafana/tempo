package errorspanlogging

import (
	"context"
	"time"

	"github.com/go-kit/log"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/prometheus/util/strutil"

	gen "github.com/grafana/tempo/modules/generator/processor"
	processor_util "github.com/grafana/tempo/modules/generator/processor/util"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	tempo_util "github.com/grafana/tempo/pkg/util"
)

type Processor struct {
	Cfg Config

	logger log.Logger

	// for testing
	now func() time.Time
}

func New(cfg Config, logger log.Logger) gen.Processor {
	return &Processor{
		Cfg:    cfg,
		logger: logger,
		now:    time.Now,
	}
}

func (p *Processor) Name() string { return Name }

func (p *Processor) PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) {
	span, _ := opentracing.StartSpanFromContext(ctx, "errorspanlogging.PushSpans")
	defer span.Finish()

	p.logIfSpanHasError(req.Batches)
}

func (p *Processor) Shutdown(_ context.Context) {
}

func (p *Processor) logIfSpanHasError(resourceSpans []*v1_trace.ResourceSpans) {
	for _, rs := range resourceSpans {
		// already extract service name, and resources attributes, so we only have to do it once per batch of spans
		svcName, _ := processor_util.FindServiceName(rs.Resource.Attributes)
		logger := p.logger
		for _, a := range rs.Resource.GetAttributes() {
			logger = log.With(
				logger,
				"span_"+strutil.SanitizeLabelName(a.GetKey()),
				tempo_util.StringifyAnyValue(a.GetValue()))
		}

		for _, ils := range rs.InstrumentationLibrarySpans {
			for _, span := range ils.Spans {
				if span.GetStatus().GetCode() != v1_trace.Status_STATUS_CODE_ERROR {
					continue
				}
				p.logSpan(svcName, span, logger)
			}
		}
	}
}

func (p *Processor) logSpan(svcName string, span *v1_trace.Span, logger log.Logger) {
	latencySeconds := float64(span.GetEndTimeUnixNano()-span.GetStartTimeUnixNano()) / float64(time.Second.Nanoseconds())

	logger = log.With(
		logger,
		"span_service_name", svcName,
		"span_name", span.GetName(),
		"span_kind", span.GetKind().String(),
		"span_status", span.GetStatus().GetCode().String(),
		"span_duration_seconds", latencySeconds,
		"span_trace_id", tempo_util.TraceIDToHexString(span.TraceId))

	for _, a := range span.GetAttributes() {
		logger = log.With(
			logger,
			"span_"+strutil.SanitizeLabelName(a.GetKey()),
			tempo_util.StringifyAnyValue(a.GetValue()))
	}

	logger.Log("msg", "error_spans_received")
}
