package kong

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
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

	// Remaining tokens after this node
	remainder []Token
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

// Visitable returns the Visitable for this path element.
func (p *Path) Visitable() Visitable {
	switch {
	case p.App != nil:
		return p.App

	case p.Argument != nil:
		return p.Argument

	case p.Command != nil:
		return p.Command

	case p.Flag != nil:
		return p.Flag

	case p.Positional != nil:
		return p.Positional
	}
	return nil
}

// Remainder returns the remaining unparsed args after this Path element.
func (p *Path) Remainder() []string {
	args := []string{}
	for _, token := range p.remainder {
		args = append(args, token.String())
	}
	return args
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

	values        map[*Value]reflect.Value // Temporary values during tracing.
	resetErrors   map[*Value]error         // Default errors that a resolver may override.
	bindings      bindings
	resolvers     []Resolver // Extra context-specific resolvers.
	scan          *Scanner
	lastWinsPeers map[*Flag][]*Flag // Other members of each flag's lastwins group.
}

// Trace path of "args" through the grammar tree.
//
// The returned Context will include a Path of all commands, arguments, positionals and flags.
//
// This just constructs a new trace. To fully apply the trace you must call Reset(), Resolve(),
// Validate() and Apply().
func Trace(k *Kong, args []string) (*Context, error) {
	s := Scan(args...).AllowHyphenPrefixedParameters(k.allowHyphenated)
	c := &Context{
		Kong: k,
		Args: args,
		Path: []*Path{
			{App: k.Model, Flags: k.Model.Flags, remainder: s.PeekAll()},
		},
		values:        map[*Value]reflect.Value{},
		scan:          s,
		bindings:      bindings{},
		lastWinsPeers: lastWinsPeers(k.Model.Node),
	}
	c.Error = c.trace(c.Model.Node)
	if c.Error == nil {
		c.filterLastWins()
	}
	return c, nil
}

// filterLastWins removes trace occurrences overridden by a later occurrence of
// another flag in the same lastwins group, as if they had never been given.
// Their values were already discarded during the trace (see parseFlag).
func (c *Context) filterLastWins() {
	if len(c.lastWinsPeers) == 0 {
		return
	}
	drop := map[*Path]bool{}
	visited := map[*Flag]bool{}
	for _, path := range c.Path {
		flag := path.Flag
		if flag == nil || visited[flag] {
			continue
		}
		peers, ok := c.lastWinsPeers[flag]
		if !ok {
			continue
		}
		members := map[*Flag]bool{flag: true}
		visited[flag] = true
		for _, peer := range peers {
			members[peer] = true
			visited[peer] = true
		}
		occurrences := []*Path{}
		for _, occurrence := range c.Path {
			if occurrence.Flag != nil && members[occurrence.Flag] {
				occurrences = append(occurrences, occurrence)
			}
		}
		// Only the trailing run of occurrences of a single flag survives:
		// everything before the last occurrence of any other member was
		// overridden.
		winner := occurrences[len(occurrences)-1].Flag
		overridden := 0
		for i, occurrence := range occurrences {
			if occurrence.Flag != winner {
				overridden = i + 1
			}
		}
		for _, occurrence := range occurrences[:overridden] {
			drop[occurrence] = true
		}
	}
	if len(drop) == 0 {
		return
	}
	path := c.Path[:0]
	for _, trace := range c.Path {
		if !drop[trace] {
			path = append(path, trace)
		}
	}
	c.Path = path
}

// lastWinsPeers maps each member of a lastwins group to the other members of
// its group. Groups are scoped to the node their flags are declared on.
func lastWinsPeers(node *Node) map[*Flag][]*Flag {
	peers := map[*Flag][]*Flag{}
	_ = Visit(node, func(v Visitable, next Next) error {
		if n, ok := v.(*Node); ok {
			groups := map[string][]*Flag{}
			for _, flag := range n.Flags {
				for _, group := range flag.LastWins {
					groups[group] = append(groups[group], flag)
				}
			}
			for _, members := range groups {
				if len(members) < 2 {
					continue
				}
				for _, member := range members {
					others := make([]*Flag, 0, len(members)-1)
					for _, other := range members {
						if other != member {
							others = append(others, other)
						}
					}
					peers[member] = others
				}
			}
		}
		return next(nil)
	})
	return peers
}

