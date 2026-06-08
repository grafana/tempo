package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/stretchr/objx"
	funk "github.com/thoas/go-funk"
)

// findContextFromEndpoint returns a valid context from $KUBECONFIG that uses the given
// apiServer endpoint.
func findContextFromEndpoint(endpoint string) (Config, error) {
	cluster, context, err := ContextFromIP(endpoint)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Context: *context,
		Cluster: *cluster,
	}, nil
}

// findContextFromNames will try to match a context name from names
func findContextFromNames(names []string) (Config, error) {
	for _, name := range names {
		cluster, context, err := ContextFromName(name)

		if _, ok := err.(ErrorNoContext); ok {
			continue
		} else if err != nil {
			return Config{}, err
		}
		return Config{
			Context: *context,
			Cluster: *cluster,
		}, nil
	}
	return Config{}, ErrorNoContext(fmt.Sprintf("%v", names))
}

func ContextFromName(contextName string) (*Cluster, *Context, error) {
	cfg, err := Kubeconfig()
	if err != nil {
		return nil, nil, err
	}

	// find the context by name
	var context Context
	contexts, err := tryMSISlice(cfg.Get("contexts"), "contexts")
	if err != nil {
		return nil, nil, err
	}

	err = find(contexts, "name", fmt.Sprintf("^%s$", contextName), &context)
	if err == ErrorNoMatch {
		return nil, nil, ErrorNoContext(contextName)
	} else if err != nil {
		return nil, nil, err
	}
	var cluster Cluster
	clusters, err := tryMSISlice(cfg.Get("clusters"), "clusters")
	if err != nil {
		return nil, nil, err
	}

	err = find(clusters, "name", fmt.Sprintf("^%s$", context.Context.Cluster), &cluster)
	if err == ErrorNoMatch {
		return nil, nil, ErrorNoCluster(contextName)
	} else if err != nil {
		return nil, nil, err
	}

	return &cluster, &context, nil
}

// Kubeconfig returns the merged $KUBECONFIG of the host
func Kubeconfig() (objx.Map, error) {
	cmd := kubectlCmd("config", "view", "-o", "json")
	cfgJSON := bytes.Buffer{}
	cmd.Stdout = &cfgJSON
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return objx.FromJSON(cfgJSON.String())
}

// Contexts returns a list of context names
func Contexts() ([]string, error) {
	cmd := kubectlCmd("config", "get-contexts", "-o=name")
	buf := bytes.Buffer{}
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return strings.Split(buf.String(), "\n"), nil
}

// ContextFromIP searches the $KUBECONFIG for a context using a cluster that matches the apiServer
func ContextFromIP(apiServer string) (*Cluster, *Context, error) {
	cfg, err := Kubeconfig()
	if err != nil {
		return nil, nil, err
	}

	// find the correct cluster
	var cluster Cluster
	clusters, err := tryMSISlice(cfg.Get("clusters"), "clusters")
	if err != nil {
		return nil, nil, err
	}

	err = find(clusters, "cluster.server", apiServer, &cluster)
	if err == ErrorNoMatch {
		return nil, nil, ErrorNoCluster(apiServer)
	} else if err != nil {
		return nil, nil, err
	}

	// find a context that uses the cluster
	var context Context
	contexts, err := tryMSISlice(cfg.Get("contexts"), "contexts")
	if err != nil {
		return nil, nil, err
	}

	// find the context that uses the cluster, it should be an exact match
	err = find(contexts, "context.cluster", fmt.Sprintf("^%s$", cluster.Name), &context)
	if err == ErrorNoMatch {
		return nil, nil, ErrorNoContext(cluster.Name)
	} else if err != nil {
		return nil, nil, err
	}

	return &cluster, &context, nil
}

// IPFromContext parses $KUBECONFIG, finds the cluster with the given name and
// returns the cluster's endpoint
func IPFromContext(name string) (ip string, err error) {
	cfg, err := Kubeconfig()
	if err != nil {
		return "", err
	}

	// find a context with the given name
	var context Context
	contexts, err := tryMSISlice(cfg.Get("contexts"), "contexts")
	if err != nil {
		return "", err
	}

	err = find(contexts, "name", fmt.Sprintf("^%s$", name), &context)
	if err == ErrorNoMatch {
		return "", ErrorNoContext(name)
	} else if err != nil {
		return "", err
	}

	// find the cluster of the context
	var cluster Cluster
	clusters, err := tryMSISlice(cfg.Get("clusters"), "clusters")
	if err != nil {
		return "", err
	}

	clusterName := context.Context.Cluster
	err = find(clusters, "name", fmt.Sprintf("^%s$", clusterName), &cluster)
	if err == ErrorNoMatch {
		return "", fmt.Errorf("no cluster named `%s` as required by context `%s` was found. Please check your $KUBECONFIG", clusterName, name)
	} else if err != nil {
		return "", err
	}

	return cluster.Cluster.Server, nil
}

func tryMSISlice(v *objx.Value, what string) ([]map[string]interface{}, error) {
	if s := v.MSISlice(); s != nil {
		return s, nil
	}

	data, ok := v.Data().([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected %s to be of type `[]map[string]interface{}`, but got `%T` instead", what, v.Data())
	}
	return data, nil
}

// ErrorNoMatch occurs when no item matched had the expected value
var ErrorNoMatch = errors.New("no matches found")

// find attempts to find an object in list whose prop equals expected.
// If found, the value is unmarshalled to ptr, otherwise errNotFound is returned.
func find(list []map[string]interface{}, prop string, expected string, ptr interface{}) error {
	var findErr error
	i := funk.Find(list, func(x map[string]interface{}) bool {
		if findErr != nil {
			return false
		}

		got := objx.New(x).Get(prop).Data()
		str, ok := got.(string)
		if !ok {
			findErr = fmt.Errorf("testing whether `%s` is `%s`: unable to parse `%v` as string", prop, expected, got)
			return false
		}
		return regexp.MustCompile(expected).MatchString(str)
	})
	if findErr != nil {
		return findErr
	}

	if i == nil {
		return ErrorNoMatch
	}

	o := objx.New(i).MustJSON()
	return json.Unmarshal([]byte(o), ptr)
}
