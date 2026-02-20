package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/go-clix/cli"
	"github.com/posener/complete"

	"github.com/grafana/tanka/pkg/process"
	"github.com/grafana/tanka/pkg/tanka"
	"github.com/grafana/tanka/pkg/term"
)

// special exit codes for tk diff
const (
	// no changes
	ExitStatusClean = 0
	// differences between the local config and the cluster
	ExitStatusDiff = 16
)

var (
	colorValues = cli.PredictSet("auto", "always", "never")
)

func setForceColor(opts *tanka.DiffBaseOpts) error {
	switch opts.Color {
	case "":
	case "auto":
	case "always":
		color.NoColor = false
	case "never":
		color.NoColor = true
	default:
		return fmt.Errorf(`--color must be either: "auto", "always", or "never"`)
	}
	return nil
}

func validateDryRun(dryRunStr string) error {
	switch dryRunStr {
	case "", "none", "client", "server":
		return nil
	}
	return fmt.Errorf(`--dry-run must be either: "", "none", "server" or "client"`)
}

func validateAutoApprove(autoApproveDeprecated bool, autoApproveString string) (tanka.AutoApproveSetting, error) {
	var result tanka.AutoApproveSetting

	if autoApproveString != "" && autoApproveDeprecated {
		return result, fmt.Errorf("--dangerous-auto-approve and --auto-approve are mutually exclusive")
	}
	if autoApproveString == "" {
		if autoApproveDeprecated {
			result = tanka.AutoApproveAlways
		} else {
			result = tanka.AutoApproveNever
		}
	} else {
		if autoApproveString != string(tanka.AutoApproveAlways) && autoApproveString != string(tanka.AutoApproveNever) && autoApproveString != string(tanka.AutoApproveNoChanges) {
			return result, fmt.Errorf("invalid value for --auto-approve: %s", autoApproveString)
		}
		result = tanka.AutoApproveSetting(autoApproveString)
	}

	return result, nil
}

func applyCmd(ctx context.Context) *cli.Command {
	cmd := &cli.Command{
		Use:   "apply <path>",
		Short: "apply the configuration to the cluster",
		Args:  generateWorkflowArgs(ctx),
		Predictors: complete.Flags{
			"color":          colorValues,
			"diff-strategy":  cli.PredictSet("native", "subset", "validate", "server", "none"),
			"apply-strategy": cli.PredictSet("client", "server"),
		},
	}

	var opts tanka.ApplyOpts
	cmd.Flags().BoolVar(&opts.Validate, "validate", true, "validation of resources (kubectl --validate=false)")
	cmd.Flags().StringVar(&opts.ApplyStrategy, "apply-strategy", "", "force the apply strategy to use. Automatically chosen if not set.")
	cmd.Flags().StringVar(&opts.DiffStrategy, "diff-strategy", "", "force the diff strategy to use. Automatically chosen if not set.")

	var (
		autoApproveDeprecated bool
		autoApproveString     string
	)
	addApplyFlags(cmd.Flags(), &opts.ApplyBaseOpts, &autoApproveDeprecated, &autoApproveString)
	addDiffFlags(cmd.Flags(), &opts.DiffBaseOpts)
	vars := workflowFlags(cmd.Flags())
	getJsonnetOpts := jsonnetFlags(cmd.Flags())

	cmd.Run = func(_ *cli.Command, args []string) error {
		ctx, span := tracer.Start(ctx, "applyCmd")
		defer span.End()
		err := validateDryRun(opts.DryRun)
		if err != nil {
			return err
		}
		if opts.AutoApprove, err = validateAutoApprove(autoApproveDeprecated, autoApproveString); err != nil {
			return err
		}
		if err := setForceColor(&opts.DiffBaseOpts); err != nil {
			return err
		}

		filters, err := process.StrExps(vars.targets...)
		if err != nil {
			return err
		}
		opts.Filters = filters
		opts.JsonnetOpts = getJsonnetOpts()
		opts.Name = vars.name
		opts.JsonnetImplementation = vars.jsonnetImplementation

		return tanka.Apply(ctx, args[0], opts)
	}
	return cmd
}

