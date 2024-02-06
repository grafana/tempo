package overrides

import (
	"bytes"
	"encoding/json"
	"flag"
	"reflect"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
)

// Copied from Cortex
func TestConfigTagsYamlMatchJson(t *testing.T) {
	overrides := reflect.TypeOf(LegacyOverrides{})
	n := overrides.NumField()
	var mismatch []string

	for i := 0; i < n; i++ {
		field := overrides.Field(i)

		// Note that we aren't requiring YAML and JSON tags to match, just that
		// they either both exist or both don't exist.
		hasYAMLTag := field.Tag.Get("yaml") != ""
		hasJSONTag := field.Tag.Get("json") != ""

		if hasYAMLTag != hasJSONTag {
			mismatch = append(mismatch, field.Name)
		}
	}

	assert.Empty(t, mismatch, "expected no mismatched JSON and YAML tags")
}

// Copied from Cortex and modified
func TestConfigYamlMatchJson(t *testing.T) {
	inputYAML := `
ingestion_rate_strategy: global
ingestion_rate_limit_bytes: 100_000
ingestion_burst_size_bytes: 100_000

max_traces_per_user: 1000
max_global_traces_per_user: 1000
max_bytes_per_trace: 100_000

block_retention: 24h
compaction_window: 4h

per_tenant_override_config: /etc/Overrides.yaml
per_tenant_override_period: 1m

metrics_generator_send_queue_size: 10
metrics_generator_send_workers: 1

max_search_duration: 5m
`
	inputJSON := `
{
	"ingestion_rate_strategy": "global",
	"ingestion_rate_limit_bytes": 100000,
	"ingestion_burst_size_bytes": 100000,

	"max_traces_per_user": 1000,
	"max_global_traces_per_user": 1000,
	"max_bytes_per_trace": 100000,

	"block_retention": "24h",
	"compaction_window": "4h",

	"per_tenant_override_config": "/etc/Overrides.yaml",
	"per_tenant_override_period": "1m",

	"metrics_generator_send_queue_size": 10,
	"metrics_generator_send_workers": 1,

	"max_search_duration": "5m"
}`

	limitsYAML := LegacyOverrides{}
	err := yaml.Unmarshal([]byte(inputYAML), &limitsYAML)
	require.NoError(t, err, "expected to be able to unmarshal from YAML")

	limitsJSON := LegacyOverrides{}
	err = json.Unmarshal([]byte(inputJSON), &limitsJSON)
	require.NoError(t, err, "expected to be able to unmarshal from JSON")

	assert.Equal(t, limitsYAML, limitsJSON)
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
metrics_generator_remote_write_headers:
  tenant-id: foo
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
user_configurable_overrides:
  enabled: true
`

	legacyCfg := Config{}
	legacyCfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})
	assert.NoError(t, yaml.UnmarshalStrict([]byte(legacyRawYaml), &legacyCfg))
	assert.Equal(t, ConfigTypeLegacy, legacyCfg.ConfigType)
	legacyCfg.ConfigType = ConfigTypeNew // For comparison vs new config

	rawYaml := `
defaults:
  ingestion:
    rate_strategy: local
    rate_limit_bytes: 12345
    burst_size_bytes: 67890
    max_traces_per_user: 1
    max_global_traces_per_user: 2
  read:
    max_bytes_per_tag_values_query: 15
    max_blocks_per_tag_values_query: 16
    max_search_duration: 17s
  compaction:
    block_retention: 14s
  metrics_generator:
    ring_size: 3
    processors:
    - span-metrics
    max_active_series: 4
    collection_interval: 5s
    disable_collection: false
    remote_write_headers:
      tenant-id: foo
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
user_configurable_overrides:
  enabled: true
`
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})
	assert.NoError(t, yaml.UnmarshalStrict([]byte(rawYaml), &cfg))

	assert.Equal(t, cfg, legacyCfg)
}

func TestNumberOfOverrides(t *testing.T) {
	// Asserts that the number of overrides in the new config is the same as the
	// number of overrides in the legacy config.
	assert.Equal(t, countOverrides(LegacyOverrides{}), countOverrides(Overrides{}))
}

// countOverrides recursively counts the number of non-struct fields in a struct.
func countOverrides(v any) int {
	return rvCountFields(reflect.ValueOf(v))
}

func rvCountFields(rv reflect.Value) int {
	if rv.Kind() != reflect.Struct {
		return 0
	}

	n := 0
	for i := 0; i < rv.NumField(); i++ {
		fv := rv.Field(i)
		if fv.Kind() == reflect.Struct {
			n += rvCountFields(fv)
		} else {
			n++
		}
	}
	return n
}

func TestOverrides_AssertUserConfigurableOverridesAreASubsetOfRuntimeOverrides(t *testing.T) {
	userConfigurableOverrides := client.Limits{}

	err := gofakeit.Struct(&userConfigurableOverrides)
	assert.NoError(t, err)

	// TODO clear out collection_interval because unmarshalling a time.Duration into overrides.Overrides
	//  fails. The JSON decoder is not able to parse creations correctly, so e.g. a string like "30s" is
	//  not considered valid.
	//  To fix this we should migrate the various time.Duration to a similar type like client.Duration and
	//  verify they operate the same when marshalling/unmshalling yaml.
	userConfigurableOverrides.MetricsGenerator.CollectionInterval = nil

	// encode to json
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	err = encoder.Encode(&userConfigurableOverrides)
	assert.NoError(t, err)

	// and decode back to overrides.Overrides
	d := json.NewDecoder(&buf)

	// all fields should be known
	d.DisallowUnknownFields()

	var runtimeOverrides Overrides
	err = d.Decode(&runtimeOverrides)
	assert.NoError(t, err)
}
