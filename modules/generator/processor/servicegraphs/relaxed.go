package servicegraphs

import (
	"errors"

	"github.com/grafana/tempo/v3/modules/generator/processor/servicegraphs/store"
	processor_util "github.com/grafana/tempo/v3/modules/generator/processor/util"
	tracev1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
)

func (p *Processor) consumeRelaxed(batches []*tracev1.ResourceSpans) error {
	dropped := 0
	for _, rs := range batches {
		service, ok := processor_util.FindServiceName(rs.Resource.Attributes)
		if !ok || service == "" {
			continue
		}
		for _, scope := range rs.ScopeSpans {
			for _, span := range scope.Spans {
				filtered := !p.filter.ApplyFilterPolicy(rs.Resource, span)
				if filtered {
					p.filteredSpansCounter.Inc()
				}
				err := p.relaxedStore.AddSpan(span.TraceId, span.SpanId, span.ParentSpanId, span.Kind, filtered, func(e *store.Edge, serverDimensions map[string]string) {
					e.SetTraceID(span.TraceId)
					e.SpanKind = span.Kind
					e.ClientService = service
					e.ClientLatencySec = spanDurationSec(span)
					e.ClientEndTimeUnixNano = span.EndTimeUnixNano
					e.ServerStartTimeUnixNano = span.StartTimeUnixNano
					e.Failed = p.spanFailed(span)
					e.SpanMultiplier = 1
					if p.usesSpanMultiplier {
						e.SpanMultiplier = processor_util.GetSpanMultiplier(p.Cfg.SpanMultiplierKey, span, rs.Resource, p.Cfg.EnableTraceStateSpanMultiplier)
					}
					p.upsertDimensions(true, e.Dimensions, rs.Resource.Attributes, span.Attributes)
					p.upsertDimensions(false, serverDimensions, rs.Resource.Attributes, span.Attributes)
					p.upsertPeerNode(e, span.Attributes)
					if isClient(span.Kind) {
						p.upsertDatabaseRequest(e, rs.Resource.Attributes, span)
					}
				})
				if errors.Is(err, store.ErrTooManyItems) {
					dropped++
					p.metricDroppedSpans.Inc()
					continue
				}
				if err != nil {
					return err
				}
			}
		}
	}
	if dropped > 0 {
		return &tooManySpansError{droppedSpans: dropped}
	}
	return nil
}
