package helm

import (
	"bytes"
	"io"
	"os"

	"github.com/grafana/tanka/pkg/kubernetes/manifest"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v3"
)

func (e ExecHelm) templateCommandArgs(name, chart string, opts TemplateOpts) []string {
	args := []string{name, chart,
		"--values", "-", // values from stdin
	}
	args = append(args, opts.Flags()...)
	return args
}

// Template expands a Helm Chart into a regular manifest.List using the `helm
// template` command
func (e ExecHelm) Template(name, chart string, opts TemplateOpts) (manifest.List, error) {
	args := e.templateCommandArgs(name, chart, opts)

	cmd := e.cmd("template", args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr

	data, err := yaml.Marshal(opts.Values)
	if err != nil {
		return nil, errors.Wrap(err, "Converting Helm values to YAML")
	}
	cmd.Stdin = bytes.NewReader(data)

	if err := cmd.Run(); err != nil {
		return nil, errors.Wrap(err, "Expanding Helm Chart")
	}

	var list manifest.List
	d := yaml.NewDecoder(&buf)
	for {
		var m manifest.Manifest
		if err := d.Decode(&m); err != nil {
			if err == io.EOF {
				break
			}
			return nil, errors.Wrap(err, "Parsing Helm output")
		}

		// Helm might return "empty" elements in the YAML stream that consist
		// only of comments. Ignore these
		if len(m) == 0 {
			continue
		}

		list = append(list, m)
	}

	return list, nil
}

// TemplateOpts are additional, non-required options for Helm.Template
type TemplateOpts struct {
	// Values to pass to Helm using --values
	Values map[string]interface{}

	// Kubernetes api versions used for Capabilities.APIVersions
	APIVersions []string
	// IncludeCRDs specifies whether CustomResourceDefinitions are included in
	// the template output
	IncludeCRDs bool
	// skip tests from templated output
	SkipTests bool
	// Kubernetes version used for Capabilities.KubeVersion
	KubeVersion string
	// Namespace scope for this request
	Namespace string
	// NoHooks specifies whether hooks should be excluded from the template output
	NoHooks bool
}

// Flags returns all options apart from Values as their respective `helm
// template` flag equivalent
func (t TemplateOpts) Flags() []string {
	var flags []string

	for _, value := range t.APIVersions {
		flags = append(flags, "--api-versions="+value)
	}

	if t.IncludeCRDs {
		flags = append(flags, "--include-crds")
	}

	if t.SkipTests {
		flags = append(flags, "--skip-tests")
	}

	if t.KubeVersion != "" {
		flags = append(flags, "--kube-version="+t.KubeVersion)
	}

	if t.NoHooks {
		flags = append(flags, "--no-hooks")
	}

	if t.Namespace != "" {
		flags = append(flags, "--namespace="+t.Namespace)
	}

	return flags
}
