package tanka

import (
	"fmt"
	"strings"
)

// ErrNoEnv means that the given jsonnet has no Environment object
// This must not be fatal, some operations work without
type ErrNoEnv struct {
	path string
}

func (e ErrNoEnv) Error() string {
	return fmt.Sprintf("unable to find an Environment in '%s'", e.path)
}

// ErrMultipleEnvs means that the given jsonnet has multiple Environment objects
type ErrMultipleEnvs struct {
	path      string
	givenName string
	foundEnvs []string
}

func (e ErrMultipleEnvs) Error() string {
	if e.givenName != "" {
		return fmt.Sprintf("found multiple Environments in %q matching %q. Provide a more specific name that matches a single one: \n - %s", e.path, e.givenName, strings.Join(e.foundEnvs, "\n - "))
	}

	return fmt.Sprintf("found multiple Environments in %q. Use `--name` to select a single one: \n - %s", e.path, strings.Join(e.foundEnvs, "\n - "))
}

// ErrParallel is an array of errors collected while processing in parallel
type ErrParallel struct {
	errors []error
}

func (e ErrParallel) Error() string {
	returnErr := "Errors occurred during parallel processing:\n\n"
	for _, err := range e.errors {
		returnErr = fmt.Sprintf("%s- %s\n\n", returnErr, err.Error())
	}
	return returnErr
}
