package main

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectMode(t *testing.T) {
	tests := []struct {
		name         string
		m            map[string]interface{}
		flagOverride string
		expected     string
	}{
		{
			name:     "target all is monolithic",
			m:        map[string]interface{}{"target": "all"},
			expected: modeMonolithic,
		},
		{
			name:     "no target field is monolithic",
			m:        map[string]interface{}{},
			expected: modeMonolithic,
		},
		{
			name:     "empty target is monolithic",
			m:        map[string]interface{}{"target": ""},
			expected: modeMonolithic,
		},
		// Empty-target key removal is asserted separately below.
		{
			name:     "distributor target is microservices",
			m:        map[string]interface{}{"target": "distributor"},
			expected: modeMicroservices,
		},
		{
			name:     "query-frontend target is microservices",
			m:        map[string]interface{}{"target": "query-frontend"},
			expected: modeMicroservices,
		},
		{
			name:         "flag override takes precedence",
			m:            map[string]interface{}{"target": "all"},
			flagOverride: modeMicroservices,
			expected:     modeMicroservices,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectMode(tt.m, tt.flagOverride, new([]string))
			assert.Equal(t, tt.expected, result)
		})
	}

	t.Run("scalable-single-binary is rewritten to all with warning", func(t *testing.T) {
		m := map[string]interface{}{"target": "scalable-single-binary"}
		var warnings []string
		result := detectMode(m, "", &warnings)
		assert.Equal(t, modeMonolithic, result)
		assert.Equal(t, "all", m["target"])
		require.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "scalable-single-binary")
		assert.Contains(t, warnings[0], "removed in Tempo 3.0")
	})

	t.Run("empty target string is deleted from map", func(t *testing.T) {
		m := map[string]interface{}{"target": ""}
		result := detectMode(m, "", new([]string))
		assert.Equal(t, modeMonolithic, result)
		// An explicit empty target overrides Tempo's default, so it must be removed.
		assert.NotContains(t, m, "target")
	})
}

