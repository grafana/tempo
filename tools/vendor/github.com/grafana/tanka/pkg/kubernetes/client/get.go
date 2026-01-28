package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/semver"

	"github.com/grafana/tanka/pkg/kubernetes/manifest"
)

// Get retrieves a single Kubernetes object from the cluster
func (k Kubectl) Get(namespace, kind, name string) (manifest.Manifest, error) {
	return k.get(namespace, kind, []string{name}, getOpts{})
}

// GetByLabels retrieves all objects matched by the given labels from the cluster.
// Set namespace to empty string for --all-namespaces
func (k Kubectl) GetByLabels(namespace, kind string, labels map[string]string) (manifest.List, error) {
	lArgs := make([]string, 0, len(labels))
	for k, v := range labels {
		lArgs = append(lArgs, fmt.Sprintf("-l=%s=%s", k, v))
	}
	// Needed to properly filter for resources that should be pruned
	if k.info.ClientVersion.GreaterThan(semver.MustParse("1.21.0")) {
		lArgs = append(lArgs, "--show-managed-fields")
	}
	var opts getOpts
	if namespace == "" {
		opts.allNamespaces = true
	}
	list, err := k.get(namespace, kind, lArgs, opts)
	if err != nil {
		return nil, err
	}

	return unwrapList(list)
}

// GetByState returns the full object, including runtime fields for each
// resource in the state
func (k Kubectl) GetByState(data manifest.List, opts GetByStateOpts) (manifest.List, error) {
	list, err := k.get("", "", []string{"-f", "-"}, getOpts{
		ignoreNotFound: opts.IgnoreNotFound,
		stdin:          data.String(),
	})
	if err != nil {
		return nil, err
	}

	return unwrapList(list)
}

type getOpts struct {
	allNamespaces  bool
	ignoreNotFound bool
	stdin          string
}

func (k Kubectl) get(namespace, kind string, selector []string, opts getOpts) (manifest.Manifest, error) {
	// build cli flags and args
	argv := []string{
		"-o", "json",
	}
	if opts.ignoreNotFound {
		argv = append(argv, "--ignore-not-found")
	}

	if opts.allNamespaces {
		argv = append(argv, "--all-namespaces")
	} else if namespace != "" {
		argv = append(argv, "-n", namespace)
	}

	if kind != "" {
		argv = append(argv, kind)
	}

	argv = append(argv, selector...)

	// setup command environment
	cmd := k.ctl("get", argv...)
	var sout, serr bytes.Buffer
	cmd.Stdout = &sout
	cmd.Stderr = &serr
	if opts.stdin != "" {
		cmd.Stdin = strings.NewReader(opts.stdin)
	}

	// run command
	if err := cmd.Run(); err != nil {
		return nil, parseGetErr(err, serr.String())
	}

	// return error if nothing was returned
	// because parsing empty output as json would cause errors
	if sout.Len() == 0 {
		return nil, ErrorNothingReturned{}
	}

	// parse result
	var m manifest.Manifest
	if err := json.Unmarshal(sout.Bytes(), &m); err != nil {
		return nil, err
	}

	return m, nil
}

func parseGetErr(err error, stderr string) error {
	if strings.HasPrefix(stderr, "Error from server (NotFound)") {
		return ErrorNotFound{stderr}
	}
	if strings.HasPrefix(stderr, "error: the server doesn't have a resource type") {
		return ErrorUnknownResource{stderr}
	}

	return errors.New(strings.TrimPrefix(fmt.Sprintf("%s\n%s", stderr, err), "\n"))
}

func unwrapList(list manifest.Manifest) (manifest.List, error) {
	if list.Kind() != "List" {
		return nil, fmt.Errorf("expected kind `List` but got `%s` instead", list.Kind())
	}

	items := list["items"].([]interface{})
	ms := make(manifest.List, 0, len(items))
	for _, i := range items {
		ms = append(ms, manifest.Manifest(i.(map[string]interface{})))
	}

	return ms, nil
}
