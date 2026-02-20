package client

import (
	"github.com/grafana/tanka/pkg/kubernetes/manifest"
)

// Client for working with Kubernetes
type Client interface {
	// Get the specified object(s) from the cluster
	Get(namespace, kind, name string) (manifest.Manifest, error)
	GetByLabels(namespace, kind string, labels map[string]string) (manifest.List, error)
	GetByState(data manifest.List, opts GetByStateOpts) (manifest.List, error)

	// Apply the configuration to the cluster. `data` must contain a plaintext
	// format that is `kubectl-apply(1)` compatible
	Apply(data manifest.List, opts ApplyOpts) error

	// DiffServerSide runs the diff operation on the server and returns the
	// result in `diff(1)` format
	DiffServerSide(data manifest.List) (*string, error)

	// DiffExitCode performs kubectl diff and returns true if there are changes, false if no changes
	DiffExitCode(data manifest.List) (bool, error)

	// Delete the specified object(s) from the cluster
	Delete(namespace, apiVersion, kind, name string, opts DeleteOpts) error

	// Namespaces the cluster currently has
	Namespaces() (map[string]bool, error)

	// Namespace retrieves a namespace from the cluster
	Namespace(namespace string) (manifest.Manifest, error)

	// Resources returns all known api-resources of the cluster
	Resources() (Resources, error)

	// Info returns known informational data about the client. Best effort based,
	// fields of `Info` that cannot be stocked with valuable data, e.g.
	// due to an error, shall be left nil.
	Info() Info

	// Close may run tasks once the client is no longer needed.
	Close() error
}

// ApplyOpts allow to specify additional parameter for apply operations
type ApplyOpts struct {
	// force allows to ignore checks and force the operation
	Force bool

	// validate allows to enable/disable kubectl validation
	Validate bool

	// autoApprove allows to skip the interactive approval
	AutoApprove bool

	// DryRun string passed to kubectl as --dry-run=<DryRun>
	DryRun string

	// ApplyStrategy to pick a final method for deploying generated objects
	ApplyStrategy string
}

// DeleteOpts allow to specify additional parameters for delete operations
// Currently not different from ApplyOpts, but may be required in the future
type DeleteOpts ApplyOpts

// GetByStateOpts allow to specify additional parameters for GetByState function
// Currently there is just ignoreNotFound parameter which is only useful for
// GetByState() so we only have GetByStateOpts instead of more generic GetOpts
// for all get operations
type GetByStateOpts struct {
	// ignoreNotFound allows to ignore errors caused by missing objects
	IgnoreNotFound bool
}
