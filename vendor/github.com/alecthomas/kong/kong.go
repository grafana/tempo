package kong

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

var (
	callbackReturnSignature = reflect.TypeOf((*error)(nil)).Elem()
)

// Error reported by Kong.
type Error struct{ msg string }

func (e Error) Error() string { return e.msg }

func fail(format string, args ...interface{}) {
	panic(Error{msg: fmt.Sprintf(format, args...)})
}

// Must creates a new Parser or panics if there is an error.
func Must(ast interface{}, options ...Option) *Kong {
	k, err := New(ast, options...)
	if err != nil {
		panic(err)
	}
	return k
}

// Kong is the main parser type.
type Kong struct {
	// Grammar model.
	Model *Application

	// Termination function (defaults to os.Exit)
	Exit func(int)

	Stdout io.Writer
	Stderr io.Writer

	bindings  bindings
	loader    ConfigurationLoader
	resolvers []Resolver
	registry  *Registry

	noDefaultHelp bool
	usageOnError  bool
	help          HelpPrinter
	helpFormatter HelpValueFormatter
	helpOptions   HelpOptions
	helpFlag      *Flag
	vars          Vars

	// Set temporarily by Options. These are applied after build().
	postBuildOptions []Option
}

// New creates a new Kong parser on grammar.
//
// See the README (https://github.com/alecthomas/kong) for usage instructions.
func New(grammar interface{}, options ...Option) (*Kong, error) {
	k := &Kong{
		Exit:          os.Exit,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
		registry:      NewRegistry().RegisterDefaults(),
		vars:          Vars{},
		bindings:      bindings{},
		helpFormatter: DefaultHelpValueFormatter,
	}

	options = append(options, Bind(k))

	for _, option := range options {
		if err := option.Apply(k); err != nil {
			return nil, err
		}
	}

	if k.help == nil {
		k.help = DefaultHelpPrinter
	}

	model, err := build(k, grammar)
	if err != nil {
		return k, err
	}
	model.Name = filepath.Base(os.Args[0])
	k.Model = model
	k.Model.HelpFlag = k.helpFlag

	for _, option := range k.postBuildOptions {
		if err = option.Apply(k); err != nil {
			return nil, err
		}
	}
	k.postBuildOptions = nil

	if err = k.interpolate(k.Model.Node); err != nil {
		return nil, err
	}

	k.bindings.add(k.vars)

	return k, nil
}

type varStack []Vars

func (v *varStack) head() Vars { return (*v)[len(*v)-1] }
func (v *varStack) pop()       { *v = (*v)[:len(*v)-1] }
func (v *varStack) push(vars Vars) Vars {
	if len(*v) != 0 {
		vars = (*v)[len(*v)-1].CloneWith(vars)
	}
	*v = append(*v, vars)
	return vars
}

// Interpolate variables into model.
func (k *Kong) interpolate(node *Node) (err error) {
	stack := varStack{}
	return Visit(node, func(node Visitable, next Next) error {
		switch node := node.(type) {
		case *Node:
			vars := stack.push(node.Vars())
			node.Help, err = interpolate(node.Help, vars, nil)
			if err != nil {
				return fmt.Errorf("help for %s: %s", node.Path(), err)
			}
			err = next(nil)
			stack.pop()
			return err

		case *Value:
			return next(k.interpolateValue(node, stack.head()))
		}
		return next(nil)
	})
}

func (k *Kong) interpolateValue(value *Value, vars Vars) (err error) {
	if len(value.Tag.Vars) > 0 {
		vars = vars.CloneWith(value.Tag.Vars)
	}
	if value.Default, err = interpolate(value.Default, vars, nil); err != nil {
		return fmt.Errorf("default value for %s: %s", value.Summary(), err)
	}
	if value.Enum, err = interpolate(value.Enum, vars, nil); err != nil {
		return fmt.Errorf("enum value for %s: %s", value.Summary(), err)
	}
	value.Help, err = interpolate(value.Help, vars, map[string]string{
		"default": value.Default,
		"enum":    value.Enum,
	})
	if err != nil {
		return fmt.Errorf("help for %s: %s", value.Summary(), err)
	}
	return nil
}

// Provide additional builtin flags, if any.
func (k *Kong) extraFlags() []*Flag {
	if k.noDefaultHelp {
		return nil
	}
	var helpTarget helpValue
	value := reflect.ValueOf(&helpTarget).Elem()
	helpFlag := &Flag{
		Short: 'h',
		Value: &Value{
			Name:         "help",
			Help:         "Show context-sensitive help.",
			Target:       value,
			Tag:          &Tag{},
			Mapper:       k.registry.ForValue(value),
			DefaultValue: reflect.ValueOf(false),
		},
	}
	helpFlag.Flag = helpFlag
	k.helpFlag = helpFlag
	return []*Flag{helpFlag}
}

