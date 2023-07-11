package overrides

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestConfig_inlineLimits(t *testing.T) {
	rawYaml := `
max_bytes_per_trace: 100
max_traces_per_user: 1
ingestion_rate_limit_bytes: 500
ingestion_burst_size_bytes: 500
per_tenant_override_config: /overrides/overrides.yaml
`

	cfg := Config{}
	assert.NoError(t, yaml.UnmarshalStrict([]byte(rawYaml), &cfg))

	expected := Config{
		DefaultLimits: Limits{
			MaxBytesPerTrace:        100,
			MaxLocalTracesPerUser:   1,
			IngestionRateLimitBytes: 500,
			IngestionBurstSizeBytes: 500,
		},
		PerTenantOverrideConfig: "/overrides/overrides.yaml",
		PerTenantOverridePeriod: 0,
	}
	assert.Equal(t, expected, cfg)
}

func TestConfig_defaultLimits(t *testing.T) {
	rawYaml := `
default_limits:
  max_bytes_per_trace: 100
  max_traces_per_user: 1
  ingestion_rate_limit_bytes: 500
  ingestion_burst_size_bytes: 500
per_tenant_override_config: /overrides/overrides.yaml
`

	cfg := Config{}
	assert.NoError(t, yaml.UnmarshalStrict([]byte(rawYaml), &cfg))

	expected := Config{
		DefaultLimits: Limits{
			MaxBytesPerTrace:        100,
			MaxLocalTracesPerUser:   1,
			IngestionRateLimitBytes: 500,
			IngestionBurstSizeBytes: 500,
		},
		PerTenantOverrideConfig: "/overrides/overrides.yaml",
		PerTenantOverridePeriod: 0,
	}
	assert.Equal(t, expected, cfg)
}
