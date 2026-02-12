package validation

import (
	"testing"

	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/stretchr/testify/require"
)

func TestValidateFilterPolicies(t *testing.T) {
	tests := []struct {
		name       string
		policies   []filterconfig.FilterPolicy
		expErr     bool
		expErrText string
	}{
		{
			name:     "nil policies",
			policies: nil,
		},
		{
			name:     "empty policies",
			policies: []filterconfig.FilterPolicy{},
		},
		{
			name: "valid include with strict match",
			policies: []filterconfig.FilterPolicy{{
				Include: &filterconfig.PolicyMatch{
					MatchType:  filterconfig.Strict,
					Attributes: []filterconfig.MatchPolicyAttribute{{Key: "resource.service.name", Value: "my-service"}},
				},
			}},
		},
		{
			name: "valid exclude with regex match",
			policies: []filterconfig.FilterPolicy{{
				Exclude: &filterconfig.PolicyMatch{
					MatchType:  filterconfig.Regex,
					Attributes: []filterconfig.MatchPolicyAttribute{{Key: "resource.service.name", Value: "unknown_.*"}},
				},
			}},
		},
		{
			name: "valid include and exclude",
			policies: []filterconfig.FilterPolicy{{
				Include: &filterconfig.PolicyMatch{
					MatchType:  filterconfig.Strict,
					Attributes: []filterconfig.MatchPolicyAttribute{{Key: "resource.service.name", Value: "my-service"}},
				},
				Exclude: &filterconfig.PolicyMatch{
					MatchType:  filterconfig.Regex,
					Attributes: []filterconfig.MatchPolicyAttribute{{Key: "span.http.url", Value: "/health.*"}},
				},
			}},
		},
		{
			name: "valid intrinsic attributes",
			policies: []filterconfig.FilterPolicy{{
				Include: &filterconfig.PolicyMatch{
					MatchType: filterconfig.Strict,
					Attributes: []filterconfig.MatchPolicyAttribute{
						{Key: "kind", Value: "SPAN_KIND_SERVER"},
						{Key: "name", Value: "HTTP GET"},
						{Key: "status", Value: "STATUS_CODE_OK"},
					},
				},
			}},
		},
		{
			name: "multiple valid policies",
			policies: []filterconfig.FilterPolicy{
				{
					Include: &filterconfig.PolicyMatch{
						MatchType:  filterconfig.Strict,
						Attributes: []filterconfig.MatchPolicyAttribute{{Key: "resource.service.name", Value: "svc-a"}},
					},
				},
				{
					Exclude: &filterconfig.PolicyMatch{
						MatchType:  filterconfig.Regex,
						Attributes: []filterconfig.MatchPolicyAttribute{{Key: "span.http.url", Value: "/health.*"}},
					},
				},
			},
		},
		{
			name: "valid span-scoped attribute",
			policies: []filterconfig.FilterPolicy{{
				Include: &filterconfig.PolicyMatch{
					MatchType:  filterconfig.Strict,
					Attributes: []filterconfig.MatchPolicyAttribute{{Key: "span.http.method", Value: "GET"}},
				},
			}},
		},
		{
			name: "valid intrinsic with regex match",
			policies: []filterconfig.FilterPolicy{{
				Include: &filterconfig.PolicyMatch{
					MatchType:  filterconfig.Regex,
					Attributes: []filterconfig.MatchPolicyAttribute{{Key: "name", Value: "HTTP.*"}},
				},
			}},
		},
		{
			name:       "no include or exclude",
			policies:   []filterconfig.FilterPolicy{{}},
			expErr:     true,
			expErrText: "must have at least an `include`, `includeAny` or `exclude`",
		},
		{
			name: "invalid match type on include",
			policies: []filterconfig.FilterPolicy{{
				Include: &filterconfig.PolicyMatch{
					MatchType:  "invalid",
					Attributes: []filterconfig.MatchPolicyAttribute{{Key: "resource.service.name", Value: "test"}},
				},
			}},
			expErr:     true,
			expErrText: "invalid match type",
		},
		{
			name: "invalid match type on exclude",
			policies: []filterconfig.FilterPolicy{{
				Exclude: &filterconfig.PolicyMatch{
					MatchType:  "invalid",
					Attributes: []filterconfig.MatchPolicyAttribute{{Key: "resource.service.name", Value: "test"}},
				},
			}},
			expErr:     true,
			expErrText: "invalid match type",
		},
		{
			name: "empty attribute key",
			policies: []filterconfig.FilterPolicy{{
				Include: &filterconfig.PolicyMatch{
					MatchType:  filterconfig.Strict,
					Attributes: []filterconfig.MatchPolicyAttribute{{Key: "", Value: "test"}},
				},
			}},
			expErr:     true,
			expErrText: "invalid attribute",
		},
		{
			name: "unsupported intrinsic",
			policies: []filterconfig.FilterPolicy{{
				Exclude: &filterconfig.PolicyMatch{
					MatchType:  filterconfig.Strict,
					Attributes: []filterconfig.MatchPolicyAttribute{{Key: "duration"}},
				},
			}},
			expErr:     true,
			expErrText: "unsupported intrinsic",
		},
		{
			name: "invalid regex pattern",
			policies: []filterconfig.FilterPolicy{{
				Include: &filterconfig.PolicyMatch{
					MatchType:  filterconfig.Regex,
					Attributes: []filterconfig.MatchPolicyAttribute{{Key: "resource.service.name", Value: ".*("}},
				},
			}},
			expErr:     true,
			expErrText: "invalid attribute filter regexp",
		},
		{
			name: "invalid intrinsic kind value",
			policies: []filterconfig.FilterPolicy{{
				Include: &filterconfig.PolicyMatch{
					MatchType:  filterconfig.Strict,
					Attributes: []filterconfig.MatchPolicyAttribute{{Key: "kind", Value: "INVALID_KIND"}},
				},
			}},
			expErr:     true,
			expErrText: "unsupported kind intrinsic string value",
		},
		{
			name: "invalid intrinsic status value",
			policies: []filterconfig.FilterPolicy{{
				Include: &filterconfig.PolicyMatch{
					MatchType:  filterconfig.Strict,
					Attributes: []filterconfig.MatchPolicyAttribute{{Key: "status", Value: "invalid"}},
				},
			}},
			expErr:     true,
			expErrText: "unsupported status intrinsic string value",
		},
		{
			name: "invalid regex on intrinsic",
			policies: []filterconfig.FilterPolicy{{
				Include: &filterconfig.PolicyMatch{
					MatchType:  filterconfig.Regex,
					Attributes: []filterconfig.MatchPolicyAttribute{{Key: "name", Value: ".*("}},
				},
			}},
			expErr:     true,
			expErrText: "invalid intrinsic filter regex",
		},
		{
			name: "unsupported event attribute scope",
			policies: []filterconfig.FilterPolicy{{
				Include: &filterconfig.PolicyMatch{
					MatchType:  filterconfig.Strict,
					Attributes: []filterconfig.MatchPolicyAttribute{{Key: "event.name", Value: "test"}},
				},
			}},
			expErr:     true,
			expErrText: "invalid or unsupported attribute scope",
		},
		{
			name: "unsupported link attribute scope",
			policies: []filterconfig.FilterPolicy{{
				Include: &filterconfig.PolicyMatch{
					MatchType:  filterconfig.Strict,
					Attributes: []filterconfig.MatchPolicyAttribute{{Key: "link.traceID", Value: "test"}},
				},
			}},
			expErr:     true,
			expErrText: "invalid or unsupported attribute scope",
		},
		{
			name: "unsupported instrumentation attribute scope",
			policies: []filterconfig.FilterPolicy{{
				Include: &filterconfig.PolicyMatch{
					MatchType:  filterconfig.Strict,
					Attributes: []filterconfig.MatchPolicyAttribute{{Key: "instrumentation.name", Value: "test"}},
				},
			}},
			expErr:     true,
			expErrText: "invalid or unsupported attribute scope",
		},
		{
			name: "second policy invalid",
			policies: []filterconfig.FilterPolicy{
				{
					Include: &filterconfig.PolicyMatch{
						MatchType:  filterconfig.Strict,
						Attributes: []filterconfig.MatchPolicyAttribute{{Key: "resource.service.name", Value: "valid"}},
					},
				},
				{},
			},
			expErr:     true,
			expErrText: "must have at least an `include`, `includeAny` or `exclude`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilterPolicies(tt.policies)
			if tt.expErr {
				require.ErrorContains(t, err, tt.expErrText)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateCostAttributionDimensions(t *testing.T) {
	tests := []struct {
		name       string
		dimensions map[string]string
		expErr     bool
		expErrText string
	}{
		{
			name: "valid dimensions with explicit prometheus label names",
			dimensions: map[string]string{
				"tempo.attribute.1": "valid_label_name",
				"tempo.attribute.2": "another_valid_label",
			},
			expErr: false,
		},
		{
			name: "valid dimensions with default mapping (empty value)",
			dimensions: map[string]string{
				"tempo.attribute.1": "",
				"tempo.attribute.2": "",
			},
			expErr: false,
		},
		{
			name: "valid dimensions with mixed explicit and default mapping",
			dimensions: map[string]string{
				"tempo.attribute.1": "custom_label",
				"tempo.attribute.2": "",
				"http.status_code":  "status_code",
				"service.namespace": "",
			},
			expErr: false,
		},
		{
			name: "valid dimensions values with sanitization required",
			dimensions: map[string]string{
				"tempo.attribute":  "label-with-dashes",
				"http.status.code": "status_code_with_underscores",
			},
			expErr: false,
		},
		{
			name:       "empty dimensions map",
			dimensions: map[string]string{},
			expErr:     false,
		},
		{
			name:       "nil dimensions map",
			dimensions: nil,
			expErr:     false,
		},
		{
			name: "dimensions with numeric suffixes",
			dimensions: map[string]string{
				"attribute1": "label_1",
				"attribute2": "label_2",
				"attribute3": "label_3",
			},
			expErr: false,
		},
		{
			name: "dimensions with numeric prefixes",
			dimensions: map[string]string{
				"1.attribute": "1label",
				"2.attribute": "2label",
			},
			expErr: true,
			// we will end up with duplicate output labels after sanitizing label names because they start with digit
			expErrText: "cost_attribution.dimensions has duplicate label name: '_label', both",
		},
		{
			name: "dimensions with numeric prefixes and underscores",
			dimensions: map[string]string{
				"1.attribute": "1_label",
				"2.attribute": "2_label",
			},
			expErr: true,
			// we will end up with invalid output label name after sanitizing label names because they start with digit
			expErrText: "cost_attribution.dimensions config has invalid label name: '__label'",
		},
		{
			name: "dimensions that need sanitization from default mapping",
			dimensions: map[string]string{
				"http.method":          "", // Will be sanitized to http_method
				"service.namespace":    "", // Will be sanitized to service_namespace
				"db.connection.string": "", // Will be sanitized to db_connection_string
			},
			expErr: false,
		},
		{
			name: "valid prometheus reserved labels",
			dimensions: map[string]string{
				"tempo.job":      "job",
				"tempo.instance": "instance",
			},
			expErr: false,
		},
		{
			name: "long dimension names",
			dimensions: map[string]string{
				"very.long.attribute.name.with.many.segments": "very_long_label_name_that_is_still_valid",
			},
			expErr: false,
		},
		{
			name: "dimensions with unicode characters get sanitized",
			dimensions: map[string]string{
				"attribute": "",
				"👻":         "",
			},
			expErr: false,
		},
		{
			name: "dimensions with reserved characters will error",
			dimensions: map[string]string{
				"attribute": "__name__",
				"👻":         "",
			},
			expErr:     true,
			expErrText: "cost_attribution.dimensions config has invalid label name: '__name__'",
		},
		{
			name: "dimensions sanitized into invalid label names will error",
			dimensions: map[string]string{
				"👻👻👻": "",
			},
			expErr:     true,
			expErrText: "cost_attribution.dimensions config has invalid label name: '___'",
		},
		{
			name: "dimensions with duplicate keys will error",
			dimensions: map[string]string{
				"attribute": "name",
				"span.name": "name",
			},
			expErr:     true,
			expErrText: "cost_attribution.dimensions has duplicate label name: 'name', both",
		},
		{
			name: "same attribute across two scopes in dimensions will error",
			dimensions: map[string]string{
				"resource.attribute": "",
				"span.attribute":     "",
			},
			expErr:     true,
			expErrText: "cost_attribution.dimensions has duplicate label name: 'attribute', both",
		},
		{
			name: "same attribute scoped and unscoped scopes in dimensions will error",
			dimensions: map[string]string{
				"resource.attribute": "",
				"attribute":          "",
			},
			expErr:     true,
			expErrText: "cost_attribution.dimensions has duplicate label name: 'attribute', both",
		},
		{
			name: "same attribute across two scopes in dimensions but with mapping",
			dimensions: map[string]string{
				"resource.attribute": "res_name",
				"span.attribute":     "span_name",
			},
			expErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCostAttributionDimensions(tt.dimensions)
			if tt.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expErrText)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