func TestDetectLegacyOverrides(t *testing.T) {
	tests := []struct {
		name      string
		m         map[string]interface{}
		expectErr bool
	}{
		{
			name: "new format with defaults key",
			m: map[string]interface{}{
				"overrides": map[string]interface{}{
					"defaults": map[string]interface{}{
						"ingestion": map[string]interface{}{
							"rate_limit_bytes": 5000000,
						},
					},
				},
			},
			expectErr: false,
		},
		{
			name: "legacy format detected",
			m: map[string]interface{}{
				"overrides": map[string]interface{}{
					"ingestion_rate_strategy":    "global",
					"ingestion_rate_limit_bytes": 5000000,
					"max_traces_per_user":        50000,
				},
			},
			expectErr: true,
		},
		{
			name:      "no overrides section",
			m:         map[string]interface{}{},
			expectErr: false,
		},
		{
			name: "unknown keys without legacy keys is fine",
			m: map[string]interface{}{
				"overrides": map[string]interface{}{
					"defaults":         map[string]interface{}{},
					"some_unknown_key": "value",
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := detectLegacyOverrides(tt.m)
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "legacy overrides format detected")
				assert.Contains(t, err.Error(), "tempo-cli migrate overrides-config")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDeleteRemovedBlocks(t *testing.T) {
	m := map[string]interface{}{
		"server":                   map[string]interface{}{"http_listen_port": 3200},
		"ingester":                 map[string]interface{}{"max_block_duration": "5m"},
		"ingester_client":          map[string]interface{}{"grpc_compression": "snappy"},
		"compactor":                map[string]interface{}{"compaction": map[string]interface{}{}},
		"metrics_generator_client": map[string]interface{}{"grpc_compression": "snappy"},
		"storage":                  map[string]interface{}{},
	}

	var warnings []string
	deleteRemovedBlocks(m, &warnings)

	assert.NotContains(t, m, "ingester")
	assert.NotContains(t, m, "ingester_client")
	assert.NotContains(t, m, "compactor")
	// metrics_generator_client is deprecated but still in 3.0 Config, so it's preserved
	assert.Contains(t, m, "metrics_generator_client")
	assert.Contains(t, m, "server")
	assert.Contains(t, m, "storage")
	assert.Len(t, warnings, 3)
}

func TestAddIngestBlocks(t *testing.T) {
	t.Run("monolithic mode skips kafka", func(t *testing.T) {
		m := map[string]interface{}{}
		err := addIngestBlocks(m, modeMonolithic, "", "")
		require.NoError(t, err)
		assert.NotContains(t, m, "ingest")
	})

	t.Run("microservices mode requires kafka address", func(t *testing.T) {
		m := map[string]interface{}{}
		err := addIngestBlocks(m, modeMicroservices, "", "tempo")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--kafka-address is required")
	})

	t.Run("microservices mode adds kafka config", func(t *testing.T) {
		m := map[string]interface{}{}
		err := addIngestBlocks(m, modeMicroservices, "kafka:9092", "my-topic")
		require.NoError(t, err)

		ingest := m["ingest"].(map[string]interface{})
		kafka := ingest["kafka"].(map[string]interface{})
		assert.Equal(t, "kafka:9092", kafka["address"])
		assert.Equal(t, "my-topic", kafka["topic"])
	})

	t.Run("microservices mode merges into existing ingest", func(t *testing.T) {
		m := map[string]interface{}{
			"ingest": map[string]interface{}{
				"kafka": map[string]interface{}{
					"client_id": "my-client",
				},
			},
		}
		err := addIngestBlocks(m, modeMicroservices, "kafka:9092", "tempo")
		require.NoError(t, err)

		ingest := m["ingest"].(map[string]interface{})
		kafka := ingest["kafka"].(map[string]interface{})
		assert.Equal(t, "kafka:9092", kafka["address"])
		assert.Equal(t, "tempo", kafka["topic"])
		assert.Equal(t, "my-client", kafka["client_id"])
	})
}

func TestModifyOverrides(t *testing.T) {
	t.Run("creates overrides and sets compaction_disabled", func(t *testing.T) {
		m := map[string]interface{}{}
		var warnings []string
		modifyOverrides(m, &warnings)

		overrides := m["overrides"].(map[string]interface{})
		defaults := overrides["defaults"].(map[string]interface{})
		compaction := defaults["compaction"].(map[string]interface{})
		assert.Equal(t, true, compaction["compaction_disabled"])
	})

	t.Run("preserves existing overrides", func(t *testing.T) {
		m := map[string]interface{}{
			"overrides": map[string]interface{}{
				"defaults": map[string]interface{}{
					"ingestion": map[string]interface{}{
						"rate_limit_bytes": 5000000,
					},
				},
			},
		}
		var warnings []string
		modifyOverrides(m, &warnings)

		overrides := m["overrides"].(map[string]interface{})
		defaults := overrides["defaults"].(map[string]interface{})
		// compaction_disabled is set
		compaction := defaults["compaction"].(map[string]interface{})
		assert.Equal(t, true, compaction["compaction_disabled"])
		// existing ingestion config is preserved
		ingestion := defaults["ingestion"].(map[string]interface{})
		assert.Equal(t, 5000000, ingestion["rate_limit_bytes"])
	})

	t.Run("warns about per_tenant_override_config", func(t *testing.T) {
		m := map[string]interface{}{
			"overrides": map[string]interface{}{
				"defaults":                   map[string]interface{}{},
				"per_tenant_override_config": "/etc/tempo/overrides.yaml",
			},
		}
		var warnings []string
		modifyOverrides(m, &warnings)

		require.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "/etc/tempo/overrides.yaml")
		assert.Contains(t, warnings[0], "compaction_disabled")
	})
}

func TestCleanLocalBlocks(t *testing.T) {
	t.Run("removes top-level local_blocks processor config", func(t *testing.T) {
		m := map[string]interface{}{
			"metrics_generator": map[string]interface{}{
				"processor": map[string]interface{}{
					"local_blocks": map[string]interface{}{
						"flush_to_metrics": true,
					},
					"service_graphs": map[string]interface{}{
						"dimensions": []interface{}{"service.namespace"},
					},
				},
			},
		}
		var warnings []string
		cleanLocalBlocks(m, &warnings)

		mg := m["metrics_generator"].(map[string]interface{})
		proc := mg["processor"].(map[string]interface{})
		assert.NotContains(t, proc, "local_blocks")
		assert.Contains(t, proc, "service_graphs")
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "local_blocks")
	})

	t.Run("removes local_blocks from overrides defaults", func(t *testing.T) {
		m := map[string]interface{}{
			"overrides": map[string]interface{}{
				"defaults": map[string]interface{}{
					"metrics_generator": map[string]interface{}{
						"processor": map[string]interface{}{
							"local_blocks": map[string]interface{}{
								"flush_to_metrics": true,
							},
						},
						"processors": []interface{}{"service-graphs", "local-blocks"},
					},
				},
			},
		}
		var warnings []string
		cleanLocalBlocks(m, &warnings)

		defaults := m["overrides"].(map[string]interface{})["defaults"].(map[string]interface{})
		mg := defaults["metrics_generator"].(map[string]interface{})
		proc := mg["processor"].(map[string]interface{})
		assert.NotContains(t, proc, "local_blocks")
		processors := mg["processors"].([]interface{})
		assert.Equal(t, []interface{}{"service-graphs"}, processors)
		assert.Len(t, warnings, 2)
	})

	t.Run("removes local-blocks from processors list in overrides", func(t *testing.T) {
		m := map[string]interface{}{
			"overrides": map[string]interface{}{
				"defaults": map[string]interface{}{
					"metrics_generator": map[string]interface{}{
						"processors": []interface{}{"service-graphs", "span-metrics", "local-blocks"},
					},
				},
			},
		}
		var warnings []string
		cleanLocalBlocks(m, &warnings)

		defaults := m["overrides"].(map[string]interface{})["defaults"].(map[string]interface{})
		mg := defaults["metrics_generator"].(map[string]interface{})
		processors := mg["processors"].([]interface{})
		assert.Equal(t, []interface{}{"service-graphs", "span-metrics"}, processors)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "local-blocks")
	})

	t.Run("no metrics_generator is a no-op", func(t *testing.T) {
		m := map[string]interface{}{}
		var warnings []string
		cleanLocalBlocks(m, &warnings)
		assert.Empty(t, warnings)
	})
}

func TestSetNestedValue(t *testing.T) {
	t.Run("creates nested path", func(t *testing.T) {
		m := map[string]interface{}{}
		setNestedValue(m, []string{"a", "b", "c"}, "value")
		assert.Equal(t, "value", m["a"].(map[string]interface{})["b"].(map[string]interface{})["c"])
	})

	t.Run("does not overwrite existing intermediate maps", func(t *testing.T) {
		m := map[string]interface{}{
			"a": map[string]interface{}{
				"existing": "preserved",
			},
		}
		setNestedValue(m, []string{"a", "b"}, "new")
		a := m["a"].(map[string]interface{})
		assert.Equal(t, "preserved", a["existing"])
		assert.Equal(t, "new", a["b"])
	})
}

// runMigrateConfig runs the full migrate config pipeline end-to-end, capturing
// stdout. Warnings printed to stderr are not captured here.
func runMigrateConfig(t *testing.T, inputFile, kafkaAddress, kafkaTopic, mode string) (stdout string, err error) {
	t.Helper()
	cmd := &migrateConfigCmd{
		ConfigFile:   inputFile,
		KafkaAddress: kafkaAddress,
		KafkaTopic:   kafkaTopic,
		Mode:         mode,
	}

	// Capture stdout.
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	err = cmd.Run(nil)
	_ = w.Close()
	<-done
	return buf.String(), err
}

func TestMigrateConfigEndToEnd(t *testing.T) {
	tests := []struct {
		name             string
		inputFile        string
		expectedFile     string
		kafkaAddress     string
		kafkaTopic       string
		mode             string
		expectErr        bool
		expectErrContain string
	}{
		{
			name:         "monolithic basic",
			inputFile:    "test-data/migrate-config/monolithic-basic-input.yaml",
			expectedFile: "test-data/migrate-config/monolithic-basic-expected.yaml",
		},
		{
			name:         "microservices basic",
			inputFile:    "test-data/migrate-config/microservices-basic-input.yaml",
			expectedFile: "test-data/migrate-config/microservices-basic-expected.yaml",
			kafkaAddress: "kafka:9092",
			kafkaTopic:   "tempo-traces",
		},
		{
			name:         "no target field defaults to monolithic",
			inputFile:    "test-data/migrate-config/no-target-input.yaml",
			expectedFile: "test-data/migrate-config/no-target-expected.yaml",
		},
		{
			name:         "with local-blocks processor",
			inputFile:    "test-data/migrate-config/with-local-blocks-input.yaml",
			expectedFile: "test-data/migrate-config/with-local-blocks-expected.yaml",
		},
		{
			name:         "with per-tenant override config",
			inputFile:    "test-data/migrate-config/with-per-tenant-override-config-input.yaml",
			expectedFile: "test-data/migrate-config/with-per-tenant-override-config-expected.yaml",
		},
		{
			name:         "with env vars preserves them",
			inputFile:    "test-data/migrate-config/with-env-vars-input.yaml",
			expectedFile: "test-data/migrate-config/with-env-vars-expected.yaml",
		},
		{
			name:         "scalable-single-binary is rewritten to all",
			inputFile:    "test-data/migrate-config/scalable-single-binary-input.yaml",
			expectedFile: "test-data/migrate-config/scalable-single-binary-expected.yaml",
		},
		{
			name:             "legacy overrides errors",
			inputFile:        "test-data/migrate-config/legacy-overrides-input.yaml",
			expectErr:        true,
			expectErrContain: "legacy overrides format detected",
		},
		{
			name:             "microservices without kafka-address errors",
			inputFile:        "test-data/migrate-config/microservices-basic-input.yaml",
			expectErr:        true,
			expectErrContain: "--kafka-address is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, err := runMigrateConfig(t, tt.inputFile, tt.kafkaAddress, tt.kafkaTopic, tt.mode)
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectErrContain)
				return
			}
			require.NoError(t, err)

			expected, err := os.ReadFile(tt.expectedFile)
			require.NoError(t, err)
			assert.Equal(t, string(expected), stdout,
				"output does not match %s — if the change is intentional, regenerate the expected file", tt.expectedFile)
		})
	}
}

