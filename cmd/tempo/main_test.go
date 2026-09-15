package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"
)

func TestLoadConfigPolicyTextSafety(t *testing.T) {
	const private = "synthetic-private-policy-marker"
	t.Setenv("TEMPO_TEST_PRIVATE_POLICY", private)
	for _, tc := range []struct {
		name   string
		policy string
		expand bool
		reject bool
	}{
		{"unknown-anchor", "custom_rules:\n- id: safe-rule\n  regex: *" + private, false, true},
		{"removed-description", "custom_rules:\n- id: safe-rule\n  regex: " + private + "\n  description: " + private, false, true},
		{"removed-entropy", "custom_rules:\n- id: safe-rule\n  regex: " + private + "\n  entropy: 1.5", false, true},
		{"removed-secret-group", "custom_rules:\n- id: safe-rule\n  regex: (" + private + ")\n  secret_group: 1", false, true},
		{"invalid-expansion", "custom_rules:\n- id: safe-rule\n  regex: ${TEMPO_TEST_PRIVATE_POLICY", true, true},
		{"valid-policy", "custom_rules:\n- id: safe-rule\n  regex: " + private, false, false},
		{"expanded-policy", "custom_rules:\n- id: safe-rule\n  regex: ${TEMPO_TEST_PRIVATE_POLICY}", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "tempo.yaml")
			data := "overrides:\n  defaults:\n    metrics_generator:\n      processor:\n        secret_detection:\n          " + strings.ReplaceAll(tc.policy, "\n", "\n          ") + "\n"
			require.NoError(t, os.WriteFile(configPath, []byte(data), 0o600))
			originalArgs, originalFlags := os.Args, flag.CommandLine
			t.Cleanup(func() {
				os.Args, flag.CommandLine = originalArgs, originalFlags
			})
			flag.CommandLine = flag.NewFlagSet("tempo-config-test", flag.ContinueOnError)
			flag.CommandLine.SetOutput(io.Discard)
			os.Args = []string{"tempo", "-config.file=" + configPath}
			if tc.expand {
				os.Args = append(os.Args, "-config.expand-env=true")
			}
			cfg, _, _, err := loadConfig()
			if tc.reject {
				require.True(t, err != nil, "invalid configuration must remain a startup error")
				require.True(t, cfg == nil, "invalid startup configuration must not be published")
				require.False(t, strings.Contains(err.Error(), private), "startup error disclosed private policy text")
				return
			}
			require.True(t, err == nil, "valid policy must load without redaction")
			policy := cfg.Overrides.Defaults.MetricsGenerator.Processor.SecretDetection
			require.True(t, policy != nil && len(policy.CustomRules) == 1, "configured policy must survive loading")
			require.True(t, policy.CustomRules[0].ID == "safe-rule" && policy.CustomRules[0].Regex == private, "policy text must survive decoding and expansion")
		})
	}
}

func TestLoadConfigOrdinaryDiagnostics(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "tempo.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("server:\n  http_listen_port: bad-port\n"), 0o600))
	originalArgs, originalFlags := os.Args, flag.CommandLine
	t.Cleanup(func() {
		os.Args, flag.CommandLine = originalArgs, originalFlags
	})
	flag.CommandLine = flag.NewFlagSet("tempo-config-test", flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
	os.Args = []string{"tempo", "-config.file=" + configPath}
	_, _, _, err := loadConfig()
	var typeError *yaml.TypeError
	require.ErrorAs(t, err, &typeError)
	require.Contains(t, err.Error(), "bad-port")
	require.Contains(t, err.Error(), configPath)
}
