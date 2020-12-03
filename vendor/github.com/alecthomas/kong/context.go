package kong

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Path records the nodes and parsed values from the current command-line.
type Path struct {
	Parent *Node

	// One of these will be non-nil.
	App        *Application
	Positional *Positional
	Flag       *Flag
	Argument   *Argument
	Command    *Command

	// Flags added by this node.
	Flags []*Flag

	// True if this Path element was created as the result of a resolver.
	Resolved bool
}

// Node returns the Node associated with this Path, or nil if Path is a non-Node.
func (p *Path) Node() *Node {
	switch {
	case p.App != nil:
		return p.App.Node

	case p.Argument != nil:
		return p.Argument

	case p.Command != nil:
		return p.Command
	}
	return nil
}

// Context contains the current parse context.
type Context struct {
	*Kong
	// A trace through parsed nodes.
	Path []*Path
	// Original command-line arguments.
	Args []string
	// Error that occurred during trace, if any.
	Error error

	values    map[*Value]reflect.Value // Temporary values during tracing.
	bindings  bindings
	resolvers []Resolver // Extra context-specific resolvers.
	scan      *Scanner
}

// Trace path of "args" through the grammar tree.
//
// The returned Context will include a Path of all commands, arguments, positionals and flags.
//
// This just constructs a new trace. To fully apply the trace you must call Reset(), Resolve(),
// Validate() and Apply().
func Trace(k *Kong, args []string) (*Context, error) {
	c := &Context{
		Kong: k,
		Args: args,
		Path: []*Path{
			{App: k.Model, Flags: k.Model.Flags},
		},
		values:   map[*Value]reflect.Value{},
		scan:     Scan(args...),
		bindings: bindings{},
	}
	c.Error = c.trace(c.Model.Node)
	return c, nil
}

// Bind adds bindings to the Context.
func (c *Context) Bind(args ...interface{}) {
	c.bindings.add(args...)
}

// BindTo adds a binding to the Context.
//
// This will typically have to be called like so:
//
//    BindTo(impl, (*MyInterface)(nil))
func (c *Context) BindTo(impl, iface interface{}) {
	valueOf := reflect.ValueOf(impl)
	c.bindings[reflect.TypeOf(iface).Elem()] = func() (reflect.Value, error) { return valueOf, nil }
}

// Value returns the value for a particular path element.
func (c *Context) Value(path *Path) reflect.Value {
	switch {
	case path.Positional != nil:
		return c.values[path.Positional]
	case path.Flag != nil:
		return c.values[path.Flag.Value]
	case path.Argument != nil:
		return c.values[path.Argument.Argument]
	}
	panic("can only retrieve value for flag, argument or positional")
}

// Selected command or argument.
func (c *Context) Selected() *Node {
	var selected *Node
	for _, path := range c.Path {
		switch {
		case path.Command != nil:
			selected = path.Command
		case path.Argument != nil:
			selected = path.Argument
		}
	}
	return selected
}

// Empty returns true if there were no arguments provided.
func (c *Context) Empty() bool {
	for _, path := range c.Path {
		if !path.Resolved && path.App == nil {
			return false
		}
	}
	return true
}

// Validate the current context.
func (c *Context) Validate() error { // nolint: gocyclo
	err := Visit(c.Model, func(node Visitable, next Next) error {
		if value, ok := node.(*Value); ok {
			if value.Enum != "" && (!value.Required || value.Default != "") {
				if err := checkEnum(value, value.Target); err != nil {
					return err
				}
			}
		}
		return next(nil)
	})
	if err != nil {
		return err
	}
	for _, resolver := range c.combineResolvers() {
		if err := resolver.Validate(c.Model); err != nil {
			return err
		}
	}
	for _, path := range c.Path {
		var value *Value
		switch {
		case path.Flag != nil:
			value = path.Flag.Value

		case path.Positional != nil:
			value = path.Positional
		}
		if value != nil && value.Tag.Enum != "" {
			if err := checkEnum(value, value.Target); err != nil {
				return err
			}
		}
		if err := checkMissingFlags(path.Flags); err != nil {
			return err
		}
	}
	// Check the terminal node.
	node := c.Selected()
	if node == nil {
		node = c.Model.Node
	}

	// Find deepest positional argument so we can check if all required positionals have been provided.
	positionals := 0
	for _, path := range c.Path {
		if path.Positional != nil {
			positionals = path.Positional.Position + 1
		}
	}

	if err := checkMissingChildren(node); err != nil {
		return err
	}
	if err := checkMissingPositionals(positionals, node.Positional); err != nil {
		return err
	}
	if err := checkXorDuplicates(c.Path); err != nil {
		return err
	}

	if node.Type == ArgumentNode {
		value := node.Argument
		if value.Required && !value.Set {
			return fmt.Errorf("%s is required", node.Summary())
		}
	}
	return nil
}

