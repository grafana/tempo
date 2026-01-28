# `go-clix`

Cli-X is a command line library for Go, inspired by
[`spf13/cobra`](https://github.com/spf13/cobra).

- :package: **`struct` based API**: Similar to `cobra`, `go-clix/cli` features a `struct` based
  API for easy composition and discovery of available options.
- :children_crossing: [**Subcommands**](#subcommands): `cli.Command` can be nested for a `git`
  like experience.
- :pushpin: [**Flags**](#flags): Every command has it's own set of flags. POSIX compliant
  using `spf13/pflag`.
- :busts_in_silhouette: [**Aliases**](#aliases): Commands can have multiple names, so
  breaking changes can easily be mitigated.
- :dart: **Go based completion**: `<TAB>` completion is supported for `bash`, `zsh` and
  `fish`. But you can generate suggestions using [**Go code**](#completion)!

## Getting started

Add the library to your project:

```bash
$ go get github.com/go-clix/cli
```

Then set up your root command:

```go
package main

import (
    "fmt"
    "github.com/go-clix/cli"
)

func main() {
    // create the root command
    rootCmd := cli.Command{
        Use: "greet",
        Short: "print a message",
        Run: func(cmd *cli.Command, args []string) error {
            fmt.Println("Hello from Cli-X!")
        },
    }

    // run and check for errors
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
```

## Subcommands

Every command may have children:

```go
// use a func to return a Command instead of
// a global variable and `init()`
func applyCmd() *cli.Command {
    cmd := &cli.Command{
        Use: "apply",
        Short: "apply the changes"
    }

    cmd.Run = func(cmd *cli.Command, args []string) error {
        fmt.Println("applied", args[0])
    }
    
    return cmd
}

func main() {
    rootCmd := &cli.Comand{
        Use: "kubectl",
        Short: "Kubernetes management tool",
    }

    // add the child command
    rootCmd.AddChildren(
       applyCmd(),
    )

    // run and check for errors
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
```

> **Note:** Do not use the `init()` function for setting up your commands.
> Create constructors as shown above!

## Flags

A `pflag.FlagSet` can be accessed per command using `*Command.Flags()`:

```go
func applyCmd() *cli.Command {
    cmd := &cli.Command{
        Use: "apply",
        Short: "apply the changes"
    }

    force := cmd.Flags().BoolP("force", "f", false, "skip checks")

    cmd.Run = func(cmd *cli.Command, args []string) error {
        fmt.Println("applied", args[0])
        if *force {
            fmt.Println("The force was with us.")
        }
    }
    return cmd
}
```

## Aliases

To make the `apply` subcommand also available as `make` and `do`:

```go
func applyCmd() *cli.Command {
    cmd := &cli.Command{
        Use: "apply",
        Aliases: []string{"make", "do"},
        Short: "apply the changes"
    }
}
```

Keep in mind that in `--help` outputs the command will still be called `apply`.

## Completion

Cli-X has a very powerful cli completion system, based on
[`posener/complete`](https://github.com/posener/complete).

It is powerful, because suggestions come directly from the Go application, not
from a `bash` script or similar.

Command and Flag names are automatically suggested, custom suggestions can be
implemented for [positional arguments](#positional-arguments) and [flag
values](#flag-values):

### Flag Values

Custom suggestions for flag values can be set up using `Command.Predictors`.

To do so, you need to add a `complete.Predictor` for that flag. Cli-X has a
number of predefined ones:

- `PredictAny()`: Predicts available files and directories, like `bash` does when
  having no better idea
- `PredictNone()`: Predicts literally nothing. No suggestions will be shown.
- `PredictSet(...string)`: Suggests from a predefined slice of options

If that's not sufficient, use `PredictFunc(func(complete.Args) []string)` to
create your own.

```go
import (
    "github.com/posener/complete"
    "github.com/go-clix/cli"
)

func logsCmd() *cli.Command {
    cmd := &cli.Command{
        Use: "logs",
        Short: "show logs of a pod",
        Predictors: map[string]complete.Predictor{
            "output": cli.PredictSet("lines", "json", "logfmt"),
        },
    }

    output := cmd.Flags().StringP("output", "o", "lines", "one of [lines, json, logfmt]")
}
```

### Positional Arguments

For positional arguments, Cli-X uses a slightly different `interface`, to allow
validation as well:

```go
interface {
    Validate(args []string) error
    Predict(Args) []string
}
```

Predefined options are also available:

- `ArgsAny()`: accepts any number args, predicts files and directories
- `ArgsExact(n)`: accepts _exactly_ `n` arguments. Predicts files and directories.
- `ArgsRange(n, m)`: accepts between `n` and `m` arguments (inclusive). Predicts files and directories.
- `ArgsNone()`: accepts _no_ args and predicts nothing.
- `ArgsSet(...string)`: Accepts _one_ argument, which MUST be included in the
  given set. Predicts the values from the set.

```go
import (
    "github.com/posener/complete"
    "github.com/go-clix/cli"
)

func applyCmd() *cli.Command {
    cmd := &cli.Command{
        Use: "logs",
        Short: "show logs of a pod",
        // we expect one argument which can be anything
        Args: cli.ArgsExact(1),
    }
}
```
