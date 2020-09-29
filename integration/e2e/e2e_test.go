package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	cortex_e2e "github.com/cortexproject/cortex/integration/e2e"
	cortex_e2e_db "github.com/cortexproject/cortex/integration/e2e/db"
	"github.com/stretchr/testify/require"
)

const (
	configAllInOne      = "./config-all-in-one.yaml"
	configMicroservices = "./config-microservices.yaml"
)

func TestAllInOne(t *testing.T) {
	s, err := cortex_e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	minio := cortex_e2e_db.NewMinio(9000, "tempo")
	require.NotNil(t, minio)
	require.NoError(t, s.StartAndWaitReady(minio))

	require.NoError(t, copyFileToSharedDir(s, configAllInOne, "config.yaml"))
	tempo := newTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	// Get port for the otlp receiver endpoint
	c, err := newJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c)
	batch := makeThriftBatch()
	require.NoError(t, c.EmitBatch(context.Background(), batch))

	// test metrics
	require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_distributor_spans_received_total"))

	hexID := fmt.Sprintf("%016x%016x", batch.Spans[0].TraceIdHigh, batch.Spans[0].TraceIdLow)

	// ensure trace is created in ingester (trace_idle_time has passed)
	require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_traces_created_total"))

	// query an in-memory trace
	queryAndAssertTrace(t, "http://"+tempo.Endpoint(3100)+"/api/traces/"+hexID, "my operation", 1)

	// flush trace to backend
	res, err := cortex_e2e.GetRequest("http://" + tempo.Endpoint(3100) + "/flush")
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)

	// sleep for one maintenance cycle
	time.Sleep(2 * time.Second)

	// test metrics
	require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
	require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempodb_blocklist_length"))

	// query trace - should fetch from backend
	queryAndAssertTrace(t, "http://"+tempo.Endpoint(3100)+"/api/traces/"+hexID, "my operation", 1)
}

func TestMicroservices(t *testing.T) {
	s, err := cortex_e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	minio := cortex_e2e_db.NewMinio(9000, "tempo")
	require.NotNil(t, minio)
	require.NoError(t, s.StartAndWaitReady(minio))

	require.NoError(t, copyFileToSharedDir(s, configMicroservices, "config.yaml"))
	tempoIngester1 := newTempoIngester(1)
	require.NoError(t, s.StartAndWaitReady(tempoIngester1))

	tempoIngester2 := newTempoIngester(2)
	require.NoError(t, s.StartAndWaitReady(tempoIngester2))

	tempoDistributor := newTempoDistributor()
	require.NoError(t, s.StartAndWaitReady(tempoDistributor))

	tempoQuerier := newTempoQuerier()
	require.NoError(t, s.StartAndWaitReady(tempoQuerier))

	// Get port for the otlp receiver endpoint
	c, err := newJaegerGRPCClient(tempoDistributor.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c)
	batch := makeThriftBatch()
	require.NoError(t, c.EmitBatch(context.Background(), batch))

	// test metrics
	require.NoError(t, tempoDistributor.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_distributor_spans_received_total"))

	hexID := fmt.Sprintf("%016x%016x", batch.Spans[0].TraceIdHigh, batch.Spans[0].TraceIdLow)

	// ensure trace is created in ingester (trace_idle_time has passed)
	require.NoError(t, tempoIngester1.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_traces_created_total"))
	require.NoError(t, tempoIngester2.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_traces_created_total"))

	// query an in-memory trace
	queryAndAssertTrace(t, "http://"+tempoQuerier.Endpoint(3100)+"/api/traces/"+hexID, "my operation", 1)

	// flush trace to backend
	res, err := cortex_e2e.GetRequest("http://" + tempoIngester1.Endpoint(3100) + "/flush")
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)

	res, err = cortex_e2e.GetRequest("http://" + tempoIngester2.Endpoint(3100) + "/flush")
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)

	// sleep for one maintenance cycle
	time.Sleep(2 * time.Second)

	// test metrics
	require.NoError(t, tempoIngester1.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
	require.NoError(t, tempoIngester1.WaitSumMetrics(cortex_e2e.Equals(2), "tempodb_blocklist_length"))
	require.NoError(t, tempoIngester2.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
	require.NoError(t, tempoIngester2.WaitSumMetrics(cortex_e2e.Equals(2), "tempodb_blocklist_length"))

	// query trace - should fetch from backend
	queryAndAssertTrace(t, "http://"+tempoQuerier.Endpoint(3100)+"/api/traces/"+hexID, "my operation", 1)

	// stop an ingester and confirm we can still write and query
	err = tempoIngester2.Stop()
	require.NoError(t, err)

	batch = makeThriftBatch()
	require.NoError(t, c.EmitBatch(context.Background(), batch))
	hexID = fmt.Sprintf("%016x%016x", batch.Spans[0].TraceIdHigh, batch.Spans[0].TraceIdLow)

	// query an in-memory trace
	queryAndAssertTrace(t, "http://"+tempoQuerier.Endpoint(3100)+"/api/traces/"+hexID, "my operation", 1)

	// stop another ingester and confirm things fail
	err = tempoIngester1.Stop()
	require.NoError(t, err)

	batch = makeThriftBatch()
	require.Error(t, c.EmitBatch(context.Background(), batch))
}