// Flags returns the accumulated available flags.
func (c *Context) Flags() (flags []*Flag) {
	for _, trace := range c.Path {
		flags = append(flags, trace.Flags...)
	}
	return
}

// Command returns the full command path.
func (c *Context) Command() string {
	command := []string{}
	for _, trace := range c.Path {
		switch {
		case trace.Positional != nil:
			command = append(command, "<"+trace.Positional.Name+">")

		case trace.Argument != nil:
			command = append(command, "<"+trace.Argument.Name+">")

		case trace.Command != nil:
			command = append(command, trace.Command.Name)
		}
	}
	return strings.Join(command, " ")
}

// AddResolver adds a context-specific resolver.
//
// This is most useful in the BeforeResolve() hook.
func (c *Context) AddResolver(resolver Resolver) {
	c.resolvers = append(c.resolvers, resolver)
}

// FlagValue returns the set value of a flag if it was encountered and exists, or its default value.
func (c *Context) FlagValue(flag *Flag) interface{} {
	for _, trace := range c.Path {
		if trace.Flag == flag {
			v, ok := c.values[trace.Flag.Value]
			if !ok {
				break
			}
			return v.Interface()
		}
	}
	if flag.Target.IsValid() {
		return flag.Target.Interface()
	}
	return flag.DefaultValue.Interface()
}

// Reset recursively resets values to defaults (as specified in the grammar) or the zero value.
func (c *Context) Reset() error {
	return Visit(c.Model.Node, func(node Visitable, next Next) error {
		if value, ok := node.(*Value); ok {
			return next(value.Reset())
		}
		return next(nil)
	})
}

func (c *Context) trace(node *Node) (err error) { // nolint: gocyclo
	positional := 0

	flags := []*Flag{}
	for _, group := range node.AllFlags(false) {
		flags = append(flags, group...)
	}

	for !c.scan.Peek().IsEOL() {
		token := c.scan.Peek()
		switch token.Type {
		case UntypedToken:
			switch v := token.Value.(type) {
			case string:

				switch {
				case v == "-":
					fallthrough
				default: // nolint
					c.scan.Pop()
					c.scan.PushTyped(token.Value, PositionalArgumentToken)

				// Indicates end of parsing. All remaining arguments are treated as positional arguments only.
				case v == "--":
					c.scan.Pop()
					args := []string{}
					for {
						token = c.scan.Pop()
						if token.Type == EOLToken {
							break
						}
						args = append(args, token.String())
					}
					// Note: tokens must be pushed in reverse order.
					for i := range args {
						c.scan.PushTyped(args[len(args)-1-i], PositionalArgumentToken)
					}

				// Long flag.
				case strings.HasPrefix(v, "--"):
					c.scan.Pop()
					// Parse it and push the tokens.
					parts := strings.SplitN(v[2:], "=", 2)
					if len(parts) > 1 {
						c.scan.PushTyped(parts[1], FlagValueToken)
					}
					c.scan.PushTyped(parts[0], FlagToken)

				// Short flag.
				case strings.HasPrefix(v, "-"):
					c.scan.Pop()
					// Note: tokens must be pushed in reverse order.
					if tail := v[2:]; tail != "" {
						c.scan.PushTyped(tail, ShortFlagTailToken)
					}
					c.scan.PushTyped(v[1:2], ShortFlagToken)
				}
			default:
				c.scan.Pop()
				c.scan.PushTyped(token.Value, PositionalArgumentToken)
			}

		case ShortFlagTailToken:
			c.scan.Pop()
			// Note: tokens must be pushed in reverse order.
			if tail := token.String()[1:]; tail != "" {
				c.scan.PushTyped(tail, ShortFlagTailToken)
			}
			c.scan.PushTyped(token.String()[0:1], ShortFlagToken)

		case FlagToken:
			if err := c.parseFlag(flags, token.String()); err != nil {
				return err
			}

		case ShortFlagToken:
			if err := c.parseFlag(flags, token.String()); err != nil {
				return err
			}

		case FlagValueToken:
			return fmt.Errorf("unexpected flag argument %q", token.Value)

		case PositionalArgumentToken:
			candidates := []string{}

			// Ensure we've consumed all positional arguments.
			if positional < len(node.Positional) {
				arg := node.Positional[positional]
				err := arg.Parse(c.scan, c.getValue(arg))
				if err != nil {
					return err
				}
				c.Path = append(c.Path, &Path{
					Parent:     node,
					Positional: arg,
				})
				positional++
				break
			}

			// After positional arguments have been consumed, check commands next...
			for _, branch := range node.Children {
				if branch.Type == CommandNode && !branch.Hidden {
					candidates = append(candidates, branch.Name)
				}
				if branch.Type == CommandNode && branch.Name == token.Value {
					c.scan.Pop()
					c.Path = append(c.Path, &Path{
						Parent:  node,
						Command: branch,
						Flags:   branch.Flags,
					})
					return c.trace(branch)
				}
			}

			// Finally, check arguments.
			for _, branch := range node.Children {
				if branch.Type == ArgumentNode {
					arg := branch.Argument
					if err := arg.Parse(c.scan, c.getValue(arg)); err == nil {
						c.Path = append(c.Path, &Path{
							Parent:   node,
							Argument: branch,
							Flags:    branch.Flags,
						})
						return c.trace(branch)
					}
				}
			}

			return findPotentialCandidates(token.String(), candidates, "unexpected argument %s", token)
		default:
			return fmt.Errorf("unexpected token %s", token)
		}
	}
	return c.maybeSelectDefault(flags, node)
}

