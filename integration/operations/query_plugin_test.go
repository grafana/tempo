package deployments

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	thrift "github.com/jaegertracing/jaeger-idl/thrift-gen/jaeger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchUsingJaegerPlugin(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		DeploymentMode: util.DeploymentModeSingleBinary,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		require.NoError(t, util.CopyFileToSharedDir(h.TestScenario, "config-tempo-query.yaml", "config-tempo-query.yaml"))

		// Start tempo-query and jaeger-query services
		tempoQuery := util.NewTempoQuery()
		jaegerQuery := util.NewJaegerQuery()

		err := h.TestScenario.StartAndWaitReady(tempoQuery, jaegerQuery)
		require.NoError(t, err)

		batch := makeThriftBatchWithSpanCountForServiceAndOp(2, "execute", "backend")
		require.NoError(t, h.WriteJaegerBatch(batch, ""))

		batch = makeThriftBatchWithSpanCountForServiceAndOp(2, "request", "frontend")
		require.NoError(t, h.WriteJaegerBatch(batch, ""))

		// wait for the 2 traces to be written to the live stores
		h.WaitTracesQueryable(t, 2)

		callJaegerQuerySearchServicesAssert(t, jaegerQuery, servicesOrOpJaegerQueryResponse{
			Data: []string{
				"frontend",
				"backend",
			},
			Total: 2,
		})

		callJaegerQuerySearchOperationAssert(t, jaegerQuery, "frontend", servicesOrOpJaegerQueryResponse{
			Data: []string{
				"execute",
				"request",
			},
			Total: 2,
		})

		callJaegerQuerySearchOperationAssert(t, jaegerQuery, "backend", servicesOrOpJaegerQueryResponse{
			Data: []string{
				"execute",
				"request",
			},
			Total: 2,
		})

		callJaegerQuerySearchTraceAssert(t, jaegerQuery, "request", "frontend")
		callJaegerQuerySearchTraceAssert(t, jaegerQuery, "execute", "backend")
	})
}

func callJaegerQuerySearchServicesAssert(t *testing.T, svc *e2e.HTTPService, expected servicesOrOpJaegerQueryResponse) {
	assert.Eventually(t, func() bool {
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
		slices.Sort(expected.Data)
		slices.Sort(response.Data)

		return assert.ObjectsAreEqual(expected, response)
	}, 1*time.Minute, 5*time.Second)
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
	slices.Sort(expected.Data)
	slices.Sort(response.Data)
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
	for range n {
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
