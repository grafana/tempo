package main

import (
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"
)

func TestMigrateOverridesConfig(t *testing.T) {
	tests := []struct {
		name         string
		inputFile    string
		expectedFile string
	}{
		{
			name:         "legacy config is migrated to new scoped format",
			inputFile:    "test-data/migrate-overrides-config/legacy-input.yaml",
			expectedFile: "test-data/migrate-overrides-config/legacy-expected.yaml",
		},
		{
			name:         "new format config is passed through unchanged",
			inputFile:    "test-data/migrate-overrides-config/new-format-input.yaml",
			expectedFile: "test-data/migrate-overrides-config/new-format-expected.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedBytes, err := os.ReadFile(tt.expectedFile)
			require.NoError(t, err)

			outputPath := t.TempDir() + "/output.yaml"
			cmd := migrateOverridesConfigCmd{
				ConfigFile: tt.inputFile,
				ConfigDest: outputPath,
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

// sortStringSlices recursively sorts []interface{} slices that contain only strings
// to handle non-deterministic ordering from map-backed types like ListToMap.
func sortStringSlices(v interface{}) {
	switch val := v.(type) {
	case map[interface{}]interface{}:
		for _, v := range val {
			sortStringSlices(v)
		}
	case []interface{}:
		allStrings := true
		for _, item := range val {
			if _, ok := item.(string); !ok {
				allStrings = false
			}
			sortStringSlices(item)
		}
		if allStrings && len(val) > 0 {
			sort.Slice(val, func(i, j int) bool {
				return val[i].(string) < val[j].(string)
			})
		}
	}
}
