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

package chlog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tools/chloggen/internal/config"
)

func TestSummary(t *testing.T) {
	brk1 := Entry{
		ChangeType: Breaking,
		Component:  "foo",
		Note:       "broke foo",
		Issues:     []int{123},
	}
	brk2 := Entry{
		ChangeType: Breaking,
		Component:  "bar",
		Note:       "broke bar",
		Issues:     []int{345, 678},
		SubText:    "more details",
	}
	dep1 := Entry{
		ChangeType: Deprecation,
		Component:  "foo",
		Note:       "deprecate foo",
		Issues:     []int{1234},
	}
	dep2 := Entry{
		ChangeType: Deprecation,
		Component:  "bar",
		Note:       "deprecate bar",
		Issues:     []int{3456, 6789},
		SubText:    "more details",
	}
	enh1 := Entry{
		ChangeType: Enhancement,
		Component:  "foo",
		Note:       "enhance foo",
		Issues:     []int{12},
	}
	enh2 := Entry{
		ChangeType: Enhancement,
		Component:  "bar",
		Note:       "enhance bar",
		Issues:     []int{34, 67},
		SubText:    "more details",
	}
	bug1 := Entry{
		ChangeType: BugFix,
		Component:  "foo",
		Note:       "bug foo",
		Issues:     []int{1},
	}
	bug2 := Entry{
		ChangeType: BugFix,
		Component:  "bar",
		Note:       "bug bar",
		Issues:     []int{3, 6},
		SubText:    "more details",
	}
	new1 := Entry{
		ChangeType: NewComponent,
		Component:  "foo",
		Note:       "new foo",
		Issues:     []int{2},
	}
	new2 := Entry{
		ChangeType: NewComponent,
		Component:  "bar",
		Note:       "new bar",
		Issues:     []int{4, 7},
		SubText:    "more details",
	}

	actual, err := GenerateSummary("1.0", []*Entry{&brk1, &brk2, &dep1, &dep2, &enh1, &enh2, &bug1, &bug2, &new1, &new2}, &config.Config{})
	assert.NoError(t, err)

	// This file is not meant to be the entire changelog so will not pass markdownlint if named with .md extension.
	expected, err := os.ReadFile(filepath.Join("testdata", "CHANGELOG"))
	require.NoError(t, err)

	assert.Equal(t, string(expected), actual)
}

func TestCustomSummary(t *testing.T) {
	brk1 := Entry{
		ChangeType: Breaking,
		Component:  "foo",
		Note:       "broke foo",
		Issues:     []int{123},
	}
	brk2 := Entry{
		ChangeType: Breaking,
		Component:  "bar",
		Note:       "broke bar",
		Issues:     []int{345, 678},
		SubText:    "more details",
	}

	actual, err := GenerateSummary("1.0", []*Entry{&brk1, &brk2}, &config.Config{SummaryTemplate: filepath.Join("testdata", "custom.tmpl")})
	require.NoError(t, err)

	// This file is not meant to be the entire changelog so will not pass markdownlint if named with .md extension.
	expected, err := os.ReadFile(filepath.Join("testdata", "CHANGELOG_custom"))
	require.NoError(t, err)

	assert.Equal(t, string(expected), actual)
}