// Bind adds bindings to the Context.
func (c *Context) Bind(args ...any) {
	c.bindings.add(args...)
}

// BindTo adds a binding to the Context.
//
// This will typically have to be called like so:
//
//	BindTo(impl, (*MyInterface)(nil))
func (c *Context) BindTo(impl, iface any) {
	c.bindings.addTo(impl, iface)
}

// BindToProvider allows binding of provider functions.
//
// This is useful when the Run() function of different commands require different values that may
// not all be initialisable from the main() function.
//
// "provider" must be a function with the signature func(...) (T, error) or func(...) T,
// where ... will be recursively injected with bound values.
func (c *Context) BindToProvider(provider any) error {
	return c.bindings.addProvider(provider, false /* singleton */)
}

// BindSingletonProvider allows binding of provider functions.
// The provider will be called once and the result cached.
//
// "provider" must be a function with the signature func(...) (T, error) or func(...) T,
// where ... will be recursively injected with bound values.
func (c *Context) BindSingletonProvider(provider any) error {
	return c.bindings.addProvider(provider, true /* singleton */)
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
func (c *Context) Validate() error { //nolint: gocyclo
	// Only check nodes on the selected command path: an envar shared with
	// another command may hold a value that is invalid there.
	for _, path := range c.Path {
		node := path.Node()
		if node == nil {
			continue
		}
		for _, value := range node.Values() {
			ok := atLeastOneEnvSet(value.Tag.Envs)
			if value.Enum != "" && (!value.Required || value.HasDefault || (len(value.Tag.Envs) != 0 && ok)) {
				if err := checkEnum(value, value.Target); err != nil {
					return err
				}
			}
		}
	}
	for _, el := range c.Path {
		var (
			value reflect.Value
			desc  string
		)
		switch node := el.Visitable().(type) {
		case *Value:
			value = node.Target
			desc = node.ShortSummary()

		case *Flag:
			value = node.Target
			desc = node.ShortSummary()

		case *Application:
			value = node.Target
			desc = ""

		case *Node:
			value = node.Target
			desc = node.Path()
		}
		for _, validate := range getValidators(value) {
			if err := validate.Validate(c); err != nil {
				if desc != "" {
					return fmt.Errorf("%s: %w", desc, err)
				}
				return err
			}
		}
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
	if err := checkXorDuplicatedAndAndMissing(c.Path); err != nil {
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
func (c *Context) FlagValue(flag *Flag) any {
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
	selected := c.selectedValues()
	c.resetErrors = map[*Value]error{}
	return Visit(c.Model.Node, func(node Visitable, next Next) error {
		value, ok := node.(*Value)
		if !ok {
			return next(nil)
		}
		err := value.Reset()
		if err != nil && !selected[value] {
			// An envar shared with a node outside the selected command path
			// may not parse there; that must not fail this parse.
			value.Target.Set(reflect.Zero(value.Target.Type()))
			err = nil
		}
		if err != nil && len(c.combineResolvers()) != 0 {
			// A resolver may supply a valid value after defaults are reset.
			// Keep the error so Resolve can return it if no value overrides it.
			c.resetErrors[value] = err
			err = nil
		}
		return next(err)
	})
}

// selectedValues returns the set of values attached to nodes on the traced
// command path.
func (c *Context) selectedValues() map[*Value]bool {
	selected := map[*Value]bool{}
	for _, path := range c.Path {
		if node := path.Node(); node != nil {
			for _, value := range node.Values() {
				selected[value] = true
			}
		}
	}
	return selected
}

func (c *Context) endParsing() {
	args := []string{}
	for {
		token := c.scan.Pop()
		if token.Type == EOLToken {
			break
		}
		args = append(args, token.String())
	}
	// Note: tokens must be pushed in reverse order.
	for i := range args {
		c.scan.PushTyped(args[len(args)-1-i], PositionalArgumentToken)
	}
}

//nolint:maintidx
func (c *Context) trace(node *Node) (err error) { //nolint: gocyclo
	positional := 0
	node.Active = true

	flags := []*Flag{}
	flagNode := node
	if node.DefaultCmd != nil && node.DefaultCmd.Tag.Default == "withargs" {
		// Add flags of the default command if the current node has one
		// and that default command allows args / flags without explicitly
		// naming the command on the CLI.
		flagNode = node.DefaultCmd
	}
	for _, group := range flagNode.AllFlags(false) {
		flags = append(flags, group...)
	}

	if node.Passthrough {
		c.endParsing()
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
				default: //nolint
					c.scan.Pop()
					c.scan.PushTyped(token.Value, PositionalArgumentToken)

				// Indicates end of parsing. All remaining arguments are treated as positional arguments only.
				case v == "--":
					c.endParsing()

					// Pop the -- token unless the next positional argument accepts passthrough arguments.
					if !(positional < len(node.Positional) && node.Positional[positional].Passthrough) {
						c.scan.Pop()
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
				if isUnknownFlagError(err) && positional < len(node.Positional) && node.Positional[positional].PassthroughMode == PassThroughModeAll {
					c.scan.Pop()
					c.scan.PushTyped(token.String(), PositionalArgumentToken)
				} else {
					return err
				}
			}

		case ShortFlagToken:
			if err := c.parseFlag(flags, token.String()); err != nil {
				if isUnknownFlagError(err) && positional < len(node.Positional) && node.Positional[positional].PassthroughMode == PassThroughModeAll {
					c.scan.Pop()
					c.scan.PushTyped(token.String(), PositionalArgumentToken)
				} else {
					return err
				}
			}

		case FlagValueToken:
			return fmt.Errorf("unexpected flag argument %q", token.Value)

		case PositionalArgumentToken:
			candidates := []string{}

			// Ensure we've consumed all positional arguments.
			if positional < len(node.Positional) {
				arg := node.Positional[positional]

				if arg.Passthrough {
					c.endParsing()
				}

				arg.Active = true
				err := arg.Parse(c.scan, c.getValue(arg))
				if err != nil {
					return err
				}
				c.Path = append(c.Path, &Path{
					Parent:     node,
					Positional: arg,
					remainder:  c.scan.PeekAll(),
				})
				positional++
				break
			}

			// Assign token value to a branch name if tagged as an alias
			// An alias will be ignored in the case of an existing command
			cmds := make(map[string]bool)
			for _, branch := range node.Children {
				if branch.Type == CommandNode {
					cmds[branch.Name] = true
				}
			}
			for _, branch := range node.Children {
				for _, a := range branch.Aliases {
					_, ok := cmds[a]
					if token.Value == a && !ok {
						token.Value = branch.Name
						break
					}
				}
			}

			// After positional arguments have been consumed, check commands next...
			for _, branch := range node.Children {
				if branch.Type == CommandNode && !branch.Hidden {
					candidates = append(candidates, branch.Name)
				}
				if branch.Type == CommandNode && branch.Name == token.Value {
					if err := c.checkDefaultCmdFlags(node, branch); err != nil {
						return err
					}
					c.scan.Pop()
					c.Path = append(c.Path, &Path{
						Parent:    node,
						Command:   branch,
						Flags:     branch.Flags,
						remainder: c.scan.PeekAll(),
					})
					return c.trace(branch)
				}
			}

			// Finally, check arguments.
			for _, branch := range node.Children {
				if branch.Type == ArgumentNode {
					arg := branch.Argument
					if err := arg.Parse(c.scan, c.getValue(arg)); err == nil {
						if err := c.checkDefaultCmdFlags(node, branch); err != nil {
							return err
						}
						c.Path = append(c.Path, &Path{
							Parent:    node,
							Argument:  branch,
							Flags:     branch.Flags,
							remainder: c.scan.PeekAll(),
						})
						return c.trace(branch)
					}
				}
			}

			// If there is a default command that allows args and nothing else
			// matches, take the branch of the default command
			if node.DefaultCmd != nil && node.DefaultCmd.Tag.Default == "withargs" {
				c.Path = append(c.Path, &Path{
					Parent:    node,
					Command:   node.DefaultCmd,
					Flags:     node.DefaultCmd.Flags,
					remainder: c.scan.PeekAll(),
				})
				return c.trace(node.DefaultCmd)
			}

			return findPotentialCandidates(token.String(), candidates, "unexpected argument %s", token)
		default:
			return fmt.Errorf("unexpected token %s", token)
		}
	}
	return c.maybeSelectDefault(flags, node)
}

// checkDefaultCmdFlags rejects flags that were parsed into node's "withargs"
// default command when a different child, selected, has been chosen instead.
//
// trace() offers the default command's flags while parsing node's own tokens,
// so a flag given before an explicitly named sibling is parsed into the default
// command. The default command is then never selected, so without this check
// the value would be silently dropped. The value is not moved to a same-named
// flag on selected because the two flags may differ in type or meaning.
func (c *Context) checkDefaultCmdFlags(node, selected *Node) error {
	defaultCmd := node.DefaultCmd
	if defaultCmd == nil || defaultCmd.Tag.Default != "withargs" || selected == defaultCmd {
		return nil
	}
	defaultFlags := map[*Flag]bool{}
	for _, flag := range defaultCmd.Flags {
		defaultFlags[flag] = true
	}
	for _, path := range c.Path {
		if path.Flag != nil && defaultFlags[path.Flag] {
			return fmt.Errorf("flag --%s applies to the default command %q, not %q", path.Flag.Name, defaultCmd.Name, selected.Name)
		}
	}
	return nil
}

// IgnoreDefault can be implemented by flags that want to be applied before any default commands.
type IgnoreDefault interface {
	IgnoreDefault()
}

// End of the line, check for a default command, but only if we're not displaying help,
// otherwise we'd only ever display the help for the default command.
func (c *Context) maybeSelectDefault(flags []*Flag, node *Node) error {
	for _, flag := range flags {
		if _, ok := flag.Target.Interface().(IgnoreDefault); ok && flag.Set {
			return nil
		}
	}
	if node.DefaultCmd != nil {
		c.Path = append(c.Path, &Path{
			Parent:    node.DefaultCmd,
			Command:   node.DefaultCmd,
			Flags:     node.DefaultCmd.Flags,
			remainder: c.scan.PeekAll(),
		})
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
			var selected any
			for _, resolver := range resolvers {
				s, err := resolver.Resolve(c, path, flag)
				if err != nil {
					return fmt.Errorf("%s: %w", flag.ShortSummary(), err)
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
				Flag:      flag,
				Resolved:  true,
				remainder: c.scan.PeekAll(),
			})
		}
	}
	c.Path = append(c.Path, inserted...)
	for value, err := range c.resetErrors {
		if _, ok := c.values[value]; !ok {
			return err
		}
	}
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
		switch v.Kind() {
		case reflect.Ptr:
			v.Set(reflect.New(v.Type().Elem()))
		case reflect.Slice:
			v.Set(reflect.MakeSlice(v.Type(), 0, 0))
		case reflect.Map:
			v.Set(reflect.MakeMap(v.Type()))
		default:
		}
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
	candidates := []string{}

	for _, flag := range flags {
		long := "--" + flag.Name
		matched := long == match
		candidates = append(candidates, long)
		if flag.Short != 0 {
			short := "-" + string(flag.Short)
			matched = matched || (short == match)
			candidates = append(candidates, short)
		}
		for _, alias := range flag.Aliases {
			aliasFlag := "--" + alias
			if utf8.RuneCountInString(alias) == 1 {
				aliasFlag = "-" + alias
			}
			matched = matched || (aliasFlag == match)
			candidates = append(candidates, aliasFlag)
		}

		neg := negatableFlagName(flag.Name, flag.Tag.Negatable)
		if !matched && match != neg {
			continue
		}
		// Found a matching flag.
		c.scan.Pop()
		flag.Negated = match == neg && flag.Tag.Negatable != ""
		if flag.Negated {
			negatedValue := true
			if c.scan.Peek().Type == FlagValueToken {
				token, err := c.scan.PopValue("bool")
				if err != nil {
					return err
				}
				negatedValue, err = parseBoolToken(token)
				if err != nil {
					return err
				}
			}
			// Invert the boolean token before Decode so custom bool mappers
			// keep ownership of their field representation.
			c.scan.PushTyped(!negatedValue, FlagValueToken)
		}
		// An occurrence of a lastwins group member overrides the accumulated
		// state of the other members, as if they had never been given.
		for _, peer := range c.lastWinsPeers[flag] {
			peer.Set = false
			delete(c.values, peer.Value)
		}
		err := flag.Parse(c.scan, c.getValue(flag.Value))
		if err != nil {
			var expected *expectedError
			if errors.As(err, &expected) && expected.token.InferredType().IsAny(FlagToken, ShortFlagToken) {
				return fmt.Errorf("%s; perhaps try %s=%q?", err.Error(), flag.ShortSummary(), expected.token)
			}
			return err
		}
		c.Path = append(c.Path, &Path{
			Flag:      flag,
			remainder: c.scan.PeekAll(),
		})
		return nil
	}
	return &unknownFlagError{Cause: findPotentialCandidates(match, candidates, "unknown flag %s", match)}
}

func isUnknownFlagError(err error) bool {
	var unknown *unknownFlagError
	return errors.As(err, &unknown)
}

type unknownFlagError struct{ Cause error }

func (e *unknownFlagError) Unwrap() error { return e.Cause }
func (e *unknownFlagError) Error() string { return e.Cause.Error() }

// Call an arbitrary function filling arguments with bound values.
func (c *Context) Call(fn any, binds ...any) (out []any, err error) {
	fv := reflect.ValueOf(fn)
	bindings := c.Kong.bindings.clone().add(binds...).add(c).merge(c.bindings)
	return callAnyFunction(fv, bindings)
}

// RunNode calls the Run() method on an arbitrary node.
//
// This is useful in conjunction with Visit(), for dynamically running commands.
//
// Any passed values will be bindable to arguments of the target Run() method. Additionally,
// all parent nodes in the command structure will be bound.
func (c *Context) RunNode(node *Node, binds ...any) (err error) {
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
			// Try value and pointer to value.
			for _, p := range []reflect.Value{p.Target, p.Target.Addr()} {
				t := p.Type()
				for i := 0; i < p.NumMethod(); i++ {
					methodt := t.Method(i)
					if strings.HasPrefix(methodt.Name, "Provide") {
						method := p.Method(i)
						if err := methodBinds.addProvider(method.Interface(), false /* singleton */); err != nil {
							return fmt.Errorf("%s.%s: %w", t.Name(), methodt.Name, err)
						}
					}
				}
			}
		}
		if method.IsValid() {
			methods = append(methods, targetMethod{node, method, methodBinds})
		}
	}
	if len(methods) == 0 {
		return fmt.Errorf("no Run() method found in hierarchy of %s", c.Selected().Summary())
	}
	for _, method := range methods {
		if err = callFunction(method.method, method.binds); err != nil {
			return err
		}
	}
	return nil
}

