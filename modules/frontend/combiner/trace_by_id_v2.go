package combiner

import (
	"fmt"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level" //nolint:all //deprecated
	spanpruningprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/spanpruning"
	"github.com/grafana/tempo/pkg/tempopb"
)

// TraceFilter filters an assembled trace. Process must not mutate its input, which may be cached.
type TraceFilter interface {
	Process(t *tempopb.Trace) (*tempopb.Trace, error)
}

// TraceByIDV2Options holds optional post-processing configuration for the v2 trace combiner.
type TraceByIDV2Options struct {
	// TraceFilter is applied first (e.g. a TraceQL spanset filter). nil = no filtering.
	TraceFilter TraceFilter
	// SpanPruningConfig is applied after TraceFilter. nil = no pruning.
	SpanPruningConfig *spanpruningprocessor.Config
	// Logger is used to log non-fatal errors such as pruning failures.
	Logger log.Logger
}

// TraceByIDV2Combiner is the concrete combiner type returned by NewTypedTraceByIDV2.
// It exposes MetricsCombiner so callers can read accumulated metrics without
// triggering a second finalize call via GRPCFinal.
type TraceByIDV2Combiner struct {
	GRPCCombiner[*tempopb.TraceByIDResponse]
	MetricsCombiner *TraceByIDMetricsCombiner
}

func NewTypedTraceByIDV2(maxBytes int, marshalingFormat api.MarshallingFormat, traceRedactor TraceRedactor, opts TraceByIDV2Options) *TraceByIDV2Combiner {
	metricsCombiner := NewTraceByIDMetricsCombiner()
	gc := NewTraceByIDV2WithMetrics(maxBytes, marshalingFormat, traceRedactor, opts, metricsCombiner)
	return &TraceByIDV2Combiner{
		GRPCCombiner:    gc.(GRPCCombiner[*tempopb.TraceByIDResponse]),
		MetricsCombiner: metricsCombiner,
	}
}

func NewTraceByIDV2(maxBytes int, marshalingFormat api.MarshallingFormat, traceRedactor TraceRedactor, opts TraceByIDV2Options) Combiner {
	return NewTraceByIDV2WithMetrics(maxBytes, marshalingFormat, traceRedactor, opts, NewTraceByIDMetricsCombiner())
}

func NewTraceByIDV2WithMetrics(maxBytes int, marshalingFormat api.MarshallingFormat, traceRedactor TraceRedactor, opts TraceByIDV2Options, metricsCombiner *TraceByIDMetricsCombiner) Combiner {
	combiner := trace.NewCombiner(maxBytes, true)
	var partialTrace bool
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

			// TraceQL spanset filter runs before pruning so pruning operates on the already-filtered set.
			if opts.TraceFilter != nil {
				filtered, err := opts.TraceFilter.Process(traceResult)
				if err != nil {
					return nil, err
				}
				traceResult = filtered
			}

			// Span pruning runs last: it collapses repetitive leaf spans in whatever trace remains.
			if opts.SpanPruningConfig != nil {
				pruned, err := spanpruning.PruneTrace(opts.SpanPruningConfig, traceResult)
				if err != nil {
					if opts.Logger != nil {
						level.Error(opts.Logger).Log("msg", "span pruning failed, returning unpruned trace", "err", err)
					}
				} else {
					traceResult = pruned
				}
			}

			resp.Trace = traceResult
			// metrics report bytes inspected to pull the whole trace; filtering/pruning only trims output, so they are unchanged.
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
