// Command assign_codeowner assigns unassigned pull requests using assigned-PR load.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func run(ctx context.Context, command string) error {
	if command != "sweep" && command != "report-failure" {
		return fmt.Errorf("unknown command %q; expected sweep or report-failure", command)
	}
	token, err := requiredEnv("GH_TOKEN")
	if err != nil {
		return err
	}
	client, err := newGitHubClient(token)
	if err != nil {
		return err
	}
	if command == "report-failure" {
		// The reporter needs only REPO (and RUN_URL). It must not inherit the
		// sweep's configuration validation, or a sweep config failure would
		// also kill the failure report that exists to surface it.
		repo, err := repoFromEnv()
		if err != nil {
			return err
		}
		runURL, err := requiredEnv("RUN_URL")
		if err != nil {
			return err
		}
		return reportFailure(ctx, client, repo, runURL)
	}
	cfg, err := configFromEnv()
	if err != nil {
		return err
	}
	return sweep(ctx, client, cfg)
}

func workflowError(err error) {
	message := strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(err.Error())
	fmt.Printf("::error::%s\n", message)
}

func main() {
	command := "sweep"
	if len(os.Args) > 2 {
		workflowError(errors.New("usage: assign_codeowner [sweep|report-failure]"))
		os.Exit(2)
	}
	if len(os.Args) == 2 {
		command = os.Args[1]
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, command); err != nil {
		workflowError(err)
		os.Exit(1)
	}
}
