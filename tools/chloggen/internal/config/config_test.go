// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNew(t *testing.T) {
	root := "/tmp"
	cfg := New(root)
	assert.Equal(t, filepath.Join(root, DefaultEntriesDir), cfg.EntriesDir)
	assert.Equal(t, filepath.Join(root, DefaultEntriesDir, DefaultTemplateYAML), cfg.TemplateYAML)

	assert.Equal(t, 1, len(cfg.ChangeLogs))
	assert.NotNil(t, cfg.ChangeLogs[DefaultChangeLogKey])
	assert.Equal(t, filepath.Join(root, DefaultChangeLogFilename), cfg.ChangeLogs[DefaultChangeLogKey])

	assert.Equal(t, 1, len(cfg.DefaultChangeLogs))
	assert.Equal(t, DefaultChangeLogKey, cfg.DefaultChangeLogs[0])
}

func TestNewFromFile(t *testing.T) {
	testCases := []struct {
		name      string
		cfg       *Config
		expectErr string
	}{
		{
			name: "empty",
			cfg:  &Config{},
		},
		{
			name: "multi-changelog-no-default",
			cfg: &Config{
				EntriesDir:   ".test",
				TemplateYAML: "TEMPLATE-custom.yaml",
				ChangeLogs: map[string]string{
					"foo": "CHANGELOG-1.md",
					"bar": "CHANGELOG-2.md",
				},
			},
		},
		{
			name: "multi-changelog-with-default",
			cfg: &Config{
				EntriesDir:   ".test",
				TemplateYAML: "TEMPLATE-custom.yaml",
				ChangeLogs: map[string]string{
					"foo": "CHANGELOG-1.md",
					"bar": "CHANGELOG-2.md",
				},
				DefaultChangeLogs: []string{"foo"},
			},
		},
		{
			name: "default-changelogs-without-changelogs",
			cfg: &Config{
				EntriesDir:        ".test",
				TemplateYAML:      "TEMPLATE-custom.yaml",
				DefaultChangeLogs: []string{"foo"},
			},
			expectErr: "cannot specify 'default_changelogs' without 'changelogs",
		},
		{
			name: "default-changelog-not-in-changelogs",
			cfg: &Config{
				EntriesDir:   ".test",
				TemplateYAML: "TEMPLATE-custom.yaml",
				ChangeLogs: map[string]string{
					"foo": "CHANGELOG-1.md",
					"bar": "CHANGELOG-2.md",
				},
				DefaultChangeLogs: []string{"foo", "bar", "fake"},
			},
			expectErr: `contains key "fake" which is not defined in 'changelogs'`,
		},
		{
			name: "absolute-entries-dir",
			cfg: &Config{
				EntriesDir: "/tmp/abs_entries",
			},
		},
		{
			name: "absolute-template-yaml",
			cfg: &Config{
				TemplateYAML: "/tmp/abs_template.yaml",
			},
		},
		{
			name: "absolute-changelog",
			cfg: &Config{
				ChangeLogs: map[string]string{
					"default": "/tmp/abs_changelog.md",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			cfgBytes, err := yaml.Marshal(tc.cfg)
			require.NoError(t, err)

			cfgFile, err := os.CreateTemp(tempDir, "*.yaml")
			require.NoError(t, err)
			defer cfgFile.Close()

			_, err = cfgFile.Write(cfgBytes)
			require.NoError(t, err)

			actualCfg, err := NewFromFile(tempDir, filepath.Base(cfgFile.Name()))
			if tc.expectErr != "" {
				require.ErrorContains(t, err, tc.expectErr)
				return
			}
			assert.NoError(t, err)

			// This would be the default config is no values were specified in the config file.
			// Instantiate it here to compare against the actual config as appropriate.
			defaultCfg := New(tempDir)

			expectedEntriesDir := defaultCfg.EntriesDir
			if tc.cfg.EntriesDir != "" {
				if filepath.IsAbs(tc.cfg.EntriesDir) {
					expectedEntriesDir = filepath.Clean(tc.cfg.EntriesDir)
				} else {
					expectedEntriesDir = filepath.Join(tempDir, tc.cfg.EntriesDir)
				}
			}
			assert.Equal(t, expectedEntriesDir, actualCfg.EntriesDir)

			expectedTemplateYAML := defaultCfg.TemplateYAML
			if tc.cfg.TemplateYAML != "" {
				if filepath.IsAbs(tc.cfg.TemplateYAML) {
					expectedTemplateYAML = filepath.Clean(tc.cfg.TemplateYAML)
				} else {
					expectedTemplateYAML = filepath.Join(tempDir, tc.cfg.TemplateYAML)
				}
			}
			assert.Equal(t, expectedTemplateYAML, actualCfg.TemplateYAML)

			if len(tc.cfg.ChangeLogs) == 0 {
				assert.Equal(t, 1, len(actualCfg.ChangeLogs))
				assert.NotNil(t, actualCfg.ChangeLogs[DefaultChangeLogKey])
				assert.Equal(t, filepath.Join(tempDir, DefaultChangeLogFilename), actualCfg.ChangeLogs[DefaultChangeLogKey])

				// When no changelogs are specified, the default changelog must be the only default changelog.
				assert.Equal(t, 1, len(actualCfg.DefaultChangeLogs))
				assert.Equal(t, DefaultChangeLogKey, actualCfg.DefaultChangeLogs[0])
			} else {
				assert.Equal(t, len(tc.cfg.ChangeLogs), len(actualCfg.ChangeLogs))
				for key, filename := range tc.cfg.ChangeLogs {
					assert.NotNil(t, actualCfg.ChangeLogs[key])
					expectedFilename := filename
					if !filepath.IsAbs(filename) {
						expectedFilename = filepath.Join(tempDir, filename)
					}
					assert.Equal(t, filepath.Clean(expectedFilename), actualCfg.ChangeLogs[key])
				}

				// When changelogs are specified, the default changelogs must be a subset of the changelogs.
				// It is acceptable to have no default changelog.
				assert.Equal(t, len(tc.cfg.DefaultChangeLogs), len(actualCfg.DefaultChangeLogs))
			}

			for _, key := range actualCfg.DefaultChangeLogs {
				assert.NotNil(t, actualCfg.ChangeLogs[key])
			}
		})
	}
}

func TestNewFromFileErr(t *testing.T) {
	tempDir := t.TempDir()

	_, err := NewFromFile(tempDir, "nonexistent.yaml")
	assert.Error(t, err)

	// Write a file with invalid YAML and then read it back to get expected error
	cfgFile, err := os.CreateTemp(tempDir, "*.yaml")
	require.NoError(t, err)
	defer cfgFile.Close()

	_, err = cfgFile.WriteString("!@#$%^&*())")
	require.NoError(t, err)

	_, err = NewFromFile(tempDir, filepath.Base(cfgFile.Name()))
	assert.Error(t, err)
}
