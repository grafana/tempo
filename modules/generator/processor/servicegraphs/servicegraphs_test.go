package servicegraphs

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/go-kit/log"
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
	p := New(cfg, "test", log.NewNopLogger())

	registry := gen.NewRegistry(nil, "test-tenant")
	err := p.RegisterMetrics(registry)
	assert.NoError(t, err)

	now := time.Now()
	registry.SetTimeNow(func() time.Time {
		return now
	})

	traces := testData(t, "testdata/test-sample.json")
	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: traces.Batches})

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

func TestServiceGraphs_tooManySpansErr(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.MaxItems = 0
	p := New(cfg, "test", log.NewNopLogger())

	traces := testData(t, "testdata/test-sample.json")
	err := p.(*processor).consume(traces.Batches)
	assert.True(t, errors.As(err, &tooManySpansError{}))
}

func testData(t *testing.T, path string) *tempopb.Trace {
	f, err := os.Open(path)
	require.NoError(t, err)

	trace := &tempopb.Trace{}
	err = jsonpb.Unmarshal(f, trace)
	require.NoError(t, err)

	return trace
}
