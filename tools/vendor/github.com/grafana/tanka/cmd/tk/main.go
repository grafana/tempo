package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-clix/cli"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"golang.org/x/term"

	"github.com/grafana/tanka/internal/telemetry"
	"github.com/grafana/tanka/pkg/tanka"
)

var interactive = term.IsTerminal(int(os.Stdout.Fd()))

var tracer = telemetry.Tracer("tanka")

func main() {
	rootCmd := &cli.Command{
		Use:     "tk",
		Short:   "tanka <3 jsonnet",
		Version: tanka.CurrentVersion,
	}

	ctx := context.Background()
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("tanka"),
	)
	shutdownOtel, err := telemetry.Setup(ctx, res)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, telemetry.LoadEnvironmentCarrier())

	// set default logging level early; not all commands parse --log-level
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// workflow commands
	addCommandsWithLogLevelOption(
		rootCmd,
		applyCmd(ctx),
		showCmd(ctx),
		diffCmd(ctx),
		pruneCmd(ctx),
		deleteCmd(ctx),
	)

	addCommandsWithLogLevelOption(
		rootCmd,
		envCmd(ctx),
		statusCmd(ctx),
		exportCmd(ctx),
	)

	// jsonnet commands
	addCommandsWithLogLevelOption(
		rootCmd,
		fmtCmd(ctx),
		lintCmd(ctx),
		evalCmd(ctx),
		initCmd(ctx),
		toolCmd(ctx),
	)

	// external commands prefixed with "tk-"
	addCommandsWithLogLevelOption(
		rootCmd,
		prefixCommands("tk-")...,
	)

	// Run!
	if err := rootCmd.Execute(); err != nil {
		if err := shutdownOtel(context.Background()); err != nil {
			fmt.Fprintln(os.Stderr, "OTEL shutdown error:", err)
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	if err := shutdownOtel(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "OTEL shutdown error:", err)
	}
}

func addCommandsWithLogLevelOption(rootCmd *cli.Command, cmds ...*cli.Command) {
	for _, cmd := range cmds {
		levels := []string{zerolog.Disabled.String(), zerolog.FatalLevel.String(), zerolog.ErrorLevel.String(), zerolog.WarnLevel.String(), zerolog.InfoLevel.String(), zerolog.DebugLevel.String(), zerolog.TraceLevel.String()}
		cmd.Flags().String("log-level", zerolog.InfoLevel.String(), "possible values: "+strings.Join(levels, ", "))

		cmdRun := cmd.Run
		cmd.Run = func(cmd *cli.Command, args []string) error {
			level, err := zerolog.ParseLevel(cmd.Flags().Lookup("log-level").Value.String())
			if err != nil {
				return err
			}
			zerolog.SetGlobalLevel(level)

			return cmdRun(cmd, args)
		}
		rootCmd.AddCommand(cmd)
	}
}
