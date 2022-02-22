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

	now := time.Now()
	registry.SetTimeNow(func() time.Time {
		return now
	})

	traces := testData(t, "testdata/test-sample.json")
	err = p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: traces.Batches})
	assert.NoError(t, err)

	// Manually call expire to force collection of edges.
	sgp := p.(*processor)
	sgp.store.Expire()

	appender := &test_util.Appender{}

	collectTime := now
	err = registry.Gather(appender)
	assert.NoError(t, err)

	assert.False(t, appender.IsCommitted)
	assert.False(t, appender.IsRolledback)

	expectedMetrics := []test_util.Metric{
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_client_seconds_bucket", le="0.1"}`, Value: 0},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_client_seconds_bucket", le="0.2"}`, Value: 0},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_client_seconds_bucket", le="0.4"}`, Value: 0},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_client_seconds_bucket", le="0.8"}`, Value: 0},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_client_seconds_bucket", le="1.6"}`, Value: 2},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_client_seconds_bucket", le="3.2"}`, Value: 3},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_client_seconds_bucket", le="6.4"}`, Value: 3},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_client_seconds_bucket", le="12.8"}`, Value: 3},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_client_seconds_bucket", le="+Inf"}`, Value: 3},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_client_seconds_count"}`, Value: 3},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_client_seconds_sum"}`, Value: 4.4},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_server_seconds_bucket", le="0.1"}`, Value: 0},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_server_seconds_bucket", le="0.2"}`, Value: 0},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_server_seconds_bucket", le="0.4"}`, Value: 0},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_server_seconds_bucket", le="0.8"}`, Value: 0},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_server_seconds_bucket", le="1.6"}`, Value: 2},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_server_seconds_bucket", le="3.2"}`, Value: 3},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_server_seconds_bucket", le="6.4"}`, Value: 3},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_server_seconds_bucket", le="12.8"}`, Value: 3},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_server_seconds_bucket", le="+Inf"}`, Value: 3},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_server_seconds_count"}`, Value: 3},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_server_seconds_sum"}`, Value: 5},
		{Labels: `{client="app", server="db", __name__="traces_service_graph_request_total"}`, Value: 3},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_bucket", le="0.1"}`, Value: 0},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_bucket", le="0.2"}`, Value: 0},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_bucket", le="0.4"}`, Value: 0},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_bucket", le="0.8"}`, Value: 0},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_bucket", le="1.6"}`, Value: 1},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_bucket", le="3.2"}`, Value: 2},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_bucket", le="6.4"}`, Value: 3},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_bucket", le="12.8"}`, Value: 3},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_bucket", le="+Inf"}`, Value: 3},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_count"}`, Value: 3},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_client_seconds_sum"}`, Value: 7.8},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_bucket", le="0.1"}`, Value: 0},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_bucket", le="0.2"}`, Value: 0},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_bucket", le="0.4"}`, Value: 0},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_bucket", le="0.8"}`, Value: 0},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_bucket", le="1.6"}`, Value: 1},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_bucket", le="3.2"}`, Value: 3},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_bucket", le="6.4"}`, Value: 3},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_bucket", le="12.8"}`, Value: 3},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_bucket", le="+Inf"}`, Value: 3},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_count"}`, Value: 3},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_server_seconds_sum"}`, Value: 6.2},
		{Labels: `{client="lb", server="app", __name__="traces_service_graph_request_total"}`, Value: 3},
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
