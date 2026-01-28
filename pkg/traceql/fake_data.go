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
	if probability < 1 && rand.Float64() > probability { //nolint:gosec // G404
		return &tempopb.SearchResponse{}
	}

	numTraces := 1 + rand.Intn(3) //nolint:gosec // G404
	traces := make([]*tempopb.TraceSearchMetadata, numTraces)

	for i := range numTraces {
		traceID := fmt.Sprintf("%016x%016x", rand.Int63(), rand.Int63())                      //nolint:gosec // G404
		startTime := time.Now().Add(-time.Duration(rand.Intn(3600)) * time.Second).UnixNano() //nolint:gosec // G404
		duration := uint32(100 + rand.Intn(900))                                              //nolint:gosec // G404

		numSpans := 1 + rand.Intn(5) //nolint:gosec // G404
		spans := make([]*tempopb.Span, numSpans)

		for k := range numSpans {
			spanID := fmt.Sprintf("%016x", rand.Int63())                                  //nolint:gosec // G404
			spanStartTime := uint64(startTime) + uint64(rand.Intn(int(duration)))*1000000 //nolint:gosec // G404
			spanDuration := uint64(rand.Intn(100)) * 1000000                              //nolint:gosec // G404

			numAttrs := 1 + rand.Intn(3) //nolint:gosec // G404
			attrs := make([]*v1.KeyValue, numAttrs)
			for l := range attrs {
				attrs[l] = &v1.KeyValue{
					Key: fmt.Sprintf("attr-%d", l),
					Value: &v1.AnyValue{
						Value: &v1.AnyValue_StringValue{
							StringValue: fmt.Sprintf("value-%d", rand.Intn(10)), //nolint:gosec // G404
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
			RootServiceName:   fmt.Sprintf("service-%d", rand.Intn(5)),    //nolint:gosec // G404
			RootTraceName:     fmt.Sprintf("operation-%d", rand.Intn(10)), //nolint:gosec // G404
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
			InspectedBytes:  uint64(rand.Intn(1000000)), //nolint:gosec // G404
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
