package servicegraphs

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	test_util "github.com/grafana/tempo/modules/generator/processor/util/test"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceGraphs(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	p := New(cfg, "test")

	traces := testData(t, "testdata/test-sample.json")
	err := p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: traces.Batches})
	assert.NoError(t, err)

	// Push empty batch to force collection of edges.
	err = p.PushSpans(context.Background(), &tempopb.PushSpansRequest{})

	appender := &test_util.Appender{}

	collectTime := time.Now()
	err = p.CollectMetrics(context.Background(), appender)
	assert.NoError(t, err)

	assert.False(t, appender.IsCommitted)
	assert.False(t, appender.IsRolledback)

	expectedMetrics := []test_util.Metric{
		{`{client="app", server="db", __name__="test_client_latency_bucket", le="+Inf"}`, 0},
		{`{client="app", server="db", __name__="test_client_latency_bucket", le="0.1"}`, 6},
		{`{client="app", server="db", __name__="test_client_latency_bucket", le="0.2"}`, 6},
		{`{client="app", server="db", __name__="test_client_latency_bucket", le="0.4"}`, 6},
		{`{client="app", server="db", __name__="test_client_latency_bucket", le="0.8"}`, 6},
		{`{client="app", server="db", __name__="test_client_latency_bucket", le="1.6"}`, 6},
		{`{client="app", server="db", __name__="test_client_latency_bucket", le="12.8"}`, 6},
		{`{client="app", server="db", __name__="test_client_latency_bucket", le="3.2"}`, 6},
		{`{client="app", server="db", __name__="test_client_latency_bucket", le="6.4"}`, 6},
		{`{client="app", server="db", __name__="test_client_latency_count"}`, 6},
		{`{client="app", server="db", __name__="test_client_latency_sum"}`, 9400},
		{`{client="app", server="db", __name__="test_client_requests_total"}`, 3},
		{`{client="lb", server="app", __name__="test_client_latency_bucket", le="+Inf"}`, 0},
		{`{client="lb", server="app", __name__="test_client_latency_bucket", le="0.1"}`, 6},
		{`{client="lb", server="app", __name__="test_client_latency_bucket", le="0.2"}`, 6},
		{`{client="lb", server="app", __name__="test_client_latency_bucket", le="0.4"}`, 6},
		{`{client="lb", server="app", __name__="test_client_latency_bucket", le="0.8"}`, 6},
		{`{client="lb", server="app", __name__="test_client_latency_bucket", le="1.6"}`, 6},
		{`{client="lb", server="app", __name__="test_client_latency_bucket", le="12.8"}`, 6},
		{`{client="lb", server="app", __name__="test_client_latency_bucket", le="3.2"}`, 6},
		{`{client="lb", server="app", __name__="test_client_latency_bucket", le="6.4"}`, 6},
		{`{client="lb", server="app", __name__="test_client_latency_count"}`, 6},
		{`{client="lb", server="app", __name__="test_client_latency_sum"}`, 14000},
		{`{client="lb", server="app", __name__="test_client_requests_total"}`, 3},
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
