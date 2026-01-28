package tanka

import (
	"encoding/json"

	"github.com/grafana/tanka/pkg/jsonnet/implementations/types"
	"github.com/grafana/tanka/pkg/spec"
	"github.com/grafana/tanka/pkg/spec/v1alpha1"
	"github.com/rs/zerolog/log"
)

// StaticLoader loads an environment from a static file called `spec.json`.
// Jsonnet is evaluated as normal
type StaticLoader struct {
	jsonnetImpl types.JsonnetImplementation
}

func (s StaticLoader) Load(path string, opts LoaderOpts) (*v1alpha1.Environment, error) {
	config, err := s.Peek(path, opts)
	if err != nil {
		return nil, err
	}

	data, err := s.Eval(path, opts)
	if err != nil {
		return nil, err
	}
	config.Data = data

	return config, nil
}

func (s StaticLoader) Peek(path string, _ LoaderOpts) (*v1alpha1.Environment, error) {
	config, err := parseStaticSpec(path)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (s StaticLoader) List(path string, opts LoaderOpts) ([]*v1alpha1.Environment, error) {
	env, err := s.Peek(path, opts)
	if err != nil {
		return nil, err
	}

	return []*v1alpha1.Environment{env}, nil
}

func (s *StaticLoader) Eval(path string, opts LoaderOpts) (interface{}, error) {
	config, err := s.Peek(path, opts)
	if err != nil {
		return nil, err
	}

	envCode, err := specToExtCode(config)
	if err != nil {
		return nil, err
	}
	opts.ExtCode.Set(environmentExtCode, envCode)

	raw, err := evalJsonnet(path, s.jsonnetImpl, opts.JsonnetOpts)
	if err != nil {
		return nil, err
	}

	var data interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, err
	}

	return data, nil
}

func specToExtCode(spec *v1alpha1.Environment) (string, error) {
	spec.Data = nil
	data, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// parseStaticSpec parses the `spec.json` of the environment and returns a
// *kubernetes.Kubernetes from it
func parseStaticSpec(path string) (*v1alpha1.Environment, error) {
	env, err := spec.ParseDir(path)
	if err != nil {
		switch err.(type) {
		// the config includes deprecated fields
		case spec.ErrDeprecated:
			log.Warn().Err(err).Msg("encountered deprecated fields in spec.json")
		// spec.json missing. we can still work with the default value
		case spec.ErrNoSpec:
			return env, nil
		// some other error
		default:
			return nil, err
		}
	}

	return env, nil
}
