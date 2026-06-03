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
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/grafana/tempo/tools/chloggen/internal/chlog"
)

const (
	insertPoint = "<!-- next version -->\n"
)

var (
	version         string
	dry             bool
	componentFilter string
)

func updateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Updates CHANGELOG.MD to include all new changes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			entriesByChangelog, err := chlog.ReadEntries(globalCfg)
			if err != nil {
				return err
			}

			// Validate before rendering/deleting so an invalid entry is never
			// silently dropped from the summary and then lost when its source
			// file is deleted.
			if err = chlog.ValidateEntries(globalCfg, entriesByChangelog); err != nil {
				return err
			}

			// Backfill empty 'issues' from git here, not in validate, so the PR-CI
			// validation of an unmerged entry doesn't depend on git history. Dedupe
			// by pointer: one entry can appear under several changelogs, and a
			// failed backfill would otherwise re-spawn git for it each time.
			backfilled := make(map[*chlog.Entry]bool)
			for _, entries := range entriesByChangelog {
				for _, e := range entries {
					if backfilled[e] {
						continue
					}
					backfilled[e] = true
					e.BackfillIssues()
				}
			}

			// Never render an entry without a PR link: fail, listing all of them.
			if missing := chlog.MissingIssues(entriesByChangelog); len(missing) > 0 {
				noun := "entry"
				if len(missing) > 1 {
					noun = "entries"
				}
				return fmt.Errorf("could not determine a PR number for %d changelog %s; set 'issues' explicitly in:\n  %s",
					len(missing), noun, strings.Join(missing, "\n  "))
			}

			for changeLogKey, entries := range entriesByChangelog {

				slices.SortFunc(entries, func(a, b *chlog.Entry) int {
					return strings.Compare(a.Component, b.Component)
				})

				if componentFilter != "" {
					filteredEntries := make([]*chlog.Entry, 0, len(entries))
					for _, e := range entries {
						if e.Component == componentFilter {
							filteredEntries = append(filteredEntries, e)
						}
					}
					entries = filteredEntries
				}
				chlogUpdate, err := chlog.GenerateSummary(version, entries, globalCfg)
				if err != nil {
					return err
				}

				if dry {
					cmd.Printf("Generated changelog updates for %s:\n", changeLogKey)
					cmd.Println(chlogUpdate)
					continue
				}

				filename := globalCfg.ChangeLogs[changeLogKey]
				oldChlogBytes, err := os.ReadFile(filepath.Clean(filename))
				if err != nil {
					return err
				}
				chlogParts := bytes.Split(oldChlogBytes, []byte(insertPoint))
				if len(chlogParts) != 2 {
					return fmt.Errorf("expected one instance of %s", insertPoint)
				}

				chlogHeader, chlogHistory := string(chlogParts[0]), string(chlogParts[1])

				var chlogBuilder strings.Builder
				chlogBuilder.WriteString(chlogHeader)
				chlogBuilder.WriteString(insertPoint)
				chlogBuilder.WriteString(chlogUpdate)
				chlogBuilder.WriteString(chlogHistory)

				tmpMD := filename + ".tmp"
				if err = os.WriteFile(filepath.Clean(tmpMD), []byte(chlogBuilder.String()), 0o644); err != nil {
					return err
				}

				if err = os.Rename(tmpMD, filename); err != nil {
					return err
				}

				cmd.Printf("Finished updating %s\n", filename)
			}

			// Delete the consumed entry files only once, after every changelog has
			// been written successfully, and never in dry-run mode. This avoids
			// losing the source YAMLs if a later changelog write fails.
			if !dry {
				if err = chlog.DeleteEntries(globalCfg); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&version, "version", "v", "vTODO", "will be rendered directly into the update text")
	cmd.Flags().BoolVarP(&dry, "dry", "d", false, "will generate the update text and print to stdout")
	cmd.Flags().StringVarP(&componentFilter, "component", "c", "", "only select entries with this exact component")
	return cmd
}
