package main

import (
	"github.com/go-clix/cli"
	"github.com/gobwas/glob"
	"github.com/posener/complete"

	"github.com/grafana/tanka/pkg/jsonnet"
)

func lintCmd() *cli.Command {
	cmd := &cli.Command{
		Use:   "lint <FILES|DIRECTORIES>",
		Short: "lint Jsonnet code",
		Args: cli.Args{
			Validator: cli.ArgsMin(1),
			Predictor: complete.PredictFiles("*.*sonnet"),
		},
	}

	exclude := cmd.Flags().StringSliceP("exclude", "e", []string{"**/.*", ".*", "**/vendor/**", "vendor/**"}, "globs to exclude")
	parallelism := cmd.Flags().IntP("parallelism", "n", 4, "amount of workers")

	// this is now always sent as debug logs
	cmd.Flags().BoolP("verbose", "v", false, "print each checked file")
	if err := cmd.Flags().MarkDeprecated("verbose", "logs are sent to debug now, this is unused"); err != nil {
		panic(err)
	}

	cmd.Run = func(_ *cli.Command, args []string) error {
		globs := make([]glob.Glob, len(*exclude))
		for i, e := range *exclude {
			g, err := glob.Compile(e)
			if err != nil {
				return err
			}
			globs[i] = g
		}

		return jsonnet.Lint(args, &jsonnet.LintOpts{Excludes: globs, Parallelism: *parallelism})
	}

	return cmd
}
