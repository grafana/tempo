package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/grafana/e2e"
	util "github.com/grafana/tempo/integration"
	thrift "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchUsingJaegerPlugin(t *testing.T) {
	s, err := e2e.NewScenario("tempo_query_plugin_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, "config-plugin-test.yaml", "config.yaml"))
	require.NoError(t, util.CopyFileToSharedDir(s, "config-tempo-query.yaml", "config-tempo-query.yaml"))

	tempo := util.NewTempoAllInOne()
	tempoQuery := util.NewTempoQuery()

	require.NoError(t, s.StartAndWaitReady(tempo))
	require.NoError(t, s.StartAndWaitReady(tempoQuery))

	jaegerClient, err := util.NewJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	batch := makeThriftBatchWithSpanCountForServiceAndOp(2, "execute", "backend")
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	batch = makeThriftBatchWithSpanCountForServiceAndOp(2, "request", "frontend")
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	// Wait for the traces to be written to the WAL
	time.Sleep(time.Second * 3)

	callJaegerQuerySearchServicesAssert(t, tempoQuery, servicesOrOpJaegerQueryResponse{
		Data: []string{
			"frontend",
			"backend",
		},
		Total: 2,
	})

	callJaegerQuerySearchOperationAssert(t, tempoQuery, "frontend", servicesOrOpJaegerQueryResponse{
		Data: []string{
			"execute",
			"request",
		},
		Total: 2,
	})

	callJaegerQuerySearchOperationAssert(t, tempoQuery, "backend", servicesOrOpJaegerQueryResponse{
		Data: []string{
			"execute",
			"request",
		},
		Total: 2,
	})

	callJaegerQuerySearchTraceAssert(t, tempoQuery, "request", "frontend")
	callJaegerQuerySearchTraceAssert(t, tempoQuery, "execute", "backend")
}

func TestSearchUsingBackendTagsService(t *testing.T) {
	s, err := e2e.NewScenario("tempo_query_plugin_backend_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, "config-plugin-test.yaml", "config.yaml"))
	require.NoError(t, util.CopyFileToSharedDir(s, "config-tempo-query.yaml", "config-tempo-query.yaml"))

	tempo := util.NewTempoAllInOne()
	tempoQuery := util.NewTempoQuery()

	require.NoError(t, s.StartAndWaitReady(tempo))
	require.NoError(t, s.StartAndWaitReady(tempoQuery))

	jaegerClient, err := util.NewJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	batch := makeThriftBatchWithSpanCountForServiceAndOp(2, "execute", "backend")
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	batch = makeThriftBatchWithSpanCountForServiceAndOp(2, "request", "frontend")
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	// Wait for the traces to be written to the WAL
	time.Sleep(time.Second * 3)

	callFlush(t, tempo)
	time.Sleep(time.Second * 1)
	callFlush(t, tempo)

	callJaegerQuerySearchServicesAssert(t, tempoQuery, servicesOrOpJaegerQueryResponse{
		Data: []string{
			"frontend",
			"backend",
		},
		Total: 2,
	})
}

func callJaegerQuerySearchServicesAssert(t *testing.T, svc *e2e.HTTPService, expected servicesOrOpJaegerQueryResponse) {
	// search for tag values
	req, err := http.NewRequest(http.MethodGet, "http://"+svc.Endpoint(16686)+"/api/services", nil)
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	// read body and print it
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	defer res.Body.Close()

	// parse response
	var response servicesOrOpJaegerQueryResponse
	require.NoError(t, json.Unmarshal(body, &response))
	sort.Slice(expected.Data, func(i, j int) bool { return expected.Data[i] < expected.Data[j] })
	sort.Slice(response.Data, func(i, j int) bool { return response.Data[i] < response.Data[j] })
	require.Equal(t, expected, response)
}

func callJaegerQuerySearchOperationAssert(t *testing.T, svc *e2e.HTTPService, operation string, expected servicesOrOpJaegerQueryResponse) {
	apiURL := fmt.Sprintf("/api/services/%s/operations", operation)

	// search for tag values
	req, err := http.NewRequest(http.MethodGet, "http://"+svc.Endpoint(16686)+apiURL, nil)
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	// read body and print it
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	defer res.Body.Close()

	// parse response
	var response servicesOrOpJaegerQueryResponse
	require.NoError(t, json.Unmarshal(body, &response))
	sort.Slice(expected.Data, func(i, j int) bool { return expected.Data[i] < expected.Data[j] })
	sort.Slice(response.Data, func(i, j int) bool { return response.Data[i] < response.Data[j] })
	require.Equal(t, expected, response)
}

func callJaegerQuerySearchTraceAssert(t *testing.T, svc *e2e.HTTPService, operation, service string) {
	start := time.Now().Add(-10 * time.Minute)
	end := start.Add(1 * time.Hour)

	apiURL := fmt.Sprintf("/api/traces?end=%d&limit=20&lookback=1h&maxDuration&minDuration&service=%s&start=%d", end.UnixMicro(), service, start.UnixMicro())

	// search for tag values
	req, err := http.NewRequest(http.MethodGet, "http://"+svc.Endpoint(16686)+apiURL, nil)
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	// read body and print it
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	defer res.Body.Close()

	// parse response
	var response tracesJaegerQueryResponse
	require.NoError(t, json.Unmarshal(body, &response))
	assert.Equal(t, 1, len(response.Data))
	assert.Equal(t, 2, len(response.Data[0].Spans))
	assert.Equal(t, operation, response.Data[0].Spans[0].OperationName)
	for _, p := range response.Data[0].Processes {
		assert.Equal(t, p.ServiceName, service)
	}
}

func makeThriftBatchWithSpanCountForServiceAndOp(n int, name, service string) *thrift.Batch {
	var spans []*thrift.Span
	traceIDLow := rand.Int63()
	traceIDHigh := rand.Int63()
	for i := 0; i < n; i++ {
		spans = append(spans, &thrift.Span{
			TraceIdLow:    traceIDLow,
			TraceIdHigh:   traceIDHigh,
			SpanId:        rand.Int63(),
			StartTime:     time.Now().UnixMicro(),
			ParentSpanId:  0,
			OperationName: name,
			References:    nil,
			Flags:         0,
			Duration:      10,
			Logs:          nil,
		})
	}

	return &thrift.Batch{
		Process: &thrift.Process{
			ServiceName: service,
		},
		Spans: spans,
	}
}

type servicesOrOpJaegerQueryResponse struct {
	Data  []string `json:"data"`
	Total int      `json:"total"`
}

type spanJaeger struct {
	OperationName string `json:"operationName"`
	ProcessID     string `json:"processId"`
}

type processJaeger struct {
	ServiceName string `json:"serviceName"`
}

type traceJaeger struct {
	Processes map[string]processJaeger `json:"processes"`
	Spans     []spanJaeger             `json:"spans"`
}

type tracesJaegerQueryResponse struct {
	Data []traceJaeger `json:"data"`
}
