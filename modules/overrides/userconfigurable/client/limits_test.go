package client

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/grafana/tempo/modules/overrides/histograms"
	"github.com/grafana/tempo/pkg/secrets"
	"github.com/grafana/tempo/pkg/sharedconfig"
	"github.com/grafana/tempo/pkg/util/listtomap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"
)

func strPtr(s string) *string {
	return &s
}

func mapBoolPtr(m map[string]bool) *map[string]bool {
	return &m
}

func TestLimits_parseJson(t *testing.T) {
	testCases := []struct {
		name     string
		json     string
		expected Limits
	}{
		{
			"empty json",
			`{}`,
			Limits{},
		},
		{
			"populated json",
			`{
  "forwarders": ["dev"],
  "metrics_generator": {
    "processors": ["service-graphs"],
    "collection_interval": "30s",
    "trace_id_label_name": "my_trace_id",
    "ingestion_time_range_slack": "45s",
    "native_histogram_bucket_factor": 1.2,
    "native_histogram_min_reset_duration": "10m",
    "native_histogram_max_bucket_number": 101,
    "generate_native_histograms": "native",
    "processor": {
      "service_graphs": {
        "dimensions": ["cluster"],
        "span_multiplier_key": "custom_key"
      },
      "span_metrics": {
        "dimensions": ["cluster"],
        "intrinsic_dimensions": {
          "service": true
        },
        "dimension_mappings": [
          {
            "name": "foo",
            "source_labels": [
              "bar",
			  "bar2"
            ],
            "join": "/"
          },
		  {
            "name": "abc",
            "source_labels": [
              "def"
            ],
            "join": ""
          }
        ],
        "histogram_buckets": [0.1, 0.2, 0.5],
        "span_multiplier_key": "custom_key"
      },
      "host_info": {
        "host_identifiers": ["k8s.node.name", "host.id"],
        "metric_name": "traces_host_info"
      },
      "secret_detection": {
        "disabled_rules": ["generic-api-key"],
        "custom_rules": [
          {"id": "customer-token", "regex": "CUSTOMER-[0-9]+"}
        ]
      }
    }
  }
}`,
			Limits{
				Forwarders: &[]string{"dev"},
				MetricsGenerator: LimitsMetricsGenerator{
					Processors:                      &listtomap.ListToMap{"service-graphs": {}},
					CollectionInterval:              &Duration{Duration: 30 * time.Second},
					TraceIDLabelName:                strPtr("my_trace_id"),
					IngestionSlack:                  &Duration{Duration: 45 * time.Second},
					NativeHistogramBucketFactor:     func(f float64) *float64 { return &f }(1.2),
					NativeHistogramMinResetDuration: &Duration{Duration: 10 * time.Minute},
					NativeHistogramMaxBucketNumber:  func(u uint32) *uint32 { return &u }(101),
					GenerateNativeHistograms:        (*histograms.HistogramMethod)(strPtr("native")),
					Processor: LimitsMetricsGeneratorProcessor{
						ServiceGraphs: LimitsMetricsGeneratorProcessorServiceGraphs{
							Dimensions:        &[]string{"cluster"},
							SpanMultiplierKey: strPtr("custom_key"),
						},
						SpanMetrics: LimitsMetricsGeneratorProcessorSpanMetrics{
							Dimensions:          &[]string{"cluster"},
							HistogramBuckets:    &[]float64{0.1, 0.2, 0.5},
							IntrinsicDimensions: mapBoolPtr(map[string]bool{"service": true}),
							DimensionMappings: &[]sharedconfig.DimensionMappings{
								{
									Name:        "foo",
									SourceLabel: []string{"bar", "bar2"},
									Join:        "/",
								},
								{
									Name:        "abc",
									SourceLabel: []string{"def"},
									Join:        "",
								},
							},
							SpanMultiplierKey: strPtr("custom_key"),
						},
						HostInfo: LimitsMetricGeneratorProcessorHostInfo{
							HostIdentifiers: &[]string{"k8s.node.name", "host.id"},
							MetricName:      strPtr("traces_host_info"),
						},
						SecretDetection: &secrets.Policy{
							DisabledRules: []string{"generic-api-key"},
							CustomRules:   []secrets.CustomRule{{ID: "customer-token", Regex: `CUSTOMER-[0-9]+`}},
						},
					},
				},
			},
		},
		{
			"span_name_sanitization",
			`{
  "metrics_generator": {
    "span_name_sanitization": "enabled"
  }
}`,
			Limits{
				MetricsGenerator: LimitsMetricsGenerator{
					SpanNameSanitization: strPtr("enabled"),
				},
			},
		},
		{
			"empty struct field",
			`{"metrics_generator": {}}`,
			Limits{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var limits Limits
			err := json.Unmarshal([]byte(tc.json), &limits)
			assert.NoError(t, err)

			assert.Equal(t, tc.expected, limits)
			assert.True(t, reflect.DeepEqual(tc.expected, limits))
		})
	}
}

