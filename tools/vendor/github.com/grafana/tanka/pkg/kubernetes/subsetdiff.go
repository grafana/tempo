package kubernetes

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/grafana/tanka/pkg/kubernetes/client"
	"github.com/grafana/tanka/pkg/kubernetes/manifest"
	"github.com/grafana/tanka/pkg/kubernetes/util"
)

type difference struct {
	name         string
	live, merged string
}

// SubsetDiffer returns a implementation of Differ that computes the diff by
// comparing only the fields present in the desired state. This algorithm might
// miss information, but is all that's possible on cluster versions lower than
// 1.13.
func SubsetDiffer(c client.Client) Differ {
	return func(state manifest.List) (*string, error) {
		docs := []difference{}

		errCh := make(chan error)
		resultCh := make(chan difference)

		for _, rawShould := range state {
			go parallelSubsetDiff(c, rawShould, resultCh, errCh)
		}

		var lastErr error
		for i := 0; i < len(state); i++ {
			select {
			case d := <-resultCh:
				docs = append(docs, d)
			case err := <-errCh:
				lastErr = err
			}
		}
		close(resultCh)
		close(errCh)

		if lastErr != nil {
			return nil, errors.Wrap(lastErr, "calculating subset")
		}

		var diffs string
		for _, d := range docs {
			diffStr, err := util.DiffStr(d.name, d.live, d.merged)
			if err != nil {
				return nil, errors.Wrap(err, "invoking diff")
			}
			if diffStr != "" {
				diffStr += "\n"
			}
			diffs += diffStr
		}
		diffs = strings.TrimSuffix(diffs, "\n")

		if diffs == "" {
			return nil, nil
		}

		return &diffs, nil
	}
}

func parallelSubsetDiff(c client.Client, should manifest.Manifest, r chan difference, e chan error) {
	diff, err := subsetDiff(c, should)
	if err != nil {
		e <- err
		return
	}
	r <- *diff
}

func subsetDiff(c client.Client, m manifest.Manifest) (*difference, error) {
	name := util.DiffName(m)

	// kubectl output -> current state
	rawIs, err := c.Get(
		m.Metadata().Namespace(),
		m.Kind(),
		m.Metadata().Name(),
	)

	if _, ok := err.(client.ErrorNotFound); ok {
		rawIs = map[string]interface{}{}
	} else if err != nil {
		return nil, errors.Wrap(err, "getting state from cluster")
	}

	should := m.String()

	sub := subset(m, rawIs)
	is := manifest.Manifest(sub).String()
	if is == "{}\n" {
		is = ""
	}

	return &difference{
		name:   name,
		live:   is,
		merged: should,
	}, nil
}

// subset removes all keys from big, that are not present in small.
// It makes big a subset of small.
// Kubernetes returns more keys than we can know about.
// This means, we need to remove all keys from the kubectl output, that are not present locally.
func subset(small, big map[string]interface{}) map[string]interface{} {
	if small["namespace"] != nil {
		big["namespace"] = small["namespace"]
	}

	// just ignore the apiVersion for now, too much bloat
	if small["apiVersion"] != nil && big["apiVersion"] != nil {
		big["apiVersion"] = small["apiVersion"]
	}

	for k, v := range big {
		if _, ok := small[k]; !ok {
			delete(big, k)
			continue
		}

		switch b := v.(type) {
		case map[string]interface{}:
			if a, ok := small[k].(map[string]interface{}); ok {
				big[k] = subset(a, b)
			}
		case []map[string]interface{}:
			for i := range b {
				if a, ok := small[k].([]map[string]interface{}); ok {
					b[i] = subset(a[i], b[i])
				}
			}
		case []interface{}:
			for i := range b {
				if a, ok := small[k].([]interface{}); ok {
					if i >= len(a) {
						// slice in config shorter than in live. Abort, as there are no entries to diff anymore
						break
					}

					// value not a dict, no recursion needed
					cShould, ok := a[i].(map[string]interface{})
					if !ok {
						continue
					}

					// value not a dict, no recursion needed
					cIs, ok := b[i].(map[string]interface{})
					if !ok {
						continue
					}
					b[i] = subset(cShould, cIs)
				}
			}
		}
	}
	return big
}
