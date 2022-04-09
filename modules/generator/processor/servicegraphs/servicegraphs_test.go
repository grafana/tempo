package servicegraphs

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/tempopb"
)

func TestServiceGraphs(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	cfg.HistogramBuckets = []float64{2.0, 3.0}
	cfg.Dimensions = []string{"component", "does-not-exist"}

	p := New(cfg, "test", testRegistry, log.NewNopLogger())
	defer p.Shutdown(context.Background())

	traces, err := loadTestData("testdata/test-sample.json")
	require.NoError(t, err)

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: traces.Batches})

	// Manually call expire to force collection of edges.
	sgp := p.(*processor)
	sgp.store.Expire()

	lbAppLabels := labels.FromMap(map[string]string{
		"client":         "lb",
		"server":         "app",
		"component":      "net/http",
		"does_not_exist": "",
	})
	appDbLabels := labels.FromMap(map[string]string{
		"client":         "app",
		"server":         "db",
		"component":      "net/http",
		"does_not_exist": "",
	})

	fmt.Println(testRegistry)

	assert.Equal(t, 3.0, testRegistry.Query(`traces_service_graph_request_total`, appDbLabels))

	assert.Equal(t, 2.0, testRegistry.Query(`traces_service_graph_request_client_seconds_bucket`, withLe(appDbLabels, 2.0)))
	assert.Equal(t, 3.0, testRegistry.Query(`traces_service_graph_request_client_seconds_bucket`, withLe(appDbLabels, 3.0)))
	assert.Equal(t, 3.0, testRegistry.Query(`traces_service_graph_request_client_seconds_bucket`, withLe(appDbLabels, math.Inf(1))))
	assert.Equal(t, 3.0, testRegistry.Query(`traces_service_graph_request_client_seconds_count`, appDbLabels))
	assert.Equal(t, 4.4, testRegistry.Query(`traces_service_graph_request_client_seconds_sum`, appDbLabels))

	assert.Equal(t, 2.0, testRegistry.Query(`traces_service_graph_request_server_seconds_bucket`, withLe(appDbLabels, 2.0)))
	assert.Equal(t, 3.0, testRegistry.Query(`traces_service_graph_request_server_seconds_bucket`, withLe(appDbLabels, 3.0)))
	assert.Equal(t, 3.0, testRegistry.Query(`traces_service_graph_request_server_seconds_bucket`, withLe(appDbLabels, math.Inf(1))))
	assert.Equal(t, 3.0, testRegistry.Query(`traces_service_graph_request_server_seconds_count`, appDbLabels))
	assert.Equal(t, 5.0, testRegistry.Query(`traces_service_graph_request_server_seconds_sum`, appDbLabels))

	assert.Equal(t, 3.0, testRegistry.Query(`traces_service_graph_request_total`, lbAppLabels))

	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_client_seconds_bucket`, withLe(lbAppLabels, 2.0)))
	assert.Equal(t, 2.0, testRegistry.Query(`traces_service_graph_request_client_seconds_bucket`, withLe(lbAppLabels, 3.0)))
	assert.Equal(t, 3.0, testRegistry.Query(`traces_service_graph_request_client_seconds_bucket`, withLe(lbAppLabels, math.Inf(1))))
	assert.Equal(t, 3.0, testRegistry.Query(`traces_service_graph_request_client_seconds_count`, lbAppLabels))
	assert.Equal(t, 7.8, testRegistry.Query(`traces_service_graph_request_client_seconds_sum`, lbAppLabels))

	assert.Equal(t, 2.0, testRegistry.Query(`traces_service_graph_request_server_seconds_bucket`, withLe(lbAppLabels, 2.0)))
	assert.Equal(t, 2.0, testRegistry.Query(`traces_service_graph_request_server_seconds_bucket`, withLe(lbAppLabels, 3.0)))
	assert.Equal(t, 3.0, testRegistry.Query(`traces_service_graph_request_server_seconds_bucket`, withLe(lbAppLabels, math.Inf(1))))
	assert.Equal(t, 3.0, testRegistry.Query(`traces_service_graph_request_server_seconds_count`, lbAppLabels))
	assert.Equal(t, 6.2, testRegistry.Query(`traces_service_graph_request_server_seconds_sum`, lbAppLabels))
}

func TestServiceGraphs_tooManySpansErr(t *testing.T) {
	testRegistry := registry.TestRegistry{}

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.MaxItems = 1
	p := New(cfg, "test", &testRegistry, log.NewNopLogger())
	defer p.Shutdown(context.Background())

	traces, err := loadTestData("testdata/test-sample.json")
	require.NoError(t, err)

	err = p.(*processor).consume(traces.Batches)
	assert.True(t, errors.As(err, &tooManySpansError{}))
}

func loadTestData(path string) (*tempopb.Trace, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	trace := &tempopb.Trace{}
	err = jsonpb.Unmarshal(f, trace)
	return trace, err
}

func withLe(lbls labels.Labels, le float64) labels.Labels {
	lb := labels.NewBuilder(lbls)
	lb = lb.Set(labels.BucketLabel, strconv.FormatFloat(le, 'f', -1, 64))
	return lb.Labels()
}
