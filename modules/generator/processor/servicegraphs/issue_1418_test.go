package servicegraphs

import (
	"context"
	"testing"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"

	"github.com/grafana/tempo/v3/modules/generator/registry"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	tracev1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
)

func TestServiceGraphsSpanKindRelaxationConfig(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	require.False(t, cfg.EnableSpanKindRelaxation, "tracking additional span kinds must require explicit opt-in")

	require.NoError(t, yaml.UnmarshalStrict([]byte("enable_span_kind_relaxation: true\n"), &cfg))
	require.True(t, cfg.EnableSpanKindRelaxation)
}

// TestServiceGraphsIssue1418 reproduces the issue reporter's cross-cluster Istio
// trace: A (client) -> istio-ingressgateway (client) -> B (server).
// The enabled case deliberately fails until relaxed edge tracking is implemented.
func TestServiceGraphsIssue1418(t *testing.T) {
	for _, tc := range []struct {
		name           string
		enabled        bool
		wantAtoIngress float64
	}{
		{name: "disabled preserves strict matching", enabled: false, wantAtoIngress: 0},
		{name: "enabled connects consecutive client spans", enabled: true, wantAtoIngress: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{}
			cfg.RegisterFlagsAndApplyDefaults("", nil)
			cfg.EnableSpanKindRelaxation = tc.enabled
			// Keep expiration deterministic: the test inspects completed edges only.
			cfg.Workers = 0
			metrics := registry.NewTestRegistry()
			p, err := New(cfg, "issue-1418", metrics, log.NewNopLogger(),
				prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
			require.NoError(t, err)
			t.Cleanup(func() { p.Shutdown(context.Background()) })

			traceID := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
			aID := []byte{0, 0, 0, 0, 0, 0, 0, 1}
			ingressID := []byte{0, 0, 0, 0, 0, 0, 0, 2}
			bID := []byte{0, 0, 0, 0, 0, 0, 0, 3}
			p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*tracev1.ResourceSpans{
				makeServiceGraphBatch("A", tracev1.Span_SPAN_KIND_CLIENT, traceID, aID, nil),
				makeServiceGraphBatch("istio-ingressgateway", tracev1.Span_SPAN_KIND_CLIENT, traceID, ingressID, aID),
				makeServiceGraphBatch("B", tracev1.Span_SPAN_KIND_SERVER, traceID, bID, ingressID),
			}})

			aToIngress := metrics.Query(metricRequestTotal, labels.FromMap(map[string]string{
				"client": "A", "server": "istio-ingressgateway",
			}))
			ingressToB := metrics.Query(metricRequestTotal, labels.FromMap(map[string]string{
				"client": "istio-ingressgateway", "server": "B",
			}))
			t.Logf("enable_span_kind_relaxation=%t: A -> istio-ingressgateway = %g, istio-ingressgateway -> B = %g", tc.enabled, aToIngress, ingressToB)
			t.Logf("generated metrics:\n%s", metrics.String())
			assert.Equal(t, tc.wantAtoIngress, aToIngress, "the client -> client edge must complete when relaxation is enabled")
			assert.Equal(t, 1.0, ingressToB, "the existing client -> server edge must remain present exactly once")
		})
	}
}
