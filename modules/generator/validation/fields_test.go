package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
