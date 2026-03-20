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
	"go.yaml.in/yaml/v2"

	"github.com/grafana/tempo/modules/overrides/histograms"
	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/util/listtomap"
	"github.com/grafana/tempo/tempodb/backend"
)

// Copied from Cortex
func TestConfigTagsYamlMatchJson(t *testing.T) {
	overrides := reflect.TypeOf(LegacyOverrides{})
	var mismatch []string

	for field := range overrides.Fields() {

		// Skip anonymous embedded fields — they are implicitly inlined by encoding/json.
		if field.Anonymous {
			continue
		}

		// Skip fields intentionally excluded from JSON
		if field.Tag.Get("json") == "-" {
			continue
		}

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
metrics_generator_generate_native_histograms: true
metrics_generator_native_histogram_bucket_factor: 1.1
metrics_generator_native_histogram_max_bucket_number: 100
metrics_generator_native_histogram_min_reset_duration: 15m
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
    native_histogram_bucket_factor: 1.1
    native_histogram_max_bucket_number: 100
    native_histogram_min_reset_duration: 15m
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
	newOverrides, err := legacyOverrides.toNewLimits()
	require.NoError(t, err)
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
		skip := []string{"IngestionArtificialDelay", "MetricsGeneratorSpanNameSanitization", "Extensions"}
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
		baseLegacyOverrides: baseLegacyOverrides{
			IngestionRateStrategy:      "local",
			IngestionRateLimitBytes:    100,
			IngestionBurstSizeBytes:    200,
			IngestionTenantShardSize:   3,
			IngestionMaxAttributeBytes: 1000,
			IngestionArtificialDelay:   durationPtr(5 * time.Minute),
			IngestionRetryInfoEnabled:  true,

			MaxLocalTracesPerUser:  1000,
			MaxGlobalTracesPerUser: 2000,

			Forwarders: []string{"forwarder-1", "forwarder-2"},

			MetricsGeneratorRingSize:                                                    3,
			MetricsGeneratorProcessors:                                                  makeListToMap([]string{"processor-1", "processor-2"}),
			MetricsGeneratorMaxActiveSeries:                                             1000,
			MetricsGeneratorMaxActiveEntities:                                           100,
			MetricsGeneratorMaxCardinalityPerLabel:                                      500,
			MetricsGeneratorCollectionInterval:                                          10 * time.Second,
			MetricsGeneratorDisableCollection:                                           false,
			MetricsGeneratorGenerateNativeHistograms:                                    histograms.HistogramMethodNative,
			MetricsGeneratorTraceIDLabelName:                                            "trace_id",
			MetricsGeneratorForwarderQueueSize:                                          100,
			MetricsGeneratorForwarderWorkers:                                            5,
			MetricsGeneratorRemoteWriteHeaders:                                          RemoteWriteHeaders{"header-1": "value-1"},
			MetricsGeneratorProcessorServiceGraphsHistogramBuckets:                      []float64{1.0, 2.0, 5.0},
			MetricsGeneratorProcessorServiceGraphsDimensions:                            []string{"dimension-1", "dimension-2"},
			MetricsGeneratorProcessorServiceGraphsPeerAttributes:                        []string{"attribute-1", "attribute-2"},
			MetricsGeneratorProcessorServiceGraphsFilterPolicies:                        []filterconfig.FilterPolicy{{Exclude: &filterconfig.PolicyMatch{MatchType: "strict", Attributes: []filterconfig.MatchPolicyAttribute{{Key: "resource.service.name", Value: "my-service"}}}}},
			MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix:              boolPtr(true),
			MetricsGeneratorProcessorServiceGraphsEnableMessagingSystemLatencyHistogram: boolPtr(true),
			MetricsGeneratorProcessorServiceGraphsEnableVirtualNodeLabel:                boolPtr(true),
			MetricsGeneratorProcessorServiceGraphsSpanMultiplierKey:                     "custom_key",
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
			MetricsGeneratorProcessorSpanMetricsEnableTargetInfo:             boolPtr(true),
			MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions: []string{"excluded-dim-1", "excluded-dim-2"},
			MetricsGeneratorProcessorSpanMetricsEnableInstanceLabel:          boolPtr(false),
			MetricsGeneratorProcessorSpanMetricsSpanMultiplierKey:            "custom_key",
			MetricsGeneratorProcessorHostInfoHostIdentifiers:                 []string{"host-id-1", "host-id-2"},
			MetricsGeneratorProcessorHostInfoMetricName:                      "host_info",
			MetricsGeneratorIngestionSlack:                                   1 * time.Minute,
			MetricsGeneratorNativeHistogramBucketFactor:                      1.5,
			MetricsGeneratorNativeHistogramMaxBucketNumber:                   200,
			MetricsGeneratorNativeHistogramMinResetDuration:                  10 * time.Minute,
			MetricsGeneratorSpanNameSanitization:                             "",

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
		},
	}
}

