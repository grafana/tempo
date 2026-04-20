package main

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"

	"go.yaml.in/yaml/v2"

	"github.com/grafana/tempo/cmd/tempo/app"
	"github.com/grafana/tempo/modules/overrides"
)

const (
	modeMonolithic    = "monolithic"
	modeMicroservices = "microservices"
)

// yamlFieldNames returns the set of YAML tag names for the given struct type.
func yamlFieldNames(v interface{}) map[string]struct{} {
	t := reflect.TypeOf(v)
	fields := make(map[string]struct{}, t.NumField())
	for i := range t.NumField() {
		tag := t.Field(i).Tag.Get("yaml")
		if tag == "" || tag == "-" || tag == ",inline" {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		if name != "" {
			fields[name] = struct{}{}
		}
	}
	return fields
}

type migrateConfigCmd struct {
	ConfigFile   string `arg:"" help:"Path to the 2.x config file"`
	KafkaAddress string `name:"kafka-address" help:"Kafka broker address (required for microservices mode)" default:""`
	KafkaTopic   string `name:"kafka-topic" help:"Kafka topic" default:"tempo"`
	Mode         string `name:"mode" help:"Override deployment mode detection" enum:",monolithic,microservices" default:""`
}

func (cmd *migrateConfigCmd) Run(_ *globalOptions) error {
	var warnings []string

	// 1. Read YAML into generic map
	m, err := readConfigMap(cmd.ConfigFile)
	if err != nil {
		return err
	}

	// 2. Detect deployment mode
	mode := detectMode(m, cmd.Mode, &warnings)

	// 3. Check for legacy overrides format
	if err := detectLegacyOverrides(m); err != nil {
		return err
	}

	// 4. Delete removed 2.x blocks
	deleteRemovedBlocks(m, &warnings)

	// 5. Add ingest/kafka blocks (microservices only)
	if err := addIngestBlocks(m, mode, cmd.KafkaAddress, cmd.KafkaTopic); err != nil {
		return err
	}

	// 6. Modify overrides for parallel operation
	modifyOverrides(m, &warnings)

	// 7. Clean metrics-generator local_blocks config
	cleanLocalBlocks(m, &warnings)

	// 8. Validate through Tempo's 3.0 config
	validationWarnings, err := validateMigratedConfig(m)
	if err != nil {
		return err
	}
	warnings = append(warnings, validationWarnings...)

	// 9. Output
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "WARN: %s\n", w)
	}
	return outputMigratedConfig(m)
}

// readConfigMap reads a YAML file into a normalized map[string]interface{}.
func readConfigMap(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}
	if m == nil {
		m = make(map[string]interface{})
	}

	return normalizeMap(m), nil
}

// detectMode determines whether the config is for monolithic or microservices deployment.
func detectMode(m map[string]interface{}, flagOverride string, warnings *[]string) string {
	if flagOverride != "" {
		return flagOverride
	}
	target, ok := m["target"]
	if !ok {
		return modeMonolithic
	}
	targetStr, _ := target.(string)
	if app.IsSingleBinary(targetStr) || targetStr == "" {
		return modeMonolithic
	}
	if targetStr == "scalable-single-binary" {
		*warnings = append(*warnings, "warning: target %q is deprecated in Tempo 3.0; rewriting to %q and treating config as monolithic\n", targetStr, "all"))
		m["target"] = "all"
		return modeMonolithic
	}

	return modeMicroservices
}

// detectLegacyOverrides checks if the config uses the legacy flat overrides format
// and returns an error directing the user to migrate overrides first.
func detectLegacyOverrides(m map[string]interface{}) error {
	overridesRaw, ok := m["overrides"]
	if !ok {
		return nil
	}
	ovrMap, ok := overridesRaw.(map[string]interface{})
	if !ok {
		return nil
	}

	// If "defaults" key exists, this is the new format (or already migrated)
	if _, hasDefaults := ovrMap["defaults"]; hasDefaults {
		return nil
	}

	// Check for known legacy flat keys
	legacyFields := yamlFieldNames(overrides.LegacyOverrides{})
	for key := range ovrMap {
		if _, isLegacy := legacyFields[key]; isLegacy {
			return fmt.Errorf("legacy overrides format detected (found key %q); run 'tempo-cli migrate overrides-config' first", key)
		}
	}

	return nil
}

