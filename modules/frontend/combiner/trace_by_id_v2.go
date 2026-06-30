package combiner

import (
	"fmt"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
)

// TraceFilter runs on a complete trace and applies a filter to it.
type TraceFilter interface {
	Process(t *tempopb.Trace) (*tempopb.Trace, error)
}

func NewTypedTraceByIDV2(maxBytes int, marshalingFormat api.MarshallingFormat, traceRedactor TraceRedactor, traceFilter TraceFilter) GRPCCombiner[*tempopb.TraceByIDResponse] {
	return NewTraceByIDV2(maxBytes, marshalingFormat, traceRedactor, traceFilter).(GRPCCombiner[*tempopb.TraceByIDResponse])
}

func NewTraceByIDV2(maxBytes int, marshalingFormat api.MarshallingFormat, traceRedactor TraceRedactor, traceFilter TraceFilter) Combiner {
	combiner := trace.NewCombiner(maxBytes, true)
	var partialTrace bool
	metricsCombiner := NewTraceByIDMetricsCombiner()
	gc := &genericCombiner[*tempopb.TraceByIDResponse]{
		combine: func(partial *tempopb.TraceByIDResponse, _ *tempopb.TraceByIDResponse, pipelineResp PipelineResponse) error {
			if partial.Status == tempopb.PartialStatus_PARTIAL {
				partialTrace = true
			}

			metricsCombiner.Combine(partial.Metrics, pipelineResp)

			_, err := combiner.Consume(partial.Trace)
			return err
		},
		finalize: func(resp *tempopb.TraceByIDResponse) (*tempopb.TraceByIDResponse, error) {
			traceResult, _ := combiner.Result()
			if traceResult == nil {
				traceResult = &tempopb.Trace{}
			}

			// dedupe duplicate span ids
			deduper := newDeduper()
			traceResult = deduper.dedupe(traceResult)
			if traceRedactor != nil {
				err := traceRedactor.RedactTraceAttributes(traceResult)
				if err != nil {
					return nil, err
				}
			}

			// filter after redaction so it runs on the final, fully redacted trace.
			if traceFilter != nil {
				filtered, err := traceFilter.Process(traceResult)
				if err != nil {
					return nil, err
				}
				traceResult = filtered
			}

			resp.Trace = traceResult
			// metrics report bytes inspected to pull the whole trace; filtering only trims output, so they are unchanged.
			resp.Metrics = metricsCombiner.Metrics

			if partialTrace || combiner.IsPartialTrace() {
				resp.Status = tempopb.PartialStatus_PARTIAL
				resp.Message = fmt.Sprintf("Trace exceeds maximum size of %d bytes, a partial trace is returned", maxBytes)
			}

			return resp, nil
		},
		new:     func() *tempopb.TraceByIDResponse { return &tempopb.TraceByIDResponse{} },
		current: &tempopb.TraceByIDResponse{},
	}
	initHTTPCombiner(gc, marshalingFormat)
	return gc
}