// Parse arguments into target.
//
// The return Context can be used to further inspect the parsed command-line, to format help, to find the
// selected command, to run command Run() methods, and so on. See Context and README for more information.
//
// Will return a ParseError if a *semantically* invalid command-line is encountered (as opposed to a syntactically
// invalid one, which will report a normal error).
func (k *Kong) Parse(args []string) (ctx *Context, err error) {
	defer catch(&err)
	ctx, err = Trace(k, args)
	if err != nil {
		return nil, err
	}
	if ctx.Error != nil {
		return nil, &ParseError{error: ctx.Error, Context: ctx}
	}
	if err = ctx.Reset(); err != nil {
		return nil, &ParseError{error: err, Context: ctx}
	}
	if err = k.applyHook(ctx, "BeforeResolve"); err != nil {
		return nil, &ParseError{error: err, Context: ctx}
	}
	if err = ctx.Resolve(); err != nil {
		return nil, &ParseError{error: err, Context: ctx}
	}
	if err = k.applyHook(ctx, "BeforeApply"); err != nil {
		return nil, &ParseError{error: err, Context: ctx}
	}
	if _, err = ctx.Apply(); err != nil {
		return nil, &ParseError{error: err, Context: ctx}
	}
	if err = ctx.Validate(); err != nil {
		return nil, &ParseError{error: err, Context: ctx}
	}
	if err = k.applyHook(ctx, "AfterApply"); err != nil {
		return nil, &ParseError{error: err, Context: ctx}
	}
	return ctx, nil
}

func (k *Kong) applyHook(ctx *Context, name string) error {
	for _, trace := range ctx.Path {
		var value reflect.Value
		switch {
		case trace.App != nil:
			value = trace.App.Target
		case trace.Argument != nil:
			value = trace.Argument.Target
		case trace.Command != nil:
			value = trace.Command.Target
		case trace.Positional != nil:
			value = trace.Positional.Target
		case trace.Flag != nil:
			value = trace.Flag.Value.Target
		default:
			panic("unsupported Path")
		}
		method := getMethod(value, name)
		if !method.IsValid() {
			continue
		}
		binds := k.bindings.clone()
		binds.add(ctx, trace)
		binds.add(trace.Node().Vars().CloneWith(k.vars))
		binds.merge(ctx.bindings)
		if err := callMethod(name, value, method, binds); err != nil {
			return err
		}
	}
	// Path[0] will always be the app root.
	return k.applyHookToDefaultFlags(ctx, ctx.Path[0].Node(), name)
}

// Call hook on any unset flags with default values.
func (k *Kong) applyHookToDefaultFlags(ctx *Context, node *Node, name string) error {
	if node == nil {
		return nil
	}
	return Visit(node, func(n Visitable, next Next) error {
		node, ok := n.(*Node)
		if !ok {
			return next(nil)
		}
		binds := k.bindings.clone().add(ctx).add(node.Vars().CloneWith(k.vars))
		for _, flag := range node.Flags {
			if flag.Default == "" || ctx.values[flag.Value].IsValid() || !flag.Target.IsValid() {
				continue
			}
			method := getMethod(flag.Target, name)
			if !method.IsValid() {
				continue
			}
			path := &Path{Flag: flag}
			if err := callMethod(name, flag.Target, method, binds.clone().add(path)); err != nil {
				return next(err)
			}
		}
		return next(nil)
	})
}

func formatMultilineMessage(w io.Writer, leaders []string, format string, args ...interface{}) {
	lines := strings.Split(fmt.Sprintf(format, args...), "\n")
	leader := ""
	for _, l := range leaders {
		if l == "" {
			continue
		}
		leader += l + ": "
	}
	fmt.Fprintf(w, "%s%s\n", leader, lines[0])
	for _, line := range lines[1:] {
		fmt.Fprintf(w, "%*s%s\n", len(leader), " ", line)
	}
}

// Printf writes a message to Kong.Stdout with the application name prefixed.
func (k *Kong) Printf(format string, args ...interface{}) *Kong {
	formatMultilineMessage(k.Stdout, []string{k.Model.Name}, format, args...)
	return k
}

// Errorf writes a message to Kong.Stderr with the application name prefixed.
func (k *Kong) Errorf(format string, args ...interface{}) *Kong {
	formatMultilineMessage(k.Stderr, []string{k.Model.Name, "error"}, format, args...)
	return k
}

// Fatalf writes a message to Kong.Stderr with the application name prefixed then exits with a non-zero status.
func (k *Kong) Fatalf(format string, args ...interface{}) {
	k.Errorf(format, args...)
	k.Exit(1)
}

// FatalIfErrorf terminates with an error message if err != nil.
func (k *Kong) FatalIfErrorf(err error, args ...interface{}) {
	if err == nil {
		return
	}
	msg := err.Error()
	if len(args) > 0 {
		msg = fmt.Sprintf(args[0].(string), args[1:]...) + ": " + err.Error()
	}
	// Maybe display usage information.
	if err, ok := err.(*ParseError); ok && k.usageOnError {
		options := k.helpOptions
		_ = k.help(options, err.Context)
		fmt.Fprintln(k.Stdout)
	}
	k.Errorf("%s", msg)
	k.Exit(1)
}

// LoadConfig from path using the loader configured via Configuration(loader).
//
// "path" will have ~ and any variables expanded.
func (k *Kong) LoadConfig(path string) (Resolver, error) {
	var err error
	path = ExpandPath(path)
	path, err = interpolate(path, k.vars, nil)
	if err != nil {
		return nil, err
	}
	r, err := os.Open(path) // nolint: gas
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return k.loader(r)
}

func catch(err *error) {
	msg := recover()
	if test, ok := msg.(Error); ok {
		*err = test
	} else if msg != nil {
		panic(msg)
	}
}
