package generator

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/v3/modules/generator/processor"
	"github.com/grafana/tempo/v3/modules/generator/processor/servicegraphs"

	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/v3/modules/generator/registry"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/v3/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/modules/overrides"
)

func TestServiceGraphsBypassTenantOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "overrides.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`overrides:
  collapsed:
    metrics_generator:
      processor:
        service_graphs:
          enable_service_graph_edge_storage: true
          bypass_services: [gateway]
          bypass_service_regex: ['^proxy-.*$']
  visible:
    metrics_generator:
      processor:
        service_graphs:
          bypass_services: []
          bypass_service_regex: []
  strict:
    metrics_generator:
      processor:
        service_graphs:
          enable_service_graph_edge_storage: false
          bypass_services: []
          bypass_service_regex: []
`), 0o600))
	o, err := overrides.NewOverrides(overrides.Config{PerTenantOverrideConfig: path, PerTenantOverridePeriod: model.Duration(time.Hour)}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(context.Background(), o))
	t.Cleanup(func() { require.NoError(t, services.StopAndAwaitTerminated(context.Background(), o)) })
	base := ProcessorConfig{}
	base.RegisterFlagsAndApplyDefaults("", nil)
	base.ServiceGraphs.EnableServiceGraphEdgeStorage = true
	base.ServiceGraphs.BypassServices = []string{"global-gateway"}
	base.ServiceGraphs.BypassServiceRegex = []string{`^global-proxy$`}
	for _, tc := range []struct {
		tenant                                    string
		enabled                                   bool
		exact, patterns                           []string
		wantCollapsed, wantIncoming, wantOutgoing float64
	}{
		{"collapsed", true, []string{"gateway"}, []string{`^proxy-.*$`}, 0, 1, 1},
		{"visible", true, []string{}, []string{}, 0, 1, 1},
		{"strict", false, []string{}, []string{}, 0, 0, 1},
		{"unconfigured", true, nil, nil, 0, 1, 1},
	} {
		t.Run(tc.tenant, func(t *testing.T) {
			copied, err := base.copyWithOverrides(o, tc.tenant)
			require.NoError(t, err)
			require.Equal(t, tc.enabled, copied.ServiceGraphs.EnableServiceGraphEdgeStorage)
			require.Equal(t, tc.exact, o.MetricsGeneratorProcessorServiceGraphsBypassServices(tc.tenant))
			require.Nil(t, copied.ServiceGraphs.BypassServices)
			require.Equal(t, tc.patterns, o.MetricsGeneratorProcessorServiceGraphsBypassServiceRegex(tc.tenant))
			require.Nil(t, copied.ServiceGraphs.BypassServiceRegex)
			require.NoError(t, copied.ServiceGraphs.Validate())
			copied.ServiceGraphs.Workers = 0
			metrics := registry.NewTestRegistry()
			p, err := servicegraphs.New(copied.ServiceGraphs, tc.tenant, metrics, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
			require.NoError(t, err)
			shutdown := sync.OnceFunc(func() { p.Shutdown(context.Background()) })
			t.Cleanup(shutdown)
			traceID := make([]byte, 16)
			traceID[15] = 1
			var batches []*tracev1.ResourceSpans
			for i, service := range []string{"A", "gateway", "proxy-east", "B"} {
				id := make([]byte, 8)
				id[7] = byte(i + 1)
				var parent []byte
				if i > 0 {
					parent = make([]byte, 8)
					parent[7] = byte(i)
				}
				kind := tracev1.Span_SPAN_KIND_CLIENT
				if service == "B" {
					kind = tracev1.Span_SPAN_KIND_SERVER
				}
				batches = append(batches, &tracev1.ResourceSpans{
					Resource:   &resourcev1.Resource{Attributes: []*commonv1.KeyValue{tempopb.MakeKeyValueStringPtr("service.name", service)}},
					ScopeSpans: []*tracev1.ScopeSpans{{Spans: []*tracev1.Span{{TraceId: traceID, SpanId: id, ParentSpanId: parent, Kind: kind, StartTimeUnixNano: 1, EndTimeUnixNano: 2}}}},
				})
			}
			p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: batches})
			shutdown()
			for _, edge := range []struct {
				client, server string
				want           float64
			}{
				{"A", "B", tc.wantCollapsed},
				{"A", "gateway", tc.wantIncoming},
				{"gateway", "proxy-east", tc.wantIncoming},
				{"proxy-east", "B", tc.wantOutgoing},
			} {
				require.Equal(t, edge.want, metrics.Query("traces_service_graph_request_total", labels.FromMap(map[string]string{"client": edge.client, "server": edge.server})), "%s -> %s", edge.client, edge.server)
			}
		})
	}
	require.Equal(t, []string{"global-gateway"}, base.ServiceGraphs.BypassServices)
	require.Equal(t, []string{`^global-proxy$`}, base.ServiceGraphs.BypassServiceRegex)
}

func TestServiceGraphStorageProcessorReload(t *testing.T) {
	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	o := &mockOverrides{processors: map[string]struct{}{processor.ServiceGraphsName: {}}, serviceGraphsEnableServiceGraphEdgeStorage: boolPtr(true)}
	instance, err := newInstance(cfg, "bypass-reload", o, &noopStorage{}, log.NewNopLogger())
	require.NoError(t, err)
	t.Cleanup(instance.shutdown)
	current := instance.processors[processor.ServiceGraphsName]
	require.NoError(t, instance.updateProcessors())
	require.Same(t, current, instance.processors[processor.ServiceGraphsName], "unchanged config must not recreate the processor")
	o.serviceGraphsBypassServices = []string{"gateway"}
	o.serviceGraphsBypassServiceRegex = []string{`^proxy-.*$`}
	require.NoError(t, instance.updateProcessors())
	updated := instance.processors[processor.ServiceGraphsName]
	require.Same(t, current, updated, "presentation-only updates must not recreate the metrics processor")
	require.Empty(t, updated.(*servicegraphs.Processor).Cfg.BypassServices)
	require.Empty(t, updated.(*servicegraphs.Processor).Cfg.BypassServiceRegex)
	// An invalid update must not tear down the working processor.
	o.serviceGraphsMaxItems = new(0)
	require.ErrorContains(t, instance.updateProcessors(), "max_items")
	require.Same(t, updated, instance.processors[processor.ServiceGraphsName])
	// Disabling storage restores strict matching; presentation rules are independent.
	o.serviceGraphsMaxItems = nil
	o.serviceGraphsBypassServices = []string{}
	o.serviceGraphsBypassServiceRegex = []string{}
	o.serviceGraphsEnableServiceGraphEdgeStorage = boolPtr(false)
	require.NoError(t, instance.updateProcessors())
	require.False(t, instance.processors[processor.ServiceGraphsName].(*servicegraphs.Processor).Cfg.EnableServiceGraphEdgeStorage)
	require.Empty(t, cfg.Processor.ServiceGraphs.BypassServices)
}
