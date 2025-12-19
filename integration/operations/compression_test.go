package deployments

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/integration/util"
	"github.com/klauspost/compress/gzhttp"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/tempopb"
	tempoUtil "github.com/grafana/tempo/pkg/util"
)

func TestCompression(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		DeploymentMode: util.DeploymentModeSingleBinary,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		// Send a trace
		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, h.WriteTraceInfo(info, ""))

		h.WaitTracesQueryable(t, 1)

		// Create client with compression
		apiClient := h.APIClientHTTP("")
		util.QueryAndAssertTrace(t, apiClient, info)

		// Query and assert trace with compression
		apiClient.WithTransport(gzhttp.Transport(http.DefaultTransport))
		queryAndAssertTraceCompression(t, apiClient, info)
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
	request, err := http.NewRequest("GET", client.BaseURL+httpclient.QueryTraceV2Endpoint+"/"+info.HexID(), nil)
	require.NoError(t, err)
	request.Header.Add("Accept-Encoding", "gzip")

	res, err := client.Do(request)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, "gzip", res.Header.Get("Content-Encoding"))

	gzipReader, err := gzip.NewReader(res.Body)
	require.NoError(t, err)
	defer gzipReader.Close()

	m := &tempopb.TraceByIDResponse{}

	bodyBytes, _ := io.ReadAll(gzipReader)
	err = jsonpb.Unmarshal(bytes.NewReader(bodyBytes), m)
	require.NoError(t, err)

	util.AssertEqualTrace(t, expected, m.Trace)
}
