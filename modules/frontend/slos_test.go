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

		totalRequests      float64
		totalCancelled     float64
		expectedWithInSLO  float64
		cancelledWithinSLO float64
	}{
		{
			name:              "no slo passes",
			expectedWithInSLO: 1.0,
			totalRequests:     1.0,
		},
		{
			name:          "no slo fails : error",
			err:           errors.New("foo"),
			totalRequests: 1.0,
		},
		{
			name:              "no slo passes : resource exhausted grpc error",
			err:               status.Error(codes.ResourceExhausted, "foo"),
			expectedWithInSLO: 1.0,
			totalRequests:     1.0,
		},
		{
			name:           "no slo fails : 5XX status code",
			httpStatusCode: http.StatusInternalServerError,
			totalRequests:  1.0,
		},
		{
			name:              "no slo passes : 4XX status code",
			httpStatusCode:    http.StatusTooManyRequests,
			expectedWithInSLO: 1.0,
			totalRequests:     1.0,
		},
		{
			name: "slo passes - both",
			cfg: SLOConfig{
				DurationSLO:        10 * time.Second,
				ThroughputBytesSLO: 100,
			},
			latency:           5 * time.Second,
			bytesProcessed:    110,
			expectedWithInSLO: 1.0,
			totalRequests:     1.0,
		},
		{
			name: "slo passes - latency",
			cfg: SLOConfig{
				DurationSLO:        10 * time.Second,
				ThroughputBytesSLO: 100,
			},
			latency:           5 * time.Second,
			bytesProcessed:    90,
			expectedWithInSLO: 1.0,
			totalRequests:     1.0,
		},
		{
			name: "slo passes - throughput",
			cfg: SLOConfig{
				DurationSLO:        10 * time.Second,
				ThroughputBytesSLO: 100,
			},
			latency:           15 * time.Second,
			bytesProcessed:    1650, // 15s * 110 bytes/s
			expectedWithInSLO: 1.0,
			totalRequests:     1.0,
		},
		{
			name: "slo passes - no throughput configured",
			cfg: SLOConfig{
				DurationSLO: 10 * time.Second,
			},
			latency:           5 * time.Second,
			bytesProcessed:    1,
			expectedWithInSLO: 1.0,
			totalRequests:     1.0,
		},
		{
			name: "slo passes - no latency configured",
			cfg: SLOConfig{
				ThroughputBytesSLO: 100,
			},
			latency:           15 * time.Second,
			bytesProcessed:    1650, // 15s * 110 bytes/s
			expectedWithInSLO: 1.0,
			totalRequests:     1.0,
		},
		{
			name: "slo fails - no throughput configured",
			cfg: SLOConfig{
				DurationSLO: 10 * time.Second,
			},
			latency:        15 * time.Second,
			bytesProcessed: 1,
			totalRequests:  1.0,
		},
		{
			name: "slo fails - no latency configured",
			cfg: SLOConfig{
				ThroughputBytesSLO: 100,
			},
			latency:        1 * time.Second,
			bytesProcessed: 90,
			totalRequests:  1.0,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			allCounter := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "all"}, []string{"tenant", "result"})
			sloCounter := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "slo"}, []string{"tenant", "result"})
			throughputVec := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "throughput"}, []string{"tenant"})

			hook := sloHook(allCounter, sloCounter, throughputVec, tc.cfg)

			resp := &http.Response{
				StatusCode: tc.httpStatusCode,
			}

			hook(resp, "test", uint64(tc.bytesProcessed), tc.latency, tc.err)

			actualCompleted, err := test.GetCounterValue(allCounter.WithLabelValues("test", resultCompleted))
			require.NoError(t, err)
			actualCancelled, err := test.GetCounterValue(allCounter.WithLabelValues("test", resultCanceled))
			require.NoError(t, err)
			require.Equal(t, tc.totalCancelled, actualCancelled)
			require.Equal(t, tc.totalRequests, actualCompleted+actualCancelled)

			completedWithInSLO, err := test.GetCounterValue(sloCounter.WithLabelValues("test", resultCompleted))
			require.NoError(t, err)
			cancelledWithInSLO, err := test.GetCounterValue(sloCounter.WithLabelValues("test", resultCanceled))
			require.NoError(t, err)
			require.Equal(t, tc.expectedWithInSLO, completedWithInSLO+cancelledWithInSLO)
			require.Equal(t, tc.cancelledWithinSLO, cancelledWithInSLO)
		})
	}
}

