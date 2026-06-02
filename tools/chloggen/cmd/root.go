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

// Package cmd provides the command line interface for the chloggen tool.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/grafana/tempo/tools/chloggen/internal/config"
)

var (
	configFile string
	globalCfg  *config.Config
)

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chloggen",
		Short: "Updates CHANGELOG.MD to include all new changes",
		Long:  `chloggen is a tool used to automate the generation of CHANGELOG files using individual yaml files as the source.`,
	}
	cmd.SetOut(os.Stdout)
	cmd.PersistentFlags().StringVar(&configFile, "config", "", "(optional) chloggen config file")
	cmd.AddCommand(newCmd())
	cmd.AddCommand(updateCmd())
	cmd.AddCommand(validateCmd())
	return cmd
}

// Execute executes the root command.
func Execute() {
	cobra.CheckErr(rootCmd().Execute())
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	// Don't override if already set in tests
	if globalCfg != nil {
		return
	}

	if configFile == "" {
		globalCfg = config.New(repoRoot())
	} else {
		var err error
		globalCfg, err = config.NewFromFile(repoRoot(), configFile)
		if err != nil {
			fmt.Printf("FAIL: Could not load config file: %s\n", err.Error())
			os.Exit(1)
		}
	}
}

func repoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		// This is not expected, but just in case
		fmt.Println("FAIL: Could not determine current working directory")
	}
	return dir
}