// End of the line, check for a default command, but only if we're not displaying help,
// otherwise we'd only ever display the help for the default command.
func (c *Context) maybeSelectDefault(flags []*Flag, node *Node) error {
	for _, flag := range flags {
		if flag.Name == "help" && flag.Set {
			return nil
		}
	}
	var defaultNode *Path
	for _, child := range node.Children {
		if child.Type == CommandNode && child.Tag.Default != "" {
			if defaultNode != nil {
				return fmt.Errorf("can't have more than one default command under %s", node.Summary())
			}
			defaultNode = &Path{
				Parent:  child,
				Command: child,
				Flags:   child.Flags,
			}
		}
	}
	if defaultNode != nil {
		c.Path = append(c.Path, defaultNode)
	}
	return nil
}

// Resolve walks through the traced path, applying resolvers to any unset flags.
func (c *Context) Resolve() error {
	resolvers := c.combineResolvers()
	if len(resolvers) == 0 {
		return nil
	}

	inserted := []*Path{}
	for _, path := range c.Path {
		for _, flag := range path.Flags {
			// Flag has already been set on the command-line.
			if _, ok := c.values[flag.Value]; ok {
				continue
			}

			// Pick the last resolved value.
			var selected interface{}
			for _, resolver := range resolvers {
				s, err := resolver.Resolve(c, path, flag)
				if err != nil {
					return errors.Wrap(err, flag.ShortSummary())
				}
				if s == nil {
					continue
				}
				selected = s
			}

			if selected == nil {
				continue
			}

			scan := Scan().PushTyped(selected, FlagValueToken)
			delete(c.values, flag.Value)
			err := flag.Parse(scan, c.getValue(flag.Value))
			if err != nil {
				return err
			}
			inserted = append(inserted, &Path{
				Flag:     flag,
				Resolved: true,
			})
		}
	}
	c.Path = append(inserted, c.Path...)
	return nil
}

// Combine application-level resolvers and context resolvers.
func (c *Context) combineResolvers() []Resolver {
	resolvers := []Resolver{}
	resolvers = append(resolvers, c.Kong.resolvers...)
	resolvers = append(resolvers, c.resolvers...)
	return resolvers
}

func (c *Context) getValue(value *Value) reflect.Value {
	v, ok := c.values[value]
	if !ok {
		v = reflect.New(value.Target.Type()).Elem()
		c.values[value] = v
	}
	return v
}