func TestSecretDetectionDisabledRulesSerialization(t *testing.T) {
	compiler, err := secrets.NewPolicyCompiler(nil)
	require.NoError(t, err)
	const value = `api_key="r9Q2m7V4x1Z8c6B3n0H5j2L9p4T7w8Y1"`
	for _, codec := range []struct {
		name      string
		marshal   func(any) ([]byte, error)
		unmarshal func([]byte, any) error
	}{
		{"json", json.Marshal, json.Unmarshal},
		{"yaml", yaml.Marshal, yaml.Unmarshal},
	} {
		for _, test := range []struct {
			name string
			json string
			yaml string
			want secrets.Verdict
		}{
			{
				name: "disabled-with-custom",
				json: `{"metrics_generator":{"processor":{"secret_detection":{"disabled_rules":["generic-api-key"],"custom_rules":[{"id":"customer-token","regex":"CUSTOMER-[0-9]+"}]}}}}`,
				yaml: "metrics_generator:\n  processor:\n    secret_detection:\n      disabled_rules: [generic-api-key]\n      custom_rules:\n        - id: customer-token\n          regex: CUSTOMER-[0-9]+\n",
				want: secrets.Verdict{Matches: []secrets.Match{{RuleID: "customer-token"}}},
			},
			{
				name: "explicit-empty",
				json: `{"metrics_generator":{"processor":{"secret_detection":{"disabled_rules":[]}}}}`,
				yaml: "metrics_generator:\n  processor:\n    secret_detection:\n      disabled_rules: []\n",
				want: secrets.Verdict{Matches: []secrets.Match{{RuleID: "generic-api-key"}}},
			},
			{
				name: "omitted-with-custom",
				json: `{"metrics_generator":{"processor":{"secret_detection":{"custom_rules":[{"id":"customer-token","regex":"CUSTOMER-[0-9]+"}]}}}}`,
				yaml: "metrics_generator:\n  processor:\n    secret_detection:\n      custom_rules:\n        - id: customer-token\n          regex: CUSTOMER-[0-9]+\n",
				want: secrets.Verdict{Matches: []secrets.Match{{RuleID: "generic-api-key"}, {RuleID: "customer-token"}}},
			},
		} {
			t.Run(codec.name+"/"+test.name, func(t *testing.T) {
				input := test.json
				if codec.name == "yaml" {
					input = test.yaml
				}
				var decoded Limits
				require.NoError(t, codec.unmarshal([]byte(input), &decoded))
				encoded, err := codec.marshal(decoded)
				require.NoError(t, err)
				var restored Limits
				require.NoError(t, codec.unmarshal(encoded, &restored))
				for _, limits := range []*Limits{&decoded, &restored} {
					policy := limits.MetricsGenerator.Processor.SecretDetection
					// An explicit empty policy must survive serialization, so
					// it can still replace inherited exclusions/custom rules.
					require.NotNil(t, policy)
					compiled, err := compiler.CompilePolicy(*policy)
					require.NoError(t, err)
					require.Equal(t, test.want, compiled.Detect(value+"\nCUSTOMER-123"))
				}
			})
		}
	}
}
