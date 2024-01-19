package overrides

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	userconfigurableoverrides "github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	tempo_api "github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/util/listtomap"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
)

const (
	tenant1 = "tenant-1"
	tenant2 = "tenant-2"
)

func TestUserConfigOverridesManager(t *testing.T) {
	defaultLimits := Overrides{
		Global: GlobalOverrides{
			MaxBytesPerTrace: 1024,
		},
		Forwarders: []string{"my-forwarder"},
	}
	_, mgr, cleanup := localUserConfigOverrides(t, defaultLimits, nil)
	defer cleanup()

	// Verify default limits are returned
	assert.Equal(t, 1024, mgr.MaxBytesPerTrace(tenant1))
	assert.Equal(t, []string{"my-forwarder"}, mgr.Forwarders(tenant1))
	assert.Equal(t, 1024, mgr.MaxBytesPerTrace(tenant2))
	assert.Equal(t, []string{"my-forwarder"}, mgr.Forwarders(tenant2))

	// Update limits for tenant-1
	userConfigurableLimits := &userconfigurableoverrides.Limits{
		Forwarders: &[]string{"my-other-forwarder"},
	}
	_, err := mgr.client.Set(context.Background(), tenant2, userConfigurableLimits, backend.VersionNew)
	assert.NoError(t, err)

	assert.NoError(t, mgr.reloadAllTenantLimits(context.Background()))

	// Verify updated limits are returned
	assert.Equal(t, 1024, mgr.MaxBytesPerTrace(tenant1))
	assert.Equal(t, []string{"my-forwarder"}, mgr.Forwarders(tenant1))
	assert.Equal(t, 1024, mgr.MaxBytesPerTrace(tenant2))
	assert.Equal(t, []string{"my-other-forwarder"}, mgr.Forwarders(tenant2))

	// Delete limits for tenant-1
	err = mgr.client.Delete(context.Background(), tenant2, backend.VersionNew)
	assert.NoError(t, err)

	assert.NoError(t, mgr.reloadAllTenantLimits(context.Background()))

	// Verify default limits are returned again
	assert.Equal(t, 1024, mgr.MaxBytesPerTrace(tenant1))
	assert.Equal(t, []string{"my-forwarder"}, mgr.Forwarders(tenant1))
	assert.Equal(t, 1024, mgr.MaxBytesPerTrace(tenant2))
	assert.Equal(t, []string{"my-forwarder"}, mgr.Forwarders(tenant2))
}

