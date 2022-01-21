package servicegraphs

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gen "github.com/grafana/tempo/modules/generator/processor"
	test_util "github.com/grafana/tempo/modules/generator/processor/util/test"
	"github.com/grafana/tempo/pkg/tempopb"
)

func TestServiceGraphs(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	p := New(cfg, "test")

	registry := gen.NewRegistry(nil)
	err := p.RegisterMetrics(registry)
	assert.NoError(t, err)

	traces := testData(t, "testdata/test-sample.json")
	err = p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: traces.Batches})
	assert.NoError(t, err)

	// Manually call expire to force collection of edges.
	sgp := p.(*processor)
	sgp.store.Expire()

	appender := &test_util.Appender{}

	collectTime := time.Now()
	err = registry.Gather(appender)
	assert.NoError(t, err)

	assert.False(t, appender.IsCommitted)
	assert.False(t, appender.IsRolledback)

	expectedMetrics := []test_util.Metric{
		//{`{client="app", server="db", __name__="traces_service_graph_request_client_seconds_bucket", le="+Inf"}`, 0},
		{`{client="app", server="db", __name__="traces_service_graph_request_client_seconds_bucket", le="0.1"}`, 0},
		{`{client="app", server="db", __name__="traces_service_graph_request_client_seconds_bucket", le="0.2"}`, 0},
		{`{client="app", server="db", __name__="traces_service_graph_request_client_seconds_bucket", le="0.4"}`, 0},
		{`{client="app", server="db", __name__="traces_service_graph_request_client_seconds_bucket", le="0.8"}`, 0},
		{`{client="app", server="db", __name__="traces_service_graph_request_client_seconds_bucket", le="1.6"}`, 2},
		{`{client="app", server="db", __name__="traces_service_graph_request_client_seconds_bucket", le="3.2"}`, 3},
		{`{client="app", server="db", __name__="traces_service_graph_request_client_seconds_bucket", le="6.4"}`, 3},
		{`{client="app", server="db", __name__="traces_service_graph_request_client_seconds_bucket", le="12.8"}`, 3},
		{`{client="app", server="db", __name__="traces_service_graph_request_client_seconds_count"}`, 3},
		{`{client="app", server="db", __name__="traces_service_graph_request_client_seconds_sum"}`, 4.4},
		//{`{client="app", server="db", __name__="traces_service_graph_request_server_seconds_bucket", le="+Inf"}`, 0},
		{`{client="app", server="db", __name__="traces_service_graph_request_server_seconds_bucket", le="0.1"}`, 0},
		{`{client="app", server="db", __name__="traces_service_graph_request_server_seconds_bucket", le="0.2"}`, 0},
		{`{client="app", server="db", __name__="traces_service_graph_request_server_seconds_bucket", le="0.4"}`, 0},
		{`{client="app", server="db", __name__="traces_service_graph_request_server_seconds_bucket", le="0.8"}`, 0},
		{`{client="app", server="db", __name__="traces_service_graph_request_server_seconds_bucket", le="1.6"}`, 2},
		{`{client="app", server="db", __name__="traces_service_graph_request_server_seconds_bucket", le="3.2"}`, 3},
		{`{client="app", server="db", __name__="traces_service_graph_request_server_seconds_bucket", le="6.4"}`, 3},
		{`{client="app", server="db", __name__="traces_service_graph_request_server_seconds_bucket", le="12.8"}`, 3},
		{`{client="app", server="db", __name__="traces_service_graph_request_server_seconds_count"}`, 3},
		{`{client="app", server="db", __name__="traces_service_graph_request_server_seconds_sum"}`, 5},
		{`{client="app", server="db", __name__="traces_service_graph_request_total"}`, 3},
		//{`{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_bucket", le="+Inf"}`, 0},
		{`{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_bucket", le="0.1"}`, 0},
		{`{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_bucket", le="0.2"}`, 0},
		{`{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_bucket", le="0.4"}`, 0},
		{`{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_bucket", le="0.8"}`, 0},
		{`{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_bucket", le="1.6"}`, 1},
		{`{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_bucket", le="3.2"}`, 2},
		{`{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_bucket", le="6.4"}`, 3},
		{`{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_bucket", le="12.8"}`, 3},
		{`{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_count"}`, 3},
		{`{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_sum"}`, 7.8},
		//{`{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_bucket", le="+Inf"}`, 0},
		{`{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_bucket", le="0.1"}`, 0},
		{`{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_bucket", le="0.2"}`, 0},
		{`{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_bucket", le="0.4"}`, 0},
		{`{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_bucket", le="0.8"}`, 0},
		{`{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_bucket", le="1.6"}`, 1},
		{`{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_bucket", le="3.2"}`, 3},
		{`{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_bucket", le="6.4"}`, 3},
		{`{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_bucket", le="12.8"}`, 3},
		{`{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_count"}`, 3},
		{`{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_sum"}`, 6.2},
		{`{client="lb", server="app", __name__="traces_service_graph_request_total"}`, 3},
	}
	appender.ContainsAll(t, expectedMetrics, collectTime)
}

func testData(t *testing.T, path string) *tempopb.Trace {
	f, err := os.Open(path)
	require.NoError(t, err)

	trace := &tempopb.Trace{}
	err = jsonpb.Unmarshal(f, trace)
	require.NoError(t, err)

	return trace
}
