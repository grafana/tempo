package frontend

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSLOHook(t *testing.T) {
	tcs := []struct {
		name           string
		cfg            SLOConfig
		bytesProcessed float64
		httpStatusCode int
		latency        time.Duration
		err            error

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
			name:            "no slo passes : resource exhausted grpc error",
			err:             status.Error(codes.ResourceExhausted, "foo"),
			expectedWithSLO: 1.0,
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
			bytesProcessed:  110,
			expectedWithSLO: 1.0,
		},
		{
			name: "slo passes - duration",
			cfg: SLOConfig{
				DurationSLO:        10 * time.Second,
				ThroughputBytesSLO: 100,
			},
			latency:         5 * time.Second,
			bytesProcessed:  90,
			expectedWithSLO: 1.0,
		},
		{
			name: "slo passes - throughput",
			cfg: SLOConfig{
				DurationSLO:        10 * time.Second,
				ThroughputBytesSLO: 100,
			},
			latency:         15 * time.Second,
			bytesProcessed:  1650, // 15s * 110 bytes/s
			expectedWithSLO: 1.0,
		},
		{
			name: "slo passes - no throughput configured",
			cfg: SLOConfig{
				DurationSLO: 10 * time.Second,
			},
			latency:         5 * time.Second,
			bytesProcessed:  1,
			expectedWithSLO: 1.0,
		},
		{
			name: "slo passes - no duration configured",
			cfg: SLOConfig{
				ThroughputBytesSLO: 100,
			},
			latency:         15 * time.Second,
			bytesProcessed:  1650, // 15s * 110 bytes/s
			expectedWithSLO: 1.0,
		},
		{
			name: "slo fails - no throughput configured",
			cfg: SLOConfig{
				DurationSLO: 10 * time.Second,
			},
			latency:        15 * time.Second,
			bytesProcessed: 1,
		},
		{
			name: "slo fails - no duration configured",
			cfg: SLOConfig{
				ThroughputBytesSLO: 100,
			},
			latency:        1 * time.Second,
			bytesProcessed: 90,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			allCounter := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "all"}, []string{"tenant"})
			sloCounter := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "slo"}, []string{"tenant"})
			throughputVec := prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "throughput"}, []string{"tenant"})

			hook := sloHook(allCounter, sloCounter, throughputVec, tc.cfg)

			resp := &http.Response{
				StatusCode: tc.httpStatusCode,
			}

			hook(resp, "tenant", uint64(tc.bytesProcessed), tc.latency, tc.err)

			actualAll, err := test.GetCounterValue(allCounter.WithLabelValues("tenant"))
			require.NoError(t, err)
			actualSLO, err := test.GetCounterValue(sloCounter.WithLabelValues("tenant"))
			require.NoError(t, err)

			require.Equal(t, 1.0, actualAll)
			require.Equal(t, tc.expectedWithSLO, actualSLO)
		})
	}
}

func TestBadRequest(t *testing.T) {
	allCounter := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "all"}, []string{"tenant"})
	sloCounter := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "slo"}, []string{"tenant"})
	throughputVec := prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "throughput"}, []string{"tenant"})

	hook := sloHook(allCounter, sloCounter, throughputVec, SLOConfig{
		DurationSLO:        10 * time.Second,
		ThroughputBytesSLO: 100,
	})

	res := &http.Response{
		StatusCode: http.StatusBadRequest,
		Status:     http.StatusText(http.StatusBadRequest),
		Body:       io.NopCloser(strings.NewReader("foo")),
	}

	hook(res, "tenant", 0, 0, nil)

	actualAll, err := test.GetCounterValue(allCounter.WithLabelValues("tenant"))
	require.NoError(t, err)
	actualSLO, err := test.GetCounterValue(sloCounter.WithLabelValues("tenant"))
	require.NoError(t, err)

	require.Equal(t, 1.0, actualAll)
	require.Equal(t, 1.0, actualSLO)
}

func TestCanceledRequest(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		err        error
		withInSLO  bool
	}{
		{
			name:       "random error with http response has 499 status code",
			statusCode: util.StatusClientClosedRequest,
			err:        errors.New("foo"),
			withInSLO:  true,
		},
		{
			name:       "context.Canceled error with http response has 499 status code",
			statusCode: util.StatusClientClosedRequest,
			err:        context.Canceled,
			withInSLO:  true,
		},
		{
			name:       "context.Canceled error with 500 status code",
			statusCode: http.StatusInternalServerError,
			err:        context.Canceled,
			withInSLO:  true,
		},
		{
			name:       "context.Canceled error with 200 status code",
			statusCode: http.StatusOK,
			err:        context.Canceled,
			withInSLO:  true,
		},
		{
			name:       "grpc codes.Canceled error with 500 status code",
			statusCode: http.StatusInternalServerError,
			err:        status.Error(codes.Canceled, "foo"),
			withInSLO:  true,
		},
		{
			name:       "grpc codes.Canceled error with 200 status code",
			statusCode: http.StatusOK,
			err:        status.Error(codes.Canceled, "foo"),
			withInSLO:  true,
		},
		{
			name:       "no error with 200 status code",
			statusCode: http.StatusOK,
			err:        nil,
			withInSLO:  false,
		},
		{
			name:       "no error with 500 status code",
			statusCode: http.StatusInternalServerError,
			err:        nil,
			withInSLO:  false,
		},
		{
			name:       "no error with http response has 499 status code",
			statusCode: util.StatusClientClosedRequest,
			err:        nil,
			withInSLO:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allCounter := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "all"}, []string{"tenant"})
			sloCounter := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "slo"}, []string{"tenant"})
			throughputVec := prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "throughput"}, []string{"tenant"})

			hook := sloHook(allCounter, sloCounter, throughputVec, SLOConfig{
				DurationSLO:        10 * time.Second,
				ThroughputBytesSLO: 100,
			})

			res := &http.Response{
				StatusCode: tt.statusCode,
				Status:     "context canceled",
				Body:       io.NopCloser(strings.NewReader("foo")),
			}

			// latency is below DurationSLO threshold
			hook(res, "tenant", 0, 15*time.Second, tt.err)

			actualAll, err := test.GetCounterValue(allCounter.WithLabelValues("tenant"))
			require.NoError(t, err)
			require.Equal(t, 1.0, actualAll)

			actualSLO, err := test.GetCounterValue(sloCounter.WithLabelValues("tenant"))
			require.NoError(t, err)
			if tt.withInSLO {
				require.Equal(t, 1.0, actualSLO)
			} else {
				require.Equal(t, 0.0, actualSLO)
			}
		})
	}
}
