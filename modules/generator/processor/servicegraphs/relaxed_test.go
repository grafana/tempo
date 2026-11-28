package servicegraphs

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/modules/generator/registry"
	filterconfig "github.com/grafana/tempo/v3/pkg/spanfilter/config"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	tracev1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
)

func TestRelaxedServiceGraphsSpanKinds(t *testing.T) {
	kinds := []tracev1.Span_SpanKind{tracev1.Span_SPAN_KIND_UNSPECIFIED, tracev1.Span_SPAN_KIND_INTERNAL, tracev1.Span_SPAN_KIND_CLIENT, tracev1.Span_SPAN_KIND_SERVER, tracev1.Span_SPAN_KIND_PRODUCER, tracev1.Span_SPAN_KIND_CONSUMER}
	for _, parentKind := range kinds {
		for _, childKind := range kinds {
			t.Run(parentKind.String()+"/"+childKind.String(), func(t *testing.T) {
				cfg := Config{}
				cfg.RegisterFlagsAndApplyDefaults("", nil)
				cfg.EnableServiceGraphEdgeStorage = true
				cfg.Workers = 0
				metrics := registry.NewTestRegistry()
				p, err := New(cfg, "relaxed-test", metrics, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
				require.NoError(t, err)
				t.Cleanup(func() { p.Shutdown(context.Background()) })
				// Deliver the child before its parent, in separate pushes.
				p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*tracev1.ResourceSpans{makeServiceGraphBatch("B", childKind, []byte{1}, []byte{2}, []byte{1})}})
				p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*tracev1.ResourceSpans{makeServiceGraphBatch("A", parentKind, []byte{1}, []byte{1}, nil)}})
				p.(*Processor).relaxedStore.Expire(time.Now().Add(2 * cfg.Wait))
				want := map[string]string{"client": "A", "server": "B"}
				if parentKind == tracev1.Span_SPAN_KIND_PRODUCER && childKind == tracev1.Span_SPAN_KIND_CONSUMER {
					want["connection_type"] = "messaging_system"
				}
				require.Equal(t, 1.0, metrics.Query(metricRequestTotal, labels.FromMap(want)))
				require.Equal(t, 1.0, metrics.Query(metricRequestClientSeconds+"_count", labels.FromMap(want)))
				require.Equal(t, 1.0, metrics.Query(metricRequestServerSeconds+"_count", labels.FromMap(want)))
			})
		}
	}
}

func TestRelaxedServiceGraphsDimensionsAndFiltering(t *testing.T) {
	for _, filtered := range []bool{false, true} {
		name := "prefixed dimensions"
		if filtered {
			name = "filtered child"
		}
		t.Run(name, func(t *testing.T) {
			cfg := Config{}
			cfg.RegisterFlagsAndApplyDefaults("", nil)
			cfg.EnableServiceGraphEdgeStorage = true
			cfg.Workers = 0
			cfg.EnableClientServerPrefix = true
			cfg.Dimensions = []string{"zone"}
			if filtered {
				cfg.FilterPolicies = []filterconfig.FilterPolicy{{Exclude: &filterconfig.PolicyMatch{
					MatchType: filterconfig.Strict, Attributes: []filterconfig.MatchPolicyAttribute{{Key: "resource.service.name", Value: "B"}},
				}}}
			}
			metrics := registry.NewTestRegistry()
			p, err := New(cfg, "relaxed-test", metrics, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
			require.NoError(t, err)
			t.Cleanup(func() { p.Shutdown(context.Background()) })
			p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*tracev1.ResourceSpans{
				makeServiceGraphBatch("A", tracev1.Span_SPAN_KIND_INTERNAL, []byte{1}, []byte{1}, nil, tempopb.MakeKeyValueStringPtr("zone", "west")),
				makeServiceGraphBatch("B", tracev1.Span_SPAN_KIND_INTERNAL, []byte{1}, []byte{2}, []byte{1}, tempopb.MakeKeyValueStringPtr("zone", "east")),
			}})
			p.(*Processor).relaxedStore.Expire(time.Now().Add(2 * cfg.Wait))
			count := metrics.Query(metricRequestTotal, labels.FromMap(map[string]string{"client": "A", "server": "B", "client_zone": "west", "server_zone": "east"}))
			if filtered {
				require.Zero(t, count)
				require.Empty(t, metrics.String())
			} else {
				require.Equal(t, 1.0, count)
			}
		})
	}
}