// Run executes the Run() method on the selected command, which must exist.
//
// Any passed values will be bindable to arguments of the target Run() method. Additionally,
// all parent nodes in the command structure will be bound.
func (c *Context) Run(binds ...any) (err error) {
	node := c.Selected()
	if node == nil {
		if len(c.Path) == 0 {
			return fmt.Errorf("no command selected")
		}
		selected := c.Path[0].Node()
		if selected.Type == ApplicationNode {
			method := getMethod(selected.Target, "Run")
			if method.IsValid() {
				node = selected
			}
		}

		if node == nil {
			return fmt.Errorf("no command selected")
		}
	}
	runErr := c.RunNode(node, binds...)
	err = c.Kong.applyHook(c, "AfterRun")
	return errors.Join(runErr, err)
}

// PrintUsage to Kong's stdout.
//
// If summary is true, a summarised version of the help will be output.
func (c *Context) PrintUsage(summary bool) error {
	options := c.helpOptions
	options.Summary = summary
	return c.printHelp(options)
}

func (c *Context) printHelp(options HelpOptions) error {
	options.ValueFormatter = c.Kong.helpFormatter
	return c.help(options, c)
}

// choiceGroup identifies a group of flags of which one is expected to be
// chosen: an xor group or a lastwins group.
type choiceGroup struct {
	kind string
	name string
}

