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
	"testing"

	"github.com/stretchr/testify/assert"
)

const rootUsage = `chloggen is a tool used to automate the generation of CHANGELOG files using individual yaml files as the source.

Usage:
  chloggen [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  new         Creates new change file
  update      Updates CHANGELOG.MD to include all new changes
  validate    Validates the files in the changelog directory

Flags:
      --config string   (optional) chloggen config file
  -h, --help            help for chloggen

Use "chloggen [command] --help" for more information about a command.`

func TestRoot(t *testing.T) {
	var out string
	var err error

	out, err = runCobra(t)
	assert.Contains(t, out, rootUsage)
	assert.Empty(t, err)

	out, err = runCobra(t, "--help")
	assert.Contains(t, out, rootUsage)
	assert.Empty(t, err)

	out, err = runCobra(t, "--config", "foo.yaml")
	assert.Contains(t, out, rootUsage)
	assert.Empty(t, err)
}

func TestRepoRoot(t *testing.T) {
	assert.DirExists(t, repoRoot())
}
