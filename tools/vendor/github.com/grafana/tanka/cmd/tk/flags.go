package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/grafana/tanka/pkg/spec/v1alpha1"
	"github.com/grafana/tanka/pkg/tanka"
)

type workflowFlagVars struct {
	name                  string
	targets               []string
	jsonnetImplementation string
}

func jsonnetImplementationFlag(fs *pflag.FlagSet, output *string) {
	fs.StringVar(output, "jsonnet-implementation", "go", "Use `go` to use native go-jsonnet implementation and `binary:<path>` to delegate evaluation to a binary (with the same API as the regular `jsonnet` binary, see the BinaryImplementation docstrings for more details)")
}

func workflowFlags(fs *pflag.FlagSet) *workflowFlagVars {
	v := workflowFlagVars{}
	fs.StringVar(&v.name, "name", "", "string that only a single inline environment contains in its name")
	fs.StringSliceVarP(&v.targets, "target", "t", nil, "Regex filter on '<kind>/<name>'. See https://tanka.dev/output-filtering")
	jsonnetImplementationFlag(fs, &v.jsonnetImplementation)
	return &v
}

func addDiffFlags(fs *pflag.FlagSet, opts *tanka.DiffBaseOpts) {
	fs.StringVar(&opts.Color, "color", "auto", `controls color in diff output, must be "auto", "always", or "never"`)
}

func addApplyFlags(fs *pflag.FlagSet, opts *tanka.ApplyBaseOpts, autoApproveDeprecated *bool, autoApprove *string) {
	fs.StringVar(&opts.DryRun, "dry-run", "", `--dry-run parameter to pass down to kubectl, must be "none", "server", or "client"`)
	fs.BoolVar(&opts.Force, "force", false, "force applying (kubectl apply --force)")

	// Parse the auto-approve flag (choice), still supporting the deprecated dangerous-auto-approve flag (boolean)
	fs.BoolVar(autoApproveDeprecated, "dangerous-auto-approve", false, "skip interactive approval. Only for automation!")
	if err := fs.MarkDeprecated("dangerous-auto-approve", "use --auto-approve instead"); err != nil {
		log.Fatal().Msgf("failed to mark deprecated flag: %s", err)
	}
	fs.StringVar(autoApprove, "auto-approve", "", "skip interactive approval. Only for automation! Allowed values: 'always', 'never', 'if-no-changes'")
}

func labelSelectorFlag(fs *pflag.FlagSet) func() labels.Selector {
	labelSelector := fs.StringP("selector", "l", "", "Label selector. Uses the same syntax as kubectl does")

	return func() labels.Selector {
		if *labelSelector != "" {
			selector, err := labels.Parse(*labelSelector)
			if err != nil {
				log.Fatal().Msgf("Could not parse selector (-l) %s", *labelSelector)
			}
			return selector
		}
		return nil
	}
}

func jsonnetFlags(fs *pflag.FlagSet) func() tanka.JsonnetOpts {
	getExtCode, getTLACode := cliCodeParser(fs)
	maxStack := fs.Int("max-stack", 0, "Jsonnet VM max stack. The default value is the value set in the go-jsonnet library. Increase this if you get: max stack frames exceeded")

	return func() tanka.JsonnetOpts {
		return tanka.JsonnetOpts{
			MaxStack: *maxStack,
			ExtCode:  getExtCode(),
			TLACode:  getTLACode(),
		}
	}
}

func cliCodeParser(fs *pflag.FlagSet) (func() map[string]string, func() map[string]string) {
	// need to use StringArray instead of StringSlice, because pflag attempts to
	// parse StringSlice using the csv parser, which breaks when passing objects
	extCode := fs.StringArray("ext-code", nil, "Set code value of extVar (Format: key=<code>)")
	extCodeFile := fs.StringArray("ext-code-file", nil, "Set code value of extVar from file (Format: key=filename)")
	extStr := fs.StringArrayP("ext-str", "V", nil, "Set string value of extVar (Format: key=value)")
	extStrFile := fs.StringArray("ext-str-file", nil, "Set string value of extVar from file (Format: key=filename)")

	tlaCode := fs.StringArray("tla-code", nil, "Set code value of top level function (Format: key=<code>)")
	tlaCodeFile := fs.StringArray("tla-code-file", nil, "Set code value of top level function from file (Format: key=filename)")
	tlaStr := fs.StringArrayP("tla-str", "A", nil, "Set string value of top level function (Format: key=value)")
	tlaStrFile := fs.StringArray("tla-str-file", nil, "Set string value of top level function from file (Format: key=filename)")

	newParser := func(kind string, code, str, codeFile, strFile *[]string) func() map[string]string {
		return func() map[string]string {
			m := make(map[string]string)
			for _, s := range *code {
				split := strings.SplitN(s, "=", 2)
				if len(split) != 2 {
					log.Fatal().Msgf(kind+"-code argument has wrong format: `%s`. Expected `key=<code>`", s)
				}
				m[split[0]] = split[1]
			}

			for _, s := range *str {
				split := strings.SplitN(s, "=", 2)
				if len(split) != 2 {
					log.Fatal().Msgf(kind+"-str argument has wrong format: `%s`. Expected `key=<value>`", s)
				}
				// Properly quote the string; note that fmt.Sprintf("%q",...) could
				// produce \U escapes which are not valid Jsonnet.
				js, err := json.Marshal(split[1])
				if err != nil {
					log.Fatal().Msgf("impossible: failed to convert string to JSON: %s", err)
				}
				m[split[0]] = string(js)
			}

			for _, x := range []struct {
				arg        *[]string
				kind2, imp string
			}{
				{arg: codeFile, kind2: "code", imp: "import"},
				{arg: strFile, kind2: "str", imp: "importstr"},
			} {
				for _, s := range *x.arg {
					split := strings.SplitN(s, "=", 2)
					if len(split) != 2 {
						log.Fatal().Msgf("%s-%s-file argument has wrong format: `%s`. Expected `key=filename`", kind, x.kind2, s)
					}
					m[split[0]] = fmt.Sprintf(`%s @"%s"`, x.imp, strings.ReplaceAll(split[1], `"`, `""`))
				}
			}
			return m
		}
	}

	return newParser("ext", extCode, extStr, extCodeFile, extStrFile),
		newParser("tla", tlaCode, tlaStr, tlaCodeFile, tlaStrFile)
}

func envSettingsFlags(env *v1alpha1.Environment, fs *pflag.FlagSet) {
	fs.StringVar(&env.Spec.APIServer, "server", env.Spec.APIServer, "endpoint of the Kubernetes API")
	fs.StringVar(&env.Spec.APIServer, "server-from-context", env.Spec.APIServer, "set the server to a known one from $KUBECONFIG")
	fs.StringSliceVar(&env.Spec.ContextNames, "context-name", env.Spec.ContextNames, "valid context name for environment, can pass multiple, regex supported.")
	fs.StringVar(&env.Spec.Namespace, "namespace", env.Spec.Namespace, "namespace to create objects in")
	fs.StringVar(&env.Spec.DiffStrategy, "diff-strategy", env.Spec.DiffStrategy, "specify diff-strategy. Automatically detected otherwise.")
	fs.BoolVar(&env.Spec.InjectLabels, "inject-labels", env.Spec.InjectLabels, "add tanka environment label to each created resource. Required for 'tk prune'.")
}
