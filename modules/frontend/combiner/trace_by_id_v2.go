package combiner

import (
	"fmt"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level" //nolint:all //deprecated
	spanpruningprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/spanpruning"
	"github.com/grafana/tempo/pkg/tempopb"
)

// TraceByIDV2Options holds optional post-processing configuration for the v2 trace combiner.
type TraceByIDV2Options struct {
	// SpanPruningConfig holds processor configuration when span pruning is active. nil = off.
	SpanPruningConfig *spanpruningprocessor.Config
	// Logger is used to log non-fatal pruning errors.
	Logger log.Logger
}

func NewTypedTraceByIDV2(maxBytes int, marshalingFormat api.MarshallingFormat, traceRedactor TraceRedactor, opts TraceByIDV2Options) GRPCCombiner[*tempopb.TraceByIDResponse] {
	return NewTraceByIDV2(maxBytes, marshalingFormat, traceRedactor, opts).(GRPCCombiner[*tempopb.TraceByIDResponse])
}

func NewTraceByIDV2(maxBytes int, marshalingFormat api.MarshallingFormat, traceRedactor TraceRedactor, opts TraceByIDV2Options) Combiner {
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

			// Pruning runs even on a partial trace (see partialTrace/combiner.IsPartialTrace below):
			// reducing the size of an already-oversized partial trace is still valuable, and the
			// resulting summary spans are simply scoped to whatever spans made it into the partial result.
			pruneStatus := spanpruning.StatusNotPruned
			if opts.SpanPruningConfig != nil {
				pruned, status, err := spanpruning.PruneTrace(opts.SpanPruningConfig, traceResult)
				if err != nil {
					if opts.Logger != nil {
						level.Error(opts.Logger).Log("msg", "span pruning failed, returning unpruned trace", "err", err)
					}
				} else {
					traceResult = pruned
					pruneStatus = status
				}
			}

			if metricsCombiner.Metrics.AdditionalMetrics == nil {
				metricsCombiner.Metrics.AdditionalMetrics = map[string]int64{}
			}
			metricsCombiner.Metrics.AdditionalMetrics[tempopb.AdditionalMetricReturnedBytes] = int64(traceResult.Size())

			resp.Trace = traceResult
			resp.Metrics = metricsCombiner.Metrics

			var messages []string
			if partialTrace || combiner.IsPartialTrace() {
				messages = append(messages, fmt.Sprintf("Trace exceeds maximum size of %d bytes, a partial trace is returned", maxBytes))
			}
			switch pruneStatus {
			case spanpruning.StatusPrunedOnWrite:
				messages = append(messages, "Trace was already pruned before it reached Tempo; this is the complete trace Tempo has stored")
			case spanpruning.StatusPrunedOnRead:
				messages = append(messages, "Trace was pruned by Tempo while serving this request; the original spans remain in storage and can be retrieved by querying again with span pruning disabled")
			}
			if len(messages) > 0 {
				resp.Status = tempopb.PartialStatus_PARTIAL
				resp.Message = strings.Join(messages, "; ")
			}

			return resp, nil
		},
		new:     func() *tempopb.TraceByIDResponse { return &tempopb.TraceByIDResponse{} },
		current: &tempopb.TraceByIDResponse{},
	}
	initHTTPCombiner(gc, marshalingFormat)
	return gc
}