func flagChoiceGroups(flag *Flag) []choiceGroup {
	groups := make([]choiceGroup, 0, len(flag.Xor)+len(flag.LastWins))
	for _, name := range flag.Xor {
		groups = append(groups, choiceGroup{kind: "xor", name: name})
	}
	for _, name := range flag.LastWins {
		groups = append(groups, choiceGroup{kind: "lastwins", name: name})
	}
	return groups
}

func checkMissingFlags(flags []*Flag) error {
	choiceGroupSet := map[choiceGroup]bool{}
	choiceGroups := map[choiceGroup][]string{}
	andGroupSet := map[string]bool{}
	andGroup := map[string][]string{}
	missing := []string{}
	andGroupRequired := getRequiredAndGroupMap(flags)
	for _, flag := range flags {
		for _, and := range flag.And {
			flag.Required = andGroupRequired[and]
		}
		groups := flagChoiceGroups(flag)
		if flag.Set {
			for _, group := range groups {
				choiceGroupSet[group] = true
			}
			for _, and := range flag.And {
				andGroupSet[and] = true
			}
		}
		if !flag.Required || flag.Set {
			continue
		}
		if len(groups) > 0 || len(flag.And) > 0 {
			for _, group := range groups {
				if choiceGroupSet[group] {
					continue
				}
				choiceGroups[group] = append(choiceGroups[group], flag.Summary())
			}
			for _, and := range flag.And {
				andGroup[and] = append(andGroup[and], flag.Summary())
			}
		} else {
			missing = append(missing, flag.Summary())
		}
	}
	for group, flags := range choiceGroups {
		if !choiceGroupSet[group] && len(flags) > 1 {
			missing = append(missing, strings.Join(flags, " or "))
		}
	}
	for _, flags := range andGroup {
		if len(flags) > 1 {
			missing = append(missing, strings.Join(flags, " and "))
		}
	}

	if len(missing) == 0 {
		return nil
	}

	sort.Strings(missing)

	return fmt.Errorf("missing flags: %s", strings.Join(missing, ", "))
}

