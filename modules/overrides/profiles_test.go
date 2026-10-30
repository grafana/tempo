package overrides

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"
)

func TestPerTenantOverridesProfiles(t *testing.T) {
	limits, err := UnmarshalPerTenantOverrides([]byte(`
profiles:
  standard:
    ingestion:
      rate_limit_bytes: 100
      retry_info_enabled: false
    global:
      max_bytes_per_trace: 0
    metrics_generator:
      processors: [service-graphs]
overrides:
  tenant-a: standard
  tenant-b: standard
  standard:
    ingestion:
      rate_limit_bytes: 200
  empty: {}
  unset: null
  "*": standard
`))
	require.NoError(t, err)
	require.Same(t, limits["tenant-a"], limits["tenant-b"])
	require.Same(t, limits["tenant-a"], limits["*"])
	require.Equal(t, 100, limits["tenant-a"].Ingestion.RateLimitBytes)
	require.NotNil(t, limits["tenant-a"].Ingestion.RetryInfoEnabled)
	require.False(t, *limits["tenant-a"].Ingestion.RetryInfoEnabled)
	require.Zero(t, limits["tenant-a"].Global.MaxBytesPerTrace)
	require.Contains(t, limits["tenant-a"].MetricsGenerator.Processors.GetMap(), "service-graphs")
	require.Equal(t, 200, limits["standard"].Ingestion.RateLimitBytes)
	require.Equal(t, &Overrides{}, limits["empty"])
	require.Nil(t, limits["unset"])
}

func TestPerTenantOverridesProfilesInvalid(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		err  string
	}{
		{"unknown profile", "overrides:\n  tenant: missing", `unknown profile "missing"`},
		{"empty reference", "overrides:\n  tenant: ''", "profile name must not be empty"},
		{"null profile", "profiles:\n  standard: null\noverrides:\n  tenant: standard", `profile "standard" must not be null`},
		{"empty profile name", "profiles:\n  '': {}", "profile name must not be empty"},
		{"unknown top-level key", "profiles: {}\nunknown: true", "field unknown not found"},
		{"unknown profile key", "profiles:\n  standard:\n    typo: 1", `unknown extension key "typo"`},
		{"unknown nested key", "profiles:\n  standard:\n    ingestion:\n      typo: 1", "field typo not found"},
		{"duplicate profile", "profiles:\n  standard: {}\n  standard: {}", "key \"standard\" already set"},
		{"duplicate tenant", "profiles:\n  standard: {}\noverrides:\n  tenant: standard\n  tenant: standard", "key \"tenant\" already set"},
		{"legacy profile", "profiles:\n  standard:\n    max_traces_per_user: 1\noverrides:\n  tenant: standard", "profiles"},
		{"profile inheritance", "profiles:\n  standard: other", "cannot unmarshal"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := UnmarshalPerTenantOverrides([]byte(tc.yaml))
			require.ErrorContains(t, err, tc.err)
		})
	}
}

func TestLoadPerTenantOverridesProfilesValidation(t *testing.T) {
	for _, referenced := range []bool{false, true} {
		t.Run(fmt.Sprint(referenced), func(t *testing.T) {
			input := "profiles:\n  standard:\n    forwarder:\n      workers: -1\n"
			if referenced {
				input += "overrides:\n  tenant: standard\n"
			}
			loader := loadPerTenantOverrides(nil, ConfigTypeNew, false, false)
			_, err := loader(strings.NewReader(input))
			require.ErrorContains(t, err, "forwarder.workers must be non-negative")
		})
	}

	calls := 0
	validator := &mockValidator{f: func(limits *Overrides) error {
		calls++
		if limits.Ingestion.RateLimitBytes == 100 {
			return errors.New("invalid rate")
		}
		return nil
	}}
	loader := loadPerTenantOverrides(validator, ConfigTypeNew, false, false)
	_, err := loader(strings.NewReader("profiles:\n  unused:\n    ingestion:\n      rate_limit_bytes: 100"))
	require.ErrorContains(t, err, "invalid rate")
	require.Equal(t, 1, calls)

	calls = 0
	_, err = loader(strings.NewReader("profiles:\n  standard: {}\noverrides:\n  tenant-a: standard\n  tenant-b: standard"))
	require.NoError(t, err)
	require.Equal(t, 1, calls, "a shared profile should be validated once per reload")
}

func TestLoadPerTenantOverridesProfilesExpandEnv(t *testing.T) {
	t.Setenv("OVERRIDES_PROFILE", "standard")
	t.Setenv("OVERRIDES_RATE", "123")
	loader := loadPerTenantOverrides(nil, ConfigTypeNew, true, false)
	loaded, err := loader(bytes.NewReader([]byte(`
profiles:
  standard:
    ingestion:
      rate_limit_bytes: ${OVERRIDES_RATE}
overrides:
  tenant: ${OVERRIDES_PROFILE}
`)))
	require.NoError(t, err)
	require.Equal(t, 123, loaded.(*perTenantOverrides).forUser("tenant").Ingestion.RateLimitBytes)
}

