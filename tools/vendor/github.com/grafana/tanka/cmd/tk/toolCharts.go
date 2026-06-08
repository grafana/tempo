package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-clix/cli"
	"github.com/grafana/tanka/pkg/helm"
	"gopkg.in/yaml.v2"
)

const repoConfigFlagUsage = "specify a local helm repository config file to use instead of the repositories in the chartfile.yaml. For use with private repositories"

func chartsCmd(ctx context.Context) *cli.Command {
	cmd := &cli.Command{
		Use:   "charts",
		Short: "Declarative vendoring of Helm Charts",
		Args:  cli.ArgsMin(1), // Make sure we print out the help if no subcommand is given, `tk tool charts` is not valid
	}

	addCommandsWithLogLevelOption(
		cmd,
		chartsInitCmd(ctx),
		chartsAddCmd(ctx),
		chartsAddRepoCmd(ctx),
		chartsVendorCmd(ctx),
		chartsConfigCmd(ctx),
		chartsVersionCheckCmd(ctx),
	)

	return cmd
}

func chartsVendorCmd(ctx context.Context) *cli.Command {
	cmd := &cli.Command{
		Use:   "vendor",
		Short: "Download Charts to a local folder",
	}
	prune := cmd.Flags().Bool("prune", false, "also remove non-vendored files from the destination directory")
	repoConfigPath := cmd.Flags().String("repository-config", "", repoConfigFlagUsage)

	cmd.Run = func(_ *cli.Command, _ []string) error {
		_, span := tracer.Start(ctx, "chartsVendorCmd")
		defer span.End()
		c, err := loadChartfile()
		if err != nil {
			return err
		}

		return c.Vendor(*prune, *repoConfigPath)
	}

	return cmd
}

func chartsAddCmd(ctx context.Context) *cli.Command {
	cmd := &cli.Command{
		Use:   "add [chart@version] [...]",
		Short: "Adds Charts to the chartfile",
	}
	repoConfigPath := cmd.Flags().String("repository-config", "", repoConfigFlagUsage)

	cmd.Run = func(_ *cli.Command, args []string) error {
		_, span := tracer.Start(ctx, "chartsVendorCmd")
		defer span.End()
		c, err := loadChartfile()
		if err != nil {
			return err
		}

		return c.Add(args, *repoConfigPath)
	}

	return cmd
}

func chartsAddRepoCmd(ctx context.Context) *cli.Command {
	cmd := &cli.Command{
		Use:   "add-repo [NAME] [URL]",
		Short: "Adds a repository to the chartfile",
		Args:  cli.ArgsExact(2),
	}

	cmd.Run = func(_ *cli.Command, args []string) error {
		_, span := tracer.Start(ctx, "chartsVendorCmd")
		defer span.End()
		c, err := loadChartfile()
		if err != nil {
			return err
		}

		return c.AddRepos(helm.Repo{
			Name: args[0],
			URL:  args[1],
		})
	}

	return cmd
}

func chartsConfigCmd(ctx context.Context) *cli.Command {
	cmd := &cli.Command{
		Use:   "config",
		Short: "Displays the current manifest",
	}

	cmd.Run = func(_ *cli.Command, _ []string) error {
		_, span := tracer.Start(ctx, "chartsVendorCmd")
		defer span.End()
		c, err := loadChartfile()
		if err != nil {
			return err
		}

		data, err := yaml.Marshal(c.Manifest)
		if err != nil {
			return err
		}

		fmt.Print(string(data))

		return nil
	}

	return cmd
}

func chartsInitCmd(ctx context.Context) *cli.Command {
	cmd := &cli.Command{
		Use:   "init",
		Short: "Create a new Chartfile",
	}

	cmd.Run = func(_ *cli.Command, _ []string) error {
		_, span := tracer.Start(ctx, "chartsVendorCmd")
		defer span.End()
		wd, err := os.Getwd()
		if err != nil {
			return err
		}

		path := filepath.Join(wd, helm.Filename)
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("chartfile at '%s' already exists. Aborting", path)
		}

		if _, err := helm.InitChartfile(path); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Success! New Chartfile created at '%s'", path)
		return nil
	}

	return cmd
}

func chartsVersionCheckCmd(ctx context.Context) *cli.Command {
	cmd := &cli.Command{
		Use:   "version-check",
		Short: "Check required charts for updated versions",
	}
	repoConfigPath := cmd.Flags().String("repository-config", "", repoConfigFlagUsage)
	prettyPrint := cmd.Flags().Bool("pretty-print", false, "pretty print json output with indents")

	cmd.Run = func(_ *cli.Command, _ []string) error {
		_, span := tracer.Start(ctx, "chartsVendorCmd")
		defer span.End()
		c, err := loadChartfile()
		if err != nil {
			return err
		}

		data, err := c.VersionCheck(*repoConfigPath)
		if err != nil {
			return err
		}

		enc := json.NewEncoder(os.Stdout)
		if *prettyPrint {
			enc.SetIndent("", "  ")
		}
		return enc.Encode(data)
	}

	return cmd
}

func loadChartfile() (*helm.Charts, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	return helm.LoadChartfile(wd)
}