func TestUserConfigOverridesManager_allFields(t *testing.T) {
	defaultLimits := Overrides{}
	_, mgr, cleanup := localUserConfigOverrides(t, defaultLimits, nil)
	defer cleanup()

	assert.Empty(t, mgr.Forwarders(tenant1))
	assert.Empty(t, mgr.MetricsGeneratorProcessors(tenant1))
	assert.Equal(t, false, mgr.MetricsGeneratorDisableCollection(tenant1))
	assert.Equal(t, 0*time.Second, mgr.MetricsGeneratorCollectionInterval(tenant1))
	assert.Empty(t, mgr.MetricsGeneratorProcessorServiceGraphsDimensions(tenant1))
	assert.Empty(t, false, mgr.MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix(tenant1))
	assert.Empty(t, mgr.MetricsGeneratorProcessorServiceGraphsPeerAttributes(tenant1))
	assert.Empty(t, mgr.MetricsGeneratorProcessorServiceGraphsHistogramBuckets(tenant1))
	assert.Empty(t, mgr.MetricsGeneratorProcessorSpanMetricsDimensions(tenant1))
	assert.Equal(t, false, mgr.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(tenant1))
	assert.Empty(t, mgr.MetricsGeneratorProcessorSpanMetricsFilterPolicies(tenant1))
	assert.Empty(t, mgr.MetricsGeneratorProcessorSpanMetricsHistogramBuckets(tenant1))
	assert.Empty(t, mgr.MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions(tenant1))

	// Inject user-configurable overrides
	mgr.tenantLimits[tenant1] = &userconfigurableoverrides.Limits{
		Forwarders: &[]string{"my-forwarder"},
		MetricsGenerator: &userconfigurableoverrides.LimitsMetricsGenerator{
			Processors:         map[string]struct{}{"service-graphs": {}},
			DisableCollection:  boolPtr(true),
			CollectionInterval: &userconfigurableoverrides.Duration{Duration: 60 * time.Second},
			Processor: &userconfigurableoverrides.LimitsMetricsGeneratorProcessor{
				ServiceGraphs: &userconfigurableoverrides.LimitsMetricsGeneratorProcessorServiceGraphs{
					Dimensions:               &[]string{"sg-dimension"},
					EnableClientServerPrefix: boolPtr(true),
					PeerAttributes:           &[]string{"attribute"},
					HistogramBuckets:         &[]float64{1, 2, 3, 4, 5},
				},
				SpanMetrics: &userconfigurableoverrides.LimitsMetricsGeneratorProcessorSpanMetrics{
					Dimensions:       &[]string{"sm-dimension"},
					EnableTargetInfo: boolPtr(true),
					FilterPolicies: &[]filterconfig.FilterPolicy{
						{
							Include: &filterconfig.PolicyMatch{
								MatchType: filterconfig.Strict,
								Attributes: []filterconfig.MatchPolicyAttribute{
									{
										Key:   "span.kind",
										Value: "SPAN_KIND_SERVER",
									},
								},
							},
							Exclude: &filterconfig.PolicyMatch{
								MatchType: filterconfig.Strict,
								Attributes: []filterconfig.MatchPolicyAttribute{
									{
										Key:   "span.kind",
										Value: "SPAN_KIND_CONSUMER",
									},
								},
							},
						},
					},
					HistogramBuckets:             &[]float64{10, 20, 30, 40, 50},
					TargetInfoExcludedDimensions: &[]string{"some-label"},
				},
			},
		},
	}

	// Verify we can get the updated overrides
	assert.Equal(t, []string{"my-forwarder"}, mgr.Forwarders(tenant1))
	assert.Equal(t, map[string]struct{}{"service-graphs": {}}, mgr.MetricsGeneratorProcessors(tenant1))
	assert.Equal(t, true, mgr.MetricsGeneratorDisableCollection(tenant1))
	assert.Equal(t, []string{"sg-dimension"}, mgr.MetricsGeneratorProcessorServiceGraphsDimensions(tenant1))
	assert.Equal(t, 60*time.Second, mgr.MetricsGeneratorCollectionInterval(tenant1))
	assert.Equal(t, true, mgr.MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix(tenant1))
	assert.Equal(t, []string{"attribute"}, mgr.MetricsGeneratorProcessorServiceGraphsPeerAttributes(tenant1))
	assert.Equal(t, []float64{1, 2, 3, 4, 5}, mgr.MetricsGeneratorProcessorServiceGraphsHistogramBuckets(tenant1))
	assert.Equal(t, []string{"sm-dimension"}, mgr.MetricsGeneratorProcessorSpanMetricsDimensions(tenant1))
	assert.Equal(t, true, mgr.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(tenant1))
	assert.Equal(t, []float64{10, 20, 30, 40, 50}, mgr.MetricsGeneratorProcessorSpanMetricsHistogramBuckets(tenant1))
	assert.Equal(t, []string{"some-label"}, mgr.MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions(tenant1))

	filterPolicies := mgr.MetricsGeneratorProcessorSpanMetricsFilterPolicies(tenant1)
	assert.NotEmpty(t, filterPolicies)
	assert.Equal(t, filterconfig.Strict, filterPolicies[0].Include.MatchType)
	assert.Equal(t, filterconfig.Strict, filterPolicies[0].Exclude.MatchType)
	assert.Equal(t, "span.kind", filterPolicies[0].Include.Attributes[0].Key)
	assert.Equal(t, "span.kind", filterPolicies[0].Exclude.Attributes[0].Key)
	assert.Equal(t, "SPAN_KIND_SERVER", filterPolicies[0].Include.Attributes[0].Value)
	assert.Equal(t, "SPAN_KIND_CONSUMER", filterPolicies[0].Exclude.Attributes[0].Value)
}

