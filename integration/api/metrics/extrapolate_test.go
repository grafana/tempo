package metrics

import (
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/v3/integration/util"
	"github.com/grafana/tempo/v3/modules/overrides"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/stretchr/testify/require"
)

// TestQueryRangeExtrapolateOverride checks that the metrics_extrapolate override is resolved by the
// query-frontend and honored by the querier and live-store, including for live-store complete blocks
// whose results were already cached without extrapolation.
func TestQueryRangeExtrapolateOverride(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		const (
			traces        = 10
			spansPerTrace = 5
			spans         = traces * spansPerTrace
		)

		for range traces {
			trace := test.MakeTraceWithSpanCount(1, spansPerTrace, test.ValidTraceID(nil))
			for _, rs := range trace.ResourceSpans {
				for _, ss := range rs.ScopeSpans {
					for _, s := range ss.Spans {
						s.TraceState = "ot=th:8" // 50% sampling, so each span extrapolates to 2
					}
				}
			}
			require.NoError(t, h.WriteTempoProtoTraces(trace, ""))
		}

		h.WaitTracesQueryable(t, traces)

		// wait for both live-stores to complete their blocks so the queries below go through the
		// complete-block cache
		for _, service := range []string{util.ServiceLiveStoreZoneA, util.ServiceLiveStoreZoneB} {
			require.NoError(t, h.Services[service].WaitSumMetricsWithOptions(
				e2e.Greater(0),
				[]string{"tempo_live_store_blocks_completed_total"},
				e2e.WaitMissingMetrics,
			))
		}

		// use the same time range for every query so the plain query hits the same complete-block cache
		// entries with the override off and on. the frontend aligns the end of the range down to the step
		// after applying query_end_cutoff, so use a short step to keep the spans just written in range.
		now := time.Now()
		start, end := now.Add(-5*time.Minute), now.Add(time.Minute)
		const step = "1s"

		requireCount := func(t *testing.T, query string, expected float64, extrapolated bool) {
			t.Helper()
			callQueryRange(t, h, queryRangeRequest{Query: query, Start: start, End: end, Step: step}, func(res *tempopb.QueryRangeResponse, err error) {
				require.NoError(t, err)
				require.Len(t, res.GetSeries(), 1)
				require.InDelta(t, expected, sumSamples(res.GetSeries()[0].GetSamples()), 0.000001)
				require.Equal(t, extrapolated, res.GetExtrapolated())
			})
		}

		const query = "{} | count_over_time()"

		// later cases depend on this one warming the cache, so stop if it fails
		require.True(t, t.Run("override off", func(t *testing.T) {
			requireCount(t, query, spans, false)
		}))

		require.NoError(t, h.UpdateOverrides(map[string]*overrides.Overrides{
			"single-tenant": {
				Read: overrides.ReadOverrides{
					MetricsExtrapolate: new(true),
				},
			},
		}))

		t.Run("override on", func(t *testing.T) {
			requireCount(t, query, 2*spans, true)
		})

		t.Run("override on, hint off", func(t *testing.T) {
			requireCount(t, query+" with(extrapolate=false)", spans, false)
		})
	})
}
