package main

import (
	"os"

	"github.com/mattn/go-colorable"
)

func main() {
	cli := &CLI{
		outStream: colorable.NewColorableStdout(),
		errStream: colorable.NewColorableStderr(),
	}
	os.Exit(cli.Run(os.Args))
}