func pruneCmd(ctx context.Context) *cli.Command {
	cmd := &cli.Command{
		Use:   "prune <path>",
		Short: "delete resources removed from Jsonnet",
		Args:  generateWorkflowArgs(ctx),
		Predictors: complete.Flags{
			"color": colorValues,
		},
	}

	var opts tanka.PruneOpts
	cmd.Flags().StringVar(&opts.Name, "name", "", "string that only a single inline environment contains in its name")
	var (
		autoApproveDeprecated bool
		autoApproveString     string
	)
	addApplyFlags(cmd.Flags(), &opts.ApplyBaseOpts, &autoApproveDeprecated, &autoApproveString)
	addDiffFlags(cmd.Flags(), &opts.DiffBaseOpts)
	getJsonnetOpts := jsonnetFlags(cmd.Flags())

	cmd.Run = func(_ *cli.Command, args []string) error {
		ctx, span := tracer.Start(ctx, "pruneCmd")
		defer span.End()
		err := validateDryRun(opts.DryRun)
		if err != nil {
			return err
		}
		if opts.AutoApprove, err = validateAutoApprove(autoApproveDeprecated, autoApproveString); err != nil {
			return err
		}
		if err := setForceColor(&opts.DiffBaseOpts); err != nil {
			return err
		}

		opts.JsonnetOpts = getJsonnetOpts()

		return tanka.Prune(ctx, args[0], opts)
	}

	return cmd
}

func deleteCmd(ctx context.Context) *cli.Command {
	cmd := &cli.Command{
		Use:   "delete <path>",
		Short: "delete the environment from cluster",
		Args:  generateWorkflowArgs(ctx),
		Predictors: complete.Flags{
			"color": colorValues,
		},
	}

	var opts tanka.DeleteOpts

	var (
		autoApproveDeprecated bool
		autoApproveString     string
	)
	addApplyFlags(cmd.Flags(), &opts.ApplyBaseOpts, &autoApproveDeprecated, &autoApproveString)
	addDiffFlags(cmd.Flags(), &opts.DiffBaseOpts)
	vars := workflowFlags(cmd.Flags())
	getJsonnetOpts := jsonnetFlags(cmd.Flags())

	cmd.Run = func(_ *cli.Command, args []string) error {
		ctx, span := tracer.Start(ctx, "deleteCmd")
		defer span.End()
		err := validateDryRun(opts.DryRun)
		if err != nil {
			return err
		}
		if opts.AutoApprove, err = validateAutoApprove(autoApproveDeprecated, autoApproveString); err != nil {
			return err
		}
		if err := setForceColor(&opts.DiffBaseOpts); err != nil {
			return err
		}

		filters, err := process.StrExps(vars.targets...)
		if err != nil {
			return err
		}
		opts.Filters = filters
		opts.JsonnetOpts = getJsonnetOpts()
		opts.Name = vars.name
		opts.JsonnetImplementation = vars.jsonnetImplementation

		return tanka.Delete(ctx, args[0], opts)
	}
	return cmd
}