func TestRuntimeConfigOverridesProfilesReload(t *testing.T) {
	overridesFile := filepath.Join(t.TempDir(), "overrides.yaml")
	writeConfig := func(input string) {
		t.Helper()
		require.NoError(t, os.WriteFile(overridesFile+".tmp", []byte(input), 0o600))
		require.NoError(t, os.Rename(overridesFile+".tmp", overridesFile))
	}
	profileConfig := func(rate int) string {
		return fmt.Sprintf("profiles:\n  standard:\n    ingestion:\n      rate_limit_bytes: %d\noverrides:\n  tenant-a: standard\n  tenant-b: standard\n  '*': standard\n", rate)
	}
	writeConfig(profileConfig(100))
	cfg := Config{
		ConfigType:              ConfigTypeNew,
		PerTenantOverrideConfig: overridesFile,
		PerTenantOverridePeriod: model.Duration(10 * time.Millisecond),
	}
	cfg.Defaults.Ingestion.RateLimitBytes = 300
	cfg.Defaults.Ingestion.BurstSizeBytes = 400
	registry := prometheus.NewRegistry()
	service, err := newRuntimeConfigOverrides(cfg, nil, registry)
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(context.Background(), service))
	t.Cleanup(func() { require.NoError(t, services.StopAndAwaitTerminated(context.Background(), service)) })
	manager := service.(*runtimeConfigOverridesManager)
	first := manager.GetRuntimeOverridesFor("tenant-a")
	require.Same(t, first, manager.GetRuntimeOverridesFor("tenant-b"))
	require.Equal(t, float64(100), manager.IngestionRateLimitBytes("unlisted"))
	require.Zero(t, manager.IngestionBurstSizeBytes("tenant-a"), "profiles preserve inline override semantics")
	require.ElementsMatch(t, []string{"tenant-a", "tenant-b", "*"}, manager.GetTenantIDs())

	status := &bytes.Buffer{}
	require.NoError(t, manager.WriteStatusRuntimeConfig(status, httptest.NewRequest("GET", "/status/runtime_config", nil)))
	var resolved struct {
		Defaults  Overrides             `yaml:"defaults"`
		Overrides map[string]*Overrides `yaml:"overrides"`
	}
	require.NoError(t, yaml.UnmarshalStrict(status.Bytes(), &resolved))
	require.Equal(t, 100, resolved.Overrides["tenant-a"].Ingestion.RateLimitBytes)

	writeConfig(profileConfig(200))
	require.Eventually(t, func() bool {
		return manager.IngestionRateLimitBytes("tenant-a") == 200 && manager.IngestionRateLimitBytes("tenant-b") == 200
	}, 5*time.Second, 10*time.Millisecond)
	require.Equal(t, 100, first.Ingestion.RateLimitBytes, "reload must not mutate an older snapshot")

	writeConfig("overrides:\n  tenant-a: missing\n")
	require.Eventually(t, func() bool {
		metrics, err := registry.Gather()
		if err != nil {
			return false
		}
		for _, metric := range metrics {
			if metric.GetName() == "tempo_runtime_config_last_reload_successful" {
				return metric.GetMetric()[0].GetGauge().GetValue() == 0
			}
		}
		return false
	}, 5*time.Second, 10*time.Millisecond)
	require.Equal(t, float64(200), manager.IngestionRateLimitBytes("tenant-a"), "invalid reload keeps previous configuration")

	writeConfig("overrides: {}")
	require.Eventually(t, func() bool { return manager.IngestionRateLimitBytes("tenant-a") == 300 }, 5*time.Second, 10*time.Millisecond)
}

func BenchmarkPerTenantOverridesProfiles(b *testing.B) {
	for _, tenants := range []int{1000, 100000} {
		for _, profiles := range []bool{false, true} {
			b.Run(fmt.Sprintf("tenants=%d/profiles=%t", tenants, profiles), func(b *testing.B) {
				var input strings.Builder
				const limits = "    ingestion:\n      rate_limit_bytes: 100\n      burst_size_bytes: 200\n    global:\n      max_bytes_per_trace: 300\n    metrics_generator:\n      processors: [service-graphs, span-metrics]\n    compaction:\n      block_retention: 24h\n"
				if profiles {
					input.WriteString("profiles:\n  standard:\n" + limits)
				}
				input.WriteString("overrides:\n")
				for i := range tenants {
					if profiles {
						fmt.Fprintf(&input, "  tenant-%d: standard\n", i)
					} else {
						fmt.Fprintf(&input, "  tenant-%d:\n%s", i, limits)
					}
				}
				data := []byte(input.String())
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					_, err := UnmarshalPerTenantOverrides(data)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func TestPerTenantOverridesProfilesExtensions(t *testing.T) {
	resetRegistryForTesting(t)
	get := RegisterExtension(&testExtension{})
	limits, err := UnmarshalPerTenantOverrides([]byte(`
profiles:
  standard:
    test_extension:
      field_a: shared
      field_b: 5
overrides:
  tenant-a: standard
  tenant-b: standard
`))
	require.NoError(t, err)
	require.Same(t, get(limits["tenant-a"]), get(limits["tenant-b"]))
	require.Equal(t, "shared", get(limits["tenant-a"]).FieldA)

	_, err = UnmarshalPerTenantOverrides([]byte(`
profiles:
  standard:
    test_extension:
      field_a: ""
overrides:
  tenant: standard
`))
	require.ErrorContains(t, err, "field_a cannot be empty")
}
