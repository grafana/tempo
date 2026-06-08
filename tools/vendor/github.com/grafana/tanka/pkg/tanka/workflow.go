package tanka

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"

	"github.com/grafana/tanka/pkg/kubernetes"
	"github.com/grafana/tanka/pkg/kubernetes/client"
	"github.com/grafana/tanka/pkg/kubernetes/manifest"
	"github.com/grafana/tanka/pkg/spec/v1alpha1"
	"github.com/grafana/tanka/pkg/term"
)

type AutoApproveSetting string

const (
	// AutoApproveNever disables auto-approval
	AutoApproveNever AutoApproveSetting = "never"
	// AutoApproveAlways enables auto-approval
	AutoApproveAlways AutoApproveSetting = "always"
	// AutoApproveNoChanges enables auto-approval if there were no changes found in the diff
	AutoApproveNoChanges AutoApproveSetting = "if-no-changes"

	ApplyStrategyServer = "server"
	ApplyStrategyClient = "client"
)

type ApplyBaseOpts struct {
	Opts
	DiffBaseOpts

	// AutoApprove skips the interactive approval
	AutoApprove AutoApproveSetting
	// DryRun string passed to kubectl as --dry-run=<DryRun>
	DryRun string
	// Force ignores any warnings kubectl might have
	Force bool
}

type DiffBaseOpts struct {
	// Color controls color output
	Color string
}

// ApplyOpts specify additional properties for the Apply action
type ApplyOpts struct {
	ApplyBaseOpts

	// DiffStrategy to use for printing the diff before approval
	DiffStrategy string
	// ApplyStrategy decides how kubectl will apply the manifest
	ApplyStrategy string
	// Validate set to false ignores invalid Kubernetes schemas
	Validate bool

	// ServerSide bool passed to kubectl as --server-side
	ServerSide bool
}

// ErrorApplyStrategyUnknown occurs when an apply-strategy is requested that does
// not exist. Unlike ErrorDiffStrategyUnknown, this needs to be used before things
// reach the `kube.Apply` function.
type ErrorApplyStrategyUnknown struct {
	Requested string
}

func (e ErrorApplyStrategyUnknown) Error() string {
	return fmt.Sprintf("apply strategy `%s` does not exist. Pick one of: [server, client].", e.Requested)
}

// Apply parses the environment at the given directory (a `baseDir`) and applies
// the evaluated jsonnet to the Kubernetes cluster defined in the environments
// `spec.json`.
func Apply(ctx context.Context, baseDir string, opts ApplyOpts) error {
	l, err := Load(ctx, baseDir, opts.Opts)
	if err != nil {
		return err
	}

	// If the apply strategy was not set on the command-line, draw from spec or use default
	if opts.ApplyStrategy == "" {
		if l.Env.Spec.ApplyStrategy != "" {
			opts.ApplyStrategy = l.Env.Spec.ApplyStrategy
		} else {
			opts.ApplyStrategy = ApplyStrategyClient
		}
	}
	if opts.ApplyStrategy != ApplyStrategyClient && opts.ApplyStrategy != ApplyStrategyServer {
		return ErrorApplyStrategyUnknown{Requested: opts.ApplyStrategy}
	}

	// Default to `server` diff in server apply mode
	if opts.ApplyStrategy == ApplyStrategyServer && opts.DiffStrategy == "" && l.Env.Spec.DiffStrategy == "" {
		l.Env.Spec.DiffStrategy = ApplyStrategyServer
	}

	kube, err := l.Connect()
	if err != nil {
		return err
	}
	defer kube.Close()

	var noChanges bool
	if opts.DiffStrategy != "none" {
		// show diff
		diff, err := kube.Diff(ctx, l.Resources, kubernetes.DiffOpts{Strategy: opts.DiffStrategy})
		switch {
		case err != nil:
			// This is not fatal, the diff is not strictly required
			log.Error().Err(err).Msg("error diffing")
		case diff == nil:
			noChanges = true
			// If using KUBECTL_INTERACTIVE_DIFF, the stdout buffer is always empty
			if os.Getenv("KUBECTL_INTERACTIVE_DIFF") == "" {
				tmp := "Warning: There are no differences. Your apply may not do anything at all."
				diff = &tmp
			}
		}

		// in case of non-fatal error diff may be nil
		if diff != nil {
			b := term.Colordiff(*diff)
			fmt.Print(b.String())
		}
	}

	// prompt for confirmation
	if opts.AutoApprove != AutoApproveAlways && !(noChanges && opts.AutoApprove == AutoApproveNoChanges) && opts.DryRun == "" {
		if err := confirmPrompt("Applying to", l.Env.Spec.Namespace, kube.Info()); err != nil {
			return err
		}
	}

	return kube.Apply(l.Resources, kubernetes.ApplyOpts{
		Force:         opts.Force,
		Validate:      opts.Validate,
		DryRun:        opts.DryRun,
		ApplyStrategy: opts.ApplyStrategy,
	})
}

// confirmPrompt asks the user for confirmation before apply
func confirmPrompt(action, namespace string, info client.Info) error {
	alert := color.New(color.FgRed, color.Bold).SprintFunc()

	return term.Confirm(
		fmt.Sprintf(`%s namespace '%s' of cluster '%s' at '%s' using context '%s'.`, action,
			alert(namespace),
			alert(info.Kubeconfig.Cluster.Name),
			alert(info.Kubeconfig.Cluster.Cluster.Server),
			alert(info.Kubeconfig.Context.Name),
		),
		"yes",
	)
}