func TestUserConfigOverridesManager_populateFromBackend(t *testing.T) {
	defaultLimits := Overrides{
		Forwarders: []string{"my-forwarder"},
	}
	tempDir, mgr, cleanup := localUserConfigOverrides(t, defaultLimits, nil)
	defer cleanup()

	assert.Equal(t, mgr.Forwarders(tenant1), []string{"my-forwarder"})

	// write directly to backend
	limits := &userconfigurableoverrides.Limits{
		Forwarders: &[]string{"my-other-forwarder"},
	}
	writeUserConfigurableOverridesToDisk(t, tempDir, tenant1, limits)

	// reload from backend
	err := mgr.reloadAllTenantLimits(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, mgr.Forwarders(tenant1), []string{"my-other-forwarder"})
}

func TestUserConfigOverridesManager_deletedFromBackend(t *testing.T) {
	defaultLimits := Overrides{
		Forwarders: []string{"my-forwarder"},
	}
	tempDir, mgr, cleanup := localUserConfigOverrides(t, defaultLimits, nil)
	defer cleanup()

	limits := &userconfigurableoverrides.Limits{
		Forwarders: &[]string{"my-other-forwarder"},
	}
	_, err := mgr.client.Set(context.Background(), tenant1, limits, backend.VersionNew)
	assert.NoError(t, err)

	assert.NoError(t, mgr.reloadAllTenantLimits(context.Background()))

	assert.Equal(t, mgr.Forwarders(tenant1), []string{"my-other-forwarder"})

	// delete overrides.json directly from the backend
	deleteUserConfigurableOverridesFromDisk(t, tempDir, tenant1)

	// reload from backend
	err = mgr.reloadAllTenantLimits(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, mgr.Forwarders("foo"), []string{"my-forwarder"})
}

func TestUserConfigOverridesManager_backendUnavailable(t *testing.T) {
	defaultLimits := Overrides{
		Forwarders: []string{"my-forwarder"},
	}
	_, mgr, cleanup := localUserConfigOverrides(t, defaultLimits, nil)
	defer cleanup()

	limits := &userconfigurableoverrides.Limits{
		Forwarders: &[]string{"my-other-forwarder"},
	}
	_, err := mgr.client.Set(context.Background(), tenant1, limits, backend.VersionNew)
	assert.NoError(t, err)

	assert.NoError(t, mgr.reloadAllTenantLimits(context.Background()))

	// replace reader by this uncooperative fella
	mgr.client = &badClient{}

	// reloading fails
	assert.Error(t, mgr.reloadAllTenantLimits(context.Background()))

	// but overrides should be cached
	assert.Equal(t, []string{"my-other-forwarder"}, mgr.Forwarders(tenant1))
}

func TestUserConfigOverridesManager_WriteStatusRuntimeConfig(t *testing.T) {
	bl := Overrides{Forwarders: []string{"my-forwarder"}}
	_, configurableOverrides, cleanup := localUserConfigOverrides(t, bl, nil)
	defer cleanup()

	// set user config limits
	configurableOverrides.tenantLimits["test"] = &userconfigurableoverrides.Limits{
		Forwarders: &[]string{"my-other-forwarder"},
	}

	tests := []struct {
		name      string
		overrides Service
		req       *http.Request
	}{
		{
			name:      "UserConfigOverrides with ucl",
			overrides: configurableOverrides,
			req:       httptest.NewRequest("GET", "/", nil),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			err := tc.overrides.WriteStatusRuntimeConfig(w, tc.req)
			require.NoError(t, err)

			data := w.Body.String()
			require.Contains(t, data, "user_configurable_overrides")
			require.Contains(t, data, "my-other-forwarder")

			res := w.Result()
			require.Equal(t, "text/plain; charset=utf-8", res.Header.Get(tempo_api.HeaderContentType))
			require.Equal(t, 200, res.StatusCode)
		})
	}
}

