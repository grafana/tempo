package tanka

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/tanka/pkg/jsonnet/implementations/binary"
	"github.com/grafana/tanka/pkg/jsonnet/implementations/goimpl"
	"github.com/grafana/tanka/pkg/jsonnet/implementations/types"
	"github.com/grafana/tanka/pkg/jsonnet/jpath"
	"github.com/grafana/tanka/pkg/kubernetes"
	"github.com/grafana/tanka/pkg/kubernetes/manifest"
	"github.com/grafana/tanka/pkg/process"
	"github.com/grafana/tanka/pkg/spec"
	"github.com/grafana/tanka/pkg/spec/v1alpha1"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// environmentExtCode is the extCode ID `tk.env` uses underneath
// TODO: remove "import tk" and replace it with tanka-util
const environmentExtCode = spec.APIGroup + "/environment"

// Load loads the Environment at `path`. It automatically detects whether to
// load inline or statically
func Load(path string, opts Opts) (*LoadResult, error) {
	env, err := LoadEnvironment(path, opts)
	if err != nil {
		return nil, err
	}

	result, err := LoadManifests(env, opts.Filters)
	if err != nil {
		return nil, err
	}

	// Check if there are still any inline environments in the manifests
	// They are not real k8s resources, and cannot be applied
	if envs := process.Filter(result.Resources, process.MustStrExps("Environment/.*")); len(envs) > 0 {
		return nil, errors.New("found a tanka Environment resource. Check that you aren't using a spec.json and inline environments simultaneously")
	}

	return result, nil
}

func LoadEnvironment(path string, opts Opts) (*v1alpha1.Environment, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		log.Info().Msgf("Path %q does not exist, trying to use it as an environment name", path)
		opts.Name = path
		path = "."
	} else if err != nil {
		return nil, err
	}

	loader, err := DetectLoader(path, opts)
	if err != nil {
		return nil, err
	}

	env, err := loader.Load(path, LoaderOpts{opts.JsonnetOpts, opts.Name})
	if err != nil {
		return nil, err
	}

	return env, nil
}

func LoadManifests(env *v1alpha1.Environment, filters process.Matchers) (*LoadResult, error) {
	if err := checkVersion(env.Spec.ExpectVersions.Tanka); err != nil {
		return nil, err
	}

	processed, err := process.Process(*env, filters)
	if err != nil {
		return nil, err
	}

	return &LoadResult{Env: env, Resources: processed}, nil
}

// Peek loads the metadata of the environment at path. To get resources as well,
// use Load
func Peek(path string, opts Opts) (*v1alpha1.Environment, error) {
	loader, err := DetectLoader(path, opts)
	if err != nil {
		return nil, err
	}

	return loader.Peek(path, LoaderOpts{opts.JsonnetOpts, opts.Name})
}

// List finds metadata of all environments at path that could possibly be
// loaded. List can be used to deal with multiple inline environments, by first
// listing them, choosing the right one and then only loading that one
func List(path string, opts Opts) ([]*v1alpha1.Environment, error) {
	loader, err := DetectLoader(path, opts)
	if err != nil {
		return nil, err
	}

	return loader.List(path, LoaderOpts{opts.JsonnetOpts, opts.Name})
}

func getJsonnetImplementation(path string, opts Opts) (types.JsonnetImplementation, error) {
	if strings.HasPrefix(opts.JsonnetImplementation, "binary:") {
		binPath := strings.TrimPrefix(opts.JsonnetImplementation, "binary:")

		// check if binary exists and is executable
		stat, err := os.Stat(binPath)
		if err != nil {
			return nil, fmt.Errorf("binary %q does not exist", binPath)
		}
		if stat.Mode()&0111 == 0 {
			return nil, fmt.Errorf("binary %q is not executable", binPath)
		}

		return &binary.JsonnetBinaryImplementation{
			BinPath: binPath,
		}, nil
	}

	switch opts.JsonnetImplementation {
	case "go", "":
		return &goimpl.JsonnetGoImplementation{
			Path: path,
		}, nil
	default:
		return nil, fmt.Errorf("unknown jsonnet implementation: %s", opts.JsonnetImplementation)
	}
}

// Eval returns the raw evaluated Jsonnet
func Eval(path string, opts Opts) (interface{}, error) {
	loader, err := DetectLoader(path, opts)
	if err != nil {
		return nil, err
	}

	return loader.Eval(path, LoaderOpts{opts.JsonnetOpts, opts.Name})
}

// DetectLoader detects whether the environment is inline or static and picks
// the approriate loader
func DetectLoader(path string, opts Opts) (Loader, error) {
	_, base, err := jpath.Dirs(path)
	if err != nil {
		return nil, err
	}

	jsonnetImpl, err := getJsonnetImplementation(base, opts)
	if err != nil {
		return nil, err
	}

	// check if spec.json exists
	_, err = os.Stat(filepath.Join(base, spec.Specfile))
	if os.IsNotExist(err) {
		return &InlineLoader{
			jsonnetImpl: jsonnetImpl,
		}, nil
	} else if err != nil {
		return nil, err
	}

	return &StaticLoader{
		jsonnetImpl: jsonnetImpl,
	}, nil
}

// Loader is an abstraction over the process of loading Environments
type Loader interface {
	// Load a single environment at path
	Load(path string, opts LoaderOpts) (*v1alpha1.Environment, error)

	// Peek only loads metadata and omits the actual resources
	Peek(path string, opts LoaderOpts) (*v1alpha1.Environment, error)

	// List returns metadata of all possible environments at path that can be
	// loaded
	List(path string, opts LoaderOpts) ([]*v1alpha1.Environment, error)

	// Eval returns the raw evaluated Jsonnet
	Eval(path string, opts LoaderOpts) (interface{}, error)
}

type LoaderOpts struct {
	JsonnetOpts
	Name string
}

type LoadResult struct {
	Env       *v1alpha1.Environment
	Resources manifest.List
}

func (l LoadResult) Connect() (*kubernetes.Kubernetes, error) {
	env := *l.Env

	// check env is complete
	s := ""
	if env.Spec.APIServer == "" && len(env.Spec.ContextNames) < 1 {
		s += "  * spec.apiServer|spec.contextNames: No Kubernetes cluster endpoint or context names specified. Please specify only one."
	} else if env.Spec.APIServer != "" && len(env.Spec.ContextNames) > 0 {
		s += "  * spec.apiServer|spec.contextNames: These fields are mutually exclusive, please only specify one."
	}
	if env.Spec.Namespace == "" {
		s += "  * spec.namespace: Default namespace missing"
	}
	if s != "" {
		return nil, fmt.Errorf("your Environment's spec.json seems incomplete:\n%s\n\nPlease see https://tanka.dev/config for reference", s)
	}

	// connect client
	kube, err := kubernetes.New(env)
	if err != nil {
		return nil, errors.Wrap(err, "connecting to Kubernetes")
	}

	return kube, nil
}
