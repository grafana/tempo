package deployments

import (
	"compress/gzip"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/tempopb"
	tempoUtil "github.com/grafana/tempo/pkg/util"
)

func TestCompression(t *testing.T) {
	util.WithTempoHarness(t, util.TestHarnessConfig{}, func(h *util.TempoHarness) {
		// Send a trace
		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, info.EmitAllBatches(h.JaegerExporter))

		liveStoreA := h.Services[util.ServiceLiveStoreZoneA]
		require.NoError(t, liveStoreA.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempo_live_store_traces_created_total"}, e2e.WaitMissingMetrics))

		// Create client with compression
		util.QueryAndAssertTrace(t, h.HTTPClient, info)

		// Query and assert trace with compression
		apiClientWithCompression := httpclient.NewWithCompression("http://"+h.QueryFrontendHTTPEndpoint, "")
		queryAndAssertTraceCompression(t, apiClientWithCompression, info)
	})
}

func queryAndAssertTraceCompression(t *testing.T, client *httpclient.Client, info *tempoUtil.TraceInfo) {
	// The received client will strip the header before we have a chance to inspect it, so just validate that the compressed client works as expected.
	result, err := client.QueryTrace(info.HexID())
	require.NoError(t, err)
	require.NotNil(t, result)

	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)
	util.AssertEqualTrace(t, result, expected)

	// Go's http.Client transparently requests gzip compression and automatically decompresses the
	// response, to disable this behaviour you have to explicitly set the Accept-Encoding header.

	// Make the call directly so we have a chance to inspect the response header and manually un-gzip it ourselves to confirm the content.
	request, err := http.NewRequest("GET", client.BaseURL+httpclient.QueryTraceEndpoint+"/"+info.HexID(), nil)
	require.NoError(t, err)
	request.Header.Add("Accept-Encoding", "gzip-foob")

	res, err := client.Do(request)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, "gzip", res.Header.Get("Content-Encoding"))

	gzipReader, err := gzip.NewReader(res.Body)
	require.NoError(t, err)
	defer gzipReader.Close()

	m := &tempopb.Trace{}

	bodyBytes, _ := io.ReadAll(gzipReader)
	err = tempopb.UnmarshalFromJSONV1(bodyBytes, m)

	require.NoError(t, err)
	util.AssertEqualTrace(t, expected, m)
}
