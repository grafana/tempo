package traceql

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
)

// generateFakeSearchResponse creates a fake SearchResponse with a chance defined by given probability.
// It must be used only for testing purposes.
func generateFakeSearchResponse(probability float64) *tempopb.SearchResponse {
	if probability <= 0 {
		return &tempopb.SearchResponse{}
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // G404
	if probability < 1 && r.Float64() > probability {    //nolint:gosec // G404
		return &tempopb.SearchResponse{}
	}

	numTraces := 1 + r.Intn(3)
	traces := make([]*tempopb.TraceSearchMetadata, numTraces)

	for i := range numTraces {
		traceID := fmt.Sprintf("%016x%016x", r.Int63(), r.Int63())
		startTime := time.Now().Add(-time.Duration(r.Intn(3600)) * time.Second).UnixNano()
		duration := uint32(100 + r.Intn(900)) // 100-1000ms

		numSpans := 1 + r.Intn(5) // 1-5 spans
		spans := make([]*tempopb.Span, numSpans)

		for k := range numSpans {
			spanID := fmt.Sprintf("%016x", r.Int63())
			spanStartTime := uint64(startTime) + uint64(r.Intn(int(duration)))*1000000 // convert ms to ns
			spanDuration := uint64(r.Intn(100)) * 1000000                              // 0-100ms in nanoseconds

			numAttrs := 1 + r.Intn(3) // 1-3 attributes per span
			attrs := make([]*v1.KeyValue, numAttrs)
			for l := range attrs {
				attrs[l] = &v1.KeyValue{
					Key: fmt.Sprintf("attr-%d", l),
					Value: &v1.AnyValue{
						Value: &v1.AnyValue_StringValue{
							StringValue: fmt.Sprintf("value-%d", r.Intn(10)),
						},
					},
				}
			}

			spans[k] = &tempopb.Span{
				SpanID:            spanID,
				Name:              fmt.Sprintf("span-%d", k),
				StartTimeUnixNano: spanStartTime,
				DurationNanos:     spanDuration,
				Attributes:        attrs,
			}
		}

		spanSet := &tempopb.SpanSet{
			Spans:   spans,
			Matched: uint32(numSpans),
		}

		traces[i] = &tempopb.TraceSearchMetadata{
			TraceID:           traceID,
			RootServiceName:   fmt.Sprintf("service-%d", r.Intn(5)),
			RootTraceName:     fmt.Sprintf("operation-%d", r.Intn(10)),
			StartTimeUnixNano: uint64(startTime),
			DurationMs:        duration,
			SpanSet:           spanSet,
			SpanSets:          []*tempopb.SpanSet{spanSet},
		}
	}

	return &tempopb.SearchResponse{
		Traces: traces,
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: uint32(numTraces),
			InspectedBytes:  uint64(r.Intn(1000000)),
		},
	}
}

// simulateLatency sleeps for the specified duration with optional standard deviation
func simulateLatency(duration time.Duration, stdDev time.Duration) {
	if stdDev > 0 {
		variation := time.Duration(rand.NormFloat64() * float64(stdDev)) //nolint:gosec // G404
		// possibility of having a crazy big number is never a zero
		// capping to 3 sigmas (0.3% possibility)
		if variation > 3*stdDev {
			variation = 3 * stdDev
		} else if variation < -3*stdDev {
			variation = -3 * stdDev
		}
		duration += variation
	}
	if duration <= 0 {
		return
	}
	time.Sleep(duration)
}
