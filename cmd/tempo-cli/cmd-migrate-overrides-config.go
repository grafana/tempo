package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/cmd/tempo/app"
	"github.com/grafana/tempo/modules/overrides"
)

type migrateOverridesConfigCmd struct {
	ConfigFile string `arg:"" help:"Path to tempo config file"`

	ConfigDest    string `type:"path" short:"d" help:"Path to tempo config file. If not specified, output to stdout"`
	OverridesDest string `type:"path" short:"o" help:"Path to tempo overrides file. If not specified, output to stdout"`
}

func (cmd *migrateOverridesConfigCmd) Run(*globalOptions) error {
	// Defaults
	cfg := app.Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})

	// Existing config
	buff, err := os.ReadFile(cmd.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read configFile %s: %w", cmd.ConfigFile, err)
	}

	if err := yaml.UnmarshalStrict(buff, &cfg); err != nil {
		return fmt.Errorf("failed to parse configFile %s: %w", cmd.ConfigFile, err)
	}

	o, err := overrides.NewOverrides(cfg.Overrides, noopValidator{}, prometheus.DefaultRegisterer)
	if err != nil {
		return fmt.Errorf("failed to load overrides module: %w", err)
	}

	if err := services.StartAndAwaitRunning(context.Background(), o); err != nil {
		return fmt.Errorf("failed to start overrides module: %w", err)
	}

	buffer := bytes.NewBuffer(make([]byte, 0))
	if err := o.WriteStatusRuntimeConfig(buffer, &http.Request{URL: &url.URL{}}); err != nil {
		return fmt.Errorf("failed to output runtime config: %w", err)
	}

	var runtimeConfig struct {
		Defaults           overrides.Overrides         `yaml:"defaults"`
		PerTenantOverrides map[string]overrides.Config `yaml:"overrides"`
	}
	if err := yaml.UnmarshalStrict(buffer.Bytes(), &runtimeConfig); err != nil {
		return fmt.Errorf("failed parsing overrides config: %w", err)
	}

	cfg.Overrides.Defaults = runtimeConfig.Defaults
	configBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if cmd.ConfigDest != "" {
		if err := os.WriteFile(cmd.ConfigDest, configBytes, 0o644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}
	} else {
		fmt.Println(cmd.ConfigFile)
		// Only print the overrides block
		partialCfg := struct {
			Overrides overrides.Config `yaml:"overrides"`
		}{Overrides: cfg.Overrides}
		overridesBytes, err := yaml.Marshal(partialCfg)
		if err != nil {
			return fmt.Errorf("failed to marshal overrides: %w", err)
		}
		fmt.Println(string(overridesBytes))
	}

	overridesBytes, err := yaml.Marshal(runtimeConfig.PerTenantOverrides)
	if err != nil {
		return fmt.Errorf("failed to marshal overrides: %w", err)
	}

	if cmd.OverridesDest != "" {
		if err := os.WriteFile(cmd.OverridesDest, overridesBytes, 0o644); err != nil {
			return fmt.Errorf("failed to write overrides file: %w", err)
		}
	} else {
		fmt.Println(cfg.Overrides.PerTenantOverrideConfig)
		fmt.Println(string(overridesBytes))
	}

	return nil
}

type noopValidator struct{}

func (n noopValidator) Validate(*overrides.Overrides) error {
	return nil
}
