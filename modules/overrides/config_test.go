package overrides

import (
	"bytes"
	"encoding/json"
	"flag"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/util/listtomap"
	"github.com/grafana/tempo/tempodb/backend"
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
ingestion_tenant_shard_size: 3
ingestion_max_attribute_bytes: 1_000

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
	"ingestion_tenant_shard_size": 3,
	"ingestion_max_attribute_bytes": 1000,

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
ingestion_tenant_shard_size: 3
ingestion_max_attribute_bytes: 1000
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
metrics_generator_processor_service_graphs_enable_messaging_system_latency_histogram: false
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
metrics_generator_generate_native_histograms: true
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
    tenant_shard_size: 3
    max_attribute_bytes: 1000
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
    generate_native_histograms: true
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
        enable_messaging_system_latency_histogram: false
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
	userConfigurableOverrides := client.Limits{
		Forwarders: &[]string{"test"},
		CostAttribution: client.CostAttribution{
			Dimensions: &map[string]string{"server": "192.168.1.1"},
		},
		MetricsGenerator: client.LimitsMetricsGenerator{
			CollectionInterval: &client.Duration{Duration: 5 * time.Minute},
			Processors:         map[string]struct{}{"service-graphs": {}},
		},
	}

	// TODO clear out collection_interval because unmarshalling a time.Duration into overrides.Overrides
	//  fails. The JSON decoder is not able to parse creations correctly, so e.g. a string like "30s" is
	//  not considered valid.
	//  To fix this we should migrate the various time.Duration to a similar type like client.Duration and
	//  verify they operate the same when marshalling/unmshalling yaml.
	userConfigurableOverrides.MetricsGenerator.CollectionInterval = nil

	// encode to json
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	err := encoder.Encode(&userConfigurableOverrides)
	assert.NoError(t, err)

	// and decode back to overrides.Overrides
	d := json.NewDecoder(&buf)

	// all fields should be known
	d.DisallowUnknownFields()

	var runtimeOverrides Overrides
	err = d.Decode(&runtimeOverrides)
	assert.NoError(t, err)
}

func TestFormatConversion(t *testing.T) {
	legacyOverrides := generateTestLegacyOverrides()

	// Verify that all fields have been populated in our test fixture
	ensureAllFieldsPopulated(t, legacyOverrides)

	// Convert to new format and back
	newOverrides := legacyOverrides.toNewLimits()
	convertedLegacyOverrides := newOverrides.toLegacy()

	// Compare original and converted
	assert.Equal(t, legacyOverrides, convertedLegacyOverrides)
}

// ensureAllFieldsPopulated checks that all fields in the struct have non-zero values
// This helps catch if a new field is added to LegacyOverrides but not included in our test fixture
func ensureAllFieldsPopulated(t *testing.T, o LegacyOverrides) {
	v := reflect.ValueOf(o)
	t.Helper()

	// Get the type of the struct
	structType := v.Type()

	// Iterate through all fields
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := structType.Field(i).Name

		// Skip certain fields that can be zero in valid configs
		skip := []string{"IngestionArtificialDelay"}
		if slices.Contains(skip, fieldName) {
			continue
		}

		// For bool fields, we consider that they're explicitly set
		// regardless of whether they're true or false
		if field.Kind() == reflect.Bool {
			continue
		}

		assert.False(t, isZeroValue(field), "Field %s has not been populated in the test fixture - add a value for it", fieldName)
	}
}

// isZeroValue checks if a reflect.Value is the zero value for its type
func isZeroValue(v reflect.Value) bool {
	// Handle nil interfaces and pointers
	if (v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr) && v.IsNil() {
		return true
	}

	// Special case for slices and maps
	if (v.Kind() == reflect.Slice || v.Kind() == reflect.Map) && v.Len() == 0 {
		return true
	}

	// For structs, recursively check fields
	if v.Kind() == reflect.Struct {
		if v.NumField() == 0 {
			return true
		}

		allZero := true
		for i := 0; i < v.NumField(); i++ {
			if !isZeroValue(v.Field(i)) {
				allZero = false
				break
			}
		}
		return allZero
	}

	// For other types, compare with zero value of that type
	zeroValue := reflect.Zero(v.Type()).Interface()
	return reflect.DeepEqual(v.Interface(), zeroValue)
}

