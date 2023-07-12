package overrides

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestConfig_inlineLimits(t *testing.T) {
	rawYaml := `
max_bytes_per_trace: 100
max_traces_per_user: 1
per_tenant_override_config: /overrides/overrides.yaml`

	cfg := Config{}
	cfg.RegisterFlags(&flag.FlagSet{})
	assert.NoError(t, yaml.UnmarshalStrict([]byte(rawYaml), &cfg))

	expected := Config{}
	expected.RegisterFlags(&flag.FlagSet{})
	expected.DefaultLimits.MaxBytesPerTrace = 100
	expected.DefaultLimits.MaxLocalTracesPerUser = 1
	expected.PerTenantOverrideConfig = "/overrides/overrides.yaml"
	assert.Equal(t, expected, cfg)
}

func TestConfig_defaultLimits(t *testing.T) {
	rawYaml := `
default_limits:
  max_bytes_per_trace: 100
  max_traces_per_user: 1
per_tenant_override_config: /overrides/overrides.yaml`

	cfg := Config{}
	cfg.RegisterFlags(&flag.FlagSet{})
	assert.NoError(t, yaml.UnmarshalStrict([]byte(rawYaml), &cfg))

	expected := Config{}
	expected.RegisterFlags(&flag.FlagSet{})
	expected.DefaultLimits.MaxBytesPerTrace = 100
	expected.DefaultLimits.MaxLocalTracesPerUser = 1
	expected.PerTenantOverrideConfig = "/overrides/overrides.yaml"
	assert.Equal(t, expected, cfg)
}

func TestConfig_mixInlineAndDefaultLimits(t *testing.T) {
	rawYaml := `
default_limits:
  max_bytes_per_trace: 100
max_traces_per_user: 1
per_tenant_override_config: /overrides/overrides.yaml`

	cfg := Config{}
	cfg.RegisterFlags(&flag.FlagSet{})
	// TODO this error isn't helpful "line 2: field default_limits not found in type overrides.legacyConfig"
	assert.Error(t, yaml.UnmarshalStrict([]byte(rawYaml), &cfg))
}