func getRequiredAndGroupMap(flags []*Flag) map[string]bool {
	andGroupRequired := map[string]bool{}
	for _, flag := range flags {
		for _, and := range flag.And {
			if flag.Required {
				andGroupRequired[and] = true
			}
		}
	}
	return andGroupRequired
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

	// A node with a Run() method may run on its own, so it does not require one
	// of its subcommands to be selected.
	runnable := node.Target.IsValid() && getMethod(node.Target, "Run").IsValid()

	for _, child := range node.Children {
		if child.Hidden {
			continue
		}
		if child.Argument != nil {
			if !child.Argument.Required {
				continue
			}
			missing = append(missing, strconv.Quote(child.Summary()))
		} else if !runnable {
			missing = append(missing, strconv.Quote(child.Name))
		}
	}
	if len(missing) == 0 {
		return nil
	}

	if len(missing) > 5 {
		missing = append(missing[:5], "...")
	}
	if len(missing) == 1 {
		return fmt.Errorf("expected %s", missing[0])
	}
	return fmt.Errorf("expected one of %s", strings.Join(missing, ", "))
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
		arg := values[positional]
		// TODO(aat): Fix hardcoding of these env checks all over the place :\
		if len(arg.Tag.Envs) != 0 {
			if atLeastOneEnvSet(arg.Tag.Envs) {
				continue
			}
		}
		missing = append(missing, "<"+arg.Name+">")
	}
	if len(missing) == 0 {
		return nil
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
		return errors.New("enum can only be applied to a slice or value")

	case reflect.Ptr:
		if target.IsNil() {
			return nil
		}
		return checkEnum(value, target.Elem())
	default:
		enumSlice := value.EnumSlice()
		v := fmt.Sprintf("%v", target)
		enums := []string{}
		for _, enum := range enumSlice {
			if enum == v {
				return nil
			}
			enums = append(enums, fmt.Sprintf("%q", enum))
		}
		return fmt.Errorf("%s must be one of %s but got %q", value.ShortSummary(), strings.Join(enums, ","), fmt.Sprintf("%v", target.Interface()))
	}
}

