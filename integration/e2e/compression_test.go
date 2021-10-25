package e2e

import (
	"compress/gzip"
	"net/http"
	"testing"
	"time"

	cortex_e2e "github.com/cortexproject/cortex/integration/e2e"
	"github.com/gogo/protobuf/jsonpb"
	util "github.com/grafana/tempo/integration"
	"github.com/grafana/tempo/pkg/tempopb"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
)

const (
	configCompression = "config-all-in-one-local.yaml"
)

func TestCompression(t *testing.T) {
	s, err := cortex_e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configCompression, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	// Get port for the Jaeger gRPC receiver endpoint
	c, err := newJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c)

	info := tempoUtil.NewTraceInfo(time.Now(), "")
	require.NoError(t, info.EmitAllBatches(c))

	apiClient := tempoUtil.NewClient("http://"+tempo.Endpoint(3200), "")

	apiClientWithCompression := tempoUtil.NewClientWithCompression("http://"+tempo.Endpoint(3200), "")

	queryAndAssertTrace(t, apiClient, info)
	queryAndAssertTraceCompression(t, apiClientWithCompression, info)
}

func queryAndAssertTraceCompression(t *testing.T, client *tempoUtil.Client, info *tempoUtil.TraceInfo) {

	// The recieved client will strip the header before we have a chance to inspect it, so just validate that the compressed client works as expected.
	result, resp, err := client.QueryTraceWithResponse(info.HexID())
	require.NoError(t, err)
	require.NotNil(t, resp)

	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)
	require.True(t, equalTraces(result, expected))

	// Go's http.Client transparently requests gzip compression and automatically decompresses the
	// response, to disable this behaviour you have to explicitly set the Accept-Encoding header.

	// Make the call directly so we have a chance to inspect the response header and manually un-gzip it ourselves to confirm the content.
	request, err := http.NewRequest("GET", client.BaseURL+tempoUtil.QueryTraceEndpoint+"/"+info.HexID(), nil)
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
	unmarshaller := &jsonpb.Unmarshaler{}
	err = unmarshaller.Unmarshal(gzipReader, m)
	require.NoError(t, err)
	require.True(t, equalTraces(expected, m))

	return
}

func queryAndAssert___(t *testing.T, client *tempoUtil.Client, info *tempoUtil.TraceInfo) {
	resp, err := client.QueryTrace(info.HexID())
	require.NoError(t, err)

	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	require.True(t, equalTraces(resp, expected))
}