// generateTestLegacyOverrides creates a test fixture with predefined values
// If a new field is added to LegacyOverrides, it must be added here as well,
// or the ensureAllFieldsPopulated check will fail.
func generateTestLegacyOverrides() LegacyOverrides {
	// Create a predefined test fixture with values for all fields
	return LegacyOverrides{
		IngestionRateStrategy:      "local",
		IngestionRateLimitBytes:    100,
		IngestionBurstSizeBytes:    200,
		IngestionTenantShardSize:   3,
		IngestionMaxAttributeBytes: 1000,
		IngestionArtificialDelay:   durationPtr(5 * time.Minute),

		MaxLocalTracesPerUser:  1000,
		MaxGlobalTracesPerUser: 2000,

		Forwarders: []string{"forwarder-1", "forwarder-2"},

		MetricsGeneratorRingSize:                                                    3,
		MetricsGeneratorProcessors:                                                  makeListToMap([]string{"processor-1", "processor-2"}),
		MetricsGeneratorMaxActiveSeries:                                             1000,
		MetricsGeneratorCollectionInterval:                                          10 * time.Second,
		MetricsGeneratorDisableCollection:                                           false,
		MetricsGeneratorGenerateNativeHistograms:                                    HistogramMethodNative,
		MetricsGeneratorTraceIDLabelName:                                            "trace_id",
		MetricsGeneratorForwarderQueueSize:                                          100,
		MetricsGeneratorForwarderWorkers:                                            5,
		MetricsGeneratorRemoteWriteHeaders:                                          RemoteWriteHeaders{"header-1": "value-1"},
		MetricsGeneratorProcessorServiceGraphsHistogramBuckets:                      []float64{1.0, 2.0, 5.0},
		MetricsGeneratorProcessorServiceGraphsDimensions:                            []string{"dimension-1", "dimension-2"},
		MetricsGeneratorProcessorServiceGraphsPeerAttributes:                        []string{"attribute-1", "attribute-2"},
		MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix:              true,
		MetricsGeneratorProcessorServiceGraphsEnableMessagingSystemLatencyHistogram: true,
		MetricsGeneratorProcessorServiceGraphsEnableVirtualNodeLabel:                true,
		MetricsGeneratorProcessorSpanMetricsHistogramBuckets:                        []float64{1.0, 2.0, 5.0},
		MetricsGeneratorProcessorSpanMetricsDimensions:                              []string{"dimension-1", "dimension-2"},
		MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions:                     map[string]bool{"dim-1": true, "dim-2": false},
		MetricsGeneratorProcessorSpanMetricsFilterPolicies: []filterconfig.FilterPolicy{
			{
				Include: &filterconfig.PolicyMatch{
					MatchType: "strict",
					Attributes: []filterconfig.MatchPolicyAttribute{
						{Key: "key-1", Value: "value-1"},
					},
				},
				Exclude: &filterconfig.PolicyMatch{
					MatchType: "strict",
					Attributes: []filterconfig.MatchPolicyAttribute{
						{Key: "key-2", Value: "value-2"},
					},
				},
			},
		},
		MetricsGeneratorProcessorSpanMetricsDimensionMappings: []sharedconfig.DimensionMappings{
			{
				Name:        "mapping-1",
				SourceLabel: []string{"source-label-1"},
				Join:        "join-1",
			},
		},
		MetricsGeneratorProcessorSpanMetricsEnableTargetInfo:             true,
		MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions: []string{"excluded-dim-1", "excluded-dim-2"},
		MetricsGeneratorProcessorLocalBlocksMaxLiveTraces:                100,
		MetricsGeneratorProcessorLocalBlocksMaxBlockDuration:             10 * time.Minute,
		MetricsGeneratorProcessorLocalBlocksMaxBlockBytes:                1024 * 1024,
		MetricsGeneratorProcessorLocalBlocksFlushCheckPeriod:             30 * time.Second,
		MetricsGeneratorProcessorLocalBlocksTraceIdlePeriod:              5 * time.Minute,
		MetricsGeneratorProcessorLocalBlocksCompleteBlockTimeout:         30 * time.Minute,
		MetricsGeneratorProcessorHostInfoHostIdentifiers:                 []string{"host-id-1", "host-id-2"},
		MetricsGeneratorProcessorHostInfoMetricName:                      "host_info",
		MetricsGeneratorIngestionSlack:                                   1 * time.Minute,

		BlockRetention:     model.Duration(7 * 24 * time.Hour),
		CompactionDisabled: true,
		CompactionWindow:   model.Duration(4 * time.Hour),

		MaxBytesPerTagValuesQuery:  1000,
		MaxBlocksPerTagValuesQuery: 100,

		MaxSearchDuration:  model.Duration(10 * time.Minute),
		MaxMetricsDuration: model.Duration(30 * time.Minute),
		UnsafeQueryHints:   true,

		MaxBytesPerTrace: 10 * 1024 * 1024,

		CostAttribution: CostAttributionOverrides{
			MaxCardinality: 1000,
			Dimensions:     map[string]string{"dim-1": "value-1", "dim-2": "value-2"},
		},

		DedicatedColumns: backend.DedicatedColumns{
			{
				Scope: backend.DedicatedColumnScopeResource,
				Name:  "resource-column",
				Type:  backend.DedicatedColumnTypeString,
			},
			{
				Scope: backend.DedicatedColumnScopeSpan,
				Name:  "span-column",
				Type:  backend.DedicatedColumnTypeString,
			},
		},
	}
}

// Helper function to create a duration pointer
func durationPtr(d time.Duration) *time.Duration {
	return &d
}

// Helper function to create ListToMap
func makeListToMap(items []string) listtomap.ListToMap {
	result := make(listtomap.ListToMap)
	for _, item := range items {
		result[item] = struct{}{}
	}
	return result
}
