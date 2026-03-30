package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"
)

func TestMigrateOverridesPerTenant(t *testing.T) {
	tests := []struct {
		name         string
		inputFile    string
		expectedFile string
	}{
		{
			name:         "legacy per-tenant overrides are migrated to new scoped format",
			inputFile:    "test-data/migrate-overrides-per-tenant/legacy-input.yaml",
			expectedFile: "test-data/migrate-overrides-per-tenant/legacy-expected.yaml",
		},
		{
			name:         "new format per-tenant overrides are passed through unchanged",
			inputFile:    "test-data/migrate-overrides-per-tenant/new-format-input.yaml",
			expectedFile: "test-data/migrate-overrides-per-tenant/new-format-expected.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedBytes, err := os.ReadFile(tt.expectedFile)
			require.NoError(t, err)

			outputPath := t.TempDir() + "/output.yaml"
			cmd := migrateOverridesPerTenantCmd{
				OverridesFile: tt.inputFile,
				OutputDest:    outputPath,
			}

			err = cmd.Run(nil)
			require.NoError(t, err)

			actualBytes, err := os.ReadFile(outputPath)
			require.NoError(t, err)

			// Compare as parsed YAML maps to avoid flaky failures from
			// non-deterministic map iteration order (e.g. ListToMap processors).
			var expectedMap, actualMap interface{}
			require.NoError(t, yaml.Unmarshal(expectedBytes, &expectedMap))
			require.NoError(t, yaml.Unmarshal(actualBytes, &actualMap))

			sortStringSlices(expectedMap)
			sortStringSlices(actualMap)

			require.Equal(t, expectedMap, actualMap)
		})
	}
}
