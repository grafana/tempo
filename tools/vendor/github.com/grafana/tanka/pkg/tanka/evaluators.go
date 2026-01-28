package tanka

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/grafana/tanka/pkg/jsonnet"
	"github.com/grafana/tanka/pkg/jsonnet/implementations/types"
	"github.com/grafana/tanka/pkg/jsonnet/jpath"
)

// EvalJsonnet evaluates the jsonnet environment at the given file system path
func evalJsonnet(path string, impl types.JsonnetImplementation, opts jsonnet.Opts) (raw string, err error) {
	entrypoint, err := jpath.Entrypoint(path)
	if err != nil {
		return "", err
	}

	// evaluate Jsonnet
	if opts.EvalScript != "" {
		// Determine if the entrypoint is a function.
		isFunction, err := jsonnet.Evaluate(path, impl, fmt.Sprintf("std.isFunction(import '%s')", entrypoint), opts)
		if err != nil {
			return "", fmt.Errorf("evaluating jsonnet in path '%s': %w", path, err)
		}
		var tla []string
		for k := range opts.TLACode {
			tla = append(tla, k+"="+k)
		}
		evalScript := fmt.Sprintf(`
  local main = (import '%s');
  %s
`, entrypoint, opts.EvalScript)

		if isFunction == "true\n" {
			tlaJoin := strings.Join(tla, ", ")
			evalScript = fmt.Sprintf(`
function(%s)
  local main = (import '%s')(%s);
  %s
`, tlaJoin, entrypoint, tlaJoin, opts.EvalScript)
		}

		raw, err = jsonnet.Evaluate(path, impl, evalScript, opts)
		if err != nil {
			return "", fmt.Errorf("evaluating jsonnet in path '%s': %w", path, err)
		}
		return raw, nil
	}

	raw, err = jsonnet.EvaluateFile(impl, entrypoint, opts)
	if err != nil {
		return "", errors.Wrap(err, "evaluating jsonnet")
	}
	return raw, nil
}

func PatternEvalScript(expr string) string {
	if strings.HasPrefix(expr, "[") {
		return fmt.Sprintf("main%s", expr)
	}
	return fmt.Sprintf("main.%s", expr)
}

// MetadataEvalScript finds the Environment object (without its .data object)
const MetadataEvalScript = `
local noDataEnv(object) =
  std.prune(
    if std.isObject(object)
    then
      if std.objectHas(object, 'apiVersion')
         && std.objectHas(object, 'kind')
      then
        if object.kind == 'Environment'
        then object { data+:: {} }
        else {}
      else
        std.mapWithKey(
          function(key, obj)
            noDataEnv(obj),
          object
        )
    else if std.isArray(object)
    then
      std.map(
        function(obj)
          noDataEnv(obj),
        object
      )
    else {}
  );

noDataEnv(main)
`

// MetadataSingleEnvEvalScript returns a Single Environment object
const MetadataSingleEnvEvalScript = `
local singleEnv(object) =
  std.prune(
    if std.isObject(object)
    then
      if std.objectHas(object, 'apiVersion')
         && std.objectHas(object, 'kind')
      then
        if object.kind == 'Environment'
        && std.member(object.metadata.name, '%s')
        then object { data:: super.data }
        else {}
      else
        std.mapWithKey(
          function(key, obj)
            singleEnv(obj),
          object
        )
    else if std.isArray(object)
    then
      std.map(
        function(obj)
          singleEnv(obj),
        object
      )
    else {}
  );

singleEnv(main)
`

// SingleEnvEvalScript returns a Single Environment object
const SingleEnvEvalScript = `
local singleEnv(object) =
  if std.isObject(object)
  then
    if std.objectHas(object, 'apiVersion')
       && std.objectHas(object, 'kind')
    then
      if object.kind == 'Environment'
      && std.member(object.metadata.name, '%s')
      then object
      else {}
    else
      std.mapWithKey(
        function(key, obj)
          singleEnv(obj),
        object
      )
  else if std.isArray(object)
  then
    std.map(
      function(obj)
        singleEnv(obj),
      object
    )
  else {};

singleEnv(main)
`
