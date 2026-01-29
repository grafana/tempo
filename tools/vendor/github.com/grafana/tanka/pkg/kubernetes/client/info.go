package client

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
)

// Info contains metadata about the client and its environment
type Info struct {
	// versions
	ClientVersion *semver.Version
	ServerVersion *semver.Version

	// kubeconfig (chosen context + cluster)
	Kubeconfig Config
}

// Config represents a single KUBECONFIG entry (single context + cluster)
// Omits the arrays of the original schema, to ease using.
type Config struct {
	Cluster Cluster `json:"cluster"`
	Context Context `json:"context"`
}

// Context is a kubectl context
type Context struct {
	Context struct {
		Cluster   string `json:"cluster"`
		User      string `json:"user"`
		Namespace string `json:"namespace"`
	} `json:"context"`
	Name string `json:"name"`
}

// Cluster is a kubectl cluster
type Cluster struct {
	Cluster struct {
		Server string `json:"server"`
	} `json:"cluster"`
	Name string `json:"name"`
}

// Version returns the version of kubectl and the Kubernetes api server
func (k Kubectl) version() (client, server *semver.Version, err error) {
	cmd := k.ctl("version", "-o", "json")

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, nil, err
	}

	// parse the result
	type ver struct {
		GitVersion string `json:"gitVersion"`
	}
	var got struct {
		ClientVersion ver `json:"clientVersion"`
		ServerVersion ver `json:"serverVersion"`
	}

	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		return nil, nil, err
	}

	// parse the versions
	client, err = semver.NewVersion(got.ClientVersion.GitVersion)
	if err != nil {
		return nil, nil, errors.Wrap(err, "client version")
	}

	server, err = semver.NewVersion(got.ServerVersion.GitVersion)
	if err != nil {
		return nil, nil, errors.Wrap(err, "server version")
	}

	return client, server, nil
}
