package kong

import (
	"encoding/json"
	"io"
	"strings"
)

// A Resolver resolves a Flag value from an external source.
type Resolver interface {
	// Validate configuration against Application.
	//
	// This can be used to validate that all provided configuration is valid within  this application.
	Validate(app *Application) error

	// Resolve the value for a Flag.
	Resolve(context *Context, parent *Path, flag *Flag) (interface{}, error)
}

// ResolverFunc is a convenience type for non-validating Resolvers.
type ResolverFunc func(context *Context, parent *Path, flag *Flag) (interface{}, error)

var _ Resolver = ResolverFunc(nil)

func (r ResolverFunc) Resolve(context *Context, parent *Path, flag *Flag) (interface{}, error) { // nolint: golint
	return r(context, parent, flag)
}
func (r ResolverFunc) Validate(app *Application) error { return nil } //  nolint: golint

// JSON returns a Resolver that retrieves values from a JSON source.
//
// Hyphens in flag names are replaced with underscores.
func JSON(r io.Reader) (Resolver, error) {
	values := map[string]interface{}{}
	err := json.NewDecoder(r).Decode(&values)
	if err != nil {
		return nil, err
	}
	var f ResolverFunc = func(context *Context, parent *Path, flag *Flag) (interface{}, error) {
		name := strings.Replace(flag.Name, "-", "_", -1)
		raw, ok := values[name]
		if !ok {
			return nil, nil
		}
		return raw, nil
	}

	return f, nil
}
