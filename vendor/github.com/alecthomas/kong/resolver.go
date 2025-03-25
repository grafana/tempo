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
	Resolve(context *Context, parent *Path, flag *Flag) (any, error)
}

// ResolverFunc is a convenience type for non-validating Resolvers.
type ResolverFunc func(context *Context, parent *Path, flag *Flag) (any, error)

var _ Resolver = ResolverFunc(nil)

func (r ResolverFunc) Resolve(context *Context, parent *Path, flag *Flag) (any, error) { //nolint: revive
	return r(context, parent, flag)
}
func (r ResolverFunc) Validate(app *Application) error { return nil } //nolint: revive

// JSON returns a Resolver that retrieves values from a JSON source.
//
// Flag names are used as JSON keys indirectly, by tring snake_case and camelCase variants.
func JSON(r io.Reader) (Resolver, error) {
	values := map[string]any{}
	err := json.NewDecoder(r).Decode(&values)
	if err != nil {
		return nil, err
	}
	var f ResolverFunc = func(context *Context, parent *Path, flag *Flag) (any, error) {
		name := strings.ReplaceAll(flag.Name, "-", "_")
		snakeCaseName := snakeCase(flag.Name)
		raw, ok := values[name]
		if ok {
			return raw, nil
		} else if raw, ok = values[snakeCaseName]; ok {
			return raw, nil
		}
		raw = values
		for _, part := range strings.Split(name, ".") {
			if values, ok := raw.(map[string]any); ok {
				raw, ok = values[part]
				if !ok {
					return nil, nil
				}
			} else {
				return nil, nil
			}
		}
		return raw, nil
	}

	return f, nil
}

func snakeCase(name string) string {
	name = strings.Join(strings.Split(strings.Title(name), "-"), "") //nolint:staticcheck // Unicode punctuation not an issue
	return strings.ToLower(name[:1]) + name[1:]
}
