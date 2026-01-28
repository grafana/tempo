package main

import (
	"errors"
	"fmt"
	"regexp"
	"runtime"

	"github.com/go-clix/cli"

	"github.com/grafana/tanka/pkg/process"
	"github.com/grafana/tanka/pkg/spec/v1alpha1"
	"github.com/grafana/tanka/pkg/tanka"
)

func exportCmd() *cli.Command {
	args := workflowArgs
	args.Validator = cli.ArgsMin(2)

	cmd := &cli.Command{
		Use:   "export <outputDir> <path> [<path>...]",
		Short: "export environments found in path(s)",
		Args:  args,
	}

	format := cmd.Flags().String(
		"format",
		"{{.apiVersion}}.{{.kind}}-{{or .metadata.name .metadata.generateName}}",
		"https://tanka.dev/exporting#filenames",
	)

	extension := cmd.Flags().String("extension", "yaml", "File extension")
	parallel := cmd.Flags().IntP("parallel", "p", 8, "Number of environments to process in parallel")
	cachePath := cmd.Flags().StringP("cache-path", "c", "", "Local file path where cached evaluations should be stored")
	cacheEnvs := cmd.Flags().StringArrayP("cache-envs", "e", nil, "Regexes which define which environment should be cached (if caching is enabled)")
	ballastBytes := cmd.Flags().Int("mem-ballast-size-bytes", 0, "Size of memory ballast to allocate. This may improve performance for large environments.")

	merge := cmd.Flags().Bool("merge", false, "Allow merging with existing directory")
	if err := cmd.Flags().MarkDeprecated("merge", "use --merge-strategy=fail-on-conflicts instead"); err != nil {
		panic(err)
	}
	mergeStrategy := cmd.Flags().String("merge-strategy", "", "What to do when exporting to an existing directory. The default setting is to disallow exporting to an existing directory. Values: 'fail-on-conflicts', 'replace-envs'")
	mergeDeletedEnvs := cmd.Flags().StringArray("merge-deleted-envs", nil, "Tanka main files that have been deleted. This is used when using a merge strategy to also delete the files of these deleted environments.")

	vars := workflowFlags(cmd.Flags())
	getJsonnetOpts := jsonnetFlags(cmd.Flags())
	getLabelSelector := labelSelectorFlag(cmd.Flags())

	recursive := cmd.Flags().BoolP("recursive", "r", false, "Look recursively for Tanka environments")

	cmd.Run = func(_ *cli.Command, args []string) error {
		// Allocate a block of memory to alter GC behaviour. See https://github.com/golang/go/issues/23044
		ballast := make([]byte, *ballastBytes)
		defer runtime.KeepAlive(ballast)

		filters, err := process.StrExps(vars.targets...)
		if err != nil {
			return err
		}

		opts := tanka.ExportEnvOpts{
			Format:    *format,
			Extension: *extension,
			Opts: tanka.Opts{
				JsonnetImplementation: vars.jsonnetImplementation,
				JsonnetOpts:           getJsonnetOpts(),
				Filters:               filters,
				Name:                  vars.name,
			},
			Selector:         getLabelSelector(),
			Parallelism:      *parallel,
			MergeDeletedEnvs: *mergeDeletedEnvs,
		}

		if opts.MergeStrategy, err = determineMergeStrategy(*merge, *mergeStrategy); err != nil {
			return err
		}

		opts.Opts.CachePath = *cachePath
		for _, expr := range *cacheEnvs {
			regex, err := regexp.Compile(expr)
			if err != nil {
				return err
			}
			opts.Opts.CachePathRegexes = append(opts.Opts.CachePathRegexes, regex)
		}

		var exportEnvs []*v1alpha1.Environment
		// find possible environments
		if *recursive {
			// get absolute path to Environment
			envs, err := tanka.FindEnvsFromPaths(args[1:], tanka.FindOpts{Selector: opts.Selector, Parallelism: opts.Parallelism, JsonnetOpts: opts.Opts.JsonnetOpts})
			if err != nil {
				return err
			}

			for _, env := range envs {
				if opts.Opts.Name != "" && opts.Opts.Name != env.Metadata.Name {
					continue
				}
				exportEnvs = append(exportEnvs, env)
			}
		} else {
			if len(args[1:]) > 1 {
				return fmt.Errorf("recursive flag is required when exporting multiple environments")
			}

			// validate environment
			env, err := tanka.Peek(args[1], opts.Opts)
			if err != nil {
				switch err.(type) {
				case tanka.ErrMultipleEnvs:
					fmt.Println("Please use --name to export a single environment or --recursive to export multiple environments.")
					return err
				default:
					return err
				}
			}

			exportEnvs = append(exportEnvs, env)
		}

		// export them
		return tanka.ExportEnvironments(exportEnvs, args[0], &opts)
	}
	return cmd
}

// `--merge` is deprecated in favor of `--merge-strategy`. However, merge has to keep working for now.
func determineMergeStrategy(deprecatedMergeFlag bool, mergeStrategy string) (tanka.ExportMergeStrategy, error) {
	if deprecatedMergeFlag && mergeStrategy != "" {
		return "", errors.New("cannot use --merge and --merge-strategy at the same time")
	}
	if deprecatedMergeFlag {
		return tanka.ExportMergeStrategyFailConflicts, nil
	}

	switch strategy := tanka.ExportMergeStrategy(mergeStrategy); strategy {
	case tanka.ExportMergeStrategyFailConflicts, tanka.ExportMergeStrategyReplaceEnvs, tanka.ExportMergeStrategyNone:
		return strategy, nil
	}

	return "", fmt.Errorf("invalid merge strategy: %q", mergeStrategy)
}