// deleteRemovedBlocks removes top-level config keys that are not recognized by the
// Tempo 3.0 Config struct. This catches 2.x-only sections like ingester, ingester_client,
// and compactor without hardcoding the list.
func deleteRemovedBlocks(m map[string]interface{}, warnings *[]string) {
	knownFields := yamlFieldNames(app.Config{})
	for key := range m {
		if _, known := knownFields[key]; !known {
			delete(m, key)
			*warnings = append(*warnings, fmt.Sprintf("removed %q section (not recognized by Tempo 3.0)", key))
		}
	}
}

// addIngestBlocks adds the ingest.kafka configuration for microservices mode.
func addIngestBlocks(m map[string]interface{}, mode, kafkaAddress, kafkaTopic string) error {
	if mode == modeMonolithic {
		return nil
	}

	if kafkaAddress == "" {
		return fmt.Errorf("--kafka-address is required in microservices mode")
	}

	setNestedValue(m, []string{"ingest", "kafka", "address"}, kafkaAddress)
	setNestedValue(m, []string{"ingest", "kafka", "topic"}, kafkaTopic)
	return nil
}

// modifyOverrides sets compaction_disabled: true in the overrides defaults and
// any inline per-tenant overrides, and warns about external per-tenant files.
func modifyOverrides(m map[string]interface{}, warnings *[]string) {
	// Ensure overrides map exists
	ovr := getOrCreateNestedMap(m, "overrides")

	// Set defaults.compaction.compaction_disabled: true
	defaults := getOrCreateNestedMap(ovr, "defaults")
	compaction := getOrCreateNestedMap(defaults, "compaction")
	compaction["compaction_disabled"] = true

	// Walk any non-standard keys (potential inline per-tenant overrides)
	knownKeys := yamlFieldNames(overrides.Config{})
	for key, val := range ovr {
		if _, known := knownKeys[key]; known {
			continue
		}
		tenantMap, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		tenantCompaction := getOrCreateNestedMap(tenantMap, "compaction")
		tenantCompaction["compaction_disabled"] = true
	}

	// Warn about external per-tenant override files
	if perTenantPath, ok := ovr["per_tenant_override_config"]; ok {
		if pathStr, ok := perTenantPath.(string); ok && pathStr != "" {
			*warnings = append(*warnings, fmt.Sprintf(
				"external per-tenant overrides file %q needs compaction_disabled: true added manually for each tenant", pathStr))
		}
	}
}

// cleanLocalBlocks removes the local_blocks processor config and the "local-blocks"
// entry from processors lists at the top-level metrics_generator config, overrides
// defaults, and any inline per-tenant overrides.
func cleanLocalBlocks(m map[string]interface{}, warnings *[]string) {
	// Clean top-level metrics_generator
	removeLocalBlocksProcessorConfig(m, "metrics_generator", warnings)

	overridesRaw, ok := m["overrides"]
	if !ok {
		return
	}
	ovrMap, ok := overridesRaw.(map[string]interface{})
	if !ok {
		return
	}

	// Clean defaults
	if defaultsRaw, ok := ovrMap["defaults"]; ok {
		if defaults, ok := defaultsRaw.(map[string]interface{}); ok {
			removeLocalBlocksProcessorConfig(defaults, "metrics_generator", warnings)
			removeLocalBlocksFromProcessorList(defaults, warnings)
		}
	}

	// Clean any inline per-tenant overrides
	knownKeys := yamlFieldNames(overrides.Config{})
	for key, val := range ovrMap {
		if _, known := knownKeys[key]; known {
			continue
		}
		if tenantMap, ok := val.(map[string]interface{}); ok {
			removeLocalBlocksProcessorConfig(tenantMap, "metrics_generator", warnings)
			removeLocalBlocksFromProcessorList(tenantMap, warnings)
		}
	}
}

