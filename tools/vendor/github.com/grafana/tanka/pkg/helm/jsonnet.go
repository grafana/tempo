package helm

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/tanka/pkg/kubernetes/manifest"
	"github.com/rs/zerolog/log"
)

// DefaultNameFormat to use when no nameFormat is supplied
const DefaultNameFormat = `{{ print .kind "_" .metadata.name | snakecase }}`

// helmTemplateCache caches the inline environments' rendered helm templates.
var helmTemplateCache sync.Map

// JsonnetOpts are additional properties the consumer of the native func might
// pass.
type JsonnetOpts struct {
	TemplateOpts

	// CalledFrom is the file that calls helmTemplate. This is used to find the
	// vendored chart relative to this file
	CalledFrom string `json:"calledFrom"`
	// NameTemplate is used to create the keys in the resulting map
	NameFormat string `json:"nameFormat"`
}

// NativeFunc returns a jsonnet native function that provides the same
// functionality as `Helm.Template` of this package. Charts are required to be
// present on the local filesystem, at a relative location to the file that
// calls `helm.template()` / `std.native('helmTemplate')`. This guarantees
// hermeticity
func NativeFunc(h Helm) *jsonnet.NativeFunction {
	return &jsonnet.NativeFunction{
		Name: "helmTemplate",
		// Similar to `helm template [NAME] [CHART] [flags]` except 'conf' is a
		// bit more elaborate and chart is a local path
		Params: ast.Identifiers{"name", "chart", "opts"},
		Func: func(data []interface{}) (interface{}, error) {
			name, ok := data[0].(string)
			if !ok {
				return nil, fmt.Errorf("first argument 'name' must be of 'string' type, got '%T' instead", data[0])
			}

			chartpath, ok := data[1].(string)
			if !ok {
				return nil, fmt.Errorf("second argument 'chart' must be of 'string' type, got '%T' instead", data[1])
			}

			// TODO: validate data[2] actually follows the struct scheme
			opts, err := parseOpts(data[2])
			if err != nil {
				return "", err
			}

			chart, err := h.ChartExists(chartpath, opts)
			if err != nil {
				return nil, fmt.Errorf("helmTemplate: Failed to find a chart at '%s': %s. See https://tanka.dev/helm#failed-to-find-chart", chart, err)
			}

			// check if resources exist in cache
			helmKey, err := templateKey(name, chartpath, opts.TemplateOpts)
			if err != nil {
				return nil, err
			}
			if entry, ok := helmTemplateCache.Load(helmKey); ok {
				log.Debug().Msgf("Using cached template for %s", name)
				return entry, nil
			}

			// render resources
			list, err := h.Template(name, chart, opts.TemplateOpts)
			if err != nil {
				return nil, err
			}

			// convert list to map
			out, err := manifest.ListAsMap(list, opts.NameFormat)
			if err != nil {
				return nil, err
			}

			helmTemplateCache.Store(helmKey, out)
			return out, nil
		},
	}
}

// templateKey returns the key identifier used in the template cache for the given helm chart.
func templateKey(chartName string, chartPath string, opts TemplateOpts) (string, error) {
	hasher := sha256.New()
	hasher.Write([]byte(chartName))
	hasher.Write([]byte(chartPath))
	valuesBytes, err := json.Marshal(opts)
	if err != nil {
		return "", err
	}
	hasher.Write(valuesBytes)
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil)), nil
}

func parseOpts(data interface{}) (*JsonnetOpts, error) {
	c, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// default IncludeCRDs to true, as this is the default in the `helm install`
	// command. Needs to be specified here because the zero value of bool is
	// false.
	opts := JsonnetOpts{
		TemplateOpts: TemplateOpts{
			IncludeCRDs: true,
		},
	}

	if err := json.Unmarshal(c, &opts); err != nil {
		return nil, err
	}

	// Charts are only allowed at relative paths. Use conf.CalledFrom to find the callers directory
	if opts.CalledFrom == "" {
		return nil, fmt.Errorf("helmTemplate: 'opts.calledFrom' is unset or empty.\nTanka needs this to find your charts. See https://tanka.dev/helm#optscalledfrom-unset")
	}

	return &opts, nil
}