// Helper function to create a duration pointer
func durationPtr(d time.Duration) *time.Duration {
	return &d
}

func TestLegacyOverridesExtra_YAML(t *testing.T) {
	// Unregistered keys in LegacyOverrides YAML must be rejected by processLegacyExtensions.
	input := `
max_traces_per_user: 1000
lbac:
  trace_redaction_mode: mode_spans
`
	var l LegacyOverrides
	err := yaml.Unmarshal([]byte(input), &l)
	require.ErrorContains(t, err, "unknown extension flat key")
}

func TestLegacyOverridesExtra_JSON(t *testing.T) {
	// Unregistered keys in LegacyOverrides JSON must be rejected by processLegacyExtensions.
	input := `{
		"max_traces_per_user": 1000,
		"lbac_mode": "mode_spans"
	}`
	var l LegacyOverrides
	err := json.Unmarshal([]byte(input), &l)
	require.ErrorContains(t, err, "unknown extension flat key")
}

func TestLegacyOverridesExtra_YAMLvsJSON(t *testing.T) {
	// Both YAML and JSON paths must reject unregistered flat keys.
	inputYAML := `
max_traces_per_user: 1000
lbac_mode: mode_spans
`
	inputJSON := `{
		"max_traces_per_user": 1000,
		"lbac_mode": "mode_spans"
	}`

	var lYAML LegacyOverrides
	require.ErrorContains(t, yaml.Unmarshal([]byte(inputYAML), &lYAML), "unknown extension flat key")

	var lJSON LegacyOverrides
	require.ErrorContains(t, json.Unmarshal([]byte(inputJSON), &lJSON), "unknown extension flat key")
}

func TestLegacyToNewLimits_ExtraPreserved(t *testing.T) {
	// Registered extensions survive toNewLimits; unregistered flat keys are rejected at unmarshal.
	ResetRegistryForTesting(t)
	getter := RegisterExtension(&testExtension{})

	input := `
max_traces_per_user: 500
test_extension_field_a: round_trip_val
`
	var l LegacyOverrides
	require.NoError(t, yaml.Unmarshal([]byte(input), &l))
	assert.Equal(t, 500, l.MaxLocalTracesPerUser)

	ext, ok := l.Extensions["test_extension"].(*testExtension)
	require.True(t, ok)
	assert.Equal(t, "round_trip_val", ext.FieldA)

	o, err := l.toNewLimits()
	require.NoError(t, err)
	assert.Equal(t, 500, o.Ingestion.MaxLocalTracesPerUser)

	oExt := getter(&o)
	require.NotNil(t, oExt)
	assert.Equal(t, "round_trip_val", oExt.FieldA)
}

func TestToLegacy_ExtraPreserved(t *testing.T) {
	ResetRegistryForTesting(t)
	getter := RegisterExtension(&testExtension{})

	fieldB := 42
	ext := &testExtension{FieldA: "mode_spans", FieldB: &fieldB}
	var o Overrides
	o.Ingestion.MaxLocalTracesPerUser = 500
	o.Extensions = map[string]Extension{"test_extension": ext}

	l := o.toLegacy()
	assert.Equal(t, 500, l.MaxLocalTracesPerUser)
	// The typed Extension instance is stored in l.Extensions under its nested key.
	assert.Equal(t, ext, l.Extensions["test_extension"], "Extension must survive toLegacy()")
	// The getter still works on the original Overrides.
	assert.Equal(t, ext, getter(&o))
}

// Helper function to create ListToMap
func makeListToMap(items []string) listtomap.ListToMap {
	result := make(listtomap.ListToMap)
	for _, item := range items {
		result[item] = struct{}{}
	}
	return result
}
