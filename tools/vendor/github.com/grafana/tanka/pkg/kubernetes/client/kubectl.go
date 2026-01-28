package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/pkg/errors"

	"github.com/grafana/tanka/pkg/kubernetes/manifest"
)

// Kubectl uses the `kubectl` command to operate on a Kubernetes cluster
type Kubectl struct {
	info Info
}

// New returns a instance of Kubectl with a correct context already discovered.
func New(endpoint string) (*Kubectl, error) {
	k := Kubectl{}

	// discover context
	var err error
	k.info.Kubeconfig, err = findContextFromEndpoint(endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "finding usable context")
	}

	// query versions (requires context)
	k.info.ClientVersion, k.info.ServerVersion, err = k.version()
	if err != nil {
		return nil, errors.Wrap(err, "obtaining versions")
	}

	return &k, nil
}

func NewFromNames(names []string) (*Kubectl, error) {
	k := Kubectl{}

	var err error
	k.info.Kubeconfig, err = findContextFromNames(names)
	if err != nil {
		return nil, errors.Wrap(err, "finding usable context")
	}

	// query versions (requires context)
	k.info.ClientVersion, k.info.ServerVersion, err = k.version()
	if err != nil {
		return nil, errors.Wrap(err, "obtaining versions")
	}

	return &k, nil
}

// Info returns known informational data about the client and its environment
func (k Kubectl) Info() Info {
	return k.info
}

// Close runs final cleanup:
func (k Kubectl) Close() error {
	return nil
}

// Namespaces of the cluster
func (k Kubectl) Namespaces() (map[string]bool, error) {
	cmd := k.ctl("get", "namespaces", "-o", "json")

	var sout bytes.Buffer
	var serr bytes.Buffer
	cmd.Stdout = &sout
	cmd.Stderr = &serr

	err := cmd.Run()
	if err != nil {
		return nil, errors.Wrap(err, serr.String())
	}

	var list manifest.Manifest
	if err := json.Unmarshal(sout.Bytes(), &list); err != nil {
		return nil, err
	}

	items, err := list.Items()
	if err != nil {
		return nil, err
	}

	namespaces := make(map[string]bool)
	for _, m := range items {
		namespaces[m.Metadata().Name()] = true
	}
	return namespaces, nil
}

type ErrNamespaceNotFound struct {
	Namespace string
}

func (e ErrNamespaceNotFound) Error() string {
	return fmt.Sprintf("Namespace not found: %s", e.Namespace)
}

// Namespace finds a single namespace in the cluster
func (k Kubectl) Namespace(namespace string) (manifest.Manifest, error) {
	cmd := k.ctl("get", "namespaces", namespace, "-o", "json", "--ignore-not-found")

	var sout bytes.Buffer
	cmd.Stdout = &sout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	if len(sout.Bytes()) == 0 {
		return nil, ErrNamespaceNotFound{
			Namespace: namespace,
		}
	}
	var ns manifest.Manifest
	if err := json.Unmarshal(sout.Bytes(), &ns); err != nil {
		return nil, err
	}
	return ns, nil
}

// FilterWriter is an io.Writer that discards every message that matches at
// least one of the regular expressions.
type FilterWriter struct {
	buf     string
	filters []*regexp.Regexp
}

func (r *FilterWriter) Write(p []byte) (n int, err error) {
	for _, re := range r.filters {
		if re.Match(p) {
			// silently discard
			return len(p), nil
		}
	}
	r.buf += string(p)
	return os.Stderr.Write(p)
}
