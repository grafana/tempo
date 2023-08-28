package frontend

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/util/test"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestSLOHook(t *testing.T) {
	tcs := []struct {
		name            string
		cfg             SLOConfig
		throughputBytes float64
		httpStatusCode  int
		latency         time.Duration
		err             error

		expectedWithSLO float64
	}{
		{
			name:            "no slo passes",
			expectedWithSLO: 1.0,
		},
		{
			name: "no slo fails : error",
			err:  errors.New("foo"),
		},
		{
			name:           "no slo fails : 5XX status code",
			httpStatusCode: http.StatusInternalServerError,
		},
		{
			name:            "no slo passes : 4XX status code",
			httpStatusCode:  http.StatusTooManyRequests,
			expectedWithSLO: 1.0,
		},
		{
			name: "slo passes - both",
			cfg: SLOConfig{
				DurationSLO:        10 * time.Second,
				ThroughputBytesSLO: 100,
			},
			latency:         5 * time.Second,
			throughputBytes: 110,
			expectedWithSLO: 1.0,
		},
		{
			name: "slo passes - duration",
			cfg: SLOConfig{
				DurationSLO:        10 * time.Second,
				ThroughputBytesSLO: 100,
			},
			latency:         5 * time.Second,
			throughputBytes: 90,
			expectedWithSLO: 1.0,
		},
		{
			name: "slo passes - throughput",
			cfg: SLOConfig{
				DurationSLO:        10 * time.Second,
				ThroughputBytesSLO: 100,
			},
			latency:         15 * time.Second,
			throughputBytes: 110,
			expectedWithSLO: 1.0,
		},
		{
			name: "slo passes - no throughput configured",
			cfg: SLOConfig{
				DurationSLO: 10 * time.Second,
			},
			latency:         5 * time.Second,
			throughputBytes: 1,
			expectedWithSLO: 1.0,
		},
		{
			name: "slo passes - no duration configured",
			cfg: SLOConfig{
				ThroughputBytesSLO: 100,
			},
			latency:         15 * time.Second,
			throughputBytes: 110,
			expectedWithSLO: 1.0,
		},
		{
			name: "slo fails - no throughput configured",
			cfg: SLOConfig{
				DurationSLO: 10 * time.Second,
			},
			latency:         15 * time.Second,
			throughputBytes: 1,
		},
		{
			name: "slo fails - no duration configured",
			cfg: SLOConfig{
				ThroughputBytesSLO: 100,
			},
			latency:         1 * time.Second,
			throughputBytes: 90,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			allCounter := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "all"}, []string{"tenant"})
			sloCounter := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "slo"}, []string{"tenant"})

			hook := sloHook(allCounter, sloCounter, tc.cfg)

			ctx := searchSLOPreHook(context.Background()) // prehook adds the throughput pointer
			addThroughputToContext(ctx, tc.throughputBytes)

			resp := &http.Response{
				StatusCode: tc.httpStatusCode,
			}

			hook(ctx, resp, "tenant", tc.latency, tc.err)

			actualAll, err := test.GetCounterValue(allCounter.WithLabelValues("tenant"))
			require.NoError(t, err)
			actualSLO, err := test.GetCounterValue(sloCounter.WithLabelValues("tenant"))
			require.NoError(t, err)

			require.Equal(t, 1.0, actualAll)
			require.Equal(t, tc.expectedWithSLO, actualSLO)
		})
	}
}
