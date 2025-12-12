package util

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"testing"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/cmd/tempo/app"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"
)

const (
	tempoConfigFile    = "config.yaml"
	tempoOverridesFile = "overrides.yaml"

	// these paths are all referenced from the other integration test folders. this is why they all have a relative
	// path ../util. this also constrains the folder structure for integration tests to only be one level deeper
	// than ./integration
	baseConfigFile         = "../util/config-base.yaml"
	singleBinaryConfigFile = "../util/config-single-binary.yaml"
	backendConfigFile      = "../util/config-backend-%s.yaml"
	queryBackendConfigFile = "../util/config-query-backend.yaml"
)

func CopyFileToSharedDir(s *e2e.Scenario, src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("unable to read local file %s: %w", src, err)
	}

	_, err = writeFileToSharedDir(s, dst, content)
	return err
}

func writeFileToSharedDir(s *e2e.Scenario, dst string, content []byte) (string, error) {
	dst = sharedContainerPath(s, dst)

	// Ensure the entire path of directories exists
	err := os.MkdirAll(filepath.Dir(dst), os.ModePerm)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(dst, content, os.ModePerm)
	if err != nil {
		return "", err
	}

	return dst, nil
}

func sharedContainerPath(s *e2e.Scenario, file string) string {
	return filepath.Join(s.SharedDir(), file)
}

// setupConfig loads and merges config files, creates the overrides file, and validates the config
func setupConfig(t *testing.T, s *e2e.Scenario, config *TestHarnessConfig, requestedBackend string, harness *TempoHarness) app.Config {
	t.Helper()

	// Initialize template data if needed
	if config.ConfigTemplateData == nil {
		config.ConfigTemplateData = make(map[string]any)
	}

	// Call ConfigTemplateFunc if provided to populate template data
	if config.PreStartHook != nil {
		err := config.PreStartHook(s, config.ConfigTemplateData)
		require.NoError(t, err, "failed to execute config template function")
	}

	// Copy base config to shared directory
	baseConfigPath := baseConfigFile
	err := CopyFileToSharedDir(s, baseConfigPath, tempoConfigFile)
	require.NoError(t, err, "failed to copy base config to shared dir")

	// Apply single binary specific config if in single binary mode
	if config.DeploymentMode == DeploymentModeSingleBinary {
		err := applyConfigOverlay(s, singleBinaryConfigFile, nil)
		require.NoError(t, err, "failed to apply single binary config overlay")
	}

	// backend overlay
	if requestedBackend != backend.Local {
		backendOverlay := fmt.Sprintf(backendConfigFile, requestedBackend)
		err := applyConfigOverlay(s, backendOverlay, nil)
		require.NoError(t, err, "failed to apply backend config overlay", requestedBackend)
	}

	// Apply config overlay if provided
	if config.ConfigOverlay != "" {
		err := applyConfigOverlay(s, config.ConfigOverlay, config.ConfigTemplateData)
		require.NoError(t, err, "failed to apply config overlay")
	}

	// Create empty overrides file
	overridesPath := sharedContainerPath(s, tempoOverridesFile)
	err = os.WriteFile(overridesPath, []byte("overrides: {}\n"), 0644)
	require.NoError(t, err, "failed to write initial overrides file")
	harness.overridesPath = overridesPath

	// Read and parse the final config
	configPath := sharedContainerPath(s, tempoConfigFile)
	configBytes, err := os.ReadFile(configPath)
	require.NoError(t, err, "failed to read merged config file")

	var cfg app.Config
	err = yaml.UnmarshalStrict(configBytes, &cfg)
	require.NoError(t, err, "failed to unmarshal merged config into app.Config")

	return cfg
}

// applyConfigOverlay applies a config overlay file onto the shared config.yaml file,
// with optional template rendering. The overlay is merged onto the existing shared config
// and written back to shared config.yaml.
func applyConfigOverlay(s *e2e.Scenario, overlayPath string, templateData map[string]any) error {
	configPath := sharedContainerPath(s, tempoConfigFile)

	// Read and parse current shared config
	baseBuff, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read shared config file: %w", err)
	}

	var baseMap map[any]any
	err = yaml.Unmarshal(baseBuff, &baseMap)
	if err != nil {
		return fmt.Errorf("failed to parse shared config file: %w", err)
	}

	// If there's an overlay, apply it
	if overlayPath != "" {
		// Read overlay file
		overlayBuff, err := os.ReadFile(overlayPath)
		if err != nil {
			return fmt.Errorf("failed to read config overlay file: %w", err)
		}

		// Apply template rendering if template data is provided
		if len(templateData) > 0 {
			tmpl, err := template.New("config").Parse(string(overlayBuff))
			if err != nil {
				return fmt.Errorf("failed to parse config overlay template: %w", err)
			}

			var renderedBuff bytes.Buffer
			err = tmpl.Execute(&renderedBuff, templateData)
			if err != nil {
				return fmt.Errorf("failed to execute config overlay template: %w", err)
			}

			overlayBuff = renderedBuff.Bytes()
		}

		// Parse overlay
		var overlayMap map[any]any
		err = yaml.Unmarshal(overlayBuff, &overlayMap)
		if err != nil {
			return fmt.Errorf("failed to parse config overlay file: %w", err)
		}

		// Merge overlay onto base
		baseMap = mergeMaps(baseMap, overlayMap)
	}

	// Marshal and write the result back to shared config
	outputBytes, err := yaml.Marshal(baseMap)
	if err != nil {
		return fmt.Errorf("failed to marshal merged config: %w", err)
	}

	err = os.WriteFile(configPath, outputBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// mergeMaps recursively merges overlay map onto base map
// Values in overlay take precedence over base values
func mergeMaps(base, overlay map[any]any) map[any]any {
	result := make(map[any]any)

	// Copy all base values
	for k, v := range base {
		result[k] = v
	}

	// Overlay values, recursively merging nested maps
	for k, v := range overlay {
		if v == nil {
			result[k] = v
			continue
		}

		// If both base and overlay have a map at this key, merge recursively
		if baseVal, exists := result[k]; exists {
			baseMap, baseIsMap := toMapAnyAny(baseVal)
			overlayMap, overlayIsMap := toMapAnyAny(v)

			if baseIsMap && overlayIsMap {
				result[k] = mergeMaps(baseMap, overlayMap)
				continue
			}
		}

		// Otherwise, overlay value replaces base value
		result[k] = v
	}

	return result
}

// toMapAnyAny converts various map types to map[any]any
func toMapAnyAny(v any) (map[any]any, bool) {
	switch m := v.(type) {
	case map[any]any:
		return m, true
	case map[string]any:
		result := make(map[any]any)
		for k, v := range m {
			result[k] = v
		}
		return result, true
	default:
		return nil, false
	}
}