// DiffOpts specify additional properties for the Diff action
type DiffOpts struct {
	DiffBaseOpts
	Opts

	// Strategy must be one of "native", "validate", "subset" or "server"
	Strategy string
	// Summarize prints a summary, instead of the actual diff
	Summarize bool
	// WithPrune includes objects to be deleted by prune command in the diff
	WithPrune bool
	// Exit with 0 even when differences are found
	ExitZero bool
	// List all available environments and exit
	ListModifiedEnvs bool
}

// Diff parses the environment at the given directory (a `baseDir`) and returns
// the differences from the live cluster state in `diff(1)` format. If the
// `WithDiffSummarize` modifier is used, a histogram is returned instead.
// The cluster information is retrieved from the environments `spec.json`.
// NOTE: This function requires on `diff(1)` and `kubectl(1)`
func Diff(ctx context.Context, baseDir string, opts DiffOpts) (*string, error) {
	if opts.ListModifiedEnvs {
		return ListChangedEnvironments(ctx, baseDir, opts)
	}

	l, err := Load(ctx, baseDir, opts.Opts)
	if err != nil {
		return nil, err
	}
	kube, err := l.Connect()
	if err != nil {
		return nil, err
	}
	defer kube.Close()

	return kube.Diff(ctx, l.Resources, kubernetes.DiffOpts{
		Summarize: opts.Summarize,
		Strategy:  opts.Strategy,
		WithPrune: opts.WithPrune,
	})
}

// ListChangedEnvironments performs a high-level check using kubectl dry-run to identify environments with changes
func ListChangedEnvironments(ctx context.Context, baseDir string, opts DiffOpts) (*string, error) {
	// Find all environments in the directory
	envMetas, err := FindEnvs(ctx, baseDir, FindOpts{
		JsonnetOpts:           opts.JsonnetOpts,
		JsonnetImplementation: opts.JsonnetImplementation,
		Parallelism:           8, // magic number for now
	})
	if err != nil {
		return nil, err
	}

	changed := CheckEnvironmentsForChanges(ctx, envMetas, opts)
	if len(changed) == 0 {
		return nil, nil
	}

	sort.Strings(changed)

	result := strings.Join(changed, "\n")
	return &result, nil
}

// CheckEnvironmentsForChanges performs a high-level parallel check using kubectl diff --exit-code
func CheckEnvironmentsForChanges(ctx context.Context, envs []*v1alpha1.Environment, opts DiffOpts) []string {
	var mu sync.Mutex
	var changed []string

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(4)

	for _, env := range envs {
		envLoop := env
		g.Go(func() error {
			if hasChanges, envName := checkSingleEnvironmentChanges(ctx, envLoop, opts); hasChanges {
				mu.Lock()
				changed = append(changed, envName)
				mu.Unlock()
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		log.Warn().Err(err).Msg("Failed to check environments for changes")
	}
	return changed
}

// checkSingleEnvironmentChanges uses kubectl diff to quickly check for changes
func checkSingleEnvironmentChanges(ctx context.Context, env *v1alpha1.Environment, opts DiffOpts) (bool, string) {
	envName := env.Spec.Namespace
	if env.Metadata.Name != "" {
		envName = env.Metadata.Name
	}

	// Load only this single environment to get its resources
	tempOpts := opts.Opts
	tempOpts.Name = env.Metadata.Name

	// Use the environment's path for loading
	envPath := env.Metadata.Namespace
	l, err := Load(ctx, envPath, tempOpts)
	if err != nil {
		log.Warn().Err(err).Str("env", envName).Msg("Failed to load environment, assuming no changes")
		return false, envName
	}

	kube, err := l.Connect()
	if err != nil {
		log.Warn().Err(err).Str("env", envName).Msg("Failed to connect, assuming no changes")
		return false, envName
	}
	defer kube.Close()

	// Use a lightweight check via `kubectl diff --exit-code`
	hasChanges, err := kube.HasChanges(l.Resources)
	if err != nil {
		log.Warn().Err(err).Str("env", envName).Msg("Failed to check changes, assuming changes exist")
		return true, envName
	}

	return hasChanges, envName
}

// DeleteOpts specify additional properties for the Delete operation
type DeleteOpts struct {
	ApplyBaseOpts
}

// Delete parses the environment at the given directory (a `baseDir`) and deletes
// the generated objects from the Kubernetes cluster defined in the environment's
// `spec.json`.
func Delete(ctx context.Context, baseDir string, opts DeleteOpts) error {
	l, err := Load(ctx, baseDir, opts.Opts)
	if err != nil {
		return err
	}
	kube, err := l.Connect()
	if err != nil {
		return err
	}
	defer kube.Close()

	if opts.DryRun == "" {
		// show diff
		// static differ will never fail and always return something if input is not nil
		diff, err := kubernetes.StaticDiffer(false)(l.Resources)

		if err != nil {
			fmt.Println("Error diffing:", err)
		}

		// in case of non-fatal error diff may be nil
		if diff != nil {
			b := term.Colordiff(*diff)
			fmt.Print(b.String())
		}
	}

	// prompt for confirmation
	if opts.AutoApprove != AutoApproveAlways && opts.DryRun == "" {
		if err := confirmPrompt("Deleting from", l.Env.Spec.Namespace, kube.Info()); err != nil {
			return err
		}
	}

	return kube.Delete(l.Resources, kubernetes.DeleteOpts{
		Force:  opts.Force,
		DryRun: opts.DryRun,
	})
}

// Show parses the environment at the given directory (a `baseDir`) and returns
// the list of Kubernetes objects.
// Tip: use the `String()` function on the returned list to get the familiar yaml stream
func Show(ctx context.Context, baseDir string, opts Opts) (manifest.List, error) {
	l, err := Load(ctx, baseDir, opts)
	if err != nil {
		return nil, err
	}

	return l.Resources, nil
}
