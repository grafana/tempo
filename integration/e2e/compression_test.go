package e2e

import (
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	cortex_e2e "github.com/cortexproject/cortex/integration/e2e"
	util "github.com/grafana/tempo/integration"
	"github.com/stretchr/testify/assert"
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

	// send a small trace, response will not be compressed
	batch := makeThriftBatch()
	require.NoError(t, c.EmitBatch(context.Background(), batch))

	hexID := fmt.Sprintf("%016x%016x", batch.Spans[0].TraceIdHigh, batch.Spans[0].TraceIdLow)

	queryAndAssertTrace(t, "http://"+tempo.Endpoint(3200)+"/api/traces/"+hexID, "my operation", 1)
	assert.False(t, queryAndAssertTraceCompression(t, "http://"+tempo.Endpoint(3200)+"/api/traces/"+hexID, "my operation", 1))

	// send a large trace, response should be compressed
	batch = makeThriftBatchWithSpanCount(50) // one span is ~70 bytes, 50 spans is ~3500 bytes
	require.NoError(t, c.EmitBatch(context.Background(), batch))

	hexID = fmt.Sprintf("%016x%016x", batch.Spans[0].TraceIdHigh, batch.Spans[0].TraceIdLow)

	queryAndAssertTrace(t, "http://"+tempo.Endpoint(3200)+"/api/traces/"+hexID, "my operation", 1)
	assert.True(t, queryAndAssertTraceCompression(t, "http://"+tempo.Endpoint(3200)+"/api/traces/"+hexID, "my operation", 1))
}

func queryAndAssertTraceCompression(t *testing.T, url string, expectedName string, expectedBatches int) (responseIsCompressed bool) {
	// Go's http.Client transparently requests gzip compression and automatically decompresses the
	// response, to disable this behaviour you have to explicitly set the Accept-Encoding header

	client := &http.Client{Timeout: 1 * time.Second}

	request, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	request.Header.Add("Accept-Encoding", "gzip")

	res, err := client.Do(request)
	require.NoError(t, err)
	defer res.Body.Close()

	if res.Header.Get("Content-Encoding") == "gzip" {
		responseIsCompressed = true

		gzipReader, err := gzip.NewReader(res.Body)
		require.NoError(t, err)
		defer gzipReader.Close()

		assertTrace(t, gzipReader, expectedBatches, expectedName)
	} else {
		assertTrace(t, res.Body, expectedBatches, expectedName)
	}

	return
}
