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
per_tenant_override_config: /Overrides/Overrides.yaml`

	cfg := Config{}
	cfg.RegisterFlags(&flag.FlagSet{})
	assert.NoError(t, yaml.UnmarshalStrict([]byte(rawYaml), &cfg))

	expected := Config{}
	expected.RegisterFlags(&flag.FlagSet{})
	expected.DefaultLimits.Global.MaxBytesPerTrace = 100
	expected.DefaultLimits.Ingestion.MaxLocalTracesPerUser = 1
	expected.PerTenantOverrideConfig = "/Overrides/Overrides.yaml"
	assert.Equal(t, expected, cfg)
}

func TestConfig_defaultLimits(t *testing.T) {
	rawYaml := `
default_limits:
  global:
    max_bytes_per_trace: 100
  ingestion:  
    max_traces_per_user: 1
per_tenant_override_config: /Overrides/Overrides.yaml`

	cfg := Config{}
	cfg.RegisterFlags(&flag.FlagSet{})
	assert.NoError(t, yaml.UnmarshalStrict([]byte(rawYaml), &cfg))

	expected := Config{}
	expected.RegisterFlags(&flag.FlagSet{})
	expected.DefaultLimits.Global.MaxBytesPerTrace = 100
	expected.DefaultLimits.Ingestion.MaxLocalTracesPerUser = 1
	expected.PerTenantOverrideConfig = "/Overrides/Overrides.yaml"
	assert.Equal(t, expected, cfg)
}

func TestConfig_mixInlineAndDefaultLimits(t *testing.T) {
	rawYaml := `
default_limits:
  max_bytes_per_trace: 100
max_traces_per_user: 1
per_tenant_override_config: /Overrides/Overrides.yaml`

	cfg := Config{}
	cfg.RegisterFlags(&flag.FlagSet{})
	// TODO this error isn't helpful "line 2: field default_limits not found in type Overrides.legacyConfig"
	assert.Error(t, yaml.UnmarshalStrict([]byte(rawYaml), &cfg))
}

func TestConfig_legacy(t *testing.T) {
	legacyRawYaml := `
ingestion_rate_strategy: local
ingestion_rate_limit_bytes: 12345
ingestion_burst_size_bytes: 67890
max_traces_per_user: 1
max_global_traces_per_user: 2
forwarders: ['foo']
metrics_generator_ring_size: 3
metrics_generator_processors: ['span-metrics']
metrics_generator_max_active_series: 4
metrics_generator_collection_interval: 5s
metrics_generator_disable_collection: false
metrics_generator_forwarder_queue_size: 6
metrics_generator_forwarder_workers: 7
metrics_generator_processor_service_graphs_histogram_buckets: [1,2]
metrics_generator_processor_service_graphs_dimensions: ['foo']
metrics_generator_processor_service_graphs_peer_attributes: ['foo']
metrics_generator_processor_service_graphs_enable_client_server_prefix: false
metrics_generator_processor_span_metrics_histogram_buckets: [3,4]
metrics_generator_processor_span_metrics_dimensions: ['foo']
metrics_generator_processor_span_metrics_intrinsic_dimensions:
  foo: true
metrics_generator_processor_span_metrics_filter_policies:
  - include:
      match_type: strict
      attributes:
        - key: foo
          value: bar
metrics_generator_processor_span_metrics_dimension_mappings:
  - name: 'foo'
    source_labels:
      - 'bar'
    join: 'baz'
metrics_generator_processor_span_metrics_enable_target_info: true
metrics_generator_processor_local_blocks_max_live_traces: 8
metrics_generator_processor_local_blocks_max_block_duration: 9s
metrics_generator_processor_local_blocks_max_block_bytes: 10
metrics_generator_processor_local_blocks_flush_check_period: 11s
metrics_generator_processor_local_blocks_trace_idle_period: 12s
metrics_generator_processor_local_blocks_complete_block_timeout: 13s
block_retention: 14s
max_bytes_per_tag_values_query: 15
max_blocks_per_tag_values_query: 16
max_search_duration: 17s
max_bytes_per_trace: 18
per_tenant_override_config: /Overrides/Overrides.yaml
per_tenant_override_period: 19s
`

	legacyCfg := Config{}
	legacyCfg.RegisterFlags(&flag.FlagSet{})
	assert.NoError(t, yaml.UnmarshalStrict([]byte(legacyRawYaml), &legacyCfg))

	rawYaml := `
default_limits:
  ingestion:
    rate_strategy: local
    rate_limit_bytes: 12345
    burst_size_bytes: 67890
    max_traces_per_user: 1
    max_global_traces_per_user: 2
z
  compaction:
    block_retention: 14s
  metrics_generator:
    ring_size: 3
    processors:
    - span-metrics
    max_active_series: 4
    collection_interval: 5s
    disable_collection: false
    forwarder:
      queue_size: 6
      workers: 7
    processor:
      service_graphs:
        histogram_buckets:
        - 1
        - 2
        dimensions:
        - foo
        peer_attributes:
        - foo
        enable_client_server_prefix: false
      span_metrics:
        histogram_buckets:
        - 3
        - 4
        dimensions:
        - foo
        intrinsic_dimensions:
          foo: true
        filter_policies:
        - include:
            match_type: strict
            attributes:
            - key: foo
              value: bar
        dimension_mappings:
        - name: foo
          source_labels:
          - bar
          join: baz
        enable_target_info: true
      local_blocks:
        max_live_traces: 8
        max_block_duration: 9s
        max_block_bytes: 10
        flush_check_period: 11s
        trace_idle_period: 12s
        complete_block_timeout: 13s
  forwarders:
  - foo
  global:
    max_bytes_per_trace: 18
per_tenant_override_config: /Overrides/Overrides.yaml
per_tenant_override_period: 19s
`
	cfg := Config{}
	cfg.RegisterFlags(&flag.FlagSet{})
	assert.NoError(t, yaml.UnmarshalStrict([]byte(rawYaml), &cfg))

	assert.Equal(t, cfg, legacyCfg)
}
