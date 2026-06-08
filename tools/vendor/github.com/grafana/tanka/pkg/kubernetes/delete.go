package kubernetes

import (
	"github.com/grafana/tanka/pkg/kubernetes/client"
	"github.com/grafana/tanka/pkg/kubernetes/manifest"
	"github.com/grafana/tanka/pkg/process"
)

type DeleteOpts client.DeleteOpts

func (k *Kubernetes) Delete(state manifest.List, opts DeleteOpts) error {
	// Sort and reverse the manifests to avoid cascading deletions
	process.Sort(state)
	for i := 0; i < len(state)/2; i++ {
		state[i], state[len(state)-1-i] = state[len(state)-1-i], state[i]
	}

	for _, m := range state {
		if err := k.ctl.Delete(m.Metadata().Namespace(), m.APIVersion(), m.Kind(), m.Metadata().Name(), client.DeleteOpts(opts)); err != nil {
			return err
		}
	}

	return nil
}
