package overrides

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"

	"github.com/grafana/tempo/v3/modules/overrides/userconfigurable/client"
)

func TestForwarderOverridesSerialization(t *testing.T) {
	for _, queueConfig := range []ForwarderOverrides{{}, {QueueSize: 250, Workers: 4}} {
		t.Run(fmt.Sprintf("size=%d/workers=%d", queueConfig.QueueSize, queueConfig.Workers), func(t *testing.T) {
			want := Overrides{
				Forwarders: []string{"external-otlp"},
				Forwarder:  queueConfig,
				MetricsGenerator: MetricsGeneratorOverrides{
					Forwarder: ForwarderOverrides{QueueSize: 500, Workers: 8},
				},
			}
			for _, format := range []struct {
				name      string
				marshal   func(any) ([]byte, error)
				unmarshal func([]byte, any) error
			}{
				{"yaml", yaml.Marshal, yaml.UnmarshalStrict},
				{"json", json.Marshal, json.Unmarshal},
			} {
				t.Run(format.name, func(t *testing.T) {
					data, err := format.marshal(want)
					require.NoError(t, err)
					var got Overrides
					require.NoError(t, format.unmarshal(data, &got))
					require.Equal(t, want, got)

					legacy := want.toLegacy()
					require.Equal(t, queueConfig.QueueSize, legacy.ForwarderQueueSize)
					require.Equal(t, queueConfig.Workers, legacy.ForwarderWorkers)
					data, err = format.marshal(legacy)
					require.NoError(t, err)
					var gotLegacy LegacyOverrides
					require.NoError(t, format.unmarshal(data, &gotLegacy))
					converted := gotLegacy.toNewLimits()
					require.Equal(t, want.Forwarder, converted.Forwarder)
					require.Equal(t, want.Forwarders, converted.Forwarders)
					require.Equal(t, want.MetricsGenerator.Forwarder, converted.MetricsGenerator.Forwarder)
				})
			}
		})
	}
}

func TestForwarderOverridesConfigParsing(t *testing.T) {
	for _, tc := range []struct {
		name string
		yaml string
		typ  ConfigType
	}{
		{"modern", "defaults:\n  forwarders: [external-otlp]\n  forwarder:\n    queue_size: 250\n    workers: 4\n", ConfigTypeNew},
		{"legacy", "forwarders: [external-otlp]\nforwarder_queue_size: 250\nforwarder_workers: 4\n", ConfigTypeLegacy},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var cfg Config
			require.NoError(t, yaml.UnmarshalStrict([]byte(tc.yaml), &cfg))
			require.Equal(t, tc.typ, cfg.ConfigType)
			require.Equal(t, []string{"external-otlp"}, cfg.Defaults.Forwarders)
			require.Equal(t, ForwarderOverrides{QueueSize: 250, Workers: 4}, cfg.Defaults.Forwarder)
		})
	}
}

func TestForwarderOverridesLookup(t *testing.T) {
	defaults := Overrides{
		Forwarders: []string{"default-forwarder"},
		Forwarder:  ForwarderOverrides{QueueSize: 250, Workers: 4},
	}
	for _, wildcard := range []bool{false, true} {
		t.Run(fmt.Sprintf("wildcard=%t", wildcard), func(t *testing.T) {
			pto := &perTenantOverrides{TenantLimits: map[string]*Overrides{
				"custom": {Forwarders: []string{"tenant-forwarder"}, Forwarder: ForwarderOverrides{QueueSize: 500, Workers: 8}},
				"zero":   {Forwarders: []string{"zero-forwarder"}},
				"null":   nil,
			}}
			fallback := defaults.Forwarder
			if wildcard {
				fallback = ForwarderOverrides{QueueSize: 300, Workers: 6}
				pto.TenantLimits["*"] = &Overrides{Forwarder: fallback}
			}
			mgr, cleanup := createAndInitializeRuntimeOverridesManager(t, defaults, toYamlBytes(t, pto))
			defer cleanup()
			for tenant, want := range map[string]ForwarderOverrides{
				"custom": {QueueSize: 500, Workers: 8},
				"zero":   {QueueSize: 100, Workers: 2},
				"other":  fallback,
				"null":   fallback,
			} {
				require.Equal(t, want.QueueSize, mgr.ForwarderQueueSize(tenant), tenant)
				require.Equal(t, want.Workers, mgr.ForwarderWorkers(tenant), tenant)
			}
			require.Equal(t, []string{"tenant-forwarder"}, mgr.Forwarders("custom"))
		})
	}
}

func TestForwarderOverridesDefaultsValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  ForwarderOverrides
		want ForwarderOverrides
		err  string
	}{
		{"zero", ForwarderOverrides{}, ForwarderOverrides{QueueSize: 100, Workers: 2}, ""},
		{"positive", ForwarderOverrides{QueueSize: 250, Workers: 4}, ForwarderOverrides{QueueSize: 250, Workers: 4}, ""},
		{"size only", ForwarderOverrides{QueueSize: 250}, ForwarderOverrides{QueueSize: 250, Workers: 2}, ""},
		{"workers only", ForwarderOverrides{Workers: 4}, ForwarderOverrides{QueueSize: 100, Workers: 4}, ""},
		{"negative size", ForwarderOverrides{QueueSize: -1, Workers: 4}, ForwarderOverrides{}, "forwarder.queue_size"},
		{"negative workers", ForwarderOverrides{QueueSize: 250, Workers: -1}, ForwarderOverrides{}, "forwarder.workers"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mgr, err := newRuntimeConfigOverrides(Config{Defaults: Overrides{Forwarder: tc.cfg}}, nil, prometheus.NewRegistry())
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
				require.Nil(t, mgr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want.QueueSize, mgr.ForwarderQueueSize("tenant"))
			require.Equal(t, tc.want.Workers, mgr.ForwarderWorkers("tenant"))
			require.Equal(t, tc.cfg, mgr.GetRuntimeOverridesFor("tenant").Forwarder)
		})
	}
}

