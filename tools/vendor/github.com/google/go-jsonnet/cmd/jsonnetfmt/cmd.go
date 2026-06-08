/*
Copyright 2019 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/cmd/internal/cmd"
	"github.com/google/go-jsonnet/internal/formatter"
)

func version(o io.Writer) {
	fmt.Fprintf(o, "Jsonnet reformatter %s\n", jsonnet.Version())
}

func usage(o io.Writer) {
	version(o)
	fmt.Fprintln(o)
	fmt.Fprintln(o, "jsonnetfmt {<option>} { <filename> }")
	fmt.Fprintln(o)
	fmt.Fprintln(o, "Available options:")
	fmt.Fprintln(o, "  -h / --help                This message")
	fmt.Fprintln(o, "  -e / --exec                Treat filename as code")
	fmt.Fprintln(o, "  -o / --output-file <file>  Write to the output file rather than stdout")
	fmt.Fprintln(o, "  -c / --create-output-dirs  Automatically creates all parent directories for")
	fmt.Fprintln(o, "                             files")
	fmt.Fprintln(o, "  -i / --in-place            Update the Jsonnet file(s) in place")
	fmt.Fprintln(o, "  --test                     Exit with failure if reformatting changed the")
	fmt.Fprintln(o, "                             file(s)")
	fmt.Fprintln(o, "  -n / --indent <n>          Number of spaces to indent by")
	fmt.Fprintln(o, "                             (default 2, 0 means no change)")
	fmt.Fprintln(o, "  --max-blank-lines <n>      Max vertical spacing (default 2, 0 means no change)")
	fmt.Fprintln(o, "  --string-style <d|s|l>     Enforce double, single (default) quotes or 'leave'")
	fmt.Fprintln(o, "  --comment-style <h|s|l>    # (h), // (s) (default), or 'leave'; never changes")
	fmt.Fprintln(o, "                             she-bang")
	fmt.Fprintln(o, "  --[no-]pretty-field-names  Use syntax sugar for fields and indexing")
	fmt.Fprintln(o, "                             (on by default)")
	fmt.Fprintln(o, "  --[no-]pad-arrays          [ 1, 2, 3 ] instead of [1, 2, 3]")
	fmt.Fprintln(o, "  --[no-]pad-objects         { x: 1, y: 2 } instead of {x: 1, y: 2}")
	fmt.Fprintln(o, "                             (on by default)")
	fmt.Fprintln(o, "  --[no-]sort-imports        Sorting of imports (on by default)")
	fmt.Fprintln(o, "  --[no-]use-implicit-plus   Remove plus signs where they are not required")
	fmt.Fprintln(o, "                             (on by default)")
	fmt.Fprintln(o, "  --version                  Print version")
	fmt.Fprintln(o)
	fmt.Fprintln(o, "In all cases:")
	fmt.Fprintln(o, "  <filename> can be - (stdin)")
	fmt.Fprintln(o, "  Multichar options are expanded e.g. -abc becomes -a -b -c.")
	fmt.Fprintln(o, "  The -- option suppresses option processing for subsequent arguments.")
	fmt.Fprintln(o, "  Note that since filenames and jsonnet programs can begin with -, it is")
	fmt.Fprintln(o, "  advised to use -- if the argument is unknown, e.g. jsonnetfmt -- \"$FILENAME\".")
}

type config struct {
	outputFile           string
	inputFiles           []string
	evalCreateOutputDirs bool
	filenameIsCode       bool
	inPlace              bool
	test                 bool
	options              formatter.Options
}

func makeConfig() config {
	return config{
		options: formatter.DefaultOptions(),
	}
}

type processArgsStatus int

const (
	processArgsStatusContinue     = iota
	processArgsStatusSuccessUsage = iota
	processArgsStatusFailureUsage = iota
	processArgsStatusSuccess      = iota
	processArgsStatusFailure      = iota
)

func processArgs(givenArgs []string, config *config, vm *jsonnet.VM) (processArgsStatus, error) {
	args := cmd.SimplifyArgs(givenArgs)
	remainingArgs := make([]string, 0, len(args))
	i := 0

	for ; i < len(args); i++ {
		arg := args[i]
		if arg == "-h" || arg == "--help" {
			return processArgsStatusSuccessUsage, nil
		} else if arg == "-v" || arg == "--version" {
			version(os.Stdout)
			return processArgsStatusSuccess, nil
		} else if arg == "-e" || arg == "--exec" {
			config.filenameIsCode = true
		} else if arg == "-o" || arg == "--output-file" {
			outputFile := cmd.NextArg(&i, args)
			if len(outputFile) == 0 {
				return processArgsStatusFailure, fmt.Errorf("-o argument was empty string")
			}
			config.outputFile = outputFile
		} else if arg == "--" {
			// All subsequent args are not options.
			i++
			for ; i < len(args); i++ {
				remainingArgs = append(remainingArgs, args[i])
			}
			break
		} else if arg == "-i" || arg == "--in-place" {
			config.inPlace = true
		} else if arg == "--test" {
			config.test = true
		} else if arg == "-n" || arg == "--indent" {
			n := cmd.SafeStrToInt(cmd.NextArg(&i, args))
			if n < 0 {
				return processArgsStatusFailure, fmt.Errorf("invalid --indent value: %d", n)
			}
			config.options.Indent = n
		} else if arg == "--max-blank-lines" {
			n := cmd.SafeStrToInt(cmd.NextArg(&i, args))
			if n < 0 {
				return processArgsStatusFailure, fmt.Errorf("invalid --max-blank-lines value: %d", n)
			}
			config.options.MaxBlankLines = n
		} else if arg == "--string-style" {
			str := cmd.NextArg(&i, args)
			switch str {
			case "d":
				config.options.StringStyle = formatter.StringStyleDouble
			case "s":
				config.options.StringStyle = formatter.StringStyleSingle
			case "l":
				config.options.StringStyle = formatter.StringStyleLeave
			default:
				return processArgsStatusFailure, fmt.Errorf("invalid --string-style value: %s", str)
			}
		} else if arg == "--comment-style" {
			str := cmd.NextArg(&i, args)
			switch str {
			case "h":
				config.options.CommentStyle = formatter.CommentStyleHash
			case "s":
				config.options.CommentStyle = formatter.CommentStyleSlash
			case "l":
				config.options.CommentStyle = formatter.CommentStyleLeave
			default:
				return processArgsStatusFailure, fmt.Errorf("invalid --comment-style value: %s", str)
			}
		} else if arg == "--use-implicit-plus" {
			config.options.UseImplicitPlus = true
		} else if arg == "--no-use-implicit-plus" {
			config.options.UseImplicitPlus = false
		} else if arg == "--pretty-field-names" {
			config.options.PrettyFieldNames = true
		} else if arg == "--no-pretty-field-names" {
			config.options.PrettyFieldNames = false
		} else if arg == "--pad-arrays" {
			config.options.PadArrays = true
		} else if arg == "--no-pad-arrays" {
			config.options.PadArrays = false
		} else if arg == "--pad-objects" {
			config.options.PadObjects = true
		} else if arg == "--no-pad-objects" {
			config.options.PadObjects = false
		} else if arg == "--sort-imports" {
			config.options.SortImports = true
		} else if arg == "--no-sort-imports" {
			config.options.SortImports = false
		} else if arg == "-c" || arg == "--create-output-dirs" {
			config.evalCreateOutputDirs = true
		} else if len(arg) > 1 && arg[0] == '-' {
			return processArgsStatusFailure, fmt.Errorf("unrecognized argument: %s", arg)
		} else {
			remainingArgs = append(remainingArgs, arg)
		}
	}

	want := "filename"
	if config.filenameIsCode {
		want = "code"
	}
	if len(remainingArgs) == 0 {
		return processArgsStatusFailureUsage, fmt.Errorf("must give %s", want)
	}

	if !config.test && !config.inPlace {
		if len(remainingArgs) > 1 {
			return processArgsStatusFailure, fmt.Errorf("only one %s is allowed", want)
		}
	}

	config.inputFiles = remainingArgs
	return processArgsStatusContinue, nil
}

func main() {
	cmd.StartCPUProfile()
	defer cmd.StopCPUProfile()

	vm := jsonnet.MakeVM()
	vm.ErrorFormatter.SetColorFormatter(color.New(color.FgRed).Fprintf)

	config := makeConfig()

	status, err := processArgs(os.Args[1:], &config, vm)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: "+err.Error())
	}
	switch status {
	case processArgsStatusContinue:
		break
	case processArgsStatusSuccessUsage:
		usage(os.Stdout)
		os.Exit(0)
	case processArgsStatusFailureUsage:
		if err != nil {
			fmt.Fprintln(os.Stderr, "")
		}
		usage(os.Stderr)
		os.Exit(1)
	case processArgsStatusSuccess:
		os.Exit(0)
	case processArgsStatusFailure:
		os.Exit(1)
	}

	if config.inPlace || config.test {
		if len(config.inputFiles) == 0 {
			// Should already have been caught by processArgs.
			panic("Internal error: expected at least one input file.")
		}
		for _, inputFile := range config.inputFiles {
			outputFile := inputFile
			if config.inPlace {
				if inputFile == "-" {
					fmt.Fprintf(os.Stderr, "ERROR: cannot use --in-place with stdin\n")
					os.Exit(1)
				}
				if config.filenameIsCode {
					fmt.Fprintf(os.Stderr, "ERROR: cannot use --in-place with --exec\n")
					os.Exit(1)
				}
			}
			input := cmd.SafeReadInput(config.filenameIsCode, &inputFile)
			output, err := formatter.Format(inputFile, input, config.options)
			cmd.MemProfile()
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}

			if output != input {
				if config.inPlace {
					err := cmd.WriteOutputFile(output, outputFile, false)
					if err != nil {
						fmt.Fprintln(os.Stderr, err.Error())
						os.Exit(1)
					}
				} else {
					os.Exit(2)
				}
			}
		}

	} else {
		if len(config.inputFiles) != 1 {
			// Should already have been caught by processArgs.
			panic("Internal error: expected a single input file.")
		}
		inputFile := config.inputFiles[0]
		input := cmd.SafeReadInput(config.filenameIsCode, &inputFile)
		output, err := formatter.Format(inputFile, input, config.options)
		cmd.MemProfile()
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

		err = cmd.WriteOutputFile(output, config.outputFile, true)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}

}
