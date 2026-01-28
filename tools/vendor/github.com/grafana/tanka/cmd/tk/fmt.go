package main

import (
	"fmt"
	"io"
	"os"

	"github.com/go-clix/cli"
	"github.com/gobwas/glob"
	"github.com/posener/complete"

	"github.com/grafana/tanka/pkg/tanka"
)

// ArgStdin is the "magic" argument for reading from stdin
const ArgStdin = "-"

func fmtCmd() *cli.Command {
	cmd := &cli.Command{
		Use:   "fmt <FILES|DIRECTORIES>",
		Short: "format Jsonnet code",
		Args: cli.Args{
			Validator: cli.ArgsMin(1),
			Predictor: complete.PredictFiles("*.*sonnet"),
		},
	}

	stdout := cmd.Flags().Bool("stdout", false, "print formatted contents to stdout instead of writing to disk")
	test := cmd.Flags().BoolP("test", "t", false, "exit with non-zero when changes would be made")
	exclude := cmd.Flags().StringSliceP("exclude", "e", []string{"**/.*", ".*", "**/vendor/**", "vendor/**"}, "globs to exclude")
	verbose := cmd.Flags().BoolP("verbose", "v", false, "print each checked file")

	cmd.Run = func(_ *cli.Command, args []string) error {
		if len(args) == 1 && args[0] == ArgStdin {
			return fmtStdin(*test)
		}

		globs := make([]glob.Glob, len(*exclude))
		for i, e := range *exclude {
			g, err := glob.Compile(e)
			if err != nil {
				return err
			}
			globs[i] = g
		}

		var outFn tanka.OutFn
		switch {
		case *test:
			outFn = func(_, _ string) error { return nil }
		case *stdout:
			outFn = func(name, content string) error {
				fmt.Printf("// %s\n%s", name, content)
				fmt.Fprintln(os.Stderr) // some spacing
				return nil
			}
		}

		opts := &tanka.FormatOpts{
			Excludes:   globs,
			OutFn:      outFn,
			PrintNames: *verbose,
		}

		changed, err := tanka.FormatFiles(args, opts)
		if err != nil {
			return err
		}

		if *verbose {
			fmt.Fprintln(os.Stderr)
		}

		switch {
		case *test && len(changed) > 0:
			fmt.Fprintln(os.Stderr, "The following files are not properly formatted:")
			for _, s := range changed {
				fmt.Fprintln(os.Stderr, s)
			}
			os.Exit(ExitStatusDiff)
		case len(changed) == 0:
			fmt.Fprintln(os.Stderr, "All discovered files are already formatted. No changes were made")
		case len(changed) > 0:
			fmt.Fprintf(os.Stderr, "Formatted %v files\n", len(changed))
		}

		return nil
	}

	return cmd
}

func fmtStdin(test bool) error {
	content, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	formatted, err := tanka.Format("<stdin>", string(content))
	if err != nil {
		return err
	}

	fmt.Print(formatted)
	if test && string(content) != formatted {
		os.Exit(ExitStatusDiff)
	}
	return nil
}