// removeLocalBlocksProcessorConfig removes processor.local_blocks from a
// metrics_generator map found at the given key.
func removeLocalBlocksProcessorConfig(m map[string]interface{}, mgKey string, warnings *[]string) {
	mgRaw, ok := m[mgKey]
	if !ok {
		return
	}
	mg, ok := mgRaw.(map[string]interface{})
	if !ok {
		return
	}
	procRaw, ok := mg["processor"]
	if !ok {
		return
	}
	proc, ok := procRaw.(map[string]interface{})
	if !ok {
		return
	}
	if _, ok := proc["local_blocks"]; ok {
		delete(proc, "local_blocks")
		*warnings = append(*warnings, "removed metrics_generator.processor.local_blocks (block building is handled by the block-builder component in 3.0)")
	}
}

// removeLocalBlocksFromProcessorList filters "local-blocks" from the
// metrics_generator.processors list within an overrides map.
func removeLocalBlocksFromProcessorList(m map[string]interface{}, warnings *[]string) {
	mgRaw, ok := m["metrics_generator"]
	if !ok {
		return
	}
	mg, ok := mgRaw.(map[string]interface{})
	if !ok {
		return
	}
	processorsRaw, ok := mg["processors"]
	if !ok {
		return
	}
	procList, ok := processorsRaw.([]interface{})
	if !ok {
		return
	}

	filtered := make([]interface{}, 0, len(procList))
	for _, p := range procList {
		if s, ok := p.(string); ok && s == "local-blocks" {
			*warnings = append(*warnings, "removed 'local-blocks' from metrics_generator.processors — block building is handled by the block-builder component in 3.0")
			continue
		}
		filtered = append(filtered, p)
	}
	mg["processors"] = filtered
}

// containsEnvVarRef checks if a YAML byte slice contains environment variable
// references like ${VAR} that would cause type errors during unmarshalling.
func containsEnvVarRef(yamlBytes []byte) bool {
	return strings.Contains(string(yamlBytes), "${")
}

// validateMigratedConfig marshals the map back to YAML and attempts to unmarshal
// it into Tempo's Config struct for semantic validation.
// Returns warnings and an error if validation fails for reasons other than env var references.
func validateMigratedConfig(m map[string]interface{}) ([]string, error) {
	var warnings []string

	yamlBytes, err := yaml.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal migrated config for validation: %w", err)
	}

	cfg := app.Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})

	if err := yaml.UnmarshalStrict(yamlBytes, &cfg); err != nil {
		if containsEnvVarRef(yamlBytes) {
			// Env vars like ${VAR} cause type errors for non-string fields.
			// This is expected — report as warning, not fatal.
			warnings = append(warnings, fmt.Sprintf("validation skipped: config contains env var references (${...}): %v", err))
			return warnings, nil
		}
		return nil, fmt.Errorf("migrated config failed validation: %w", err)
	}

	for _, w := range cfg.CheckConfig() {
		warnings = append(warnings, fmt.Sprintf("validation: %s", w.Message))
	}

	return warnings, nil
}

// outputMigratedConfig marshals the map to YAML and prints it to stdout with a header comment.
func outputMigratedConfig(m map[string]interface{}) error {
	yamlBytes, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal migrated config: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("# Generated by tempo-cli migrate config\n")
	sb.WriteString("# Review before deploying. Remove compaction_disabled after decommissioning 2.x.\n")
	sb.Write(yamlBytes)

	fmt.Print(sb.String())
	return nil
}

// setNestedValue sets a value at a path of keys in a nested map structure,
// creating intermediate maps as needed without overwriting existing ones.
func setNestedValue(m map[string]interface{}, path []string, value interface{}) {
	current := m
	for _, key := range path[:len(path)-1] {
		next, ok := current[key]
		if !ok {
			next = make(map[string]interface{})
			current[key] = next
		}
		nextMap, ok := next.(map[string]interface{})
		if !ok {
			nextMap = make(map[string]interface{})
			current[key] = nextMap
		}
		current = nextMap
	}
	current[path[len(path)-1]] = value
}

// getOrCreateNestedMap gets or creates a map at the given key.
func getOrCreateNestedMap(m map[string]interface{}, key string) map[string]interface{} {
	if existing, ok := m[key]; ok {
		if existingMap, ok := existing.(map[string]interface{}); ok {
			return existingMap
		}
	}
	newMap := make(map[string]interface{})
	m[key] = newMap
	return newMap
}
