package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"

	"go.yaml.in/yaml/v2"

	"github.com/grafana/tempo/cmd/tempo/app"
)

type migrateOverridesConfigCmd struct {
	ConfigFile string `arg:"" help:"Path to the full tempo config file"`

	ConfigDest string `type:"path" short:"d" help:"Path to write the migrated overrides section. If not specified, output to stdout"`
}

func (cmd *migrateOverridesConfigCmd) Run(*globalOptions) error {
	cfg := app.Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})

	// Build the default overrides config for comparison from a separate instance
	// to avoid shared pointers between defaultOverrides and cfg.Overrides.
	defaultCfg := app.Config{}
	defaultCfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	defaultOverrides := defaultCfg.Overrides

	buff, err := os.ReadFile(cmd.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read configFile %s: %w", cmd.ConfigFile, err)
	}

	// Legacy overrides are automatically converted to a new format during unmarshaling
	// via Config.UnmarshalYAML.
	if err := yaml.UnmarshalStrict(buff, &cfg); err != nil {
		return fmt.Errorf("failed to parse configFile %s: %w", cmd.ConfigFile, err)
	}

	// Marshal both configs to maps so we can diff them.
	loadedMap, err := toMap(cfg.Overrides)
	if err != nil {
		return fmt.Errorf("failed to convert loaded overrides to map: %w", err)
	}

	defaultMap, err := toMap(defaultOverrides)
	if err != nil {
		return fmt.Errorf("failed to convert default overrides to map: %w", err)
	}

	// Remove keys that match the defaults so we only output user-set values.
	removeDefaults(loadedMap, defaultMap)

	result := map[string]interface{}{
		"overrides": loadedMap,
	}

	outputBytes, err := yaml.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal overrides: %w", err)
	}

	printWarnings(os.Stderr)

	if cmd.ConfigDest != "" {
		if err := os.WriteFile(cmd.ConfigDest, outputBytes, 0o600); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Migrated overrides written to %s\n", cmd.ConfigDest)
		fmt.Fprintln(os.Stderr, "Replace the overrides section in your config file with the contents of this file.")
	} else {
		fmt.Fprintln(os.Stderr, "Migrated overrides config. Replace the overrides section in your config file with the output below:")
		fmt.Fprintln(os.Stderr, "---")
		fmt.Print(string(outputBytes))
	}

	return nil
}

// toMap converts a struct to a normalized map[string]interface{} via yaml round-trip.
// All nested maps are normalized to map[string]interface{}.
func toMap(v interface{}) (map[string]interface{}, error) {
	b, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return normalizeMap(m), nil
}

// normalizeMap recursively converts all map[interface{}]interface{} to map[string]interface{}.
func normalizeMap(m map[string]interface{}) map[string]interface{} {
	for k, v := range m {
		m[k] = normalizeValue(v)
	}
	return m
}

func normalizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, v := range val {
			out[fmt.Sprintf("%v", k)] = normalizeValue(v)
		}
		return out
	case map[string]interface{}:
		return normalizeMap(val)
	case []interface{}:
		for i, item := range val {
			val[i] = normalizeValue(item)
		}
		return val
	default:
		return v
	}
}

// removeDefaults recursively removes keys from loaded where the value matches the default.
func removeDefaults(loaded, defaults map[string]interface{}) {
	for key, defaultVal := range defaults {
		loadedVal, ok := loaded[key]
		if !ok {
			continue
		}

		loadedChild, loadedIsMap := loadedVal.(map[string]interface{})
		defaultChild, defaultIsMap := defaultVal.(map[string]interface{})

		if loadedIsMap && defaultIsMap {
			removeDefaults(loadedChild, defaultChild)
			if len(loadedChild) == 0 {
				delete(loaded, key)
			}
			continue
		}

		if reflect.DeepEqual(loadedVal, defaultVal) {
			delete(loaded, key)
		}
	}
}

func printWarnings(w io.Writer) {
	fmt.Fprintln(w, "WARNING: Please verify the migrated output carefully before using it.")
	fmt.Fprintln(w, "- Fields set to Go zero values (false, 0, \"\") may be silently dropped")
	fmt.Fprintln(w, "  due to omitempty tags. Compare against your original config to ensure nothing is lost.")
	fmt.Fprintln(w, "- Secret values (e.g. remote_write_headers) are masked as '<secret>' in the output.")
	fmt.Fprintln(w, "  You must manually restore the original values.")
	fmt.Fprintln(w, "- Some struct fields without omitempty may appear with zero values (e.g. 'exclude: null')")
	fmt.Fprintln(w, "  that were not in your original config. These can be removed.")
	fmt.Fprintln(w, "NOTE: This tool is provided for convenience only. Always double check the output against your original config.")
	fmt.Fprintln(w, "For full details on overrides configuration, see: https://grafana.com/docs/tempo/latest/configuration/#overrides")
}
