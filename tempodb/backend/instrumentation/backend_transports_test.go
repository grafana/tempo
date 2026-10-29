package instrumentation

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestCountDials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	transport := &http.Transport{}
	t.Cleanup(transport.CloseIdleConnections)
	CountDials(transport, "test")
	client := &http.Client{Transport: transport}
	dials := connectionsDialed.WithLabelValues("test")

	get := func() {
		resp, err := client.Get(server.URL)
		require.NoError(t, err)
		_, err = io.Copy(io.Discard, resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
	}

	before := testutil.ToFloat64(dials)

	// The second request reuses the idle connection from the first.
	get()
	get()
	require.Equal(t, before+1, testutil.ToFloat64(dials))

	// Without an idle connection the next request dials again.
	transport.CloseIdleConnections()
	get()
	require.Equal(t, before+2, testutil.ToFloat64(dials))
}

func TestTransportRoleLabel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	client := &http.Client{Transport: NewTransport(http.DefaultTransport)}

	get := func(ctx context.Context) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		_, err = io.Copy(io.Discard, resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
	}

	count := func(role string) uint64 {
		m := &dto.Metric{}
		require.NoError(t, requestDuration.WithLabelValues(http.MethodGet, "200", role).(prometheus.Histogram).Write(m))
		return m.GetHistogram().GetSampleCount()
	}

	bloom, none := count("bloom"), count(roleNone)
	get(ContextWithRole(context.Background(), "bloom"))
	get(context.Background())
	require.Equal(t, bloom+1, count("bloom"))
	require.Equal(t, none+1, count(roleNone))
}
