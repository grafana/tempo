package cli

import (
	"fmt"

	"github.com/posener/complete"
	"github.com/posener/complete/cmd/install"
	"github.com/spf13/pflag"
)

// predict attempts to send <TAB> suggestions to bash.
// Returns false if not running in a completion context
func predict(c *Command) bool {

	cmp := complete.New(c.Name(), createCmp(c))
	return cmp.Complete()
}

// createCmd returns the structure of the Command as a `complete.Command` for
// the posener/complete library, including subcommands and flags.
func createCmp(c *Command) complete.Command {
	rootCmp := complete.Command{}

	rootCmp.Flags = complete.Flags{
		"-h":     complete.PredictNothing,
		"--help": complete.PredictNothing,
	}

	c.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden {
			return
		}

		p, ok := c.Predictors[flag.Name]
		if !ok {
			p = complete.PredictNothing
		}

		if len(flag.Shorthand) > 0 {
			rootCmp.Flags["-"+flag.Shorthand] = p
		}
		rootCmp.Flags["--"+flag.Name] = p
	})

	if c.children != nil {
		rootCmp.Sub = make(complete.Commands)
		for _, child := range c.children {
			rootCmp.Sub[child.Name()] = createCmp(child)
		}
	}

	// Positional Arguments, default to ArgsAny for all leaf commands
	if c.Args == nil && len(c.children) == 0 {
		c.Args = ArgsAny()
	}
	rootCmp.Args = c.Args

	return rootCmp
}

// completionCmd returns a command that installs native completions into the
// users shell.
func completionCmd(name string) *Command {
	cmd := &Command{
		Use:   "complete",
		Short: "install CLI completions",
		Long: fmt.Sprintf(`Registers the %s binary as its own completion handler.
This allows for richer <TAB> suggestions, because the actual application logic can be used to compute them.
Installation is done by injecting a line into your shells startup script (.bashrc, .zshrc, .fishrc)`,
			name),
		Args: ArgsNone(),
	}

	del := cmd.Flags().Bool("remove", false, "uninstall completions")

	cmd.Run = func(cmd *Command, args []string) error {
		if *del {
			return install.Uninstall(name)
		}
		return install.Install(name)
	}

	return cmd
}
