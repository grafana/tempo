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

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/grafana/tempo/tools/chloggen/internal/chlog"
	"github.com/grafana/tempo/tools/chloggen/internal/config"
)

func getSampleEntries() []*chlog.Entry {
	return []*chlog.Entry{
		enhancementEntry(),
		bugFixEntry(),
		deprecationEntry(),
		newComponentEntry(),
		breakingEntry(),
		entryWithSubtext(),
	}
}

func enhancementEntry() *chlog.Entry {
	return &chlog.Entry{
		ChangeType: chlog.Enhancement,
		Component:  "receiver/foo",
		Note:       "Add some bar",
		Issues:     []int{12345},
	}
}

func bugFixEntry() *chlog.Entry {
	return &chlog.Entry{
		ChangeType: chlog.BugFix,
		Component:  "testbed",
		Note:       "Fix blah",
		Issues:     []int{12346, 12347},
	}
}

func deprecationEntry() *chlog.Entry {
	return &chlog.Entry{
		ChangeType: chlog.Deprecation,
		Component:  "exporter/old",
		Note:       "Deprecate old",
		Issues:     []int{12348},
	}
}

func newComponentEntry() *chlog.Entry {
	return &chlog.Entry{
		ChangeType: chlog.NewComponent,
		Component:  "exporter/new",
		Note:       "Add new exporter ...",
		Issues:     []int{12349},
	}
}

func breakingEntry() *chlog.Entry {
	return &chlog.Entry{
		ChangeType: chlog.Breaking,
		Component:  "processor/oops",
		Note:       "Change behavior when ...",
		Issues:     []int{12350},
	}
}

func entryWithSubtext() *chlog.Entry {
	lines := []string{"- foo\n  - bar\n- blah\n  - 1234567"}

	return &chlog.Entry{
		ChangeType: chlog.Breaking,
		Component:  "processor/oops",
		Note:       "Change behavior when ...",
		Issues:     []int{12350},
		SubText:    strings.Join(lines, "\n"),
	}
}

func entryForChangelogs(changeType string, issue int, keys ...string) *chlog.Entry {
	keyStr := "default"
	if len(keys) > 0 {
		keyStr = strings.Join(keys, ",")
	}
	return &chlog.Entry{
		ChangeLogs: keys,
		ChangeType: changeType,
		Component:  "receiver/foo",
		Note:       fmt.Sprintf("Some change relevant to [%s]", keyStr),
		Issues:     []int{issue},
	}
}

func setupTestDir(t *testing.T, entries []*chlog.Entry) {
	require.NotNil(t, globalCfg, "test should instantiate globalCfg before calling setupTestDir")

	// Create dummy changelogs which may be updated by the test
	changelogBytes, err := readBytes(filepath.Join("testdata", config.DefaultChangeLogFilename))
	require.NoError(t, err)
	for _, filename := range globalCfg.ChangeLogs {
		require.NoError(t, os.MkdirAll(filepath.Dir(filename), os.FileMode(0o755)))
		require.NoError(t, os.WriteFile(filename, changelogBytes, os.FileMode(0o755)))
	}

	// Create the entries directory
	require.NoError(t, os.MkdirAll(globalCfg.EntriesDir, os.FileMode(0o755)))

	// Copy the entry template, for tests that ensure it is not deleted
	templateInRootDir := config.New("testdata").TemplateYAML
	templateBytes, err := readBytes(filepath.Clean(templateInRootDir))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(globalCfg.TemplateYAML, templateBytes, os.FileMode(0o755)))

	// Write the entries to the entries directory
	for i, entry := range entries {
		entryBytes, err := yaml.Marshal(entry)
		require.NoError(t, err)
		path := filepath.Join(globalCfg.EntriesDir, fmt.Sprintf("%d.yaml", i))
		require.NoError(t, os.WriteFile(path, entryBytes, os.FileMode(0o755)))
	}
}

func runCobra(t *testing.T, args ...string) (string, error) {
	cmd := rootCmd()

	outBytes := bytes.NewBufferString("")
	cmd.SetOut(outBytes)

	cmd.SetArgs(args)
	err := cmd.Execute()

	out, ioErr := io.ReadAll(outBytes)
	require.NoError(t, ioErr, "read stdout")

	return string(out), err
}

func readBytes(path string) ([]byte, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}