// ApplyDefaults if they are not already set.
func (c *Context) ApplyDefaults() error {
	return Visit(c.Model.Node, func(node Visitable, next Next) error {
		var value *Value
		switch node := node.(type) {
		case *Flag:
			value = node.Value
		case *Node:
			value = node.Argument
		case *Value:
			value = node
		default:
		}
		if value != nil {
			if err := value.ApplyDefault(); err != nil {
				return err
			}
		}
		return next(nil)
	})
}

// Apply traced context to the target grammar.
func (c *Context) Apply() (string, error) {
	path := []string{}

	for _, trace := range c.Path {
		var value *Value
		switch {
		case trace.App != nil:
		case trace.Argument != nil:
			path = append(path, "<"+trace.Argument.Name+">")
			value = trace.Argument.Argument
		case trace.Command != nil:
			path = append(path, trace.Command.Name)
		case trace.Flag != nil:
			value = trace.Flag.Value
		case trace.Positional != nil:
			path = append(path, "<"+trace.Positional.Name+">")
			value = trace.Positional
		default:
			panic("unsupported path ?!")
		}
		if value != nil {
			value.Apply(c.getValue(value))
		}
	}

	return strings.Join(path, " "), nil
}

func (c *Context) parseFlag(flags []*Flag, match string) (err error) {
	defer catch(&err)
	candidates := []string{}
	for _, flag := range flags {
		long := "--" + flag.Name
		short := "-" + string(flag.Short)
		candidates = append(candidates, long)
		if flag.Short != 0 {
			candidates = append(candidates, short)
		}
		if short != match && long != match {
			continue
		}
		// Found a matching flag.
		c.scan.Pop()
		err := flag.Parse(c.scan, c.getValue(flag.Value))
		if err != nil {
			if e, ok := errors.Cause(err).(*expectedError); ok && e.token.InferredType().IsAny(FlagToken, ShortFlagToken) {
				return errors.Errorf("%s; perhaps try %s=%q?", err, flag.ShortSummary(), e.token)
			}
			return err
		}
		c.Path = append(c.Path, &Path{Flag: flag})
		return nil
	}
	return findPotentialCandidates(match, candidates, "unknown flag %s", match)
}

// RunNode calls the Run() method on an arbitrary node.
//
// This is useful in conjunction with Visit(), for dynamically running commands.
//
// Any passed values will be bindable to arguments of the target Run() method. Additionally,
// all parent nodes in the command structure will be bound.
func (c *Context) RunNode(node *Node, binds ...interface{}) (err error) {
	type targetMethod struct {
		node   *Node
		method reflect.Value
		binds  bindings
	}
	methodBinds := c.Kong.bindings.clone().add(binds...).add(c).merge(c.bindings)
	methods := []targetMethod{}
	for i := 0; node != nil; i, node = i+1, node.Parent {
		method := getMethod(node.Target, "Run")
		methodBinds = methodBinds.clone()
		for p := node; p != nil; p = p.Parent {
			methodBinds = methodBinds.add(p.Target.Addr().Interface())
		}
		if method.IsValid() {
			methods = append(methods, targetMethod{node, method, methodBinds})
		}
	}
	if len(methods) == 0 {
		return fmt.Errorf("no Run() method found in hierarchy of %s", c.Selected().Summary())
	}
	_, err = c.Apply()
	if err != nil {
		return err
	}

	for _, method := range methods {
		if err = callMethod("Run", method.node.Target, method.method, method.binds); err != nil {
			return err
		}
	}
	return nil
}

// Run executes the Run() method on the selected command, which must exist.
//
// Any passed values will be bindable to arguments of the target Run() method. Additionally,
// all parent nodes in the command structure will be bound.
func (c *Context) Run(binds ...interface{}) (err error) {
	defer catch(&err)
	node := c.Selected()
	if node == nil {
		return fmt.Errorf("no command selected")
	}
	return c.RunNode(node, binds...)
}

// PrintUsage to Kong's stdout.
//
// If summary is true, a summarised version of the help will be output.
func (c *Context) PrintUsage(summary bool) error {
	options := c.helpOptions
	options.Summary = summary
	_ = c.help(options, c)
	return nil
}

