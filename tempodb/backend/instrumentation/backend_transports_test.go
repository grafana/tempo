package instrumentation

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
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