func TestBadRequest(t *testing.T) {
	allCounter := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "all"}, []string{"tenant", "result"})
	sloCounter := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "slo"}, []string{"tenant", "result"})
	throughputVec := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "throughput"}, []string{"tenant"})

	hook := sloHook(allCounter, sloCounter, throughputVec, SLOConfig{
		DurationSLO:        10 * time.Second,
		ThroughputBytesSLO: 100,
	})

	res := &http.Response{
		StatusCode: http.StatusBadRequest,
		Status:     http.StatusText(http.StatusBadRequest),
		Body:       io.NopCloser(strings.NewReader("foo")),
	}

	hook(res, "test", 0, 0, nil)

	actualCompleted, err := test.GetCounterValue(allCounter.WithLabelValues("test", resultCompleted))
	require.NoError(t, err)
	require.Equal(t, 1.0, actualCompleted)
	actualCancelled, err := test.GetCounterValue(allCounter.WithLabelValues("test", resultCanceled))
	require.NoError(t, err)
	require.Equal(t, 0.0, actualCancelled)
	require.Equal(t, 1.0, actualCompleted+actualCancelled)

	completedWithInSLO, err := test.GetCounterValue(sloCounter.WithLabelValues("test", resultCompleted))
	require.NoError(t, err)
	cancelledWithInSLO, err := test.GetCounterValue(sloCounter.WithLabelValues("test", resultCanceled))
	require.NoError(t, err)
	require.Equal(t, 1.0, completedWithInSLO+cancelledWithInSLO)
	require.Equal(t, 0.0, cancelledWithInSLO)
}

