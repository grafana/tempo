package main

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v2"

	"github.com/grafana/tempo/modules/overrides"
)

type migrateOverridesPerTenantCmd struct {
	OverridesFile string `arg:"" help:"Path to the per-tenant overrides file"`

	OutputDest string `type:"path" short:"d" help:"Path to write the migrated per-tenant overrides. If not specified, output to stdout"`
}

func (cmd *migrateOverridesPerTenantCmd) Run(*globalOptions) error {
	buff, err := os.ReadFile(cmd.OverridesFile)
	if err != nil {
		return fmt.Errorf("failed to read overrides file %s: %w", cmd.OverridesFile, err)
	}

	// Legacy per-tenant overrides are automatically converted to the new format
	// during unmarshaling via perTenantOverrides.UnmarshalYAML.
	tenantOverrides, err := overrides.UnmarshalPerTenantOverrides(buff)
	if err != nil {
		return fmt.Errorf("failed to parse overrides file %s: %w", cmd.OverridesFile, err)
	}

	// Diff each tenant's overrides against a zero-value Overrides to strip defaults.
	defaultMap, err := toMap(overrides.Overrides{})
	if err != nil {
		return fmt.Errorf("failed to convert default overrides to map: %w", err)
	}

	result := make(map[string]interface{}, len(tenantOverrides))
	for tenant, o := range tenantOverrides {
		if o == nil {
			continue
		}
		tenantMap, err := toMap(o)
		if err != nil {
			return fmt.Errorf("failed to convert overrides for tenant %s to map: %w", tenant, err)
		}
		removeDefaults(tenantMap, defaultMap)
		if len(tenantMap) > 0 {
			result[tenant] = tenantMap
		}
	}

	output := map[string]interface{}{
		"overrides": result,
	}

	outputBytes, err := yaml.Marshal(output)
	if err != nil {
		return fmt.Errorf("failed to marshal per-tenant overrides: %w", err)
	}

	printWarnings(os.Stderr)

	if cmd.OutputDest != "" {
		if err := os.WriteFile(cmd.OutputDest, outputBytes, 0o600); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Migrated per-tenant overrides written to %s\n", cmd.OutputDest)
	} else {
		fmt.Fprintln(os.Stderr, "Migrated per-tenant overrides. Replace your per-tenant overrides file with the output below:")
		fmt.Fprintln(os.Stderr, "---")
		fmt.Print(string(outputBytes))
	}

	return nil
}
