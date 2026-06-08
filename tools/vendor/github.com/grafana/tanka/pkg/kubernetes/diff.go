package kubernetes

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"

	"github.com/grafana/tanka/pkg/kubernetes/client"
	"github.com/grafana/tanka/pkg/kubernetes/manifest"
	"github.com/grafana/tanka/pkg/kubernetes/util"
)

// Diff takes the desired state and returns the differences from the cluster
func (k *Kubernetes) Diff(ctx context.Context, state manifest.List, opts DiffOpts) (*string, error) {
	_, span := tracer.Start(ctx, "kubernetes.Diff")
	span.End()
	// prevent https://github.com/kubernetes/kubernetes/issues/89762 until fixed
	if k.ctl.Info().ClientVersion.Equal(semver.MustParse("1.18.0")) {
		return nil, fmt.Errorf(`you seem to be using kubectl 1.18.0, which contains an unfixed issue
that makes 'kubectl diff' modify resources in your cluster.
Please upgrade kubectl to at least version 1.18.1`)
	}

	// required for separating
	namespaces, err := k.ctl.Namespaces()
	if err != nil {
		resourceNamespaces := state.Namespaces()
		namespaces = map[string]bool{}
		for _, namespace := range resourceNamespaces {
			_, err = k.ctl.Namespace(namespace)
			if err != nil {
				if errors.As(err, &client.ErrNamespaceNotFound{}) {
					continue
				}
				return nil, errors.Wrap(err, "retrieving namespaces")
			}
			namespaces[namespace] = true
		}
	}
	resources, err := k.ctl.Resources()
	if err != nil {
		return nil, errors.Wrap(err, "listing known api-resources")
	}

	// separate resources in groups
	//
	// soon: resources that have unmet dependencies that will be met during
	// apply. These will be diffed statically, because checking with the cluster
	// would cause an error
	//
	// live: all other resources
	live, soon := separate(state, k.Env.Spec.Namespace, separateOpts{
		namespaces: namespaces,
		resources:  resources,
	})

	// differ for live resources
	liveDiff, err := k.differ(opts.Strategy)
	if err != nil {
		return nil, err
	}

	// reports all resources as created
	staticDiffAllCreated := StaticDiffer(true)

	// reports all resources as deleted
	staticDiffAllDeleted := StaticDiffer(false)

	// include orphaned resources in the diff if it was requested by the user
	orphaned := manifest.List{}
	if opts.WithPrune {
		// find orphaned resources
		orphaned, err = k.Orphaned(state)
		if err != nil {
			return nil, err
		}
	}

	// run the diff
	d, err := multiDiff{
		{differ: liveDiff, state: live},
		{differ: staticDiffAllCreated, state: soon},
		{differ: staticDiffAllDeleted, state: orphaned},
	}.diff()

	switch {
	case err != nil:
		return nil, err
	case d == nil:
		return nil, nil
	}

	if opts.Summarize {
		result, err := util.DiffStat(*d)
		return &result, err
	}

	return d, nil
}

// HasChanges performs a lightweight check to determine if there are any changes
// between the desired state and cluster using kubectl diff --exit-code (no output)
func (k *Kubernetes) HasChanges(state manifest.List) (bool, error) {
	return k.ctl.DiffExitCode(state)
}

type separateOpts struct {
	namespaces map[string]bool
	resources  client.Resources
}

func separate(state manifest.List, defaultNs string, opts separateOpts) (live manifest.List, soon manifest.List) {
	soonNs := make(map[string]bool)
	for _, m := range state {
		if m.Kind() != "Namespace" {
			continue
		}
		soonNs[m.Metadata().Name()] = true
	}

	for _, m := range state {
		// non-namespaced always live
		if !opts.resources.Namespaced(m) {
			live = append(live, m)
			continue
		}

		// handle implicit default
		ns := m.Metadata().Namespace()
		if ns == "" {
			ns = defaultNs
		}

		// special case: namespace missing, BUT will be created during apply
		if !opts.namespaces[ns] && soonNs[ns] {
			soon = append(soon, m)
			continue
		}

		// everything else
		live = append(live, m)
	}

	return live, soon
}

// ErrorDiffStrategyUnknown occurs when a diff-strategy is requested that does
// not exist.
type ErrorDiffStrategyUnknown struct {
	Requested string
	differs   map[string]Differ
}

func (e ErrorDiffStrategyUnknown) Error() string {
	strats := []string{}
	for s := range e.differs {
		strats = append(strats, s)
	}
	return fmt.Sprintf("diff strategy `%s` does not exist. Pick one of: %v", e.Requested, strats)
}

func (k *Kubernetes) differ(override string) (Differ, error) {
	strategy := k.Env.Spec.DiffStrategy
	if override != "" {
		strategy = override
	}

	d, ok := k.differs[strategy]
	if !ok {
		return nil, ErrorDiffStrategyUnknown{
			Requested: strategy,
			differs:   k.differs,
		}
	}

	return d, nil
}

// StaticDiffer returns a differ that reports all resources as either created or
// deleted.
func StaticDiffer(create bool) Differ {
	return func(state manifest.List) (*string, error) {
		s := ""
		for _, m := range state {
			is, should := m.String(), ""
			if create {
				is, should = should, is
			}

			d, err := util.DiffStr(util.DiffName(m), is, should)
			if err != nil {
				return nil, err
			}
			s += d
		}

		if s == "" {
			return nil, nil
		}

		return &s, nil
	}
}

// multiDiff runs multiple differs (in series). In the future it might be worth
// parallelizing this.
type multiDiff []struct {
	differ Differ
	state  manifest.List
}

func (m multiDiff) diff() (*string, error) {
	diff := ""
	for _, d := range m {
		s, err := d.differ(d.state)
		if err != nil {
			return nil, err
		}

		if s == nil {
			continue
		}
		diff += *s
	}

	if diff == "" {
		return nil, nil
	}
	return &diff, nil
}