func TestCanceledRequest(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		latency    time.Duration
		err        error

		totalRequests      float64
		totalCancelled     float64
		totalWithInSLO     float64
		cancelledWithinSLO float64
	}{
		{
			name:               "random error with http response has 499 status code",
			statusCode:         util.StatusClientClosedRequest,
			err:                errors.New("foo"),
			latency:            5 * time.Second,
			totalWithInSLO:     1.0,
			cancelledWithinSLO: 1.0,
			totalRequests:      1.0,
			totalCancelled:     1.0,
		},
		{
			name:               "random error with http response has 499 status code and 15s latency",
			statusCode:         util.StatusClientClosedRequest,
			err:                errors.New("foo"),
			latency:            15 * time.Second, // latency > DurationSLO shouldn't be impacted
			totalWithInSLO:     1.0,
			cancelledWithinSLO: 1.0,
			totalRequests:      1.0,
			totalCancelled:     1.0,
		},
		{
			name:               "nil error with http response has 499 status code",
			statusCode:         util.StatusClientClosedRequest,
			err:                nil,
			latency:            5 * time.Second,
			totalWithInSLO:     1.0,
			cancelledWithinSLO: 1.0,
			totalRequests:      1.0,
			totalCancelled:     1.0,
		},
		{
			name:               "nil error with http response has 499 status code and 15s latency",
			statusCode:         util.StatusClientClosedRequest,
			err:                nil,
			latency:            15 * time.Second,
			totalWithInSLO:     1.0,
			cancelledWithinSLO: 1.0,
			totalRequests:      1.0,
			totalCancelled:     1.0,
		},
		{
			name:               "context.Canceled error with http response has 499 status code",
			statusCode:         util.StatusClientClosedRequest,
			err:                context.Canceled,
			latency:            15 * time.Second,
			totalWithInSLO:     1.0,
			cancelledWithinSLO: 1.0,
			totalRequests:      1.0,
			totalCancelled:     1.0,
		},
		{
			name:               "context.Canceled error with 500 status code",
			statusCode:         http.StatusInternalServerError,
			err:                context.Canceled,
			latency:            15 * time.Second,
			totalWithInSLO:     1.0,
			cancelledWithinSLO: 1.0,
			totalRequests:      1.0,
			totalCancelled:     1.0,
		},
		{
			name:               "context.Canceled error with 200 status code",
			statusCode:         http.StatusOK,
			err:                context.Canceled,
			latency:            15 * time.Second,
			totalWithInSLO:     1.0,
			cancelledWithinSLO: 1.0,
			totalRequests:      1.0,
			totalCancelled:     1.0,
		},
		{
			name:               "grpc codes.Canceled error with 500 status code",
			statusCode:         http.StatusInternalServerError,
			err:                status.Error(codes.Canceled, "foo"),
			latency:            15 * time.Second,
			totalWithInSLO:     1.0,
			cancelledWithinSLO: 1.0,
			totalRequests:      1.0,
			totalCancelled:     1.0,
		},
		{
			name:           "grpc codes.ResourceExhausted error with 500 status code",
			statusCode:     http.StatusInternalServerError,
			err:            status.Error(codes.ResourceExhausted, "foo"),
			latency:        15 * time.Second,
			totalWithInSLO: 1.0,
			totalRequests:  1.0,
		},
		{
			name:           "grpc codes.InvalidArgument error with 500 status code",
			statusCode:     http.StatusInternalServerError,
			err:            status.Error(codes.InvalidArgument, "foo"),
			latency:        15 * time.Second,
			totalWithInSLO: 1.0,
			totalRequests:  1.0,
		},
		{
			name:           "grpc codes.NotFound error with 500 status code",
			statusCode:     http.StatusInternalServerError,
			err:            status.Error(codes.NotFound, "foo"),
			latency:        15 * time.Second,
			totalWithInSLO: 1.0,
			totalRequests:  1.0,
		},
		{
			name:          "nil error with 500 status code and 15s latency",
			statusCode:    http.StatusInternalServerError,
			err:           nil,
			latency:       15 * time.Second,
			totalRequests: 1.0,
		},
		{
			name:               "nil error with http response has 499 status code",
			statusCode:         util.StatusClientClosedRequest,
			err:                nil,
			latency:            15 * time.Second,
			totalWithInSLO:     1.0,
			cancelledWithinSLO: 1.0,
			totalRequests:      1.0,
			totalCancelled:     1.0,
		},
		{
			name:               "grpc codes.Canceled error with 200 status code",
			statusCode:         http.StatusOK,
			err:                status.Error(codes.Canceled, "foo"),
			latency:            15 * time.Second,
			totalWithInSLO:     1.0,
			cancelledWithinSLO: 1.0,
			totalRequests:      1.0,
			totalCancelled:     1.0,
		},
		{
			name:          "nil error with 200 status code and 15s latency",
			statusCode:    http.StatusOK,
			err:           nil,
			latency:       15 * time.Second,
			totalRequests: 1.0,
		},
		{
			name:           "nil error with 200 status code and 5s latency",
			statusCode:     http.StatusOK,
			err:            nil,
			latency:        5 * time.Second,
			totalWithInSLO: 1.0,
			totalRequests:  1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allCounter := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "all"}, []string{"tenant", "result"})
			sloCounter := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "slo"}, []string{"tenant", "result"})
			throughputVec := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "throughput"}, []string{"tenant"})

			// anything over 10s is considered outside SLO
			hook := sloHook(allCounter, sloCounter, throughputVec, SLOConfig{
				DurationSLO:        10 * time.Second,
				ThroughputBytesSLO: 100,
			})

			res := &http.Response{
				StatusCode: tt.statusCode,
				Status:     "context canceled",
				Body:       io.NopCloser(strings.NewReader("foo")),
			}

			hook(res, "test", 0, tt.latency, tt.err)

			actualCompleted, err := test.GetCounterValue(allCounter.WithLabelValues("test", resultCompleted))
			require.NoError(t, err)
			actualCancelled, err := test.GetCounterValue(allCounter.WithLabelValues("test", resultCanceled))
			require.NoError(t, err)
			require.Equal(t, tt.totalRequests, actualCompleted+actualCancelled)
			require.Equal(t, tt.totalCancelled, actualCancelled)

			completedWithInSLO, err := test.GetCounterValue(sloCounter.WithLabelValues("test", resultCompleted))
			require.NoError(t, err)
			cancelledWithInSLO, err := test.GetCounterValue(sloCounter.WithLabelValues("test", resultCanceled))
			require.NoError(t, err)
			require.Equal(t, tt.totalWithInSLO, completedWithInSLO+cancelledWithInSLO)
			require.Equal(t, tt.cancelledWithinSLO, cancelledWithInSLO)
		})
	}
}
