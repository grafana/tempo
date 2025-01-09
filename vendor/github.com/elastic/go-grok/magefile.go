// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build mage
// +build mage

package main

import (
	"fmt"
	"path/filepath"

	"github.com/magefile/mage/mg"

	// mage:import
	"github.com/elastic/go-grok/dev-tools/mage"

	devtools "github.com/elastic/go-grok/dev-tools/mage"
	"github.com/elastic/go-grok/dev-tools/mage/gotool"
)

// Aliases are shortcuts to long target names.
// nolint: deadcode // it's used by `mage`.
var Aliases = map[string]interface{}{
	"llc":  mage.Linter.LastChange,
	"lint": mage.Linter.All,
}

// Check runs all the checks
// nolint: deadcode,unparam // it's used as a `mage` target and requires returning an error
func Check() error {
	mg.Deps(devtools.InstallGoLicenser)
	mg.Deps(devtools.Deps.CheckModuleTidy, CheckLicenseHeaders)
	mg.Deps(devtools.CheckNoChanges)
	return nil
}

// Fmt formats code and adds license headers.
func Fmt() {
	mg.Deps(devtools.GoImports.Run)
	mg.Deps(AddLicenseHeaders)
}

// AddLicenseHeaders adds ASL2 headers to .go files
func AddLicenseHeaders() error {
	fmt.Println(">> fmt - go-licenser: Adding missing headers")

	mg.Deps(devtools.InstallGoLicenser)

	licenser := gotool.Licenser

	return licenser(
		licenser.License("ASL2"),
	)
}

// CheckLicenseHeaders checks ASL2 headers in .go files
func CheckLicenseHeaders() error {
	mg.Deps(devtools.InstallGoLicenser)

	licenser := gotool.Licenser

	return licenser(
		licenser.Check(),
		licenser.License("ASL2"),
	)
}

// Notice generates a NOTICE.txt file for the module.
func Notice() error {
	return devtools.GenerateNotice(
		filepath.Join("dev-tools", "templates", "notice", "overrides.json"),
		filepath.Join("dev-tools", "templates", "notice", "rules.json"),
		filepath.Join("dev-tools", "templates", "notice", "NOTICE.txt.tmpl"),
	)
}