func checkPassthroughArg(target reflect.Value) bool {
	typ := target.Type()
	switch typ.Kind() {
	case reflect.Slice:
		return typ.Elem().Kind() == reflect.String
	default:
		return false
	}
}

func checkXorDuplicatedAndAndMissing(paths []*Path) error {
	errs := []string{}
	if err := checkXorDuplicates(paths); err != nil {
		errs = append(errs, err.Error())
	}
	if err := checkAndMissing(paths); err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, ", "))
	}
	return nil
}

func checkXorDuplicates(paths []*Path) error {
	for _, path := range paths {
		seen := map[string]*Flag{}
		for _, flag := range path.Flags {
			if !flag.Set {
				continue
			}
			for _, xor := range flag.Xor {
				if seen[xor] != nil {
					return fmt.Errorf("--%s and --%s can't be used together", seen[xor].Name, flag.Name)
				}
				seen[xor] = flag
			}
		}
	}
	return nil
}

func checkAndMissing(paths []*Path) error {
	for _, path := range paths {
		missingMsgs := []string{}
		andGroups := map[string][]*Flag{}
		for _, flag := range path.Flags {
			for _, and := range flag.And {
				andGroups[and] = append(andGroups[and], flag)
			}
		}
		for _, flags := range andGroups {
			oneSet := false
			notSet := []*Flag{}
			flagNames := []string{}
			for _, flag := range flags {
				flagNames = append(flagNames, flag.Name)
				if flag.Set {
					oneSet = true
				} else {
					notSet = append(notSet, flag)
				}
			}
			if len(notSet) > 0 && oneSet {
				missingMsgs = append(missingMsgs, fmt.Sprintf("--%s must be used together", strings.Join(flagNames, " and --")))
			}
		}
		if len(missingMsgs) > 0 {
			return fmt.Errorf("%s", strings.Join(missingMsgs, ", "))
		}
	}
	return nil
}

