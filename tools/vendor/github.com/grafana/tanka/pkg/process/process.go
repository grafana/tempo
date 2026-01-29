package process

import (
	"errors"
	"fmt"

	"github.com/grafana/tanka/pkg/kubernetes/manifest"
	"github.com/grafana/tanka/pkg/spec/v1alpha1"
)

const (
	MetadataPrefix   = "tanka.dev"
	LabelEnvironment = MetadataPrefix + "/environment"
)

// Process converts the raw Jsonnet evaluation result (JSON tree) into a flat
// list of Kubernetes objects, also applying some transformations:
// - tanka.dev/** labels
// - filtering
// - best-effort sorting
func Process(cfg v1alpha1.Environment, exprs Matchers) (manifest.List, error) {
	raw := cfg.Data

	if raw == nil {
		return manifest.List{}, nil
	}

	// Scan for everything that looks like a Kubernetes object
	extracted, err := Extract(raw)
	if err != nil {
		return nil, fmt.Errorf("got an error while extracting env `%s`: %w", cfg.Metadata.Name, err)
	}

	// Unwrap *List types
	if err := Unwrap(extracted); err != nil {
		return nil, err
	}

	out := make(manifest.List, 0, len(extracted))
	for _, m := range extracted {
		out = append(out, m)
	}

	// set default namespace
	out = Namespace(out, cfg.Spec.Namespace)

	// tanka.dev/** labels
	out, err = Label(out, cfg)
	if err != nil {
		return nil, err
	}

	// arbitrary labels and annotations from spec
	out = ResourceDefaults(out, cfg)

	// Perhaps filter for kind/name expressions
	if len(exprs) > 0 {
		out = Filter(out, exprs)
	}

	// Best-effort dependency sort
	Sort(out)

	return out, nil
}

// Label conditionally adds tanka.dev/** labels to each manifest in the List
func Label(list manifest.List, cfg v1alpha1.Environment) (manifest.List, error) {
	for i, m := range list {
		// inject tanka.dev/environment label
		if cfg.Spec.InjectLabels {
			label, err := cfg.NameLabel()
			if err != nil {
				return nil, fmt.Errorf("failed to get name label: %w", err)
			}

			m.Metadata().Labels()[LabelEnvironment] = label
		}
		list[i] = m
	}

	return list, nil
}

func ResourceDefaults(list manifest.List, cfg v1alpha1.Environment) manifest.List {
	for i, m := range list {
		for k, v := range cfg.Spec.ResourceDefaults.Annotations {
			annotations := m.Metadata().Annotations()
			if _, ok := annotations[k]; !ok {
				annotations[k] = v
			}
		}

		for k, v := range cfg.Spec.ResourceDefaults.Labels {
			labels := m.Metadata().Labels()
			if _, ok := labels[k]; !ok {
				labels[k] = v
			}
		}

		list[i] = m
	}
	return list
}

// Unwrap returns all Kubernetes objects in the manifest. If m is not a List
// type, a one item List is returned
func Unwrap(manifests map[string]manifest.Manifest) error {
	for path, m := range manifests {
		if !m.IsList() {
			continue
		}

		items, err := m.Items()
		if err != nil {
			return err
		}

		for index, i := range items {
			name := fmt.Sprintf("%s.items[%v]", path, index)

			var e *manifest.SchemaError
			if errors.As(i.Verify(), &e) {
				e.Name = name
				return e
			}

			manifests[name] = i
		}

		delete(manifests, path)
	}

	return nil
}
