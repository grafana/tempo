package spec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"

	"github.com/grafana/tanka/pkg/jsonnet/jpath"
	"github.com/grafana/tanka/pkg/spec/v1alpha1"
)

// APIGroup is the prefix used for `kind`
const APIGroup = "tanka.dev"

// Specfile is the filename for the environment config
const Specfile = "spec.json"

// ParseDir parses the given environments `spec.json` into a `v1alpha1.Environment`
// object with the name set to the directories name
func ParseDir(path string) (*v1alpha1.Environment, error) {
	root, base, err := jpath.Dirs(path)
	if err != nil {
		return nil, err
	}

	// name of the environment: relative path from rootDir
	name, err := filepath.Rel(root, base)
	if err != nil {
		return nil, err
	}

	file, err := jpath.Entrypoint(path)
	if err != nil {
		return nil, err
	}

	namespace, err := filepath.Rel(root, file)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Join(base, Specfile))
	if err != nil {
		if os.IsNotExist(err) {
			c := v1alpha1.New()
			c.Metadata.Name = name // legacy behavior
			c.Metadata.Namespace = namespace
			return c, ErrNoSpec{path}
		}
		return nil, err
	}

	c, err := Parse(data, namespace)
	if c != nil {
		// set the name field
		c.Metadata.Name = name // legacy behavior
	}

	return c, err
}

// Parse parses the json `data` into a `v1alpha1.Environment` object.
func Parse(data []byte, namespace string) (*v1alpha1.Environment, error) {
	config := v1alpha1.New()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, errors.Wrap(err, "parsing spec.json")
	}

	if err := handleDeprecated(config, data); err != nil {
		return config, err
	}

	// default apiServer URL to https
	if config.Spec.APIServer != "" && !regexp.MustCompile("^.+://").MatchString(config.Spec.APIServer) {
		config.Spec.APIServer = "https://" + config.Spec.APIServer
	}

	config.Metadata.Namespace = namespace

	return config, nil
}

func handleDeprecated(c *v1alpha1.Environment, data []byte) error {
	var errDepr ErrDeprecated

	var msi map[string]interface{}
	if err := json.Unmarshal(data, &msi); err != nil {
		return err
	}

	// namespace -> spec.namespace
	if n, ok := msi["namespace"]; ok && c.Spec.Namespace == "" {
		n, ok := n.(string)
		if !ok {
			return ErrMistypedField{"namespace", n}
		}

		errDepr = append(errDepr, depreciation{"namespace", "spec.namespace"})
		c.Spec.Namespace = n
	}

	// server -> spec.apiServer
	if s, ok := msi["server"]; ok && c.Spec.APIServer == "" {
		s, ok := s.(string)
		if !ok {
			return ErrMistypedField{"server", s}
		}

		errDepr = append(errDepr, depreciation{"server", "spec.apiServer"})
		c.Spec.APIServer = s
	}

	// team -> metadata.labels.team
	_, hasTeam := c.Metadata.Labels["team"]
	if t, ok := msi["team"]; ok && !hasTeam {
		t, ok := t.(string)
		if !ok {
			return ErrMistypedField{"team", t}
		}

		errDepr = append(errDepr, depreciation{"team", "metadata.labels.team"})
		c.Metadata.Labels["team"] = t
	}

	if len(errDepr) != 0 {
		return errDepr
	}

	return nil
}
