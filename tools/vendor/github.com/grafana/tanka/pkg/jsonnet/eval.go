package jsonnet

import (
	"os"
	"regexp"
	"time"

	jsonnet "github.com/google/go-jsonnet"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/grafana/tanka/pkg/jsonnet/implementations/goimpl"
	"github.com/grafana/tanka/pkg/jsonnet/implementations/types"
	"github.com/grafana/tanka/pkg/jsonnet/jpath"
)

// Modifier allows to set optional parameters on the Jsonnet VM.
// See jsonnet.With* for this.
type Modifier func(vm *jsonnet.VM) error

// InjectedCode holds data that is "late-bound" into the VM
type InjectedCode map[string]string

// Set allows to set values on an InjectedCode, even when it is nil
func (i *InjectedCode) Set(key, value string) {
	if *i == nil {
		*i = make(InjectedCode)
	}

	(*i)[key] = value
}

// Opts are additional properties for the Jsonnet VM
type Opts struct {
	MaxStack    int
	ExtCode     InjectedCode
	TLACode     InjectedCode
	ImportPaths []string
	EvalScript  string
	CachePath   string

	CachePathRegexes []*regexp.Regexp
}

// PathIsCached determines if a given path is matched by any of the configured cached path regexes
// If no path regexes are defined, all paths are matched
func (o Opts) PathIsCached(path string) bool {
	for _, regex := range o.CachePathRegexes {
		if regex.MatchString(path) {
			return true
		}
	}
	return len(o.CachePathRegexes) == 0
}

// Clone returns a deep copy of Opts
func (o Opts) Clone() Opts {
	extCode, tlaCode := InjectedCode{}, InjectedCode{}

	for k, v := range o.ExtCode {
		extCode[k] = v
	}

	for k, v := range o.TLACode {
		tlaCode[k] = v
	}

	return Opts{
		TLACode:     tlaCode,
		ExtCode:     extCode,
		ImportPaths: append([]string{}, o.ImportPaths...),
		EvalScript:  o.EvalScript,

		CachePath:        o.CachePath,
		CachePathRegexes: o.CachePathRegexes,
	}
}

// EvaluateFile evaluates the Jsonnet code in the given file and returns the
// result in JSON form. It disregards opts.ImportPaths in favor of automatically
// resolving these according to the specified file.
func EvaluateFile(impl types.JsonnetImplementation, jsonnetFile string, opts Opts) (string, error) {
	evalFunc := func(evaluator types.JsonnetEvaluator) (string, error) {
		return evaluator.EvaluateFile(jsonnetFile)
	}
	data, err := os.ReadFile(jsonnetFile)
	if err != nil {
		return "", err
	}
	return evaluateSnippet(impl, evalFunc, jsonnetFile, string(data), opts)
}

// Evaluate renders the given jsonnet into a string
// If cache options are given, a hash from the data will be computed and
// the resulting string will be cached for future retrieval
func Evaluate(path string, impl types.JsonnetImplementation, data string, opts Opts) (string, error) {
	evalFunc := func(evaluator types.JsonnetEvaluator) (string, error) {
		return evaluator.EvaluateAnonymousSnippet(data)
	}
	return evaluateSnippet(impl, evalFunc, path, data, opts)
}

type evalFunc func(evaluator types.JsonnetEvaluator) (string, error)

func evaluateSnippet(jsonnetImpl types.JsonnetImplementation, evalFunc evalFunc, path, data string, opts Opts) (string, error) {
	var cache *FileEvalCache
	if opts.CachePath != "" && opts.PathIsCached(path) {
		cache = NewFileEvalCache(opts.CachePath)
	}

	jpath, _, _, err := jpath.Resolve(path, false)
	if err != nil {
		return "", errors.Wrap(err, "resolving import paths")
	}
	opts.ImportPaths = jpath
	evaluator := jsonnetImpl.MakeEvaluator(opts.ImportPaths, opts.ExtCode, opts.TLACode, opts.MaxStack)
	// We're using the go implementation to deal with imports because we're not evaluating, we're reading the AST
	importVM := goimpl.MakeRawVM(opts.ImportPaths, opts.ExtCode, opts.TLACode, opts.MaxStack)

	var hash string
	if cache != nil {
		startTime := time.Now()
		if hash, err = getSnippetHash(importVM, path, data); err != nil {
			return "", err
		}
		cacheLog := log.Debug().Str("path", path).Str("hash", hash).Dur("duration_ms", time.Since(startTime))
		if v, err := cache.Get(hash); err != nil {
			return "", err
		} else if v != "" {
			cacheLog.Bool("cache_hit", true).Msg("computed snippet hash")
			return v, nil
		}
		cacheLog.Bool("cache_hit", false).Msg("computed snippet hash")
	}

	content, err := evalFunc(evaluator)
	if err != nil {
		return "", err
	}

	if cache != nil {
		return content, cache.Store(hash, content)
	}

	return content, nil
}
