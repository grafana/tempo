package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	cortex_e2e "github.com/cortexproject/cortex/integration/e2e"
	cortex_e2e_db "github.com/cortexproject/cortex/integration/e2e/db"
	"github.com/golang/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
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
	tempo, err := newTempoAllInOne()
	require.NoError(t, err)
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
	res, err := cortex_e2e.GetRequest("http://" + tempo.Endpoint(3100) + "/api/traces/" + hexID)
	require.NoError(t, err)
	out := &tempopb.Trace{}
	unmarshaller := &jsonpb.Unmarshaler{}
	assert.NoError(t, unmarshaller.Unmarshal(res.Body, out))
	assert.Len(t, out.Batches, 1)
	assert.Equal(t, out.Batches[0].InstrumentationLibrarySpans[0].Spans[0].Name, "my operation")
	defer res.Body.Close()

	// flush trace to backend
	res, err = cortex_e2e.GetRequest("http://" + tempo.Endpoint(3100) + "/flush")
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)

	// sleep for one maintenance cycle
	time.Sleep(2 * time.Second)

	// test metrics
	require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
	require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempodb_blocklist_length"))

	// query trace - should fetch from backend
	resp, err := cortex_e2e.GetRequest("http://" + tempo.Endpoint(3100) + "/api/traces/" + hexID)
	require.NoError(t, err)
	assert.NoError(t, unmarshaller.Unmarshal(resp.Body, out))
	assert.Len(t, out.Batches, 1)
	assert.Equal(t, out.Batches[0].InstrumentationLibrarySpans[0].Spans[0].Name, "my operation")
	defer resp.Body.Close()
}

func TestMicroservices(t *testing.T) {
	s, err := cortex_e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	minio := cortex_e2e_db.NewMinio(9000, "tempo")
	require.NotNil(t, minio)
	require.NoError(t, s.StartAndWaitReady(minio))

	require.NoError(t, copyFileToSharedDir(s, configMicroservices, "config.yaml"))
	tempoIngester1, err := newTempoIngester(1)
	require.NoError(t, err)
	require.NoError(t, s.StartAndWaitReady(tempoIngester1))

	tempoIngester2, err := newTempoIngester(2)
	require.NoError(t, err)
	require.NoError(t, s.StartAndWaitReady(tempoIngester2))

	tempoDistributor, err := newTempoDistributor()
	require.NoError(t, err)
	require.NoError(t, s.StartAndWaitReady(tempoDistributor))

	tempoQuerier, err := newTempoQuerier()
	require.NoError(t, err)
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
	res, err := cortex_e2e.GetRequest("http://" + tempoQuerier.Endpoint(3100) + "/api/traces/" + hexID)
	require.NoError(t, err)
	out := &tempopb.Trace{}
	unmarshaller := &jsonpb.Unmarshaler{}
	assert.NoError(t, unmarshaller.Unmarshal(res.Body, out))
	assert.Len(t, out.Batches, 1)
	assert.Equal(t, out.Batches[0].InstrumentationLibrarySpans[0].Spans[0].Name, "my operation")
	defer res.Body.Close()

	// flush trace to backend
	res, err = cortex_e2e.GetRequest("http://" + tempoIngester1.Endpoint(3100) + "/flush")
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
	resp, err := cortex_e2e.GetRequest("http://" + tempoQuerier.Endpoint(3100) + "/api/traces/" + hexID)
	require.NoError(t, err)
	assert.NoError(t, unmarshaller.Unmarshal(resp.Body, out))
	assert.Len(t, out.Batches, 1)
	assert.Equal(t, out.Batches[0].InstrumentationLibrarySpans[0].Spans[0].Name, "my operation")
	defer resp.Body.Close()
}