func localUserConfigOverrides(t *testing.T, baseLimits Overrides, perTenantOverrides []byte) (string, *userConfigurableOverridesManager, func()) {
	path := t.TempDir()

	cfg := &UserConfigurableOverridesConfig{
		Enabled: true,
		Client: userconfigurableoverrides.Config{
			Backend: backend.Local,
			Local:   &local.Config{Path: path},
		},
		PollInterval: time.Second,
	}

	baseCfg := Config{
		Defaults: baseLimits,
	}

	if perTenantOverrides != nil {
		overridesFile := filepath.Join(t.TempDir(), "Overrides.yaml")

		err := os.WriteFile(overridesFile, perTenantOverrides, os.ModePerm)
		require.NoError(t, err)

		baseCfg.PerTenantOverrideConfig = overridesFile
		baseCfg.PerTenantOverridePeriod = model.Duration(time.Hour)
	}

	baseOverrides, err := NewOverrides(baseCfg, prometheus.NewRegistry())
	assert.NoError(t, err)

	// have to overwrite the registry or test panics with multiple metric reg
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	configurableOverrides, err := newUserConfigOverrides(cfg, baseOverrides)
	assert.NoError(t, err)

	// wait for service and subservices to start and load runtime config
	err = services.StartAndAwaitRunning(context.TODO(), configurableOverrides)
	require.NoError(t, err)

	return path, configurableOverrides, func() {
		err := services.StopAndAwaitTerminated(context.TODO(), configurableOverrides)
		require.NoError(t, err)
	}
}

func writeUserConfigurableOverridesToDisk(t *testing.T, dir string, tenant string, limits *userconfigurableoverrides.Limits) {
	client, err := userconfigurableoverrides.New(&userconfigurableoverrides.Config{
		Backend: backend.Local,
		Local:   &local.Config{Path: dir},
	})
	assert.NoError(t, err)

	_, err = client.Set(context.Background(), tenant, limits, backend.VersionNew)
	assert.NoError(t, err)
}

func deleteUserConfigurableOverridesFromDisk(t *testing.T, dir string, tenant string) {
	client, err := userconfigurableoverrides.New(&userconfigurableoverrides.Config{
		Backend: backend.Local,
		Local:   &local.Config{Path: dir},
	})
	assert.NoError(t, err)

	err = client.Delete(context.Background(), tenant, backend.VersionNew)
	assert.NoError(t, err)
}

type badClient struct{}

var _ userconfigurableoverrides.Client = (*badClient)(nil)

func (b *badClient) List(context.Context) ([]string, error) {
	return nil, errors.New("no")
}

func (b *badClient) Get(context.Context, string) (*userconfigurableoverrides.Limits, backend.Version, error) {
	return nil, "", errors.New("no")
}

func (b *badClient) Set(context.Context, string, *userconfigurableoverrides.Limits, backend.Version) (backend.Version, error) {
	return "", errors.New("no")
}

func (b *badClient) Delete(context.Context, string, backend.Version) error {
	return errors.New("no")
}

func (b badClient) Shutdown() {
}

func boolPtr(b bool) *bool {
	return &b
}