func checkMissingFlags(flags []*Flag) error {
	missing := []string{}
	for _, flag := range flags {
		if !flag.Required || flag.Set {
			continue
		}
		missing = append(missing, flag.Summary())
	}
	if len(missing) == 0 {
		return nil
	}

	return fmt.Errorf("missing flags: %s", strings.Join(missing, ", "))
}

func checkMissingChildren(node *Node) error {
	missing := []string{}

	missingArgs := []string{}
	for _, arg := range node.Positional {
		if arg.Required && !arg.Set {
			missingArgs = append(missingArgs, arg.Summary())
		}
	}
	if len(missingArgs) > 0 {
		missing = append(missing, strconv.Quote(strings.Join(missingArgs, " ")))
	}

	haveDefault := 0
	for _, child := range node.Children {
		if child.Hidden {
			continue
		}
		if child.Argument != nil {
			if !child.Argument.Required {
				continue
			}
			missing = append(missing, strconv.Quote(child.Summary()))
		} else {
			if child.Tag.Default != "" {
				if len(child.Children) > 0 {
					return fmt.Errorf("default command %s must not have subcommands or arguments", child.Summary())
				}
				haveDefault++
			}
			missing = append(missing, strconv.Quote(child.Name))
		}
	}
	if haveDefault > 1 {
		return fmt.Errorf("more than one default command found under %s", node.Summary())
	}
	if len(missing) == 0 || haveDefault > 0 {
		return nil
	}

	if len(missing) > 5 {
		missing = append(missing[:5], "...")
	}
	if len(missing) == 1 {
		return fmt.Errorf("expected %s", missing[0])
	}
	return fmt.Errorf("expected one of %s", strings.Join(missing, ",  "))
}

// If we're missing any positionals and they're required, return an error.
func checkMissingPositionals(positional int, values []*Value) error {
	// All the positionals are in.
	if positional >= len(values) {
		return nil
	}

	// We're low on supplied positionals, but the missing one is optional.
	if !values[positional].Required {
		return nil
	}

	missing := []string{}
	for ; positional < len(values); positional++ {
		missing = append(missing, "<"+values[positional].Name+">")
	}
	return fmt.Errorf("missing positional arguments %s", strings.Join(missing, " "))
}

func checkEnum(value *Value, target reflect.Value) error {
	switch target.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < target.Len(); i++ {
			if err := checkEnum(value, target.Index(i)); err != nil {
				return err
			}
		}
		return nil

	case reflect.Map, reflect.Struct:
		return errors.Errorf("enum can only be applied to a slice or value")

	default:
		enumMap := value.EnumMap()
		v := fmt.Sprintf("%v", target)
		if enumMap[v] {
			return nil
		}
		enums := []string{}
		for enum := range enumMap {
			enums = append(enums, fmt.Sprintf("%q", enum))
		}
		sort.Strings(enums)
		return fmt.Errorf("%s must be one of %s but got %q", value.ShortSummary(), strings.Join(enums, ","), target.Interface())
	}
}

func checkXorDuplicates(paths []*Path) error {
	for _, path := range paths {
		seen := map[string]*Flag{}
		for _, flag := range path.Flags {
			if !flag.Set {
				continue
			}
			if flag.Xor == "" {
				continue
			}
			if seen[flag.Xor] != nil {
				return fmt.Errorf("--%s and --%s can't be used together", seen[flag.Xor].Name, flag.Name)
			}
			seen[flag.Xor] = flag
		}
	}
	return nil
}

func findPotentialCandidates(needle string, haystack []string, format string, args ...interface{}) error {
	if len(haystack) == 0 {
		return fmt.Errorf(format, args...)
	}
	closestCandidates := []string{}
	for _, candidate := range haystack {
		if strings.HasPrefix(candidate, needle) || levenshtein(candidate, needle) <= 2 {
			closestCandidates = append(closestCandidates, fmt.Sprintf("%q", candidate))
		}
	}
	prefix := fmt.Sprintf(format, args...)
	if len(closestCandidates) == 1 {
		return fmt.Errorf("%s, did you mean %s?", prefix, closestCandidates[0])
	} else if len(closestCandidates) > 1 {
		return fmt.Errorf("%s, did you mean one of %s?", prefix, strings.Join(closestCandidates, ", "))
	}
	return fmt.Errorf("%s", prefix)
}
