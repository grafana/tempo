package deployments

import (
	"fmt"
	"testing"
	"time"

	"github.com/grafana/e2e"
	e2edb "github.com/grafana/e2e/db"
	"github.com/grafana/tempo/integration/util"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

const configKVStore = "config-kvstore.yaml"

func TestKVStores(t *testing.T) {
	testKVStores := []struct {
		name          string
		setupKVStore  func(*e2e.Scenario, map[string]any) error
		kvstoreConfig string
	}{
		{
			name: "memberlist",
			setupKVStore: func(_ *e2e.Scenario, templateData map[string]any) error {
				templateData["KVStoreConfig"] = `
      store: memberlist`
				return nil
			},
		},
		{
			name: "etcd",
			setupKVStore: func(s *e2e.Scenario, templateData map[string]any) error {
				etcd := e2edb.NewETCD()
				if err := s.StartAndWaitReady(etcd); err != nil {
					return err
				}
				templateData["KVStoreConfig"] = fmt.Sprintf(`
      store: etcd
      etcd:
        endpoints:
          - http://%s:%d`, etcd.Name(), etcd.HTTPPort())
				return nil
			},
		},
		{
			name: "consul",
			setupKVStore: func(s *e2e.Scenario, templateData map[string]any) error {
				consul := e2edb.NewConsul()
				if err := s.StartAndWaitReady(consul); err != nil {
					return err
				}
				templateData["KVStoreConfig"] = fmt.Sprintf(`
      store: consul
      consul:
        host: http://%s:%d`, consul.Name(), consul.HTTPPort())
				return nil
			},
		},
	}

	for _, tc := range testKVStores {
		t.Run(tc.name, func(t *testing.T) {
			util.WithTempoHarness(t, util.TestHarnessConfig{
				ConfigOverlay: configKVStore,
				PreStartHook:  tc.setupKVStore,
			}, func(h *util.TempoHarness) {
				h.WaitTracesWritable(t)

				liveStoreA := h.Services[util.ServiceLiveStoreZoneA]
				liveStoreB := h.Services[util.ServiceLiveStoreZoneB]

				matchers := []*labels.Matcher{
					{Type: labels.MatchEqual, Name: "state", Value: "ACTIVE"},
					{Type: labels.MatchEqual, Name: "name", Value: "live-store"},
				}
				require.NoError(t, liveStoreA.WaitSumMetricsWithOptions(
					e2e.Equals(float64(2)),
					[]string{"tempo_ring_members"},
					e2e.WithLabelMatchers(matchers...),
				), "live stores failed to join ring with %s", tc.name)
				require.NoError(t, liveStoreB.WaitSumMetricsWithOptions(
					e2e.Equals(float64(2)),
					[]string{"tempo_ring_members"},
					e2e.WithLabelMatchers(matchers...),
				), "live stores failed to join ring with %s", tc.name)

				// Send a trace
				info := tempoUtil.NewTraceInfo(time.Now(), "")
				require.NoError(t, h.WriteTraceInfo(info, ""))

				// Wait for trace to be created in live stores
				h.WaitTracesQueryable(t, 1)

				// Find trace
				apiClient := h.APIClientHTTP("")
				util.QueryAndAssertTrace(t, apiClient, info)
				util.SearchTraceQLAndAssertTrace(t, apiClient, info)
			})
		})
	}
}