func findPotentialCandidates(needle string, haystack []string, format string, args ...any) error {
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

type validatable interface{ Validate() error }
type extendedValidatable interface {
	Validate(kctx *Context) error
}

// Proxy a validatable function to the extendedValidatable interface
type validatableFunc func() error

func (f validatableFunc) Validate(kctx *Context) error { return f() }

func isValidatable(v reflect.Value) extendedValidatable {
	if !v.IsValid() || (v.Kind() == reflect.Ptr || v.Kind() == reflect.Slice || v.Kind() == reflect.Map) && v.IsNil() {
		return nil
	}
	if validate, ok := v.Interface().(validatable); ok {
		return validatableFunc(validate.Validate)
	}
	if validate, ok := v.Interface().(extendedValidatable); ok {
		return validate
	}
	if v.CanAddr() {
		return isValidatable(v.Addr())
	}
	return nil
}

// getValidators returns validators implemented by v and by any embedded fields,
// matching how hooks are discovered (see getMethods).
func getValidators(v reflect.Value) (validators []extendedValidatable) {
	walkEmbedded(v, func(v reflect.Value) {
		if validate := isValidatable(v); validate != nil {
			validators = append(validators, validate)
		}
	})
	return
}

func atLeastOneEnvSet(envs []string) bool {
	for _, env := range envs {
		if _, ok := os.LookupEnv(env); ok {
			return true
		}
	}
	return false
}
