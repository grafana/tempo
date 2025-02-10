package kong

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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
	name = strings.Join(strings.Split(strings.Title(name), "-"), "")
	return strings.ToLower(name[:1]) + name[1:]
}

// EnvResolver provides a resolver for environment variables tags
func EnvResolver() Resolver {
	// Resolvers are typically only invoked for flags, as shown here:
	// https://github.com/alecthomas/kong/blob/v1.6.0/context.go#L567
	// However, environment variable annotations can also apply to arguments,
	// as demonstrated in this test:
	// https://github.com/alecthomas/kong/blob/v1.6.0/kong_test.go#L1226-L1244
	// To handle this, we ensure that arguments are resolved as well.
	// Since the resolution only needs to happen once, we use this boolean
	// to track whether the resolution process has already been performed.
	argsResolved := false
	return ResolverFunc(func(context *Context, parent *Path, flag *Flag) (interface{}, error) {
		if !argsResolved {
			if err := resolveArgs(context.Path); err != nil {
				return nil, err
			}
			// once resolved we do not want to run this anymore
			argsResolved = true
		}
		for _, env := range flag.Tag.Envs {
			envar, ok := os.LookupEnv(env)
			// Parse the first non-empty ENV in the list
			if ok {
				return envar, nil
			}
		}
		return nil, nil
	})
}

func resolveArgs(paths []*Path) error {
	for _, path := range paths {
		if path.Command == nil {
			continue
		}
		for _, positional := range path.Command.Positional {
			if positional.Tag == nil {
				continue
			}
			if err := visitValue(positional); err != nil {
				return err
			}
		}
		if path.Command.Argument != nil {
			if err := visitValue(path.Command.Argument); err != nil {
				return err
			}
		}
	}
	return nil
}

func visitValue(value *Value) error {
	for _, env := range value.Tag.Envs {
		envar, ok := os.LookupEnv(env)
		if !ok {
			continue
		}
		token := Token{Type: FlagValueToken, Value: envar}
		if err := value.Parse(ScanFromTokens(token), value.Target); err != nil {
			return fmt.Errorf("%s (from envar %s=%q)", err, env, envar)
		}
	}
	return nil
}
