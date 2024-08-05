package e2e

import (
	"compress/gzip"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/v2/integration/util"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v2/pkg/httpclient"
	"github.com/grafana/tempo/v2/pkg/tempopb"
	tempoUtil "github.com/grafana/tempo/v2/pkg/util"
)

const (
	configCompression = "deployments/config-all-in-one-local.yaml"
)

func TestCompression(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configCompression, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	// Get port for the Jaeger gRPC receiver endpoint
	c, err := util.NewJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c)

	info := tempoUtil.NewTraceInfo(time.Now(), "")
	require.NoError(t, info.EmitAllBatches(c))

	apiClient := httpclient.New("http://"+tempo.Endpoint(3200), "")

	apiClientWithCompression := httpclient.NewWithCompression("http://"+tempo.Endpoint(3200), "")

	util.QueryAndAssertTrace(t, apiClient, info)
	queryAndAssertTraceCompression(t, apiClientWithCompression, info)
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
	request.Header.Add("Accept-Encoding", "gzip")

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
