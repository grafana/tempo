package kustomize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/tanka/pkg/kubernetes/manifest"
)

// JsonnetOpts are additional properties the consumer of the native func might
// pass.
type JsonnetOpts struct {
	// CalledFrom is the file that calls kustomizeBuild. This is used to find the
	// vendored Kustomize relative to this file
	CalledFrom string `json:"calledFrom"`
	// NameBuild is used to create the keys in the resulting map
	NameFormat string `json:"nameFormat"`
}

// NativeFunc returns a jsonnet native function that provides the same
// functionality as `Kustomize.Build` of this package. Kustomize yamls are required to be
// present on the local filesystem, at a relative location to the file that
// calls `kustomize.build()` / `std.native('kustomizeBuild')`. This guarantees
// hermeticity
func NativeFunc(k Kustomize) *jsonnet.NativeFunction {
	return &jsonnet.NativeFunction{
		Name: "kustomizeBuild",
		// Similar to `kustomize build {path}` where {path} is a local path
		Params: ast.Identifiers{"path", "opts"},
		Func: func(data []interface{}) (interface{}, error) {
			path, ok := data[0].(string)
			if !ok {
				return nil, fmt.Errorf("argument 'path' must be of 'string' type, got '%T' instead", data[0])
			}

			// TODO: validate data[1] actually follows the struct scheme
			opts, err := parseOpts(data[1])
			if err != nil {
				return "", err
			}

			// resolve the Kustomize path relative to the caller
			callerDir := filepath.Dir(opts.CalledFrom)
			actualPath := filepath.Join(callerDir, path)
			if _, err := os.Stat(actualPath); err != nil {
				return nil, fmt.Errorf("kustomizeBuild: Failed to find kustomization at '%s': %s. See https://tanka.dev/kustomize#failed-to-find-kustomization", actualPath, err)
			}

			// render resources
			list, err := k.Build(actualPath)
			if err != nil {
				return nil, err
			}

			// convert list to map
			out, err := manifest.ListAsMap(list, opts.NameFormat)
			if err != nil {
				return nil, err
			}

			return out, nil
		},
	}
}

func parseOpts(data interface{}) (*JsonnetOpts, error) {
	c, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var opts JsonnetOpts
	if err := json.Unmarshal(c, &opts); err != nil {
		return nil, err
	}

	// Kustomize paths are only allowed at relative paths. Use conf.CalledFrom to find the callers directory
	if opts.CalledFrom == "" {
		return nil, fmt.Errorf("kustomizeBuild: 'opts.calledFrom' is unset or empty.\nTanka needs this to find your Kustomize. See https://tanka.dev/kustomize#optscalledfrom-unset")
	}

	return &opts, nil
}
