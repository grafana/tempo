package cli

import (
	"strings"

	"github.com/spf13/pflag"
)

// Flags returns the `*pflag.FlagSet` of this command
func (c *Command) Flags() *pflag.FlagSet {
	if c.flags == nil {
		c.flags = pflag.NewFlagSet(c.Name(), pflag.ContinueOnError)
	}
	return c.flags
}

// stripFlags removes flags from the argument line, leaving only subcommands and
// positional arguments.
func stripFlags(args []string, c *Command) []string {
	if len(args) == 0 {
		return args
	}

	commands := []string{}
	flags := c.Flags()

Loop:
	for len(args) > 0 {
		s := args[0]
		args = args[1:]
		switch {
		case s == "--":
			// "--" terminates the flags
			break Loop
		case strings.HasPrefix(s, "--") && !strings.Contains(s, "=") && !hasNoOptDefVal(s[2:], flags):
			// If '--flag arg' then
			// delete arg from args.
			fallthrough // (do the same as below)
		case strings.HasPrefix(s, "-") && !strings.Contains(s, "=") && len(s) == 2 && !shortHasNoOptDefVal(s[1:], flags):
			// If '-f arg' then
			// delete 'arg' from args or break the loop if len(args) <= 1.
			if len(args) <= 1 {
				break Loop
			} else {
				args = args[1:]
				continue
			}
		case s != "" && !strings.HasPrefix(s, "-"):
			commands = append(commands, s)
		}
	}

	return commands
}

func hasNoOptDefVal(name string, fs *pflag.FlagSet) bool {
	flag := fs.Lookup(name)
	if flag == nil {
		return false
	}
	return flag.NoOptDefVal != ""
}

func shortHasNoOptDefVal(name string, fs *pflag.FlagSet) bool {
	if len(name) == 0 {
		return false
	}

	flag := fs.ShorthandLookup(name[:1])
	if flag == nil {
		return false
	}
	return flag.NoOptDefVal != ""
}