func TestValidateMigratedConfig(t *testing.T) {
	t.Run("unknown nested key fails validation", func(t *testing.T) {
		m := map[string]interface{}{
			"target": "all",
			"server": map[string]interface{}{
				"bogus_unknown_field": 42,
			},
		}
		_, err := validateMigratedConfig(m)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed validation")
		assert.Contains(t, err.Error(), "bogus_unknown_field")
	})

	t.Run("unsupported target fails validation", func(t *testing.T) {
		m := map[string]interface{}{
			"target": "bogus-target",
		}
		_, err := validateMigratedConfig(m)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported target")
		assert.Contains(t, err.Error(), "bogus-target")
	})

	t.Run("env var placeholder downgrades type error to warning", func(t *testing.T) {
		m := map[string]interface{}{
			"target": "all",
			"server": map[string]interface{}{
				"http_listen_port": "${HTTP_PORT}",
			},
		}
		warnings, err := validateMigratedConfig(m)
		require.NoError(t, err)
		require.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "validation skipped for env var placeholders")
	})

	t.Run("valid config passes without warnings", func(t *testing.T) {
		m := map[string]interface{}{
			"target": "all",
			"server": map[string]interface{}{
				"http_listen_port": 3200,
			},
		}
		warnings, err := validateMigratedConfig(m)
		require.NoError(t, err)
		assert.Empty(t, warnings)
	})
}
