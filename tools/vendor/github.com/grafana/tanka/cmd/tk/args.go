package main

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/go-clix/cli"
	"github.com/posener/complete"

	"github.com/grafana/tanka/pkg/jsonnet/jpath"
	"github.com/grafana/tanka/pkg/tanka"
)

var workflowArgs = cli.Args{
	Validator: cli.ValidateExact(1),
	Predictor: cli.PredictFunc(func(args complete.Args) []string {
		pwd, err := os.Getwd()
		if err != nil {
			return nil
		}

		root, err := jpath.FindRoot(pwd)
		if err != nil {
			return nil
		}

		envs, err := tanka.FindEnvs(pwd, tanka.FindOpts{})
		if err != nil && !errors.As(err, &tanka.ErrParallel{}) {
			return nil
		}

		var reldirs []string
		for _, env := range envs {
			path := filepath.Join(root, env.Metadata.Namespace) // namespace == path on disk
			reldir, err := filepath.Rel(pwd, path)
			if err == nil {
				reldirs = append(reldirs, reldir)
			}
		}

		if len(reldirs) != 0 {
			return reldirs
		}

		return complete.PredictFiles("*").Predict(args)
	}),
}