// TestUserConfigOverridesManager_MergeRuntimeConfig tests that per tenant runtime overrides
// are loaded correctly when userconfigurableoverrides are enabled
func TestUserConfigOverridesManager_MergeRuntimeConfig(t *testing.T) {
	tenantID := "test"

	// setup per tenant runtime override for tenant "test"
	pto := perTenantRuntimeOverrides(tenantID)

	_, mgr, cleanup := localUserConfigOverrides(t, Overrides{}, toYamlBytes(t, pto))
	defer cleanup()
	// mgr.Interface will call baseOverrides manager, which is runtime config overrides.
	baseMgr := mgr.Interface

	// Set Forwarders in UserConfigOverrides limits
	mgr.tenantLimits[tenantID] = &userconfigurableoverrides.Limits{
		Forwarders: &[]string{"my-other-forwarder"},
		MetricsGenerator: &userconfigurableoverrides.LimitsMetricsGenerator{
			Processors: map[string]struct{}{"local-blocks": {}},
		},
	}

	// Test all override methods

	// Forwarders are set in user-configurable overrides and will override runtime overrides
	assert.NotEqual(t, mgr.Forwarders(tenantID), baseMgr.Forwarders(tenantID))

	// Processors will be the merged result between runtime and user-configurable overrides
	assert.Equal(t, mgr.MetricsGeneratorProcessors(tenantID), map[string]struct{}{"local-blocks": {}, "service-graphs": {}, "span-metrics": {}})

	// For the remaining settings runtime overrides will bubble up
	assert.Equal(t, mgr.IngestionRateStrategy(), baseMgr.IngestionRateStrategy())
	assert.Equal(t, mgr.MaxLocalTracesPerUser(tenantID), baseMgr.MaxLocalTracesPerUser(tenantID))
	assert.Equal(t, mgr.MaxGlobalTracesPerUser(tenantID), baseMgr.MaxGlobalTracesPerUser(tenantID))
	assert.Equal(t, mgr.MaxBytesPerTrace(tenantID), baseMgr.MaxBytesPerTrace(tenantID))
	assert.Equal(t, mgr.Forwarders(tenantID), []string{"my-other-forwarder"})
	assert.Equal(t, baseMgr.Forwarders(tenantID), []string{"fwd", "fwd-2"})

	assert.Equal(t, mgr.MaxBytesPerTagValuesQuery(tenantID), baseMgr.MaxBytesPerTagValuesQuery(tenantID))
	assert.Equal(t, mgr.MaxBlocksPerTagValuesQuery(tenantID), baseMgr.MaxBlocksPerTagValuesQuery(tenantID))
	assert.Equal(t, mgr.IngestionRateLimitBytes(tenantID), baseMgr.IngestionRateLimitBytes(tenantID))
	assert.Equal(t, mgr.IngestionBurstSizeBytes(tenantID), baseMgr.IngestionBurstSizeBytes(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorIngestionSlack(tenantID), baseMgr.MetricsGeneratorIngestionSlack(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorRingSize(tenantID), baseMgr.MetricsGeneratorRingSize(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorMaxActiveSeries(tenantID), baseMgr.MetricsGeneratorMaxActiveSeries(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorCollectionInterval(tenantID), baseMgr.MetricsGeneratorCollectionInterval(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorDisableCollection(tenantID), baseMgr.MetricsGeneratorDisableCollection(tenantID))
	assert.Equal(t, mgr.MetricsGenerationTraceIDLabelName(tenantID), baseMgr.MetricsGenerationTraceIDLabelName(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorForwarderQueueSize(tenantID), baseMgr.MetricsGeneratorForwarderQueueSize(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorForwarderWorkers(tenantID), baseMgr.MetricsGeneratorForwarderWorkers(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorProcessorServiceGraphsHistogramBuckets(tenantID), baseMgr.MetricsGeneratorProcessorServiceGraphsHistogramBuckets(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorProcessorServiceGraphsDimensions(tenantID), baseMgr.MetricsGeneratorProcessorServiceGraphsDimensions(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorProcessorServiceGraphsPeerAttributes(tenantID), baseMgr.MetricsGeneratorProcessorServiceGraphsPeerAttributes(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorProcessorSpanMetricsHistogramBuckets(tenantID), baseMgr.MetricsGeneratorProcessorSpanMetricsHistogramBuckets(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorProcessorSpanMetricsDimensions(tenantID), baseMgr.MetricsGeneratorProcessorSpanMetricsDimensions(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions(tenantID), baseMgr.MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorProcessorSpanMetricsFilterPolicies(tenantID), baseMgr.MetricsGeneratorProcessorSpanMetricsFilterPolicies(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorProcessorLocalBlocksMaxLiveTraces(tenantID), baseMgr.MetricsGeneratorProcessorLocalBlocksMaxLiveTraces(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorProcessorLocalBlocksMaxBlockDuration(tenantID), baseMgr.MetricsGeneratorProcessorLocalBlocksMaxBlockDuration(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorProcessorLocalBlocksMaxBlockBytes(tenantID), baseMgr.MetricsGeneratorProcessorLocalBlocksMaxBlockBytes(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorProcessorLocalBlocksTraceIdlePeriod(tenantID), baseMgr.MetricsGeneratorProcessorLocalBlocksTraceIdlePeriod(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorProcessorLocalBlocksFlushCheckPeriod(tenantID), baseMgr.MetricsGeneratorProcessorLocalBlocksFlushCheckPeriod(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorProcessorLocalBlocksCompleteBlockTimeout(tenantID), baseMgr.MetricsGeneratorProcessorLocalBlocksCompleteBlockTimeout(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorProcessorSpanMetricsDimensionMappings(tenantID), baseMgr.MetricsGeneratorProcessorSpanMetricsDimensionMappings(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(tenantID), baseMgr.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix(tenantID), baseMgr.MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix(tenantID))
	assert.Equal(t, mgr.MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions(tenantID), baseMgr.MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions(tenantID))
	assert.Equal(t, mgr.BlockRetention(tenantID), baseMgr.BlockRetention(tenantID))
	assert.Equal(t, mgr.MaxSearchDuration(tenantID), baseMgr.MaxSearchDuration(tenantID))
	assert.Equal(t, mgr.DedicatedColumns(tenantID), baseMgr.DedicatedColumns(tenantID))
}

func perTenantRuntimeOverrides(tenantID string) *perTenantOverrides {
	pto := &perTenantOverrides{
		TenantLimits: map[string]*Overrides{
			tenantID: {
				Ingestion: IngestionOverrides{
					RateStrategy:           LocalIngestionRateStrategy,
					RateLimitBytes:         400,
					BurstSizeBytes:         400,
					MaxLocalTracesPerUser:  500,
					MaxGlobalTracesPerUser: 5000,
				},
				Read: ReadOverrides{
					MaxBytesPerTagValuesQuery:  1000,
					MaxBlocksPerTagValuesQuery: 100,
					MaxSearchDuration:          model.Duration(1000 * time.Hour),
				},
				Compaction: CompactionOverrides{
					BlockRetention: model.Duration(360 * time.Hour),
				},
				MetricsGenerator: MetricsGeneratorOverrides{
					RingSize:           2,
					Processors:         listtomap.ListToMap{"span-metrics": {}, "service-graphs": {}},
					MaxActiveSeries:    60000,
					CollectionInterval: 15 * time.Second,
					DisableCollection:  false,
					Forwarder: ForwarderOverrides{
						QueueSize: 400,
						Workers:   3,
					},
					Processor: ProcessorOverrides{
						ServiceGraphs: ServiceGraphsOverrides{
							HistogramBuckets:         []float64{0.002, 0.004, 0.008, 0.016, 0.032, 0.064},
							Dimensions:               []string{"k8s.cluster-name", "k8s.namespace.name", "http.method", "http.route", "http.status_code", "service.version"},
							PeerAttributes:           []string{"foo", "bar"},
							EnableClientServerPrefix: true,
						},
						SpanMetrics: SpanMetricsOverrides{
							HistogramBuckets:             []float64{0.002, 0.004, 0.008, 0.016, 0.032, 0.064},
							Dimensions:                   []string{"k8s.cluster-name", "k8s.namespace.name", "http.method", "http.route", "http.status_code", "service.version"},
							IntrinsicDimensions:          map[string]bool{"foo": true, "bar": true},
							FilterPolicies:               []filterconfig.FilterPolicy{{Exclude: &filterconfig.PolicyMatch{MatchType: filterconfig.Regex, Attributes: []filterconfig.MatchPolicyAttribute{{Key: "resource.service.name", Value: "unknown_service:myservice"}}}}},
							DimensionMappings:            []sharedconfig.DimensionMappings{{Name: "foo", SourceLabel: []string{"bar"}, Join: "baz"}},
							EnableTargetInfo:             true,
							TargetInfoExcludedDimensions: []string{"bar", "namespace", "env"},
						},
						LocalBlocks: LocalBlocksOverrides{
							MaxLiveTraces:        100,
							MaxBlockDuration:     100 * time.Second,
							MaxBlockBytes:        4000,
							FlushCheckPeriod:     10 * time.Second,
							TraceIdlePeriod:      20 * time.Second,
							CompleteBlockTimeout: 30 * time.Second,
						},
					},
					IngestionSlack: 0,
				},
				Forwarders: []string{"fwd", "fwd-2"},
				Global:     GlobalOverrides{MaxBytesPerTrace: 5000000},
				Storage: StorageOverrides{
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "resource", Name: "dedicated.resource.foo", Type: "string"},
						{Scope: "span", Name: "dedicated.span.bar", Type: "string"},
					},
				},
			},
		},
	}

	return pto
}
