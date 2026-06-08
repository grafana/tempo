package tanka

import (
	"context"

	"github.com/grafana/tanka/pkg/kubernetes/client"
	"github.com/grafana/tanka/pkg/kubernetes/manifest"
	"github.com/grafana/tanka/pkg/spec/v1alpha1"
)

// Info holds information about a particular environment, including its Config,
// the individual resources of the desired state and also the status of the
// client.
type Info struct {
	Env       *v1alpha1.Environment
	Resources manifest.List
	Client    client.Info
}

// Status returns information about the particular environment
func Status(ctx context.Context, baseDir string, opts Opts) (*Info, error) {
	r, err := Load(ctx, baseDir, opts)
	if err != nil {
		return nil, err
	}
	kube, err := r.Connect()
	if err != nil {
		return nil, err
	}

	r.Env.Spec.DiffStrategy = kube.Env.Spec.DiffStrategy

	return &Info{
		Env:       r.Env,
		Resources: r.Resources,
		Client:    kube.Info(),
	}, nil
}
