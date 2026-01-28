package main

import (
	"encoding/json"

	"github.com/go-clix/cli"

	"github.com/grafana/tanka/pkg/tanka"
)

func evalCmd() *cli.Command {
	cmd := &cli.Command{
		Short: "evaluate the jsonnet to json",
		Use:   "eval <path>",
		Args:  workflowArgs,
	}

	var jsonnetImplementation string
	evalPattern := cmd.Flags().StringP("eval", "e", "", "Evaluate expression on output of jsonnet")
	jsonnetImplementationFlag(cmd.Flags(), &jsonnetImplementation)

	getJsonnetOpts := jsonnetFlags(cmd.Flags())

	cmd.Run = func(_ *cli.Command, args []string) error {
		jsonnetOpts := tanka.Opts{
			JsonnetImplementation: jsonnetImplementation,
			JsonnetOpts:           getJsonnetOpts(),
		}
		if *evalPattern != "" {
			jsonnetOpts.EvalScript = tanka.PatternEvalScript(*evalPattern)
		}
		raw, err := tanka.Eval(args[0], jsonnetOpts)

		if raw == nil && err != nil {
			return err
		}

		out, err := json.MarshalIndent(raw, "", "  ")
		if err != nil {
			return err
		}

		return pageln(string(out))
	}

	return cmd
}
