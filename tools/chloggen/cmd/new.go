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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var filename string

func newCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Creates new change file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			sanitizedName := filepath.Base(cleanFileName(filename))
			if sanitizedName == "." || sanitizedName == ".." || sanitizedName == "" {
				return fmt.Errorf("invalid filename %q", filename)
			}

			path := filepath.Join(globalCfg.EntriesDir, sanitizedName)
			var pathWithExt string
			switch ext := filepath.Ext(path); ext {
			case ".yaml":
				pathWithExt = path
			case ".yml":
				pathWithExt = strings.TrimSuffix(path, ".yml") + ".yaml"
			default:
				pathWithExt = path + ".yaml"
			}

			templateFile, err := os.Open(filepath.Clean(globalCfg.TemplateYAML))
			if err != nil {
				return err
			}
			defer templateFile.Close()

			templateBytes, err := io.ReadAll(templateFile)
			if err != nil {
				return err
			}
			err = os.WriteFile(filepath.Clean(pathWithExt), templateBytes, os.FileMode(0o644))
			if err != nil {
				return err
			}
			cmd.Printf("Changelog entry template copied to: %s\n", pathWithExt)
			return nil
		},
	}
	cmd.Flags().StringVarP(&filename, "filename", "f", "", "name of the file to add")
	if err := cmd.MarkFlagRequired("filename"); err != nil {
		cmd.PrintErrf("could not mark filename flag as required: %v", err)
		os.Exit(1)
	}
	return cmd
}

func cleanFileName(filename string) string {
	replace := strings.NewReplacer("/", "_", "\\", "_")
	return replace.Replace(filename)
}
