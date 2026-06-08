package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/fatih/structs"

	"github.com/go-clix/cli"

	"github.com/grafana/tanka/pkg/tanka"
)

func statusCmd(ctx context.Context) *cli.Command {
	cmd := &cli.Command{
		Use:   "status <path>",
		Short: "display an overview of the environment, including contents and metadata.",
		Args:  generateWorkflowArgs(ctx),
	}

	vars := workflowFlags(cmd.Flags())
	getJsonnetOpts := jsonnetFlags(cmd.Flags())

	cmd.Run = func(_ *cli.Command, args []string) error {
		status, err := tanka.Status(ctx, args[0], tanka.Opts{
			JsonnetImplementation: vars.jsonnetImplementation,
			JsonnetOpts:           getJsonnetOpts(),
			Name:                  vars.name,
		})
		if err != nil {
			return err
		}

		context := status.Client.Kubeconfig.Context
		fmt.Println("Context:", context.Name)
		fmt.Println("Cluster:", context.Context.Cluster)
		fmt.Println("Environment:")

		specMap := structs.Map(status.Env.Spec)
		var keys []string
		for k := range specMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := specMap[k]
			fmt.Printf("  %s: %v\n", k, v)
		}

		fmt.Println("Resources:")
		f := "  %s\t%s/%s\n"
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
		fmt.Fprintln(w, "  NAMESPACE\tOBJECTSPEC")
		for _, r := range status.Resources {
			fmt.Fprintf(w, f, r.Metadata().Namespace(), r.Kind(), r.Metadata().Name())
		}
		w.Flush()

		return nil
	}
	return cmd
}
