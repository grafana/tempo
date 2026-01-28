package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-clix/cli"
	"github.com/posener/complete"

	"github.com/grafana/tanka/pkg/jsonnet"
	"github.com/grafana/tanka/pkg/jsonnet/jpath"
)

func toolCmd() *cli.Command {
	cmd := &cli.Command{
		Short: "handy utilities for working with jsonnet",
		Use:   "tool [command]",
		Args:  cli.ArgsMin(1), // Make sure we print out the help if no subcommand is given, `tk tool` is not valid
	}

	addCommandsWithLogLevelOption(
		cmd,
		jpathCmd(),
		importsCmd(),
		importersCmd(),
		importersCountCmd(),
		chartsCmd(),
	)
	return cmd
}

func jpathCmd() *cli.Command {
	cmd := &cli.Command{
		Short: "export JSONNET_PATH for use with other jsonnet tools",
		Use:   "jpath <file/dir>",
		Args:  workflowArgs,
	}

	debug := cmd.Flags().BoolP("debug", "d", false, "show debug info")

	cmd.Run = func(_ *cli.Command, args []string) error {
		path := args[0]

		entrypoint, err := jpath.Entrypoint(path)
		if err != nil {
			return fmt.Errorf("resolving JPATH: %s", err)
		}

		jsonnetpath, base, root, err := jpath.Resolve(entrypoint, false)
		if err != nil {
			return fmt.Errorf("resolving JPATH: %s", err)
		}

		if *debug {
			// log to debug info to stderr
			fmt.Fprintln(os.Stderr, "main:", entrypoint)
			fmt.Fprintln(os.Stderr, "rootDir:", root)
			fmt.Fprintln(os.Stderr, "baseDir:", base)
			fmt.Fprintln(os.Stderr, "jpath:", jsonnetpath)
		}

		// print export JSONNET_PATH to stdout
		fmt.Printf("%s", strings.Join(jsonnetpath, ":"))

		return nil
	}

	return cmd
}

func importsCmd() *cli.Command {
	cmd := &cli.Command{
		Use:   "imports <path>",
		Short: "list all transitive imports of an environment",
		Args:  workflowArgs,
	}

	check := cmd.Flags().StringP("check", "c", "", "git commit hash to check against")

	cmd.Run = func(_ *cli.Command, args []string) error {
		var modFiles []string
		if *check != "" {
			var err error
			modFiles, err = gitChangedFiles(*check)
			if err != nil {
				return fmt.Errorf("invoking git: %s", err)
			}
		}

		path, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("loading environment: %s", err)
		}

		deps, err := jsonnet.TransitiveImports(path)
		if err != nil {
			return fmt.Errorf("resolving imports: %s", err)
		}

		root, err := gitRoot()
		if err != nil {
			return fmt.Errorf("invoking git: %s", err)
		}
		if modFiles != nil {
			for _, m := range modFiles {
				mod := filepath.Join(root, m)
				if err != nil {
					return err
				}

				for _, dep := range deps {
					if mod == dep {
						fmt.Printf("Rebuild required. File `%s` imports `%s`, which has been changed in `%s`.\n", args[0], dep, *check)
						os.Exit(16)
					}
				}
			}
			fmt.Printf("Rebuild not required, because no imported files have been changed in `%s`.\n", *check)
			os.Exit(0)
		}

		s, err := json.Marshal(deps)
		if err != nil {
			return fmt.Errorf("formatting: %s", err)
		}
		fmt.Println(string(s))

		return nil
	}

	return cmd
}

func importersCmd() *cli.Command {
	cmd := &cli.Command{
		Use:   "importers <file> <file...> <deleted:file...>",
		Short: "list all environments that either directly or transitively import the given files",
		Long: `list all environments that either directly or transitively import the given files
If the file being looked up was deleted, it should be prefixed with "deleted:".

As optimization,
if the file is not a vendored (located at <tk-root>/vendor/) or a lib file (located at <tk-root>/lib/), we assume:
- it is used in a Tanka environment
- it will not be imported by any lib or vendor files
- the environment base (closest main file in parent dirs) will be considered an importer
- if no base is found, all main files in child dirs will be considered importers
`,
		Args: cli.Args{
			Validator: cli.ArgsMin(1),
			Predictor: complete.PredictFiles("*"),
		},
	}

	root := cmd.Flags().String("root", ".", "root directory to search for environments")
	cmd.Run = func(_ *cli.Command, args []string) error {
		root, err := filepath.Abs(*root)
		if err != nil {
			return fmt.Errorf("resolving root: %w", err)
		}

		for _, f := range args {
			if strings.HasPrefix(f, "deleted:") {
				continue
			}
			if _, err := os.Stat(f); os.IsNotExist(err) {
				return fmt.Errorf("file %q does not exist", f)
			}
		}

		envs, err := jsonnet.FindImporterForFiles(root, args)
		if err != nil {
			return fmt.Errorf("resolving imports: %s", err)
		}

		fmt.Println(strings.Join(envs, "\n"))

		return nil
	}

	return cmd
}

func importersCountCmd() *cli.Command {
	cmd := &cli.Command{
		Use:   "importers-count <dir>",
		Short: "for each file in the given directory, list the number of environments that import it",
		Long: `for each file in the given directory, list the number of environments that import it

As optimization,
if the file is not a vendored (located at <tk-root>/vendor/) or a lib file (located at <tk-root>/lib/), we assume:
- it is used in a Tanka environment
- it will not be imported by any lib or vendor files
- the environment base (closest main file in parent dirs) will be considered an importer
- if no base is found, all main files in child dirs will be considered importers
`,
		Args: cli.Args{
			Validator: cli.ArgsExact(1),
			Predictor: complete.PredictDirs("*"),
		},
	}

	root := cmd.Flags().String("root", ".", "root directory to search for environments")
	recursive := cmd.Flags().Bool("recursive", false, "find files recursively")
	filenameRegex := cmd.Flags().String("filename-regex", "", "only count files that match the given regex. Matches only jsonnet files by default")

	cmd.Run = func(_ *cli.Command, args []string) error {
		dir := args[0]

		root, err := filepath.Abs(*root)
		if err != nil {
			return fmt.Errorf("resolving root: %w", err)
		}

		result, err := jsonnet.CountImporters(root, dir, *recursive, *filenameRegex)
		if err != nil {
			return fmt.Errorf("resolving imports: %s", err)
		}

		fmt.Println(result)

		return nil
	}

	return cmd
}

func gitRoot() (string, error) {
	s, err := git("rev-parse", "--show-toplevel")
	return strings.TrimRight(s, "\n"), err
}

func gitChangedFiles(sha string) ([]string, error) {
	f, err := git("diff-tree", "--no-commit-id", "--name-only", "-r", sha)
	if err != nil {
		return nil, err
	}
	return strings.Split(f, "\n"), nil
}

func git(argv ...string) (string, error) {
	cmd := exec.Command("git", argv...)
	cmd.Stderr = os.Stderr
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return buf.String(), nil
}