func diffCmd(ctx context.Context) *cli.Command {
	cmd := &cli.Command{
		Use:   "diff <path>",
		Short: "differences between the configuration and the cluster",
		Args:  generateWorkflowArgs(ctx),
		Predictors: complete.Flags{
			"color":         colorValues,
			"diff-strategy": cli.PredictSet("native", "subset", "validate", "server"),
		},
	}

	var opts tanka.DiffOpts
	addDiffFlags(cmd.Flags(), &opts.DiffBaseOpts)
	cmd.Flags().StringVar(&opts.Strategy, "diff-strategy", "", "force the diff-strategy to use. Automatically chosen if not set.")
	cmd.Flags().BoolVarP(&opts.Summarize, "summarize", "s", false, "print summary of the differences, not the actual contents")
	cmd.Flags().BoolVarP(&opts.WithPrune, "with-prune", "p", false, "include objects deleted from the configuration in the differences")
	cmd.Flags().BoolVarP(&opts.ExitZero, "exit-zero", "z", false, "Exit with 0 even when differences are found.")
	cmd.Flags().BoolVar(&opts.ListModifiedEnvs, "list-modified-envs", false, "List environments with changes")

	vars := workflowFlags(cmd.Flags())
	getJsonnetOpts := jsonnetFlags(cmd.Flags())

	cmd.Run = func(_ *cli.Command, args []string) error {
		ctx, span := tracer.Start(ctx, "diffCmd")
		defer span.End()
		if err := setForceColor(&opts.DiffBaseOpts); err != nil {
			return err
		}
		filters, err := process.StrExps(vars.targets...)
		if err != nil {
			return err
		}
		opts.Filters = filters
		opts.JsonnetOpts = getJsonnetOpts()
		opts.Name = vars.name
		opts.JsonnetImplementation = vars.jsonnetImplementation

		changes, err := tanka.Diff(ctx, args[0], opts)
		if err != nil {
			return err
		}

		if changes == nil {
			if opts.ListModifiedEnvs {
				fmt.Fprintln(os.Stderr, "No environments with changes.")
			} else {
				fmt.Fprintln(os.Stderr, "No differences.")
			}
			os.Exit(ExitStatusClean)
		}

		// For special modes, print output directly without color processing
		if opts.ListModifiedEnvs {
			fmt.Print(*changes)
		} else {
			r := term.Colordiff(*changes)
			if err := fPageln(r); err != nil {
				return err
			}
		}

		// For --list-modified-envs, always exit with success code
		if opts.ListModifiedEnvs {
			os.Exit(ExitStatusClean)
		}

		exitStatusDiff := ExitStatusDiff
		if opts.ExitZero {
			exitStatusDiff = ExitStatusClean
		}
		span.End()
		os.Exit(exitStatusDiff)
		return nil
	}

	return cmd
}

func showCmd(ctx context.Context) *cli.Command {
	cmd := &cli.Command{
		Use:   "show <path>",
		Short: "jsonnet as yaml",
		Args:  generateWorkflowArgs(ctx),
	}

	allowRedirectFlag := cmd.Flags().Bool("dangerous-allow-redirect", false, "allow redirecting output to a file or a pipe.")

	vars := workflowFlags(cmd.Flags())
	getJsonnetOpts := jsonnetFlags(cmd.Flags())

	cmd.Run = func(_ *cli.Command, args []string) error {
		ctx, span := tracer.Start(ctx, "showCmd")
		defer span.End()
		allowRedirectEnv := os.Getenv("TANKA_DANGEROUS_ALLOW_REDIRECT") == "true"
		allowRedirect := allowRedirectEnv || *allowRedirectFlag

		if !interactive && !allowRedirect {
			fmt.Fprintln(os.Stderr, `Redirection of the output of tk show is discouraged and disabled by default.
If you want to export .yaml files for use with other tools, try 'tk export'.
Otherwise run:
  tk show --dangerous-allow-redirect 
or set the environment variable 
  TANKA_DANGEROUS_ALLOW_REDIRECT=true 
to bypass this check.`)
			return nil
		}

		filters, err := process.StrExps(vars.targets...)
		if err != nil {
			return err
		}

		pretty, err := tanka.Show(ctx, args[0], tanka.Opts{
			JsonnetOpts:           getJsonnetOpts(),
			Filters:               filters,
			Name:                  vars.name,
			JsonnetImplementation: vars.jsonnetImplementation,
		})

		if err != nil {
			return err
		}

		return pageln(pretty.String())
	}
	return cmd
}