func TestRelaxedServiceGraphsDatabaseAndVirtualNodes(t *testing.T) {
	for _, tc := range []struct {
		name                                        string
		kind                                        tracev1.Span_SpanKind
		attribute, peer, client, server, connection string
	}{
		{"database", tracev1.Span_SPAN_KIND_CLIENT, "db.name", "postgres", "A", "postgres", "database"},
		{"virtual client", tracev1.Span_SPAN_KIND_SERVER, "peer.service", "frontend", "frontend", "A", "virtual_node"},
		{"virtual server", tracev1.Span_SPAN_KIND_CLIENT, "peer.service", "payments", "A", "payments", "virtual_node"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{}
			cfg.RegisterFlagsAndApplyDefaults("", nil)
			cfg.EnableServiceGraphEdgeStorage = true
			cfg.Workers = 0
			cfg.Wait = time.Nanosecond
			metrics := registry.NewTestRegistry()
			p, err := New(cfg, "relaxed-test", metrics, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
			require.NoError(t, err)
			t.Cleanup(func() { p.Shutdown(context.Background()) })
			p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*tracev1.ResourceSpans{
				makeServiceGraphBatch("A", tc.kind, []byte{1}, []byte{1}, nil, tempopb.MakeKeyValueStringPtr(tc.attribute, tc.peer)),
			}})
			p.(*Processor).relaxedStore.Expire(time.Now())
			want := labels.FromMap(map[string]string{"client": tc.client, "server": tc.server, "connection_type": tc.connection})
			require.Equal(t, 1.0, metrics.Query(metricRequestTotal, want))
			require.Equal(t, 1.0, metrics.Query(metricRequestServerSeconds+"_count", want))
		})
	}
}

func TestRelaxedServiceGraphsPreservesBypassedServices(t *testing.T) {
	var orders [][]int
	var permute func([]int, int)
	permute = func(order []int, pos int) {
		if pos == len(order) {
			orders = append(orders, append([]int(nil), order...))
			return
		}
		for i := pos; i < len(order); i++ {
			order[pos], order[i] = order[i], order[pos]
			permute(order, pos+1)
			order[pos], order[i] = order[i], order[pos]
		}
	}
	permute([]int{0, 1, 2, 3}, 0)
	for _, order := range orders {
		t.Run(fmt.Sprint(order), func(t *testing.T) {
			cfg := Config{}
			cfg.RegisterFlagsAndApplyDefaults("", nil)
			cfg.EnableServiceGraphEdgeStorage = true
			cfg.BypassServices = []string{"gateway", "proxy"}
			cfg.Workers = 0
			metrics := registry.NewTestRegistry()
			p, err := New(cfg, "bypass-test", metrics, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
			require.NoError(t, err)
			t.Cleanup(func() { p.Shutdown(context.Background()) })
			services := []string{"A", "gateway", "proxy", "B"}
			for _, i := range order {
				var parent []byte
				if i > 0 {
					parent = []byte{byte(i)}
				}
				p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*tracev1.ResourceSpans{
					makeServiceGraphBatch(services[i], tracev1.Span_SPAN_KIND_INTERNAL, []byte{1}, []byte{byte(i + 1)}, parent),
				}})
			}
			p.(*Processor).relaxedStore.Expire(time.Now().Add(2 * cfg.Wait))
			for _, edge := range [][2]string{{"A", "gateway"}, {"gateway", "proxy"}, {"proxy", "B"}} {
				require.Equal(t, 1.0, metrics.Query(metricRequestTotal, labels.FromMap(map[string]string{"client": edge[0], "server": edge[1]})))
			}
		})
	}
}

func TestServiceGraphsPresentationConfigDoesNotEnableStorage(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.BypassServices = []string{"gateway"}
	require.NoError(t, cfg.Validate())
	require.False(t, cfg.EnableServiceGraphEdgeStorage)
	cfg.EnableServiceGraphEdgeStorage = true
	require.NoError(t, cfg.Validate())
}

// Stopping a processor must settle complete pending relationships before the
// matching window closes, rather than losing them during configuration changes.
func TestRelaxedServiceGraphsShutdownSettlesWindow(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.EnableServiceGraphEdgeStorage = true
	cfg.Workers = 0
	cfg.Wait = time.Hour
	metrics := registry.NewTestRegistry()
	p, err := New(cfg, "shutdown-window", metrics, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
	require.NoError(t, err)
	shutdown := sync.OnceFunc(func() { p.Shutdown(context.Background()) })
	t.Cleanup(shutdown)
	batches := []*tracev1.ResourceSpans{
		makeServiceGraphBatch("A", tracev1.Span_SPAN_KIND_CLIENT, []byte{1}, []byte{1}, nil),
		makeServiceGraphBatch("B", tracev1.Span_SPAN_KIND_SERVER, []byte{1}, []byte{2}, []byte{1}),
	}
	for i := 0; i < 2; i++ {
		p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: batches})
	}
	want := labels.FromMap(map[string]string{"client": "A", "server": "B"})
	require.Zero(t, metrics.Query(metricRequestTotal, want))
	shutdown()
	require.Equal(t, 1.0, metrics.Query(metricRequestTotal, want))
}