func TestForwarderOverridesAccessorDefaults(t *testing.T) {
	for _, source := range []string{"defaults", "tenant", "wildcard"} {
		for _, tc := range []struct {
			name string
			cfg  ForwarderOverrides
			want ForwarderOverrides
		}{
			{"zero", ForwarderOverrides{}, ForwarderOverrides{QueueSize: 100, Workers: 2}},
			{"positive", ForwarderOverrides{QueueSize: 250, Workers: 4}, ForwarderOverrides{QueueSize: 250, Workers: 4}},
			{"size only", ForwarderOverrides{QueueSize: 250}, ForwarderOverrides{QueueSize: 250, Workers: 2}},
			{"workers only", ForwarderOverrides{Workers: 4}, ForwarderOverrides{QueueSize: 100, Workers: 4}},
		} {
			t.Run(source+"/"+tc.name, func(t *testing.T) {
				defaults := Overrides{Forwarder: ForwarderOverrides{QueueSize: 700, Workers: 9}}
				pto := &perTenantOverrides{TenantLimits: map[string]*Overrides{}}
				switch source {
				case "defaults":
					defaults.Forwarder = tc.cfg
				case "tenant":
					pto.TenantLimits["tenant"] = &Overrides{Forwarder: tc.cfg}
				case "wildcard":
					pto.TenantLimits["*"] = &Overrides{Forwarder: tc.cfg}
				}
				mgr, cleanup := createAndInitializeRuntimeOverridesManager(t, defaults, toYamlBytes(t, pto))
				defer cleanup()
				require.Equal(t, tc.want.QueueSize, mgr.ForwarderQueueSize("tenant"))
				require.Equal(t, tc.want.Workers, mgr.ForwarderWorkers("tenant"))
				require.Equal(t, tc.cfg, mgr.GetRuntimeOverridesFor("tenant").Forwarder)
			})
		}
	}
}

func TestForwarderOverridesRuntimeLoading(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		for _, withValidator := range []bool{false, true} {
			for _, tc := range []struct {
				name string
				cfg  ForwarderOverrides
				err  string
			}{
				{"zero", ForwarderOverrides{}, ""},
				{"positive", ForwarderOverrides{QueueSize: 250, Workers: 4}, ""},
				{"negative size", ForwarderOverrides{QueueSize: -1, Workers: 4}, "forwarder.queue_size"},
				{"negative workers", ForwarderOverrides{QueueSize: 250, Workers: -1}, "forwarder.workers"},
			} {
				t.Run(fmt.Sprintf("legacy=%t/validator=%t/%s", legacy, withValidator, tc.name), func(t *testing.T) {
					input := fmt.Sprintf("overrides:\n  tenant:\n    forwarders: [external-otlp]\n    forwarder:\n      queue_size: %d\n      workers: %d\n", tc.cfg.QueueSize, tc.cfg.Workers)
					typ := ConfigTypeNew
					if legacy {
						typ = ConfigTypeLegacy
						input = fmt.Sprintf("overrides:\n  tenant:\n    forwarders: [external-otlp]\n    forwarder_queue_size: %d\n    forwarder_workers: %d\n", tc.cfg.QueueSize, tc.cfg.Workers)
					}
					var validator Validator
					if withValidator {
						validator = &mockValidator{}
					}
					got, err := loadPerTenantOverrides(validator, typ, false, legacy)(bytes.NewBufferString(input))
					if tc.err != "" {
						require.ErrorContains(t, err, "tenant")
						require.ErrorContains(t, err, tc.err)
						require.Nil(t, got)
						return
					}
					require.NoError(t, err)
					limits := got.(*perTenantOverrides).TenantLimits["tenant"]
					require.Equal(t, tc.cfg, limits.Forwarder)
					require.Equal(t, []string{"external-otlp"}, limits.Forwarders)
				})
			}
		}
	}
}

func TestForwarderOverridesUserConfigDelegation(t *testing.T) {
	defaults := Overrides{Forwarder: ForwarderOverrides{QueueSize: 250, Workers: 4}}
	pto := &perTenantOverrides{TenantLimits: map[string]*Overrides{
		"custom": {Forwarder: ForwarderOverrides{QueueSize: 500, Workers: 8}},
		"zero":   {},
	}}
	_, mgr, cleanup := localUserConfigOverrides(t, defaults, toYamlBytes(t, pto))
	defer cleanup()
	mgr.setTenantLimit("custom", &client.Limits{Forwarders: &[]string{"user-forwarder"}})
	for tenant, want := range map[string]ForwarderOverrides{
		"custom": {QueueSize: 500, Workers: 8},
		"other":  defaults.Forwarder,
		"zero":   {QueueSize: 100, Workers: 2},
	} {
		require.Equal(t, want.QueueSize, mgr.ForwarderQueueSize(tenant))
		require.Equal(t, want.Workers, mgr.ForwarderWorkers(tenant))
	}
	require.Equal(t, []string{"user-forwarder"}, mgr.Forwarders("custom"))
}
