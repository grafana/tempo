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
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/grafana/tempo/tools/chloggen/internal/chlog"
)

func validateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validates the files in the changelog directory",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := os.Stat(globalCfg.EntriesDir); err != nil {
				return err
			}

			entriesByChangelog, err := chlog.ReadEntries(globalCfg)
			if err != nil {
				return err
			}
			var errs error
			for _, entries := range entriesByChangelog {
				for _, entry := range entries {
					changelogRequired := len(globalCfg.DefaultChangeLogs) == 0
					validChangeLogs := []string{}
					for changeLogKey := range globalCfg.ChangeLogs {
						validChangeLogs = append(validChangeLogs, changeLogKey)
					}
					if err = entry.Validate(changelogRequired, globalCfg.Components, validChangeLogs...); err != nil {
						errs = errors.Join(errs, err)
					}
				}
			}
			if errs != nil {
				return errs
			}
			cmd.Printf("PASS: all files in %s/ are valid\n", globalCfg.EntriesDir)
			return nil
		},
	}
	return cmd
}
